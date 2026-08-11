package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/medflow/backend/internal/models"
)

func TestCardProgressRepo_CreateBatchDefault_And_Find(t *testing.T) {
	pool := setupTestDB(t)
	taskRepo := NewCardTaskRepo(pool)
	cardRepo := NewCardRepo(pool)
	progressRepo := NewCardProgressRepo(pool)
	ctx := context.Background()
	user := createTestForumUser(t, pool)
	task := createTestCardTask(t, pool, taskRepo, user.ID)
	cards := createTestCards(t, pool, cardRepo, task.ID, 2)
	cardIDs := []uuid.UUID{cards[0].ID, cards[1].ID}

	if err := progressRepo.CreateBatchDefault(ctx, user.ID, cardIDs); err != nil {
		t.Fatalf("CreateBatchDefault() error = %v", err)
	}

	p, err := progressRepo.FindByUserAndCard(ctx, user.ID, cards[0].ID)
	if err != nil {
		t.Fatalf("FindByUserAndCard() error = %v", err)
	}
	if p.EaseFactor != 2.5 || p.IntervalDays != 0 || p.Repetitions != 0 {
		t.Errorf("default progress = %+v, want {2.5, 0, 0}", p)
	}
	if p.NextReviewAt.After(time.Now().Add(time.Minute)) {
		t.Errorf("NextReviewAt = %v, want ~now (immediately due)", p.NextReviewAt)
	}
}

func TestCardProgressRepo_FindByUserAndCard_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewCardProgressRepo(pool)
	user := createTestForumUser(t, pool)

	_, err := repo.FindByUserAndCard(context.Background(), user.ID, uuid.New())
	if err != models.ErrCardProgressNotFound {
		t.Fatalf("FindByUserAndCard() error = %v, want ErrCardProgressNotFound", err)
	}
}

func TestCardProgressRepo_Update(t *testing.T) {
	pool := setupTestDB(t)
	taskRepo := NewCardTaskRepo(pool)
	cardRepo := NewCardRepo(pool)
	progressRepo := NewCardProgressRepo(pool)
	ctx := context.Background()
	user := createTestForumUser(t, pool)
	task := createTestCardTask(t, pool, taskRepo, user.ID)
	cards := createTestCards(t, pool, cardRepo, task.ID, 1)
	if err := progressRepo.CreateBatchDefault(ctx, user.ID, []uuid.UUID{cards[0].ID}); err != nil {
		t.Fatalf("CreateBatchDefault() error = %v", err)
	}
	p, err := progressRepo.FindByUserAndCard(ctx, user.ID, cards[0].ID)
	if err != nil {
		t.Fatalf("FindByUserAndCard() error = %v", err)
	}

	p.EaseFactor = 2.7
	p.IntervalDays = 6
	p.Repetitions = 2
	p.NextReviewAt = time.Now().Add(6 * 24 * time.Hour)
	now := time.Now()
	p.LastReviewAt = &now
	grade := 3
	p.LastGrade = &grade

	if err := progressRepo.Update(ctx, p); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	found, err := progressRepo.FindByUserAndCard(ctx, user.ID, cards[0].ID)
	if err != nil {
		t.Fatalf("FindByUserAndCard() error = %v", err)
	}
	if found.EaseFactor != 2.7 || found.IntervalDays != 6 || found.Repetitions != 2 {
		t.Errorf("updated progress = %+v, want {2.7, 6, 2}", found)
	}
	if found.LastGrade == nil || *found.LastGrade != 3 {
		t.Errorf("LastGrade = %v, want 3", found.LastGrade)
	}
}

func TestCardProgressRepo_ListDueForUser(t *testing.T) {
	pool := setupTestDB(t)
	taskRepo := NewCardTaskRepo(pool)
	cardRepo := NewCardRepo(pool)
	progressRepo := NewCardProgressRepo(pool)
	ctx := context.Background()
	user := createTestForumUser(t, pool)
	task := createTestCardTask(t, pool, taskRepo, user.ID)
	cards := createTestCards(t, pool, cardRepo, task.ID, 2)

	if err := progressRepo.CreateBatchDefault(ctx, user.ID, []uuid.UUID{cards[0].ID, cards[1].ID}); err != nil {
		t.Fatalf("CreateBatchDefault() error = %v", err)
	}
	// отодвигаем прогресс второй карточки в будущее - не должна попасть в due
	p1, _ := progressRepo.FindByUserAndCard(ctx, user.ID, cards[1].ID)
	p1.NextReviewAt = time.Now().Add(48 * time.Hour)
	if err := progressRepo.Update(ctx, p1); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	due, err := progressRepo.ListDueForUser(ctx, user.ID, 20)
	if err != nil {
		t.Fatalf("ListDueForUser() error = %v", err)
	}
	if len(due) != 1 || due[0].ID != cards[0].ID {
		t.Fatalf("ListDueForUser() = %v, want only %v", due, cards[0].ID)
	}
	if due[0].Progress.UserID != user.ID {
		t.Errorf("Progress.UserID = %v, want %v", due[0].Progress.UserID, user.ID)
	}
}

