package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/medflow/backend/internal/models"
)

type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

func (r *UserRepo) Create(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (
			id, email, password_hash, nickname, role, university, course, faculty
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		)
		RETURNING created_at, updated_at
	`

	err := r.pool.QueryRow(ctx, query,
		user.ID, user.Email, user.PasswordHash, user.Nickname,
		user.Role, user.University, user.Course, user.Faculty,
	).Scan(&user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			if strings.Contains(pgErr.ConstraintName, "email") {
				return models.ErrEmailAlreadyExists
			}
			if strings.Contains(pgErr.ConstraintName, "nickname") {
				return models.ErrNicknameExists
			}
		}
		return err
	}

	return nil
}

func (r *UserRepo) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, nickname, role, university, course, faculty,
		       email_verified_at, banned_at, ban_reason, banned_by, deleted_at, created_at, updated_at
		FROM users
		WHERE email = $1 AND deleted_at IS NULL
	`
	return r.scanUser(ctx, query, email)
}

func (r *UserRepo) FindByNickname(ctx context.Context, nickname string) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, nickname, role, university, course, faculty,
		       email_verified_at, banned_at, ban_reason, banned_by, deleted_at, created_at, updated_at
		FROM users
		WHERE nickname = $1 AND deleted_at IS NULL
	`
	return r.scanUser(ctx, query, nickname)
}

func (r *UserRepo) FindByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, nickname, role, university, course, faculty,
		       email_verified_at, banned_at, ban_reason, banned_by, deleted_at, created_at, updated_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`
	return r.scanUser(ctx, query, id)
}

// Update редактирует профиль (nickname/university/course/faculty) - только
// эти поля разрешены к изменению самим пользователем per UpdateProfileRequest.
func (r *UserRepo) Update(ctx context.Context, id uuid.UUID, nickname string, university *models.University, course *int, faculty *string) (*models.User, error) {
	query := `
		UPDATE users SET nickname = $2, university = $3, course = $4, faculty = $5, updated_at = now()
		WHERE id = $1 AND deleted_at IS NULL
	`
	cmd, err := r.pool.Exec(ctx, query, id, nickname, university, course, faculty)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && strings.Contains(pgErr.ConstraintName, "nickname") {
			return nil, models.ErrNicknameExists
		}
		return nil, err
	}
	if cmd.RowsAffected() == 0 {
		return nil, models.ErrUserNotFound
	}
	return r.FindByID(ctx, id)
}

func (r *UserRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	cmd, err := r.pool.Exec(ctx, `UPDATE users SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return models.ErrUserNotFound
	}
	return nil
}

// FindPublicByID собирает models.PublicUser (тот же тип, что использует
// форум) джойном с подсчётом тредов - отдельного репозитория для этой
// read-модели нет, см. models/forum.go.
func (r *UserRepo) FindPublicByID(ctx context.Context, id uuid.UUID) (*models.PublicUser, error) {
	query := `
		SELECT u.id, u.nickname, u.university, u.course, u.faculty, u.created_at,
		       (SELECT count(*) FROM threads t WHERE t.author_id = u.id AND t.deleted_at IS NULL)
		FROM users u
		WHERE u.id = $1 AND u.deleted_at IS NULL
	`
	var pu models.PublicUser
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&pu.ID, &pu.Nickname, &pu.University, &pu.Course, &pu.Faculty, &pu.CreatedAt, &pu.ThreadsCount,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrUserNotFound
		}
		return nil, err
	}
	return &pu, nil
}

func (r *UserRepo) scanUser(ctx context.Context, query string, args ...any) (*models.User, error) {
	var user models.User

	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Nickname,
		&user.Role, &user.University, &user.Course, &user.Faculty,
		&user.EmailVerifiedAt, &user.BannedAt, &user.BanReason, &user.BannedBy, &user.DeletedAt,
		&user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrUserNotFound
		}
		return nil, err
	}

	return &user, nil
}
