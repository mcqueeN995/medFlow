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

type LoginChangeRepo struct {
	pool *pgxpool.Pool
}

func NewLoginChangeRepo(pool *pgxpool.Pool) *LoginChangeRepo {
	return &LoginChangeRepo{pool: pool}
}

func (r *LoginChangeRepo) Save(ctx context.Context, req *models.LoginChangeRequest) error {
	query := `
		INSERT INTO login_change_requests (id, user_id, new_login, code_hash, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at
	`
	err := r.pool.QueryRow(ctx, query, req.ID, req.UserID, req.NewLogin, req.CodeHash, req.ExpiresAt).
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

func (r *LoginChangeRepo) FindByCodeHash(ctx context.Context, codeHash string) (*models.LoginChangeRequest, error) {
	query := `
		SELECT id, user_id, new_login, code_hash, expires_at, created_at
		FROM login_change_requests
		WHERE code_hash = $1
	`
	var req models.LoginChangeRequest
	err := r.pool.QueryRow(ctx, query, codeHash).Scan(
		&req.ID, &req.UserID, &req.NewLogin, &req.CodeHash, &req.ExpiresAt, &req.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrLoginChangeRequestNotFound
		}
		return nil, err
	}
	return &req, nil
}

func (r *LoginChangeRepo) DeleteByID(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM login_change_requests WHERE id = $1`, id)
	return err
}

// DeleteByUserID - вызывается при создании новой заявки, чтобы у
// пользователя не копились старые неиспользованные коды (только один
// активный запрос на смену login одновременно).
func (r *LoginChangeRepo) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM login_change_requests WHERE user_id = $1`, userID)
	return err
}
