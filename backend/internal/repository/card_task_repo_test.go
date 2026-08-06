package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/medflow/backend/internal/models"
)

func createTestCardTask(t *testing.T, pool *pgxpool.Pool, repo *CardTaskRepo, userID uuid.UUID, opts ...func(*models.CardTask)) *models.CardTask {
	t.Helper()
	task := &models.CardTask{
		UserID:     userID,
		SourceType: models.SourceUserUpload,
		Topic:      ptr("Строение сердца"),
		Difficulty: models.DifficultyMedium,
		CardsCount: ptr(10),
	}
	for _, opt := range opts {
		opt(task)
	}
	created, err := repo.Create(context.Background(), task)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM cards WHERE task_id = $1", created.ID)
		_, _ = pool.Exec(context.Background(), "DELETE FROM card_tasks WHERE id = $1", created.ID)
	})
	return created
}

func TestCardTaskRepo_Create_And_FindByID(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewCardTaskRepo(pool)
	user := createTestForumUser(t, pool)

	created := createTestCardTask(t, pool, repo, user.ID)
	if created.Status != models.CardTaskPending {
		t.Errorf("Status = %v, want pending", created.Status)
	}

	found, err := repo.FindByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if found.UserID != user.ID {
		t.Errorf("UserID = %v, want %v", found.UserID, user.ID)
	}
	if found.Topic == nil || *found.Topic != "Строение сердца" {
		t.Errorf("Topic = %v, want 'Строение сердца'", found.Topic)
	}
}

func TestCardTaskRepo_FindByID_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewCardTaskRepo(pool)

	_, err := repo.FindByID(context.Background(), uuid.New())
	if err != models.ErrCardTaskNotFound {
		t.Fatalf("FindByID() error = %v, want ErrCardTaskNotFound", err)
	}
}

func TestCardTaskRepo_StatusTransitions(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewCardTaskRepo(pool)
	ctx := context.Background()
	user := createTestForumUser(t, pool)
	task := createTestCardTask(t, pool, repo, user.ID)

	if err := repo.MarkProcessing(ctx, task.ID); err != nil {
		t.Fatalf("MarkProcessing() error = %v", err)
	}
	found, _ := repo.FindByID(ctx, task.ID)
	if found.Status != models.CardTaskProcessing || found.StartedAt == nil {
		t.Fatalf("after MarkProcessing: %+v", found)
	}

	if err := repo.MarkDone(ctx, task.ID, 7); err != nil {
		t.Fatalf("MarkDone() error = %v", err)
	}
	found, _ = repo.FindByID(ctx, task.ID)
	if found.Status != models.CardTaskDone || found.FinishedAt == nil || found.CardsCount == nil || *found.CardsCount != 7 {
		t.Fatalf("after MarkDone: %+v", found)
	}
}

func TestCardTaskRepo_MarkFailed(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewCardTaskRepo(pool)
	ctx := context.Background()
	user := createTestForumUser(t, pool)
	task := createTestCardTask(t, pool, repo, user.ID)

	if err := repo.MarkFailed(ctx, task.ID, "llm unavailable"); err != nil {
		t.Fatalf("MarkFailed() error = %v", err)
	}
	found, _ := repo.FindByID(ctx, task.ID)
	if found.Status != models.CardTaskFailed || found.ErrorMessage == nil || *found.ErrorMessage != "llm unavailable" {
		t.Fatalf("after MarkFailed: %+v", found)
	}
}

func TestCardTaskRepo_CountActive(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewCardTaskRepo(pool)
	ctx := context.Background()
	user := createTestForumUser(t, pool)

	createTestCardTask(t, pool, repo, user.ID)
	second := createTestCardTask(t, pool, repo, user.ID)
	if err := repo.MarkDone(ctx, second.ID, 1); err != nil {
		t.Fatalf("MarkDone() error = %v", err)
	}

	n, err := repo.CountActive(ctx, user.ID)
	if err != nil {
		t.Fatalf("CountActive() error = %v", err)
	}
	if n != 1 {
		t.Errorf("CountActive() = %d, want 1 (only the pending one)", n)
	}
}

func TestCardTaskRepo_FindDoneByCacheKey(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewCardTaskRepo(pool)
	ctx := context.Background()
	user := createTestForumUser(t, pool)
	cacheKey := "cachekey-" + uuid.NewString()

	task := createTestCardTask(t, pool, repo, user.ID, func(ct *models.CardTask) {
		ct.SourceType = models.SourceCatalogTextbook
		ct.CacheKey = &cacheKey
	})

	if _, err := repo.FindDoneByCacheKey(ctx, cacheKey); err != models.ErrCardTaskNotFound {
		t.Fatalf("FindDoneByCacheKey() before done error = %v, want ErrCardTaskNotFound", err)
	}

	if err := repo.MarkDone(ctx, task.ID, 5); err != nil {
		t.Fatalf("MarkDone() error = %v", err)
	}

	found, err := repo.FindDoneByCacheKey(ctx, cacheKey)
	if err != nil {
		t.Fatalf("FindDoneByCacheKey() error = %v", err)
	}
	if found.ID != task.ID {
		t.Errorf("FindDoneByCacheKey() = %v, want %v", found.ID, task.ID)
	}
}

func TestCardTaskRepo_CountPendingBefore(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewCardTaskRepo(pool)
	ctx := context.Background()
	user := createTestForumUser(t, pool)

	createTestCardTask(t, pool, repo, user.ID)
	future := time.Now().Add(time.Hour)

	n, err := repo.CountPendingBefore(ctx, future)
	if err != nil {
		t.Fatalf("CountPendingBefore() error = %v", err)
	}
	if n < 1 {
		t.Errorf("CountPendingBefore(future) = %d, want >= 1", n)
	}

	past := time.Now().Add(-time.Hour)
	n, err = repo.CountPendingBefore(ctx, past)
	if err != nil {
		t.Fatalf("CountPendingBefore() error = %v", err)
	}
	if n != 0 {
		t.Errorf("CountPendingBefore(past) = %d, want 0", n)
	}
}

func TestCardTaskRepo_List_FiltersByStatus(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewCardTaskRepo(pool)
	ctx := context.Background()
	user := createTestForumUser(t, pool)

	pending := createTestCardTask(t, pool, repo, user.ID)
	done := createTestCardTask(t, pool, repo, user.ID)
	if err := repo.MarkDone(ctx, done.ID, 3); err != nil {
		t.Fatalf("MarkDone() error = %v", err)
	}

	status := models.CardTaskPending
	items, total, err := repo.List(ctx, models.CardTaskListFilter{UserID: user.ID, Status: &status, Page: 1, Limit: 20})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if total != 1 || len(items) != 1 || items[0].ID != pending.ID {
		t.Fatalf("List(status=pending) = %v (total=%d), want only %v", items, total, pending.ID)
	}
}
