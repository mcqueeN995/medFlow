package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/medflow/backend/internal/models"
)

func TestAdminService_ListReports(t *testing.T) {
	reportRepo := &mockReportRepository{
		listFn: func(ctx context.Context, f models.ReportListFilter) ([]models.Report, int, error) {
			return []models.Report{{ID: uuid.New(), TargetType: "thread"}}, 1, nil
		},
	}
	svc := NewAdminService(reportRepo, &mockAuditLogRepository{}, &mockAdminStatsRepository{}, &mockThreadRepository{}, &mockCommentRepository{}, &mockCardRepository{})

	pagination, items, err := svc.ListReports(context.Background(), models.ReportListFilter{})
	if err != nil {
		t.Fatalf("ListReports() error = %v", err)
	}
	if len(items) != 1 || pagination.Total != 1 {
		t.Fatalf("ListReports() = %v (pagination=%+v), want 1 item", items, pagination)
	}
}

func TestAdminService_ListReports_EnrichesThreadContext(t *testing.T) {
	threadID := uuid.New()
	reportRepo := &mockReportRepository{
		listFn: func(ctx context.Context, f models.ReportListFilter) ([]models.Report, int, error) {
			return []models.Report{{ID: uuid.New(), TargetType: "thread", TargetID: threadID}}, 1, nil
		},
	}
	threadRepo := &mockThreadRepository{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.Thread, error) {
			return &models.Thread{ID: threadID, Title: "Как сдать анатомию", Content: "Помогите с материалами"}, nil
		},
	}
	svc := NewAdminService(reportRepo, &mockAuditLogRepository{}, &mockAdminStatsRepository{}, threadRepo, &mockCommentRepository{}, &mockCardRepository{})

	_, items, err := svc.ListReports(context.Background(), models.ReportListFilter{})
	if err != nil {
		t.Fatalf("ListReports() error = %v", err)
	}
	got := items[0]
	if got.TargetThreadID == nil || *got.TargetThreadID != threadID.String() {
		t.Errorf("TargetThreadID = %v, want %v", got.TargetThreadID, threadID)
	}
	if got.TargetTitle == nil || *got.TargetTitle != "Как сдать анатомию" {
		t.Errorf("TargetTitle = %v, want title", got.TargetTitle)
	}
	if got.TargetSnippet == nil || *got.TargetSnippet != "Помогите с материалами" {
		t.Errorf("TargetSnippet = %v, want content", got.TargetSnippet)
	}
	if got.TargetRemoved {
		t.Error("TargetRemoved = true, want false for a live thread")
	}
}

func TestAdminService_ListReports_EnrichesCommentContext_WithParentThreadTitle(t *testing.T) {
	threadID, commentID := uuid.New(), uuid.New()
	reportRepo := &mockReportRepository{
		listFn: func(ctx context.Context, f models.ReportListFilter) ([]models.Report, int, error) {
			return []models.Report{{ID: uuid.New(), TargetType: "comment", TargetID: commentID}}, 1, nil
		},
	}
	commentRepo := &mockCommentRepository{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.Comment, error) {
			return &models.Comment{ID: commentID, ThreadID: threadID, Content: "оскорбительный текст"}, nil
		},
	}
	threadRepo := &mockThreadRepository{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.Thread, error) {
			return &models.Thread{ID: threadID, Title: "Тред-родитель"}, nil
		},
	}
	svc := NewAdminService(reportRepo, &mockAuditLogRepository{}, &mockAdminStatsRepository{}, threadRepo, commentRepo, &mockCardRepository{})

	_, items, err := svc.ListReports(context.Background(), models.ReportListFilter{})
	if err != nil {
		t.Fatalf("ListReports() error = %v", err)
	}
	got := items[0]
	if got.TargetThreadID == nil || *got.TargetThreadID != threadID.String() {
		t.Errorf("TargetThreadID = %v, want parent thread %v", got.TargetThreadID, threadID)
	}
	if got.TargetTitle == nil || *got.TargetTitle != "Тред-родитель" {
		t.Errorf("TargetTitle = %v, want parent thread title", got.TargetTitle)
	}
	if got.TargetSnippet == nil || *got.TargetSnippet != "оскорбительный текст" {
		t.Errorf("TargetSnippet = %v, want comment content", got.TargetSnippet)
	}
}

