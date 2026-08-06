package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/medflow/backend/internal/models"
)

func TestReportRepo_Create(t *testing.T) {
	pool := setupTestDB(t)
	threadRepo := NewThreadRepo(pool)
	reportRepo := NewReportRepo(pool)
	ctx := context.Background()
	author := createTestForumUser(t, pool)
	thread := createTestThread(t, pool, threadRepo, author.ID, "reportable thread", nil)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM reports WHERE target_id = $1", thread.ID)
	})

	report, err := reportRepo.Create(ctx, author.ID, "thread", thread.ID, "спам")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if report.Status != models.ReportStatusPending {
		t.Errorf("Status = %q, want pending", report.Status)
	}
	if report.Reason != "спам" {
		t.Errorf("Reason = %q, want спам", report.Reason)
	}
}

func TestReportRepo_FindByID_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewReportRepo(pool)

	_, err := repo.FindByID(context.Background(), uuid.New())
	if err != models.ErrReportNotFound {
		t.Fatalf("FindByID() error = %v, want ErrReportNotFound", err)
	}
}

func TestReportRepo_List_FiltersByStatus(t *testing.T) {
	pool := setupTestDB(t)
	threadRepo := NewThreadRepo(pool)
	reportRepo := NewReportRepo(pool)
	ctx := context.Background()
	author := createTestForumUser(t, pool)
	thread := createTestThread(t, pool, threadRepo, author.ID, "reportable thread 2", nil)
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), "DELETE FROM reports WHERE target_id = $1", thread.ID) })

	pending, err := reportRepo.Create(ctx, author.ID, "thread", thread.ID, "спам1")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	reviewed, err := reportRepo.Create(ctx, author.ID, "thread", thread.ID, "спам2")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := reportRepo.Review(ctx, reviewed.ID, author.ID, models.ReportStatusDismissed, nil); err != nil {
		t.Fatalf("Review() error = %v", err)
	}

	// List() не фильтрует по репортёру/треду (это глобальный список для
	// админки) - в общей тестовой БД могут быть pending-репорты из других
	// тестов, поэтому проверяем присутствие/отсутствие конкретных ID, а не
	// точный total.
	status := models.ReportStatusPending
	items, _, err := reportRepo.List(ctx, models.ReportListFilter{Status: &status, Page: 1, Limit: 100})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	var sawPending, sawReviewed bool
	for _, it := range items {
		if it.ID == pending.ID {
			sawPending = true
		}
		if it.ID == reviewed.ID {
			sawReviewed = true
		}
	}
	if !sawPending {
		t.Errorf("List(status=pending) missing the pending report %v", pending.ID)
	}
	if sawReviewed {
		t.Errorf("List(status=pending) should not include the dismissed report %v", reviewed.ID)
	}
}

func TestReportRepo_Review(t *testing.T) {
	pool := setupTestDB(t)
	threadRepo := NewThreadRepo(pool)
	reportRepo := NewReportRepo(pool)
	ctx := context.Background()
	author := createTestForumUser(t, pool)
	reviewer := createTestForumUser(t, pool)
	thread := createTestThread(t, pool, threadRepo, author.ID, "reportable thread 3", nil)
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), "DELETE FROM reports WHERE target_id = $1", thread.ID) })

	report, err := reportRepo.Create(ctx, author.ID, "thread", thread.ID, "спам")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	note := "проверено, нарушение подтверждено"
	reviewed, err := reportRepo.Review(ctx, report.ID, reviewer.ID, models.ReportStatusReviewed, &note)
	if err != nil {
		t.Fatalf("Review() error = %v", err)
	}
	if reviewed.Status != models.ReportStatusReviewed {
		t.Errorf("Status = %v, want reviewed", reviewed.Status)
	}
	if reviewed.ReviewedBy == nil || *reviewed.ReviewedBy != reviewer.ID {
		t.Errorf("ReviewedBy = %v, want %v", reviewed.ReviewedBy, reviewer.ID)
	}
	if reviewed.ReviewedAt == nil {
		t.Error("ReviewedAt = nil, want set")
	}
	if reviewed.ResolutionNote == nil || *reviewed.ResolutionNote != note {
		t.Errorf("ResolutionNote = %v, want %q", reviewed.ResolutionNote, note)
	}
}

func TestReportRepo_Review_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewReportRepo(pool)

	_, err := repo.Review(context.Background(), uuid.New(), uuid.New(), models.ReportStatusReviewed, nil)
	if err != models.ErrReportNotFound {
		t.Fatalf("Review() error = %v, want ErrReportNotFound", err)
	}
}
