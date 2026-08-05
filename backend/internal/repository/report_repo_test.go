package repository

import (
	"context"
	"testing"

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
