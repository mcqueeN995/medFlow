package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/medflow/backend/internal/models"
)

type ReportRepo struct {
	pool *pgxpool.Pool
}

func NewReportRepo(pool *pgxpool.Pool) *ReportRepo {
	return &ReportRepo{pool: pool}
}

const reportSelectColumns = `
	id, reporter_id, target_type, target_id, reason, status, reviewed_by, reviewed_at, resolution_note, created_at
`

func (r *ReportRepo) scan(row pgx.Row) (*models.Report, error) {
	var rep models.Report
	err := row.Scan(&rep.ID, &rep.ReporterID, &rep.TargetType, &rep.TargetID, &rep.Reason, &rep.Status,
		&rep.ReviewedBy, &rep.ReviewedAt, &rep.ResolutionNote, &rep.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &rep, nil
}

func (r *ReportRepo) Create(ctx context.Context, reporterID uuid.UUID, targetType string, targetID uuid.UUID, reason string) (*models.Report, error) {
	report := &models.Report{
		ReporterID: reporterID,
		TargetType: targetType,
		TargetID:   targetID,
		Reason:     reason,
		Status:     models.ReportStatusPending,
	}

	err := r.pool.QueryRow(ctx, `
		INSERT INTO reports (reporter_id, target_type, target_id, reason)
		VALUES ($1, $2, $3, $4)
		RETURNING id, status, created_at
	`, reporterID, targetType, targetID, reason).Scan(&report.ID, &report.Status, &report.CreatedAt)
	if err != nil {
		return nil, err
	}
	return report, nil
}

func (r *ReportRepo) FindByID(ctx context.Context, id uuid.UUID) (*models.Report, error) {
	query := `SELECT ` + reportSelectColumns + ` FROM reports WHERE id = $1`
	rep, err := r.scan(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrReportNotFound
		}
		return nil, err
	}
	return rep, nil
}

func (r *ReportRepo) List(ctx context.Context, f models.ReportListFilter) ([]models.Report, int, error) {
	where := "1=1"
	var args []any
	argN := 1

	if f.Status != nil {
		where += fmt.Sprintf(" AND status = $%d::report_status", argN)
		args = append(args, string(*f.Status))
		argN++
	}
	if f.TargetType != nil && *f.TargetType != "" {
		where += fmt.Sprintf(" AND target_type = $%d", argN)
		args = append(args, *f.TargetType)
		argN++
	}

	var total int
	if err := r.pool.QueryRow(ctx, "SELECT count(*) FROM reports WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limitArg, offsetArg := argN, argN+1
	query := fmt.Sprintf(`SELECT %s FROM reports WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		reportSelectColumns, where, limitArg, offsetArg)
	dataArgs := append(append([]any{}, args...), f.Limit, (f.Page-1)*f.Limit)

	rows, err := r.pool.Query(ctx, query, dataArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []models.Report
	for rows.Next() {
		rep, err := r.scan(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *rep)
	}
	return out, total, rows.Err()
}

func (r *ReportRepo) Review(ctx context.Context, id, reviewedBy uuid.UUID, status models.ReportStatus, note *string) (*models.Report, error) {
	cmd, err := r.pool.Exec(ctx, `
		UPDATE reports SET status = $2::report_status, reviewed_by = $3, reviewed_at = now(), resolution_note = $4
		WHERE id = $1
	`, id, string(status), reviewedBy, note)
	if err != nil {
		return nil, err
	}
	if cmd.RowsAffected() == 0 {
		return nil, models.ErrReportNotFound
	}
	return r.FindByID(ctx, id)
}
