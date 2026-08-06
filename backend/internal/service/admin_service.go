package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/medflow/backend/internal/dto"
	"github.com/medflow/backend/internal/models"
)

var ErrReportNotFound = errors.New("report not found")

// AdminService - Reports/Stats/AuditLogs: части админки без естественного
// дома в доменном сервисе (Report не принадлежит только форуму - target_type
// может быть "card", см. CardService.ReportCard; Stats/AuditLogs изначально
// межмодульные).
type AdminService struct {
	reportRepo   ReportRepository
	auditLogRepo AuditLogRepository
	statsRepo    AdminStatsRepository
}

func NewAdminService(reportRepo ReportRepository, auditLogRepo AuditLogRepository, statsRepo AdminStatsRepository) *AdminService {
	return &AdminService{reportRepo: reportRepo, auditLogRepo: auditLogRepo, statsRepo: statsRepo}
}

func (s *AdminService) ListReports(ctx context.Context, f models.ReportListFilter) (*dto.Pagination, []dto.Report, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Limit < 1 || f.Limit > 100 {
		f.Limit = 20
	}

	reports, total, err := s.reportRepo.List(ctx, f)
	if err != nil {
		return nil, nil, err
	}
	items := make([]dto.Report, len(reports))
	for i := range reports {
		items[i] = dto.ToReport(&reports[i])
	}
	pagination := dto.NewPagination(f.Page, f.Limit, total)
	return &pagination, items, nil
}

// ReviewReport. AuditAction не содержит отдельного значения под "рассмотрение
// жалобы" (см. план модуля) - в audit_logs не пишем, чтобы не изобретать
// несуществующий в контракте action.
func (s *AdminService) ReviewReport(ctx context.Context, reviewerID, id uuid.UUID, status models.ReportStatus, note *string) (*dto.Report, error) {
	reviewed, err := s.reportRepo.Review(ctx, id, reviewerID, status, note)
	if err != nil {
		if errors.Is(err, models.ErrReportNotFound) {
			return nil, ErrReportNotFound
		}
		return nil, err
	}
	out := dto.ToReport(reviewed)
	return &out, nil
}

func (s *AdminService) Stats(ctx context.Context) (*dto.AdminStats, error) {
	stats, err := s.statsRepo.Stats(ctx)
	if err != nil {
		return nil, err
	}
	out := dto.ToAdminStats(stats)
	return &out, nil
}

func (s *AdminService) AuditLogs(ctx context.Context, f models.AuditLogListFilter) (*dto.Pagination, []dto.AuditLog, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Limit < 1 || f.Limit > 100 {
		f.Limit = 50
	}

	logs, total, err := s.auditLogRepo.List(ctx, f)
	if err != nil {
		return nil, nil, err
	}
	items := make([]dto.AuditLog, len(logs))
	for i := range logs {
		items[i] = dto.ToAuditLog(&logs[i])
	}
	pagination := dto.NewPagination(f.Page, f.Limit, total)
	return &pagination, items, nil
}
