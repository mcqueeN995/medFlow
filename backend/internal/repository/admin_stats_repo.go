package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/medflow/backend/internal/models"
)

type AdminStatsRepo struct {
	pool *pgxpool.Pool
}

func NewAdminStatsRepo(pool *pgxpool.Pool) *AdminStatsRepo {
	return &AdminStatsRepo{pool: pool}
}

func (r *AdminStatsRepo) Stats(ctx context.Context) (models.AdminStats, error) {
	var s models.AdminStats
	err := r.pool.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM users WHERE deleted_at IS NULL),
			(SELECT count(*) FROM users WHERE deleted_at IS NULL AND banned_at IS NOT NULL),
			(SELECT count(*) FROM threads WHERE deleted_at IS NULL),
			(SELECT count(*) FROM card_tasks),
			(SELECT count(*) FROM card_tasks WHERE status = 'pending'),
			(SELECT count(*) FROM refresh_tokens WHERE expires_at > now())
	`).Scan(&s.UsersTotal, &s.UsersBanned, &s.ThreadsTotal, &s.CardTasksTotal, &s.CardTasksPending, &s.ActiveSessions)
	return s, err
}
