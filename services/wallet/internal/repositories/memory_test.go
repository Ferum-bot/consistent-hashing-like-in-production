package repositories

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/Ferum-bot/consistent-hashing-like-in-production/services/wallet/internal/models"
)

func TestMemoryRepository_GetMissingUser(t *testing.T) {
	repo := NewMemoryRepository()

	_, err := repo.Get(context.Background(), 1)
	if !errors.Is(err, models.ErrUserNotFound) {
		t.Fatalf("err = %v, want ErrUserNotFound", err)
	}
}

func TestMemoryRepository_ReturnedUserIsACopy(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()
	mustSave(t, repo, models.User{ID: 1, Wallets: []models.Wallet{{Currency: "USD", Balance: 100}}})

	user, _ := repo.Get(ctx, 1)
	user.Wallets[0].Balance = 0

	stored, _ := repo.Get(ctx, 1)
	if stored.Wallets[0].Balance != 100 {
		t.Fatalf("stored balance changed through a returned copy: %d", stored.Wallets[0].Balance)
	}
}

func TestMemoryRepository_FailedUpdateLeavesUserUntouched(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()
	mustSave(t, repo, models.User{ID: 1, Wallets: []models.Wallet{{Currency: "USD", Balance: 100}}})

	errBoom := errors.New("boom")
	err := repo.Update(ctx, 1, func(user *models.User) error {
		user.Wallets[0].Balance = 0
		return errBoom
	})
	if !errors.Is(err, errBoom) {
		t.Fatalf("err = %v, want %v", err, errBoom)
	}

	stored, _ := repo.Get(ctx, 1)
	if stored.Wallets[0].Balance != 100 {
		t.Fatalf("balance = %d, want 100", stored.Wallets[0].Balance)
	}
}

func TestMemoryRepository_ConcurrentUpdates(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()

	const writers, perWriter = 8, 1000
	var wg sync.WaitGroup
	for range writers {
		wg.Go(func() {
			for range perWriter {
				_ = repo.Update(ctx, 1, func(user *models.User) error {
					user.OpenWallet("USD").Balance++
					return nil
				})
			}
		})
	}
	wg.Wait()

	user, _ := repo.Get(ctx, 1)
	if got := user.Wallets[0].Balance; got != writers*perWriter {
		t.Fatalf("balance = %d, want %d", got, writers*perWriter)
	}
}

func TestMemoryRepository_SnapshotAndStats(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()
	mustSave(t, repo,
		models.User{ID: 3, Wallets: []models.Wallet{{Currency: "USD"}}},
		models.User{ID: 1, Wallets: []models.Wallet{{Currency: "USD"}, {Currency: "EUR"}}},
		models.User{ID: 2},
	)
	_, _ = repo.DeleteMany(ctx, []int64{2})

	users, _ := repo.Snapshot(ctx)
	if len(users) != 2 || users[0].ID != 1 || users[1].ID != 3 {
		t.Fatalf("snapshot = %+v, want users 1 and 3 sorted by id", users)
	}

	stats, _ := repo.Stats(ctx)
	if stats != (models.Stats{Users: 2, Wallets: 3}) {
		t.Fatalf("stats = %+v", stats)
	}
}

func TestMemoryRepository_StatsFollowEveryChange(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()

	mustSave(t, repo, models.User{ID: 1, Wallets: []models.Wallet{{Currency: "USD"}, {Currency: "EUR"}}})
	assertStats(t, repo, models.Stats{Users: 1, Wallets: 2})

	mustSave(t, repo, models.User{ID: 1, Wallets: []models.Wallet{{Currency: "USD"}}})
	assertStats(t, repo, models.Stats{Users: 1, Wallets: 1})

	_ = repo.Update(ctx, 2, func(user *models.User) error {
		user.OpenWallet("USD")
		user.OpenWallet("GBP")
		return nil
	})
	assertStats(t, repo, models.Stats{Users: 2, Wallets: 3})

	_ = repo.Update(ctx, 3, func(user *models.User) error {
		user.OpenWallet("USD")
		return errors.New("rejected")
	})
	assertStats(t, repo, models.Stats{Users: 2, Wallets: 3})

	_, _ = repo.DeleteMany(ctx, []int64{42})
	assertStats(t, repo, models.Stats{Users: 2, Wallets: 3})

	_, _ = repo.DeleteMany(ctx, []int64{2})
	assertStats(t, repo, models.Stats{Users: 1, Wallets: 1})
}

func TestMemoryRepository_StatsUnderConcurrentChanges(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()

	const workers, users = 8, 2000

	var wg sync.WaitGroup

	for w := range workers {
		wg.Go(func() {
			for id := int64(1); id <= users; id++ {
				switch (id + int64(w)) % 3 {
				case 0:
					_, _ = repo.DeleteMany(ctx, []int64{id})
				case 1:
					_ = repo.Update(ctx, id, func(user *models.User) error {
						user.OpenWallet("USD").Balance++
						return nil
					})
				default:
					_ = repo.Update(ctx, id, func(user *models.User) error {
						user.OpenWallet("EUR").Balance++
						return nil
					})
				}
			}
		})
	}

	wg.Wait()

	snapshot, _ := repo.Snapshot(ctx)
	want := models.Stats{Users: len(snapshot)}

	for _, user := range snapshot {
		want.Wallets += len(user.Wallets)
	}

	assertStats(t, repo, want)
}

func assertStats(t *testing.T, repo *MemoryRepository, want models.Stats) {
	t.Helper()

	got, _ := repo.Stats(context.Background())
	if got != want {
		t.Fatalf("stats = %+v, want %+v", got, want)
	}
}

func mustSave(t *testing.T, repo *MemoryRepository, users ...models.User) {
	t.Helper()
	if err := repo.Save(context.Background(), users); err != nil {
		t.Fatal(err)
	}
}

func TestMemoryRepository_DeleteMany(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()

	mustSave(t, repo,
		models.User{ID: 1, Wallets: []models.Wallet{{Currency: "USD"}}},
		models.User{ID: 2, Wallets: []models.Wallet{{Currency: "USD"}, {Currency: "EUR"}}},
		models.User{ID: 3},
	)

	deleted, err := repo.DeleteMany(ctx, []int64{1, 2, 42})
	if err != nil {
		t.Fatal(err)
	}

	if deleted != 2 {
		t.Fatalf("deleted = %d, want 2", deleted)
	}

	assertStats(t, repo, models.Stats{Users: 1, Wallets: 0})
}
