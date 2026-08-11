package service

import (
	"context"
	"errors"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/medflow/backend/internal/dto"
	"github.com/medflow/backend/internal/models"
)

var ErrReportNotFound = errors.New("report not found")

// targetSnippetMaxRunes - длина превью текста цели жалобы в admin-панели;
// режем по рунам, а не байтам, чтобы не порвать кириллицу/юникод посередине.
const targetSnippetMaxRunes = 200

// AdminService - Reports/Stats/AuditLogs: части админки без естественного
// дома в доменном сервисе (Report не принадлежит только форуму - target_type
// может быть "card", см. CardService.ReportCard; Stats/AuditLogs изначально
// межмодульные).
//
// threadRepo/commentRepo/cardRepo нужны только для того, чтобы обогатить
// список жалоб контекстом цели (заголовок треда, превью текста, id для
// перехода) - см. reportContext.
type AdminService struct {
	reportRepo   ReportRepository
	auditLogRepo AuditLogRepository
	statsRepo    AdminStatsRepository
	threadRepo   ThreadRepository
	commentRepo  CommentRepository
	cardRepo     CardRepository
}

func NewAdminService(
	reportRepo ReportRepository,
	auditLogRepo AuditLogRepository,
	statsRepo AdminStatsRepository,
	threadRepo ThreadRepository,
	commentRepo CommentRepository,
	cardRepo CardRepository,
) *AdminService {
	return &AdminService{
		reportRepo:   reportRepo,
		auditLogRepo: auditLogRepo,
		statsRepo:    statsRepo,
		threadRepo:   threadRepo,
		commentRepo:  commentRepo,
		cardRepo:     cardRepo,
	}
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
		items[i] = dto.ToReport(&reports[i], s.reportContext(ctx, &reports[i]))
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
	out := dto.ToReport(reviewed, s.reportContext(ctx, reviewed))
	return &out, nil
}

// reportContext подтягивает заголовок/превью/ссылочные id цели жалобы по
// target_type - Report/ReportRepo сами по себе полиморфны и ничего не знают
// о threads/comments/cards. Не найдена цель (уже жёстко удалена) -> Removed.
func (s *AdminService) reportContext(ctx context.Context, r *models.Report) *dto.ReportTargetContext {
	switch r.TargetType {
	case "thread":
		t, err := s.threadRepo.FindByID(ctx, r.TargetID)
		if err != nil {
			return &dto.ReportTargetContext{Removed: true}
		}
		title, snippet := t.Title, truncateSnippet(t.Content)
		return &dto.ReportTargetContext{
			ThreadID: &t.ID,
			Title:    &title,
			Snippet:  &snippet,
			Removed:  t.HiddenAt != nil || t.DeletedAt != nil,
		}
	case "comment":
		c, err := s.commentRepo.FindByID(ctx, r.TargetID)
		if err != nil {
			return &dto.ReportTargetContext{Removed: true}
		}
		snippet := truncateSnippet(c.Content)
		out := &dto.ReportTargetContext{
			ThreadID: &c.ThreadID,
			Snippet:  &snippet,
			Removed:  c.HiddenAt != nil || c.DeletedAt != nil,
		}
		if t, err := s.threadRepo.FindByID(ctx, c.ThreadID); err == nil {
			out.Title = &t.Title
		}
		return out
	case "card":
		card, err := s.cardRepo.FindByID(ctx, r.TargetID)
		if err != nil {
			return &dto.ReportTargetContext{Removed: true}
		}
		snippet := truncateSnippet(card.Question)
		return &dto.ReportTargetContext{TaskID: &card.TaskID, Snippet: &snippet}
	default:
		return nil
	}
}

func truncateSnippet(s string) string {
	if utf8.RuneCountInString(s) <= targetSnippetMaxRunes {
		return s
	}
	runes := []rune(s)
	return string(runes[:targetSnippetMaxRunes]) + "…"
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