func TestAdminService_ListReports_TargetRemovedWhenNotFound(t *testing.T) {
	reportRepo := &mockReportRepository{
		listFn: func(ctx context.Context, f models.ReportListFilter) ([]models.Report, int, error) {
			return []models.Report{{ID: uuid.New(), TargetType: "thread", TargetID: uuid.New()}}, 1, nil
		},
	}
	svc := NewAdminService(reportRepo, &mockAuditLogRepository{}, &mockAdminStatsRepository{}, &mockThreadRepository{}, &mockCommentRepository{}, &mockCardRepository{})

	_, items, err := svc.ListReports(context.Background(), models.ReportListFilter{})
	if err != nil {
		t.Fatalf("ListReports() error = %v", err)
	}
	if !items[0].TargetRemoved {
		t.Error("TargetRemoved = false, want true when the underlying thread lookup fails")
	}
}

func TestAdminService_ReviewReport_DoesNotWriteAuditLog(t *testing.T) {
	reportID, reviewerID := uuid.New(), uuid.New()
	note := "проверено"
	reportRepo := &mockReportRepository{
		reviewFn: func(ctx context.Context, id, reviewedBy uuid.UUID, status models.ReportStatus, note *string) (*models.Report, error) {
			return &models.Report{ID: id, Status: status, ReviewedBy: &reviewedBy, ResolutionNote: note}, nil
		},
	}
	auditCalled := false
	auditRepo := &mockAuditLogRepository{
		createFn: func(ctx context.Context, entry *models.AuditLog) error { auditCalled = true; return nil },
	}
	svc := NewAdminService(reportRepo, auditRepo, &mockAdminStatsRepository{}, &mockThreadRepository{}, &mockCommentRepository{}, &mockCardRepository{})

	out, err := svc.ReviewReport(context.Background(), reviewerID, reportID, models.ReportStatusReviewed, &note)
	if err != nil {
		t.Fatalf("ReviewReport() error = %v", err)
	}
	if out.Status != models.ReportStatusReviewed {
		t.Errorf("Status = %v, want reviewed", out.Status)
	}
	// AuditAction в контракте не содержит отдельного значения под "рассмотрение
	// жалобы" (см. план модуля) - намеренно не пишем несуществующий action.
	if auditCalled {
		t.Error("ReviewReport() should not write an audit log entry - no matching AuditAction in the contract")
	}
}

func TestAdminService_ReviewReport_NotFound(t *testing.T) {
	reportRepo := &mockReportRepository{
		reviewFn: func(ctx context.Context, id, reviewedBy uuid.UUID, status models.ReportStatus, note *string) (*models.Report, error) {
			return nil, models.ErrReportNotFound
		},
	}
	svc := NewAdminService(reportRepo, &mockAuditLogRepository{}, &mockAdminStatsRepository{}, &mockThreadRepository{}, &mockCommentRepository{}, &mockCardRepository{})

	_, err := svc.ReviewReport(context.Background(), uuid.New(), uuid.New(), models.ReportStatusDismissed, nil)
	if !errors.Is(err, ErrReportNotFound) {
		t.Fatalf("ReviewReport() error = %v, want ErrReportNotFound", err)
	}
}

func TestAdminService_Stats(t *testing.T) {
	statsRepo := &mockAdminStatsRepository{
		statsFn: func(ctx context.Context) (models.AdminStats, error) {
			return models.AdminStats{UsersTotal: 42, UsersBanned: 3}, nil
		},
	}
	svc := NewAdminService(&mockReportRepository{}, &mockAuditLogRepository{}, statsRepo, &mockThreadRepository{}, &mockCommentRepository{}, &mockCardRepository{})

	stats, err := svc.Stats(context.Background())
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}
	if stats.UsersTotal != 42 || stats.UsersBanned != 3 {
		t.Errorf("Stats() = %+v, unexpected", stats)
	}
}

func TestAdminService_AuditLogs(t *testing.T) {
	actorID := uuid.New()
	var gotFilter models.AuditLogListFilter
	auditRepo := &mockAuditLogRepository{
		listFn: func(ctx context.Context, f models.AuditLogListFilter) ([]models.AuditLog, int, error) {
			gotFilter = f
			return []models.AuditLog{{ID: uuid.New(), ActorID: actorID, Action: models.AuditUserBan}}, 1, nil
		},
	}
	svc := NewAdminService(&mockReportRepository{}, auditRepo, &mockAdminStatsRepository{}, &mockThreadRepository{}, &mockCommentRepository{}, &mockCardRepository{})

	pagination, items, err := svc.AuditLogs(context.Background(), models.AuditLogListFilter{ActorID: &actorID})
	if err != nil {
		t.Fatalf("AuditLogs() error = %v", err)
	}
	if len(items) != 1 || pagination.Total != 1 {
		t.Fatalf("AuditLogs() = %v (pagination=%+v), want 1 item", items, pagination)
	}
	if gotFilter.ActorID == nil || *gotFilter.ActorID != actorID {
		t.Errorf("filter.ActorID = %v, want %v", gotFilter.ActorID, actorID)
	}
}
