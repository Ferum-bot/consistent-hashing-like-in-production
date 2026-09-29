package repositories

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"sync/atomic"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/libs/snapshot"
	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/loadgen/internal/models"
)

type book struct {
	epoch uint64

	offsets    []int32
	currencies []uint8
	balances   []int64
	codes      []string

	seq       []uint32
	inflight  []int32
	uncertain []uint32

	uncertainCount atomic.Int64
}

type LedgerRepository struct {
	current atomic.Pointer[book]
	epochs  atomic.Uint64
}

type snapshotUser struct {
	UserID  int64 `json:"user_id"`
	Wallets []struct {
		Currency string `json:"currency"`
		Balance  int64  `json:"balance"`
	} `json:"wallets"`
}

func NewLedgerRepository() *LedgerRepository {
	return &LedgerRepository{}
}

func (r *LedgerRepository) Load(ctx context.Context, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return models.NewErrReadSnapshot(err)
	}
	defer file.Close()

	scanner := snapshot.NewScanner(file)

	next := &book{epoch: r.epochs.Add(1), offsets: []int32{0}}
	codeIndex := map[string]uint8{}

	for line := 1; scanner.Scan(); line++ {
		if line%100_000 == 0 && ctx.Err() != nil {
			return ctx.Err()
		}

		var user snapshotUser
		if err := json.Unmarshal(scanner.Bytes(), &user); err != nil {
			return models.NewErrSnapshotLine(line, err)
		}

		if user.UserID != int64(line) {
			return models.NewErrSnapshotOrder(line, user.UserID)
		}

		for _, wallet := range user.Wallets {
			code, err := next.codeOf(codeIndex, wallet.Currency)
			if err != nil {
				return err
			}

			next.currencies = append(next.currencies, code)
			next.balances = append(next.balances, wallet.Balance)
		}

		next.offsets = append(next.offsets, int32(len(next.balances)))
	}

	if err := scanner.Err(); err != nil {
		return models.NewErrReadSnapshot(err)
	}

	users := len(next.offsets) - 1

	next.seq = make([]uint32, users+1)
	next.inflight = make([]int32, users+1)
	next.uncertain = make([]uint32, users+1)

	r.current.Store(next)

	return nil
}

func (r *LedgerRepository) Users() int {
	b := r.current.Load()
	if b == nil {
		return 0
	}

	return len(b.offsets) - 1
}

func (r *LedgerRepository) Wallets(userID int64) []models.Wallet {
	b, ok := r.currentBook(userID)
	if !ok {
		return nil
	}

	from, to := b.offsets[userID-1], b.offsets[userID]
	wallets := make([]models.Wallet, 0, to-from)

	for i := from; i < to; i++ {
		wallets = append(wallets, models.Wallet{
			Currency: b.codes[b.currencies[i]],
			Balance:  atomic.LoadInt64(&b.balances[i]),
		})
	}

	return wallets
}

func (r *LedgerRepository) BeginWrite(userID int64) models.WriteTicket {
	b, ok := r.currentBook(userID)
	if !ok {
		return models.WriteTicket{UserID: userID}
	}

	atomic.AddInt32(&b.inflight[userID], 1)
	atomic.AddUint32(&b.seq[userID], 1)

	return models.WriteTicket{UserID: userID, Epoch: b.epoch}
}

func (r *LedgerRepository) Apply(ticket models.WriteTicket, currency string, delta int64) {
	b, ok := r.bookOfTicket(ticket)
	if !ok {
		return
	}

	from, to := b.offsets[ticket.UserID-1], b.offsets[ticket.UserID]

	for i := from; i < to; i++ {
		if b.codes[b.currencies[i]] == currency {
			atomic.AddInt64(&b.balances[i], delta)
			return
		}
	}
}

func (r *LedgerRepository) MarkUncertain(ticket models.WriteTicket) {
	b, ok := r.bookOfTicket(ticket)
	if !ok {
		return
	}

	if atomic.CompareAndSwapUint32(&b.uncertain[ticket.UserID], 0, 1) {
		b.uncertainCount.Add(1)
	}
}

func (r *LedgerRepository) EndWrite(ticket models.WriteTicket) {
	b, ok := r.bookOfTicket(ticket)
	if !ok {
		return
	}

	atomic.AddInt32(&b.inflight[ticket.UserID], -1)
}

func (r *LedgerRepository) Observe(userID int64) models.Observation {
	b, ok := r.currentBook(userID)
	if !ok {
		return models.Observation{UserID: userID, Busy: true}
	}

	return models.Observation{
		UserID: userID,
		Epoch:  b.epoch,
		Seq:    atomic.LoadUint32(&b.seq[userID]),
		Busy:   atomic.LoadInt32(&b.inflight[userID]) > 0 || atomic.LoadUint32(&b.uncertain[userID]) == 1,
	}
}

func (r *LedgerRepository) Unchanged(before models.Observation) bool {
	after := r.Observe(before.UserID)

	return !after.Busy && after.Epoch == before.Epoch && after.Seq == before.Seq
}

func (r *LedgerRepository) Stats() models.LedgerStats {
	b := r.current.Load()
	if b == nil {
		return models.LedgerStats{}
	}

	stats := models.LedgerStats{
		Loaded:    true,
		Users:     len(b.offsets) - 1,
		Wallets:   len(b.balances),
		Uncertain: int(b.uncertainCount.Load()),
	}

	for i := range b.balances {
		stats.Money += atomic.LoadInt64(&b.balances[i])
	}

	return stats
}

func (r *LedgerRepository) UncertainUsers() int {
	b := r.current.Load()
	if b == nil {
		return 0
	}

	return int(b.uncertainCount.Load())
}

func (r *LedgerRepository) currentBook(userID int64) (*book, bool) {
	b := r.current.Load()
	if b == nil || userID < 1 || userID >= int64(len(b.offsets)) {
		return nil, false
	}

	return b, true
}

func (r *LedgerRepository) bookOfTicket(ticket models.WriteTicket) (*book, bool) {
	b, ok := r.currentBook(ticket.UserID)
	if !ok || b.epoch != ticket.Epoch {
		return nil, false
	}

	return b, true
}

func (b *book) codeOf(index map[string]uint8, currency string) (uint8, error) {
	if code, ok := index[currency]; ok {
		return code, nil
	}

	if len(b.codes) > math.MaxUint8 {
		return 0, models.NewErrTooManyCurrencies(currency)
	}

	code := uint8(len(b.codes))
	index[currency] = code
	b.codes = append(b.codes, currency)

	return code, nil
}
