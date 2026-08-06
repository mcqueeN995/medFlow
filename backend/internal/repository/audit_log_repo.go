package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/medflow/backend/internal/models"
)

type AuditLogRepo struct {
	pool *pgxpool.Pool
}

func NewAuditLogRepo(pool *pgxpool.Pool) *AuditLogRepo {
	return &AuditLogRepo{pool: pool}
}

// Create - metadata/ip_address не заполняются: ничего в этом модуле пока не
// генерирует структурированные метаданные сверх reason/target, а IP не
// прокидывается через сервисный слой (не специфицировано в openapi.yaml
// AuditLog как обязательное для использования). Обе колонки в БД остаются
// NULL - намеренный, видимый пробел, а не забытая часть.
func (r *AuditLogRepo) Create(ctx context.Context, entry *models.AuditLog) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO audit_logs (actor_id, action, target_type, target_id, reason)
		VALUES ($1, $2::audit_action, $3, $4, $5)
		RETURNING id, created_at
	`, entry.ActorID, string(entry.Action), entry.TargetType, entry.TargetID, entry.Reason).Scan(&entry.ID, &entry.CreatedAt)
}

func (r *AuditLogRepo) List(ctx context.Context, f models.AuditLogListFilter) ([]models.AuditLog, int, error) {
	where := "1=1"
	var args []any
	argN := 1

	if f.ActorID != nil {
		where += fmt.Sprintf(" AND al.actor_id = $%d", argN)
		args = append(args, *f.ActorID)
		argN++
	}
	if f.Action != nil {
		where += fmt.Sprintf(" AND al.action = $%d::audit_action", argN)
		args = append(args, string(*f.Action))
		argN++
	}
	if f.TargetType != nil && *f.TargetType != "" {
		where += fmt.Sprintf(" AND al.target_type = $%d", argN)
		args = append(args, *f.TargetType)
		argN++
	}
	if f.From != nil {
		where += fmt.Sprintf(" AND al.created_at >= $%d", argN)
		args = append(args, *f.From)
		argN++
	}
	if f.To != nil {
		where += fmt.Sprintf(" AND al.created_at <= $%d", argN)
		args = append(args, *f.To)
		argN++
	}

	var total int
	if err := r.pool.QueryRow(ctx, "SELECT count(*) FROM audit_logs al WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limitArg, offsetArg := argN, argN+1
	query := fmt.Sprintf(`
		SELECT al.id, al.actor_id, u.nickname, al.action, al.target_type, al.target_id, al.reason, al.metadata, al.ip_address, al.created_at
		FROM audit_logs al JOIN users u ON u.id = al.actor_id
		WHERE %s ORDER BY al.created_at DESC LIMIT $%d OFFSET $%d
	`, where, limitArg, offsetArg)
	dataArgs := append(append([]any{}, args...), f.Limit, (f.Page-1)*f.Limit)

	rows, err := r.pool.Query(ctx, query, dataArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []models.AuditLog
	for rows.Next() {
		var a models.AuditLog
		var action string
		if err := rows.Scan(&a.ID, &a.ActorID, &a.ActorNickname, &action, &a.TargetType, &a.TargetID, &a.Reason, &a.Metadata, &a.IPAddress, &a.CreatedAt); err != nil {
			return nil, 0, err
		}
		a.Action = models.AuditAction(action)
		out = append(out, a)
	}
	return out, total, rows.Err()
}
