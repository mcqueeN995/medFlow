package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/medflow/backend/internal/models"
)

type UploadRepo struct {
	pool *pgxpool.Pool
}

func NewUploadRepo(pool *pgxpool.Pool) *UploadRepo {
	return &UploadRepo{pool: pool}
}

func (r *UploadRepo) Create(ctx context.Context, u *models.Upload) (*models.Upload, error) {
	var id uuid.UUID
	var createdAt = u.CreatedAt
	err := r.pool.QueryRow(ctx, `
		INSERT INTO uploads (uploader_id, upload_type, s3_key, mime_type, size_bytes, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`, u.UploaderID, u.UploadType, u.S3Key, u.MimeType, u.SizeBytes, u.ExpiresAt).Scan(&id, &createdAt)
	if err != nil {
		return nil, err
	}
	u.ID = id
	u.CreatedAt = createdAt
	return u, nil
}

func (r *UploadRepo) FindByID(ctx context.Context, id uuid.UUID) (*models.Upload, error) {
	var u models.Upload
	err := r.pool.QueryRow(ctx, `
		SELECT id, uploader_id, upload_type, s3_key, mime_type, size_bytes, expires_at, created_at
		FROM uploads WHERE id = $1
	`, id).Scan(&u.ID, &u.UploaderID, &u.UploadType, &u.S3Key, &u.MimeType, &u.SizeBytes, &u.ExpiresAt, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrUploadNotFound
		}
		return nil, err
	}
	return &u, nil
}
