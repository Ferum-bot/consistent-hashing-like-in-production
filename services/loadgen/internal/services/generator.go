package services

import (
	"context"
	"math/rand/v2"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/wait"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/loadgen/internal/models"
)

const (
	schedulerTick = 2 * time.Millisecond
	maxBacklog    = time.Second
	idleTick      = 50 * time.Millisecond
)

type Generator struct {
	ledger   Ledger
	client   RouterClient
	observer Observer
	checker  *Checker

	routers    []string
	nextRouter atomic.Uint64
	workers    int
	jobs       chan models.Operation

	mu      sync.Mutex
	profile atomic.Pointer[models.Profile]
}

func NewGenerator(
	ledger Ledger,
	client RouterClient,
	observer Observer,
	checker *Checker,
	routers []string,
	workers int,
	profile models.Profile,
) (*Generator, error) {
	if err := validateProfile(profile); err != nil {
		return nil, err
	}

	g := &Generator{
		ledger:   ledger,
		client:   client,
		observer: observer,
		checker:  checker,
		routers:  routers,
		workers:  workers,
		jobs:     make(chan models.Operation, workers),
	}

	g.profile.Store(&profile)
	observer.ObserveProfile(profile)

	return g, nil
}

func (g *Generator) Profile() models.Profile {
	return *g.profile.Load()
}

func (g *Generator) UpdateProfile(change func(*models.Profile)) (models.Profile, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	next := *g.profile.Load()
	change(&next)

	if err := validateProfile(next); err != nil {
		return models.Profile{}, err
	}

	g.profile.Store(&next)
	g.observer.ObserveProfile(next)

	return next, nil
}

func (g *Generator) Run(ctx context.Context) {
	for range g.workers {
		go g.work(ctx)
	}

	go g.verify(ctx)

	g.schedule(ctx)
}

func (g *Generator) schedule(ctx context.Context) {
	random := rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
	ticker := time.NewTicker(schedulerTick)
	defer ticker.Stop()

	var (
		active  *models.Profile
		pick    *picker
		started time.Time
		sent    int64
	)

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			profile := g.profile.Load()
			users := g.ledger.Users()

			if !profile.Running || profile.RPS == 0 || users == 0 {
				active = nil
				continue
			}

			if profile != active || pick.users != users {
				active = profile
				pick = newPicker(random, *profile, users)
				started = now
				sent = 0
			}

			rps := float64(profile.RPS)
			due := int64(now.Sub(started).Seconds() * rps)

			if backlog := due - sent; backlog > int64(rps*maxBacklog.Seconds()) {
				for range backlog {
					g.observer.ObserveDropped()
				}

				sent = due
			}

			for ; sent < due; sent++ {
				scheduledAt := started.Add(time.Duration(float64(sent) / rps * float64(time.Second)))

				select {
				case g.jobs <- pick.next(g.ledger, scheduledAt):
				default:
					g.observer.ObserveDropped()
				}
			}
		}
	}
}

func (g *Generator) work(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case op := <-g.jobs:
			if g.profile.Load().Running {
				g.execute(ctx, op)
			}
		}
	}
}

func (g *Generator) execute(ctx context.Context, op models.Operation) {
	router := g.router()

	if op.Kind == models.OperationRead {
		g.read(ctx, router, op)
		return
	}

	g.write(ctx, router, op)
}

func (g *Generator) read(ctx context.Context, router string, op models.Operation) {
	before := g.ledger.Observe(op.UserID)

	response, err := g.client.Get(ctx, router, op.UserID)

	g.observer.ObserveRequest(op.Kind, router, response.Status, time.Since(op.ScheduledAt))

	if err != nil {
		return
	}

	g.checker.Check(before, response)
}

func (g *Generator) write(ctx context.Context, router string, op models.Operation) {
	ticket := g.ledger.BeginWrite(op.UserID)
	defer g.ledger.EndWrite(ticket)

	var (
		status int
		err    error
		delta  int64
	)

	if op.Kind == models.OperationDeposit {
		status, err = g.client.Deposit(ctx, router, op.UserID, op.Currency, op.Amount)
		delta = op.Amount
	} else {
		status, err = g.client.Withdraw(ctx, router, op.UserID, op.Currency, op.Amount)
		delta = -op.Amount
	}

	g.observer.ObserveRequest(op.Kind, router, status, time.Since(op.ScheduledAt))

	if status == http.StatusOK {
		g.ledger.Apply(ticket, op.Currency, delta)
		g.checker.Touch(op.UserID)

		return
	}

	if mayBeApplied(status, err) {
		g.ledger.MarkUncertain(ticket)
	}
}

func mayBeApplied(status int, err error) bool {
	if err != nil {
		return true
	}

	return status >= http.StatusInternalServerError && status != http.StatusServiceUnavailable
}

func (g *Generator) verify(ctx context.Context) {
	for {
		profile := g.profile.Load()

		if !profile.Running || profile.VerifyRPS == 0 {
			if !wait.For(ctx, idleTick) {
				return
			}

			continue
		}

		if userID, ok := g.checker.RecentlyTouched(); ok {
			g.read(ctx, g.router(), models.Operation{
				Kind:        models.OperationVerify,
				UserID:      userID,
				ScheduledAt: time.Now(),
			})
		}

		if !wait.For(ctx, time.Second/time.Duration(profile.VerifyRPS)) {
			return
		}
	}
}

func (g *Generator) router() string {
	return g.routers[g.nextRouter.Add(1)%uint64(len(g.routers))]
}
