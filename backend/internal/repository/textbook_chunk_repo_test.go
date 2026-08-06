package repository

import (
	"context"
	"testing"

	"github.com/medflow/backend/internal/models"
)

// syntheticVector строит детерминированный 1024-мерный вектор с единицей на
// позиции dim и нулями всюду - удобно для проверки, что ближайшим соседом
// становится вектор с максимально похожей "активной" позицией.
func syntheticVector(dim int) []float32 {
	v := make([]float32, 1024)
	v[dim] = 1
	return v
}

func TestTextbookChunkRepo_CreateBatch_And_ExistsForTextbook(t *testing.T) {
	pool := setupTestDB(t)
	textbookRepo := NewTextbookRepo(pool)
	chunkRepo := NewTextbookChunkRepo(pool)
	ctx := context.Background()
	textbook := createTestTextbookA(t, pool, textbookRepo, "учебник для чанков")

	exists, err := chunkRepo.ExistsForTextbook(ctx, textbook.ID)
	if err != nil {
		t.Fatalf("ExistsForTextbook() error = %v", err)
	}
	if exists {
		t.Fatal("ExistsForTextbook() = true before any chunks created")
	}

	err = chunkRepo.CreateBatch(ctx, []models.TextbookChunk{
		{TextbookID: &textbook.ID, ChunkIndex: 0, Content: "первый кусок", PageNumber: ptr(1), Embedding: syntheticVector(0)},
		{TextbookID: &textbook.ID, ChunkIndex: 1, Content: "второй кусок", PageNumber: ptr(2), Embedding: syntheticVector(1)},
	})
	if err != nil {
		t.Fatalf("CreateBatch() error = %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM textbook_chunks WHERE textbook_id = $1", textbook.ID)
	})

	exists, err = chunkRepo.ExistsForTextbook(ctx, textbook.ID)
	if err != nil {
		t.Fatalf("ExistsForTextbook() error = %v", err)
	}
	if !exists {
		t.Fatal("ExistsForTextbook() = false after chunks created")
	}
}

func TestTextbookChunkRepo_SearchNearest_OrdersByDistance(t *testing.T) {
	pool := setupTestDB(t)
	textbookRepo := NewTextbookRepo(pool)
	chunkRepo := NewTextbookChunkRepo(pool)
	ctx := context.Background()
	textbook := createTestTextbookA(t, pool, textbookRepo, "учебник для поиска")

	err := chunkRepo.CreateBatch(ctx, []models.TextbookChunk{
		{TextbookID: &textbook.ID, ChunkIndex: 0, Content: "далёкий", PageNumber: ptr(1), Embedding: syntheticVector(500)},
		{TextbookID: &textbook.ID, ChunkIndex: 1, Content: "близкий", PageNumber: ptr(2), Embedding: syntheticVector(1)},
		{TextbookID: &textbook.ID, ChunkIndex: 2, Content: "средний", PageNumber: ptr(3), Embedding: syntheticVector(100)},
	})
	if err != nil {
		t.Fatalf("CreateBatch() error = %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM textbook_chunks WHERE textbook_id = $1", textbook.ID)
	})

	query := syntheticVector(1) // совпадает с "близкий"
	results, err := chunkRepo.SearchNearest(ctx, &textbook.ID, nil, query, 2)
	if err != nil {
		t.Fatalf("SearchNearest() error = %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2 (topK)", len(results))
	}
	if results[0].Content != "близкий" {
		t.Errorf("results[0].Content = %q, want %q (nearest neighbor first)", results[0].Content, "близкий")
	}
}

func TestTextbookChunkRepo_SearchNearest_ScopedByTaskID(t *testing.T) {
	pool := setupTestDB(t)
	taskRepo := NewCardTaskRepo(pool)
	chunkRepo := NewTextbookChunkRepo(pool)
	ctx := context.Background()
	user := createTestForumUser(t, pool)
	task := createTestCardTask(t, pool, taskRepo, user.ID)

	err := chunkRepo.CreateBatch(ctx, []models.TextbookChunk{
		{TaskID: &task.ID, ChunkIndex: 0, Content: "личный чанк", PageNumber: ptr(1), Embedding: syntheticVector(1)},
	})
	if err != nil {
		t.Fatalf("CreateBatch() error = %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM textbook_chunks WHERE task_id = $1", task.ID)
	})

	results, err := chunkRepo.SearchNearest(ctx, nil, &task.ID, syntheticVector(1), 5)
	if err != nil {
		t.Fatalf("SearchNearest() error = %v", err)
	}
	if len(results) != 1 || results[0].Content != "личный чанк" {
		t.Fatalf("SearchNearest(taskID) = %v, want the one chunk scoped to this task", results)
	}
}

func TestTextbookChunkRepo_SearchNearest_RequiresSource(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewTextbookChunkRepo(pool)

	_, err := repo.SearchNearest(context.Background(), nil, nil, syntheticVector(0), 5)
	if err == nil {
		t.Fatal("SearchNearest(nil, nil) error = nil, want error")
	}
}
