package repository

import (
	"context"
	"testing"
)

// Stats агрегирует по всей БД, а не только по данным теста - сравниваем
// дельту до/после создания известных строк, а не абсолютные числа (в общей
// тестовой БД могут быть данные других тестов, выполняющихся до/после).
func TestAdminStatsRepo_Stats_ReflectsNewUser(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewAdminStatsRepo(pool)
	userRepo := NewUserRepo(pool)
	ctx := context.Background()

	before, err := repo.Stats(ctx)
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}

	user := createTestUser(t, userRepo, ctx)
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), "DELETE FROM users WHERE id = $1", user.ID) })

	after, err := repo.Stats(ctx)
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}

	if after.UsersTotal != before.UsersTotal+1 {
		t.Errorf("UsersTotal = %d, want %d", after.UsersTotal, before.UsersTotal+1)
	}
}

func TestAdminStatsRepo_Stats_ReflectsBannedUser(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewAdminStatsRepo(pool)
	userRepo := NewUserRepo(pool)
	ctx := context.Background()
	user := createTestUser(t, userRepo, ctx)
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), "DELETE FROM users WHERE id = $1", user.ID) })

	before, err := repo.Stats(ctx)
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}

	if _, err := userRepo.Ban(ctx, user.ID, user.ID, "test"); err != nil {
		t.Fatalf("Ban() error = %v", err)
	}

	after, err := repo.Stats(ctx)
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}
	if after.UsersBanned != before.UsersBanned+1 {
		t.Errorf("UsersBanned = %d, want %d", after.UsersBanned, before.UsersBanned+1)
	}
}
