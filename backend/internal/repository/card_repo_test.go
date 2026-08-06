package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/medflow/backend/internal/models"
)

func createTestCards(t *testing.T, pool *pgxpool.Pool, repo *CardRepo, taskID uuid.UUID, n int) []models.Card {
	t.Helper()
	cards := make([]models.Card, n)
	for i := range cards {
		cards[i] = models.Card{
			TaskID:     taskID,
			Question:   "Вопрос?",
			Answer:     "Ответ.",
			Difficulty: models.DifficultyMedium,
		}
	}
	created, err := repo.CreateBatch(context.Background(), cards)
	if err != nil {
		t.Fatalf("CreateBatch() error = %v", err)
	}
	t.Cleanup(func() {
		for _, c := range created {
			_, _ = pool.Exec(context.Background(), "DELETE FROM card_progress WHERE card_id = $1", c.ID)
			_, _ = pool.Exec(context.Background(), "DELETE FROM cards WHERE id = $1", c.ID)
		}
	})
	return created
}

func TestCardRepo_CreateBatch_And_ListByTask(t *testing.T) {
	pool := setupTestDB(t)
	taskRepo := NewCardTaskRepo(pool)
	cardRepo := NewCardRepo(pool)
	user := createTestForumUser(t, pool)
	task := createTestCardTask(t, pool, taskRepo, user.ID)

	created := createTestCards(t, pool, cardRepo, task.ID, 3)
	if len(created) != 3 {
		t.Fatalf("CreateBatch() returned %d cards, want 3", len(created))
	}
	for _, c := range created {
		if c.ID == uuid.Nil {
			t.Errorf("card ID not set: %+v", c)
		}
	}

	items, total, err := cardRepo.ListByTask(context.Background(), task.ID, 1, 20)
	if err != nil {
		t.Fatalf("ListByTask() error = %v", err)
	}
	if total != 3 || len(items) != 3 {
		t.Fatalf("ListByTask() = %d items (total=%d), want 3", len(items), total)
	}
}

func TestCardRepo_FindByID_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewCardRepo(pool)

	_, err := repo.FindByID(context.Background(), uuid.New())
	if err != models.ErrCardNotFound {
		t.Fatalf("FindByID() error = %v, want ErrCardNotFound", err)
	}
}

func TestCardRepo_CloneForTask(t *testing.T) {
	pool := setupTestDB(t)
	taskRepo := NewCardTaskRepo(pool)
	cardRepo := NewCardRepo(pool)
	user := createTestForumUser(t, pool)
	sourceTask := createTestCardTask(t, pool, taskRepo, user.ID)
	newTask := createTestCardTask(t, pool, taskRepo, user.ID)

	createTestCards(t, pool, cardRepo, sourceTask.ID, 2)

	cloned, err := cardRepo.CloneForTask(context.Background(), sourceTask.ID, newTask.ID)
	if err != nil {
		t.Fatalf("CloneForTask() error = %v", err)
	}
	t.Cleanup(func() {
		for _, c := range cloned {
			_, _ = pool.Exec(context.Background(), "DELETE FROM cards WHERE id = $1", c.ID)
		}
	})
	if len(cloned) != 2 {
		t.Fatalf("CloneForTask() returned %d cards, want 2", len(cloned))
	}
	for _, c := range cloned {
		if c.TaskID != newTask.ID {
			t.Errorf("cloned card TaskID = %v, want %v", c.TaskID, newTask.ID)
		}
		if c.Question != "Вопрос?" || c.Answer != "Ответ." {
			t.Errorf("cloned card content = %+v, want copied from source", c)
		}
	}
}

func TestCardRepo_IncrementReportCount(t *testing.T) {
	pool := setupTestDB(t)
	taskRepo := NewCardTaskRepo(pool)
	cardRepo := NewCardRepo(pool)
	user := createTestForumUser(t, pool)
	task := createTestCardTask(t, pool, taskRepo, user.ID)
	cards := createTestCards(t, pool, cardRepo, task.ID, 1)

	if err := cardRepo.IncrementReportCount(context.Background(), cards[0].ID); err != nil {
		t.Fatalf("IncrementReportCount() error = %v", err)
	}
	found, err := cardRepo.FindByID(context.Background(), cards[0].ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if found.ReportCount != 1 {
		t.Errorf("ReportCount = %d, want 1", found.ReportCount)
	}
}
