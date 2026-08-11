package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/medflow/backend/internal/models"
)

func TestCardRatingRepo_Upsert_ChangesStars_DoesNotDuplicate(t *testing.T) {
	pool := setupTestDB(t)
	taskRepo := NewCardTaskRepo(pool)
	cardRepo := NewCardRepo(pool)
	ratingRepo := NewCardRatingRepo(pool)
	ctx := context.Background()
	user := createTestForumUser(t, pool)
	task := createTestCardTask(t, pool, taskRepo, user.ID)
	cards := createTestCards(t, pool, cardRepo, task.ID, 1)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM card_ratings WHERE user_id = $1", user.ID)
	})

	if err := ratingRepo.Upsert(ctx, user.ID, cards[0].ID, 3); err != nil {
		t.Fatalf("Upsert() [1st] error = %v", err)
	}
	if err := ratingRepo.Upsert(ctx, user.ID, cards[0].ID, 5); err != nil {
		t.Fatalf("Upsert() [2nd] error = %v", err)
	}

	my, err := ratingRepo.MyRatingsBatch(ctx, user.ID, []uuid.UUID{cards[0].ID})
	if err != nil {
		t.Fatalf("MyRatingsBatch() error = %v", err)
	}
	if my[cards[0].ID] != 5 {
		t.Errorf("MyRatingsBatch()[%v] = %d, want 5 (updated, not a second row)", cards[0].ID, my[cards[0].ID])
	}
}

func TestCardRatingRepo_Delete_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	ratingRepo := NewCardRatingRepo(pool)
	user := createTestForumUser(t, pool)

	err := ratingRepo.Delete(context.Background(), user.ID, uuid.New())
	if err != models.ErrCardRatingNotFound {
		t.Fatalf("Delete() error = %v, want ErrCardRatingNotFound", err)
	}
}

func TestCardRatingRepo_AggregateForCardsBatch(t *testing.T) {
	pool := setupTestDB(t)
	taskRepo := NewCardTaskRepo(pool)
	cardRepo := NewCardRepo(pool)
	ratingRepo := NewCardRatingRepo(pool)
	ctx := context.Background()
	author := createTestForumUser(t, pool)
	voter1 := createTestForumUser(t, pool)
	voter2 := createTestForumUser(t, pool)
	task := createTestCardTask(t, pool, taskRepo, author.ID)
	cards := createTestCards(t, pool, cardRepo, task.ID, 1)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM card_ratings WHERE card_id = $1", cards[0].ID)
	})

	if err := ratingRepo.Upsert(ctx, voter1.ID, cards[0].ID, 4); err != nil {
		t.Fatalf("Upsert() [voter1] error = %v", err)
	}
	if err := ratingRepo.Upsert(ctx, voter2.ID, cards[0].ID, 2); err != nil {
		t.Fatalf("Upsert() [voter2] error = %v", err)
	}

	agg, err := ratingRepo.AggregateForCardsBatch(ctx, []uuid.UUID{cards[0].ID})
	if err != nil {
		t.Fatalf("AggregateForCardsBatch() error = %v", err)
	}
	result := agg[cards[0].ID]
	if result.RatingsCount != 2 {
		t.Errorf("RatingsCount = %d, want 2", result.RatingsCount)
	}
	if result.AverageStars != 3 {
		t.Errorf("AverageStars = %v, want 3 ((4+2)/2)", result.AverageStars)
	}
}

func TestCardRatingRepo_ListRatedByUser(t *testing.T) {
	pool := setupTestDB(t)
	taskRepo := NewCardTaskRepo(pool)
	cardRepo := NewCardRepo(pool)
	ratingRepo := NewCardRatingRepo(pool)
	ctx := context.Background()
	user := createTestForumUser(t, pool)
	task := createTestCardTask(t, pool, taskRepo, user.ID)
	cards := createTestCards(t, pool, cardRepo, task.ID, 2)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM card_ratings WHERE user_id = $1", user.ID)
	})

	if err := ratingRepo.Upsert(ctx, user.ID, cards[0].ID, 5); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}

	list, total, err := ratingRepo.ListRatedByUser(ctx, user.ID, 1, 20)
	if err != nil {
		t.Fatalf("ListRatedByUser() error = %v", err)
	}
	if total != 1 || len(list) != 1 || list[0].ID != cards[0].ID {
		t.Fatalf("ListRatedByUser() = %v (total=%d), want only %v", list, total, cards[0].ID)
	}
}
