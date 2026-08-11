package repository

import (
	"context"
	"errors"
	"fmt"
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
			id, email, password_hash, login, nickname, role, university, course, faculty
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)
		RETURNING created_at, updated_at
	`

	err := r.pool.QueryRow(ctx, query,
		user.ID, user.Email, user.PasswordHash, user.Login, user.Nickname,
		user.Role, user.University, user.Course, user.Faculty,
	).Scan(&user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			if strings.Contains(pgErr.ConstraintName, "email") {
				return models.ErrEmailAlreadyExists
			}
			if strings.Contains(pgErr.ConstraintName, "login") {
				return models.ErrLoginExists
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
		SELECT id, email, password_hash, login, nickname, role, university, course, faculty,
		       email_verified_at, banned_at, ban_reason, banned_by, deleted_at, created_at, updated_at
		FROM users
		WHERE email = $1 AND deleted_at IS NULL
	`
	return r.scanUser(ctx, query, email)
}

func (r *UserRepo) FindByLogin(ctx context.Context, login string) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, login, nickname, role, university, course, faculty,
		       email_verified_at, banned_at, ban_reason, banned_by, deleted_at, created_at, updated_at
		FROM users
		WHERE login = $1 AND deleted_at IS NULL
	`
	return r.scanUser(ctx, query, login)
}

func (r *UserRepo) FindByNickname(ctx context.Context, nickname string) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, login, nickname, role, university, course, faculty,
		       email_verified_at, banned_at, ban_reason, banned_by, deleted_at, created_at, updated_at
		FROM users
		WHERE nickname = $1 AND deleted_at IS NULL
	`
	return r.scanUser(ctx, query, nickname)
}

func (r *UserRepo) FindByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, login, nickname, role, university, course, faculty,
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

// UpdateLogin - финальный шаг смены login (см. UserService.ConfirmLoginChange),
// отдельно от Update, т.к. login не входит в UpdateProfileRequest и требует
// собственной проверки уникальности constraint'а.
func (r *UserRepo) UpdateLogin(ctx context.Context, id uuid.UUID, login string) (*models.User, error) {
	cmd, err := r.pool.Exec(ctx, `UPDATE users SET login = $2, updated_at = now() WHERE id = $1 AND deleted_at IS NULL`, id, login)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && strings.Contains(pgErr.ConstraintName, "login") {
			return nil, models.ErrLoginExists
		}
		return nil, err
	}
	if cmd.RowsAffected() == 0 {
		return nil, models.ErrUserNotFound
	}
	return r.FindByID(ctx, id)
}

// UpdatePassword - используется восстановлением пароля (AuthService.ConfirmPasswordReset),
// отдельно от Update, т.к. password_hash не входит в UpdateProfileRequest.
func (r *UserRepo) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	cmd, err := r.pool.Exec(ctx, `UPDATE users SET password_hash = $2, updated_at = now() WHERE id = $1 AND deleted_at IS NULL`, id, passwordHash)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return models.ErrUserNotFound
	}
	return nil
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

const userSelectColumns = `
	id, email, password_hash, login, nickname, role, university, course, faculty,
	email_verified_at, banned_at, ban_reason, banned_by, deleted_at, created_at, updated_at
`

// AdminList - для /admin/users: живые (не мягко удалённые) пользователи с
// фильтрами по роли/бану/университету/поиску по nickname+email.
func (r *UserRepo) AdminList(ctx context.Context, f models.AdminUserListFilter) ([]models.User, int, error) {
	where := "deleted_at IS NULL"
	var args []any
	argN := 1

	if f.Role != nil {
		where += fmt.Sprintf(" AND role = $%d::user_role", argN)
		args = append(args, string(*f.Role))
		argN++
	}
	if f.Banned != nil {
		if *f.Banned {
			where += " AND banned_at IS NOT NULL"
		} else {
			where += " AND banned_at IS NULL"
		}
	}
	if f.University != nil {
		where += fmt.Sprintf(" AND university = $%d::university", argN)
		args = append(args, string(*f.University))
		argN++
	}
	if f.Q != nil && *f.Q != "" {
		where += fmt.Sprintf(" AND (nickname ILIKE $%d OR email ILIKE $%d)", argN, argN)
		args = append(args, "%"+*f.Q+"%")
		argN++
	}

	var total int
	if err := r.pool.QueryRow(ctx, "SELECT count(*) FROM users WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limitArg, offsetArg := argN, argN+1
	query := fmt.Sprintf(`SELECT %s FROM users WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		userSelectColumns, where, limitArg, offsetArg)
	dataArgs := append(append([]any{}, args...), f.Limit, (f.Page-1)*f.Limit)

	rows, err := r.pool.Query(ctx, query, dataArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(
			&u.ID, &u.Email, &u.PasswordHash, &u.Login, &u.Nickname, &u.Role, &u.University, &u.Course, &u.Faculty,
			&u.EmailVerifiedAt, &u.BannedAt, &u.BanReason, &u.BannedBy, &u.DeletedAt, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		out = append(out, u)
	}
	return out, total, rows.Err()
}

func (r *UserRepo) ChangeRole(ctx context.Context, id uuid.UUID, role models.UserRole) (*models.User, error) {
	cmd, err := r.pool.Exec(ctx, `UPDATE users SET role = $2::user_role, updated_at = now() WHERE id = $1 AND deleted_at IS NULL`, id, string(role))
	if err != nil {
		return nil, err
	}
	if cmd.RowsAffected() == 0 {
		return nil, models.ErrUserNotFound
	}
	return r.FindByID(ctx, id)
}

func (r *UserRepo) Ban(ctx context.Context, id, bannedBy uuid.UUID, reason string) (*models.User, error) {
	cmd, err := r.pool.Exec(ctx, `
		UPDATE users SET banned_at = now(), ban_reason = $2, banned_by = $3, updated_at = now()
		WHERE id = $1 AND deleted_at IS NULL
	`, id, reason, bannedBy)
	if err != nil {
		return nil, err
	}
	if cmd.RowsAffected() == 0 {
		return nil, models.ErrUserNotFound
	}
	return r.FindByID(ctx, id)
}

func (r *UserRepo) Unban(ctx context.Context, id uuid.UUID) (*models.User, error) {
	cmd, err := r.pool.Exec(ctx, `
		UPDATE users SET banned_at = NULL, ban_reason = NULL, banned_by = NULL, updated_at = now()
		WHERE id = $1 AND deleted_at IS NULL
	`, id)
	if err != nil {
		return nil, err
	}
	if cmd.RowsAffected() == 0 {
		return nil, models.ErrUserNotFound
	}
	return r.FindByID(ctx, id)
}

func (r *UserRepo) scanUser(ctx context.Context, query string, args ...any) (*models.User, error) {
	var user models.User

	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Login, &user.Nickname,
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