func TestCardProgressRepo_StatsForUser(t *testing.T) {
	pool := setupTestDB(t)
	taskRepo := NewCardTaskRepo(pool)
	cardRepo := NewCardRepo(pool)
	progressRepo := NewCardProgressRepo(pool)
	ctx := context.Background()
	user := createTestForumUser(t, pool)
	task := createTestCardTask(t, pool, taskRepo, user.ID)
	cards := createTestCards(t, pool, cardRepo, task.ID, 2)
	if err := progressRepo.CreateBatchDefault(ctx, user.ID, []uuid.UUID{cards[0].ID, cards[1].ID}); err != nil {
		t.Fatalf("CreateBatchDefault() error = %v", err)
	}

	p0, _ := progressRepo.FindByUserAndCard(ctx, user.ID, cards[0].ID)
	p0.Repetitions = 1
	if err := progressRepo.Update(ctx, p0); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	stats, err := progressRepo.StatsForUser(ctx, user.ID)
	if err != nil {
		t.Fatalf("StatsForUser() error = %v", err)
	}
	if stats.TotalCardsLearned != 1 {
		t.Errorf("TotalCardsLearned = %d, want 1", stats.TotalCardsLearned)
	}
	if stats.DueToday != 2 {
		t.Errorf("DueToday = %d, want 2", stats.DueToday)
	}
	if stats.ByDifficulty[models.DifficultyMedium] != 2 {
		t.Errorf("ByDifficulty[medium] = %d, want 2", stats.ByDifficulty[models.DifficultyMedium])
	}
}

func TestCardProgressRepo_ListDueFavoritesForUser_OnlyFavorited(t *testing.T) {
	pool := setupTestDB(t)
	taskRepo := NewCardTaskRepo(pool)
	cardRepo := NewCardRepo(pool)
	progressRepo := NewCardProgressRepo(pool)
	favoriteRepo := NewCardFavoriteRepo(pool)
	ctx := context.Background()
	user := createTestForumUser(t, pool)
	task := createTestCardTask(t, pool, taskRepo, user.ID)
	cards := createTestCards(t, pool, cardRepo, task.ID, 2)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM card_favorites WHERE user_id = $1", user.ID)
	})

	if err := progressRepo.CreateBatchDefault(ctx, user.ID, []uuid.UUID{cards[0].ID, cards[1].ID}); err != nil {
		t.Fatalf("CreateBatchDefault() error = %v", err)
	}
	// Обе карточки due (next_review_at=now() по умолчанию), но только первая избранная.
	if err := favoriteRepo.Add(ctx, user.ID, cards[0].ID); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	due, err := progressRepo.ListDueFavoritesForUser(ctx, user.ID, 20)
	if err != nil {
		t.Fatalf("ListDueFavoritesForUser() error = %v", err)
	}
	if len(due) != 1 || due[0].ID != cards[0].ID {
		t.Fatalf("ListDueFavoritesForUser() = %v, want only %v (избранная)", due, cards[0].ID)
	}

	count, err := progressRepo.CountDueFavoritesForUser(ctx, user.ID)
	if err != nil {
		t.Fatalf("CountDueFavoritesForUser() error = %v", err)
	}
	if count != 1 {
		t.Errorf("CountDueFavoritesForUser() = %d, want 1", count)
	}
}

func TestCardProgressRepo_DistinctReviewDaysForUser(t *testing.T) {
	pool := setupTestDB(t)
	taskRepo := NewCardTaskRepo(pool)
	cardRepo := NewCardRepo(pool)
	progressRepo := NewCardProgressRepo(pool)
	ctx := context.Background()
	user := createTestForumUser(t, pool)
	task := createTestCardTask(t, pool, taskRepo, user.ID)
	cards := createTestCards(t, pool, cardRepo, task.ID, 1)
	if err := progressRepo.CreateBatchDefault(ctx, user.ID, []uuid.UUID{cards[0].ID}); err != nil {
		t.Fatalf("CreateBatchDefault() error = %v", err)
	}

	p, _ := progressRepo.FindByUserAndCard(ctx, user.ID, cards[0].ID)
	now := time.Now()
	p.LastReviewAt = &now
	if err := progressRepo.Update(ctx, p); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	days, err := progressRepo.DistinctReviewDaysForUser(ctx, user.ID, 30)
	if err != nil {
		t.Fatalf("DistinctReviewDaysForUser() error = %v", err)
	}
	if len(days) != 1 {
		t.Fatalf("DistinctReviewDaysForUser() = %v, want 1 day", days)
	}
}
