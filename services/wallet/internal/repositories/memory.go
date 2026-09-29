package repositories

import (
	"cmp"
	"context"
	"slices"
	"sync"
	"sync/atomic"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/wallet/internal/models"
)

const shardsCount = 256

type MemoryRepository struct {
	shards [shardsCount]shard

	users   atomic.Int64
	wallets atomic.Int64
}

type shard struct {
	mu    sync.RWMutex
	users map[int64]models.User
}

func NewMemoryRepository() *MemoryRepository {
	r := &MemoryRepository{}

	for i := range r.shards {
		r.shards[i].users = make(map[int64]models.User)
	}

	return r
}

func (r *MemoryRepository) Get(_ context.Context, id int64) (models.User, error) {
	sh := r.shardOf(id)

	sh.mu.RLock()
	defer sh.mu.RUnlock()

	user, ok := sh.users[id]
	if !ok {
		return models.User{}, models.ErrUserNotFound
	}

	return user.Clone(), nil
}

func (r *MemoryRepository) Update(_ context.Context, id int64, fn func(*models.User) error) error {
	sh := r.shardOf(id)

	sh.mu.Lock()
	defer sh.mu.Unlock()

	user := models.User{ID: id}

	if current, ok := sh.users[id]; ok {
		user = current.Clone()
	}

	if err := fn(&user); err != nil {
		return err
	}

	r.putLocked(sh, user)

	return nil
}

func (r *MemoryRepository) Save(_ context.Context, users []models.User) error {
	for _, user := range users {
		r.save(user.Clone())
	}

	return nil
}

func (r *MemoryRepository) DeleteMany(_ context.Context, ids []int64) (int, error) {
	deleted := 0

	for _, id := range ids {
		if r.deleteOne(id) {
			deleted++
		}
	}

	return deleted, nil
}

func (r *MemoryRepository) Snapshot(_ context.Context) ([]models.User, error) {
	users := make([]models.User, 0, r.users.Load())

	for i := range r.shards {
		users = r.shards[i].appendClones(users)
	}

	slices.SortFunc(users, func(a, b models.User) int {
		return cmp.Compare(a.ID, b.ID)
	})

	return users, nil
}

func (r *MemoryRepository) Stats(_ context.Context) (models.Stats, error) {
	return models.Stats{
		Users:   int(r.users.Load()),
		Wallets: int(r.wallets.Load()),
	}, nil
}

func (r *MemoryRepository) shardOf(id int64) *shard {
	return &r.shards[uint64(id)%shardsCount]
}

func (r *MemoryRepository) deleteOne(id int64) bool {
	sh := r.shardOf(id)

	sh.mu.Lock()
	defer sh.mu.Unlock()

	user, ok := sh.users[id]
	if !ok {
		return false
	}

	delete(sh.users, id)

	r.users.Add(-1)
	r.wallets.Add(-int64(len(user.Wallets)))

	return true
}

func (r *MemoryRepository) save(user models.User) {
	sh := r.shardOf(user.ID)

	sh.mu.Lock()
	defer sh.mu.Unlock()

	r.putLocked(sh, user)
}

func (r *MemoryRepository) putLocked(sh *shard, user models.User) {
	previous, existed := sh.users[user.ID]

	if !existed {
		r.users.Add(1)
	}

	r.wallets.Add(int64(len(user.Wallets) - len(previous.Wallets)))

	sh.users[user.ID] = user
}

func (s *shard) appendClones(users []models.User) []models.User {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, user := range s.users {
		users = append(users, user.Clone())
	}

	return users
}
