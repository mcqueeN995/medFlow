package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/medflow/backend/internal/models"
)

type ThreadRepo struct {
	pool *pgxpool.Pool
}

func NewThreadRepo(pool *pgxpool.Pool) *ThreadRepo {
	return &ThreadRepo{pool: pool}
}

// tags хранится как thread_tag[] (enum-массив без известного pgx OID), поэтому
// везде читаем/пишем через явный ::text[]/::thread_tag[] каст вместо
// регистрации типа в pgtype.Map.
const threadSelectColumns = `
	t.id, t.title, t.content, t.tags::text[], t.views_count, t.likes_count, t.comments_count,
	t.hidden_at, t.hidden_by, t.hidden_reason, t.deleted_at, t.created_at, t.updated_at,
	u.id, u.nickname, u.university, u.course, u.faculty, u.created_at,
	(SELECT count(*) FROM threads t2 WHERE t2.author_id = u.id AND t2.deleted_at IS NULL)
`

func (r *ThreadRepo) scanThread(row pgx.Row) (*models.Thread, error) {
	var t models.Thread
	var tags []string
	err := row.Scan(
		&t.ID, &t.Title, &t.Content, &tags, &t.ViewsCount, &t.LikesCount, &t.CommentsCount,
		&t.HiddenAt, &t.HiddenBy, &t.HiddenReason, &t.DeletedAt, &t.CreatedAt, &t.UpdatedAt,
		&t.Author.ID, &t.Author.Nickname, &t.Author.University, &t.Author.Course, &t.Author.Faculty, &t.Author.CreatedAt,
		&t.Author.ThreadsCount,
	)
	if err != nil {
		return nil, err
	}
	t.Tags = stringsToTags(tags)
	return &t, nil
}

func (r *ThreadRepo) Create(ctx context.Context, authorID uuid.UUID, title, content string, tags []models.ThreadTag) (*models.Thread, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `
		INSERT INTO threads (author_id, title, content, tags)
		VALUES ($1, $2, $3, $4::thread_tag[])
		RETURNING id
	`, authorID, title, content, tagsToStrings(tags)).Scan(&id)
	if err != nil {
		return nil, err
	}
	return r.FindByID(ctx, id)
}

func (r *ThreadRepo) FindByID(ctx context.Context, id uuid.UUID) (*models.Thread, error) {
	query := `SELECT ` + threadSelectColumns + ` FROM threads t JOIN users u ON u.id = t.author_id WHERE t.id = $1 AND t.deleted_at IS NULL`
	t, err := r.scanThread(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrThreadNotFound
		}
		return nil, err
	}
	return t, nil
}

// IncrementViewsIfNotRecentlyViewed засчитывает просмотр не чаще раза в 24
// часа на пользователя - см. миграцию 000015_thread_views. WHERE в ON
// CONFLICT DO UPDATE решает, обновлять ли строку: если last_viewed_at свежее
// 24 часов, обновления не происходит и RowsAffected() == 0 (аналог DO
// NOTHING для этой конкретной строки), views_count не трогаем.
func (r *ThreadRepo) IncrementViewsIfNotRecentlyViewed(ctx context.Context, threadID, userID uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	cmd, err := tx.Exec(ctx, `
		INSERT INTO thread_views (user_id, thread_id, last_viewed_at)
		VALUES ($1, $2, now())
		ON CONFLICT (user_id, thread_id) DO UPDATE SET last_viewed_at = now()
		WHERE thread_views.last_viewed_at < now() - interval '24 hours'
	`, userID, threadID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return models.ErrThreadNotFound
		}
		return err
	}

	if cmd.RowsAffected() > 0 {
		if _, err := tx.Exec(ctx, `UPDATE threads SET views_count = views_count + 1 WHERE id = $1`, threadID); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *ThreadRepo) Update(ctx context.Context, id uuid.UUID, title, content string, tags []models.ThreadTag) (*models.Thread, error) {
	cmd, err := r.pool.Exec(ctx, `
		UPDATE threads SET title = $2, content = $3, tags = $4::thread_tag[], updated_at = now()
		WHERE id = $1 AND deleted_at IS NULL
	`, id, title, content, tagsToStrings(tags))
	if err != nil {
		return nil, err
	}
	if cmd.RowsAffected() == 0 {
		return nil, models.ErrThreadNotFound
	}
	return r.FindByID(ctx, id)
}

func (r *ThreadRepo) Hide(ctx context.Context, id, hiddenBy uuid.UUID, reason string) (*models.Thread, error) {
	cmd, err := r.pool.Exec(ctx, `
		UPDATE threads SET hidden_at = now(), hidden_by = $2, hidden_reason = $3 WHERE id = $1 AND deleted_at IS NULL
	`, id, hiddenBy, reason)
	if err != nil {
		return nil, err
	}
	if cmd.RowsAffected() == 0 {
		return nil, models.ErrThreadNotFound
	}
	return r.FindByID(ctx, id)
}

func (r *ThreadRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	cmd, err := r.pool.Exec(ctx, `UPDATE threads SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return models.ErrThreadNotFound
	}
	return nil
}

func (r *ThreadRepo) List(ctx context.Context, f models.ThreadListFilter) ([]models.Thread, int, error) {
	where := "t.deleted_at IS NULL AND t.hidden_at IS NULL"
	var args []any
	argN := 1
	if f.Tag != nil {
		where += fmt.Sprintf(" AND $%d = ANY(t.tags::text[])", argN)
		args = append(args, string(*f.Tag))
		argN++
	}
	if f.AuthorID != nil {
		where += fmt.Sprintf(" AND t.author_id = $%d", argN)
		args = append(args, *f.AuthorID)
		argN++
	}

	var total int
	countQuery := "SELECT count(*) FROM threads t WHERE " + where
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	orderBy := "t.created_at DESC"
	if f.Sort == "popular" {
		orderBy = "(t.likes_count + t.comments_count) DESC, t.created_at DESC"
	}

	limitArg, offsetArg := argN, argN+1
	dataQuery := fmt.Sprintf(`
		SELECT %s FROM threads t JOIN users u ON u.id = t.author_id
		WHERE %s ORDER BY %s LIMIT $%d OFFSET $%d
	`, threadSelectColumns, where, orderBy, limitArg, offsetArg)

	dataArgs := append(append([]any{}, args...), f.Limit, (f.Page-1)*f.Limit)

	rows, err := r.pool.Query(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var threads []models.Thread
	for rows.Next() {
		t, err := r.scanThread(rows)
		if err != nil {
			return nil, 0, err
		}
		threads = append(threads, *t)
	}
	return threads, total, rows.Err()
}

func tagsToStrings(tags []models.ThreadTag) []string {
	out := make([]string, len(tags))
	for i, t := range tags {
		out[i] = string(t)
	}
	return out
}

func stringsToTags(tags []string) []models.ThreadTag {
	if tags == nil {
		return nil
	}
	out := make([]models.ThreadTag, len(tags))
	for i, t := range tags {
		out[i] = models.ThreadTag(t)
	}
	return out
}
