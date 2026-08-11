package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/medflow/backend/internal/models"
)

type PasswordResetRepo struct {
	pool *pgxpool.Pool
}

func NewPasswordResetRepo(pool *pgxpool.Pool) *PasswordResetRepo {
	return &PasswordResetRepo{pool: pool}
}

func (r *PasswordResetRepo) Save(ctx context.Context, req *models.PasswordResetRequest) error {
	query := `
		INSERT INTO password_reset_requests (id, user_id, code_hash, expires_at)
		VALUES ($1, $2, $3, $4)
		RETURNING created_at
	`
	err := r.pool.QueryRow(ctx, query, req.ID, req.UserID, req.CodeHash, req.ExpiresAt).
		Scan(&req.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return models.ErrTokenHashExists
		}
		return err
	}
	return nil
}

func (r *PasswordResetRepo) FindByCodeHash(ctx context.Context, codeHash string) (*models.PasswordResetRequest, error) {
	query := `
		SELECT id, user_id, code_hash, expires_at, created_at
		FROM password_reset_requests
		WHERE code_hash = $1
	`
	var req models.PasswordResetRequest
	err := r.pool.QueryRow(ctx, query, codeHash).Scan(
		&req.ID, &req.UserID, &req.CodeHash, &req.ExpiresAt, &req.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrPasswordResetRequestNotFound
		}
		return nil, err
	}
	return &req, nil
}

func (r *PasswordResetRepo) DeleteByID(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM password_reset_requests WHERE id = $1`, id)
	return err
}

// DeleteByUserID - вызывается при создании новой заявки, чтобы у
// пользователя не копились старые неиспользованные коды (только один
// активный запрос на восстановление пароля одновременно).
func (r *PasswordResetRepo) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM password_reset_requests WHERE user_id = $1`, userID)
	return err
}
