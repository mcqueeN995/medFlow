package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/medflow/backend/internal/models"
)

func TestCardFavoriteRepo_Add_And_ListForUser(t *testing.T) {
	pool := setupTestDB(t)
	taskRepo := NewCardTaskRepo(pool)
	cardRepo := NewCardRepo(pool)
	favoriteRepo := NewCardFavoriteRepo(pool)
	ctx := context.Background()
	user := createTestForumUser(t, pool)
	task := createTestCardTask(t, pool, taskRepo, user.ID)
	cards := createTestCards(t, pool, cardRepo, task.ID, 2)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM card_favorites WHERE user_id = $1", user.ID)
	})

	if err := favoriteRepo.Add(ctx, user.ID, cards[0].ID); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	// повторное добавление уже избранной карточки - идемпотентно, не ошибка.
	if err := favoriteRepo.Add(ctx, user.ID, cards[0].ID); err != nil {
		t.Fatalf("Add() [repeat] error = %v", err)
	}

	list, total, err := favoriteRepo.ListForUser(ctx, user.ID, 1, 20)
	if err != nil {
		t.Fatalf("ListForUser() error = %v", err)
	}
	if total != 1 || len(list) != 1 || list[0].ID != cards[0].ID {
		t.Fatalf("ListForUser() = %v (total=%d), want only %v", list, total, cards[0].ID)
	}
}

func TestCardFavoriteRepo_Remove_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	favoriteRepo := NewCardFavoriteRepo(pool)
	user := createTestForumUser(t, pool)

	err := favoriteRepo.Remove(context.Background(), user.ID, uuid.New())
	if err != models.ErrCardFavoriteNotFound {
		t.Fatalf("Remove() error = %v, want ErrCardFavoriteNotFound", err)
	}
}

func TestCardFavoriteRepo_Remove_Success(t *testing.T) {
	pool := setupTestDB(t)
	taskRepo := NewCardTaskRepo(pool)
	cardRepo := NewCardRepo(pool)
	favoriteRepo := NewCardFavoriteRepo(pool)
	ctx := context.Background()
	user := createTestForumUser(t, pool)
	task := createTestCardTask(t, pool, taskRepo, user.ID)
	cards := createTestCards(t, pool, cardRepo, task.ID, 1)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM card_favorites WHERE user_id = $1", user.ID)
	})

	if err := favoriteRepo.Add(ctx, user.ID, cards[0].ID); err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if err := favoriteRepo.Remove(ctx, user.ID, cards[0].ID); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}

	_, total, err := favoriteRepo.ListForUser(ctx, user.ID, 1, 20)
	if err != nil {
		t.Fatalf("ListForUser() error = %v", err)
	}
	if total != 0 {
		t.Errorf("total = %d, want 0 after Remove", total)
	}
}

func TestCardFavoriteRepo_IsFavoritedBatch(t *testing.T) {
	pool := setupTestDB(t)
	taskRepo := NewCardTaskRepo(pool)
	cardRepo := NewCardRepo(pool)
	favoriteRepo := NewCardFavoriteRepo(pool)
	ctx := context.Background()
	user := createTestForumUser(t, pool)
	task := createTestCardTask(t, pool, taskRepo, user.ID)
	cards := createTestCards(t, pool, cardRepo, task.ID, 2)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM card_favorites WHERE user_id = $1", user.ID)
	})

	if err := favoriteRepo.Add(ctx, user.ID, cards[0].ID); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	result, err := favoriteRepo.IsFavoritedBatch(ctx, user.ID, []uuid.UUID{cards[0].ID, cards[1].ID})
	if err != nil {
		t.Fatalf("IsFavoritedBatch() error = %v", err)
	}
	if !result[cards[0].ID] {
		t.Errorf("result[%v] = false, want true", cards[0].ID)
	}
	if result[cards[1].ID] {
		t.Errorf("result[%v] = true, want false (not favorited)", cards[1].ID)
	}
}
