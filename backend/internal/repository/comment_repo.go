package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/medflow/backend/internal/models"
)

type CommentRepo struct {
	pool *pgxpool.Pool
}

func NewCommentRepo(pool *pgxpool.Pool) *CommentRepo {
	return &CommentRepo{pool: pool}
}

const commentSelectColumns = `
	c.id, c.thread_id, c.parent_id, c.content, c.depth, c.likes_count,
	c.hidden_at, c.deleted_at, c.created_at, c.updated_at,
	u.id, u.nickname, u.university, u.course, u.faculty, u.created_at,
	(SELECT count(*) FROM threads t2 WHERE t2.author_id = u.id AND t2.deleted_at IS NULL)
`

func (r *CommentRepo) scanComment(row pgx.Row) (*models.Comment, error) {
	var c models.Comment
	err := row.Scan(
		&c.ID, &c.ThreadID, &c.ParentID, &c.Content, &c.Depth, &c.LikesCount,
		&c.HiddenAt, &c.DeletedAt, &c.CreatedAt, &c.UpdatedAt,
		&c.Author.ID, &c.Author.Nickname, &c.Author.University, &c.Author.Course, &c.Author.Faculty, &c.Author.CreatedAt,
		&c.Author.ThreadsCount,
	)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// Create вставляет комментарий и одновременно инкрементирует threads.comments_count
// в одной транзакции - счётчик обязан оставаться согласованным со строками comments.
func (r *CommentRepo) Create(ctx context.Context, threadID, authorID uuid.UUID, parentID *uuid.UUID, depth int, content string) (*models.Comment, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var id uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO comments (thread_id, parent_id, author_id, content, depth)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, threadID, parentID, authorID, content, depth).Scan(&id)
	if err != nil {
		return nil, err
	}

	if _, err := tx.Exec(ctx, `UPDATE threads SET comments_count = comments_count + 1 WHERE id = $1`, threadID); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.FindByID(ctx, id)
}

func (r *CommentRepo) FindByID(ctx context.Context, id uuid.UUID) (*models.Comment, error) {
	query := `SELECT ` + commentSelectColumns + ` FROM comments c JOIN users u ON u.id = c.author_id WHERE c.id = $1 AND c.deleted_at IS NULL`
	c, err := r.scanComment(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrCommentNotFound
		}
		return nil, err
	}
	return c, nil
}

func (r *CommentRepo) Update(ctx context.Context, id uuid.UUID, content string) (*models.Comment, error) {
	cmd, err := r.pool.Exec(ctx, `UPDATE comments SET content = $2, updated_at = now() WHERE id = $1 AND deleted_at IS NULL`, id, content)
	if err != nil {
		return nil, err
	}
	if cmd.RowsAffected() == 0 {
		return nil, models.ErrCommentNotFound
	}
	return r.FindByID(ctx, id)
}

func (r *CommentRepo) SoftDelete(ctx context.Context, id, threadID uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	cmd, err := tx.Exec(ctx, `UPDATE comments SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return models.ErrCommentNotFound
	}

	if _, err := tx.Exec(ctx, `UPDATE threads SET comments_count = GREATEST(comments_count - 1, 0) WHERE id = $1`, threadID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// ListByThread возвращает страницу комментариев верхнего уровня (depth=0) вместе
// со всеми их прямыми ответами (depth=1) - дерево ограничено двумя уровнями, см.
// UpdateComment/ForumService.CreateComment, где реплаи на реплаи "схлопываются".
func (r *CommentRepo) ListByThread(ctx context.Context, threadID uuid.UUID, page, limit int) ([]models.Comment, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM comments WHERE thread_id = $1 AND parent_id IS NULL AND deleted_at IS NULL`, threadID).Scan(&total); err != nil {
		return nil, 0, err
	}

	topQuery := `SELECT ` + commentSelectColumns + ` FROM comments c JOIN users u ON u.id = c.author_id
		WHERE c.thread_id = $1 AND c.parent_id IS NULL AND c.deleted_at IS NULL
		ORDER BY c.created_at ASC LIMIT $2 OFFSET $3`
	rows, err := r.pool.Query(ctx, topQuery, threadID, limit, (page-1)*limit)
	if err != nil {
		return nil, 0, err
	}

	var top []models.Comment
	var topIDs []uuid.UUID
	for rows.Next() {
		c, err := r.scanComment(rows)
		if err != nil {
			rows.Close()
			return nil, 0, err
		}
		top = append(top, *c)
		topIDs = append(topIDs, c.ID)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	if len(topIDs) == 0 {
		return top, total, nil
	}

	replyQuery := `SELECT ` + commentSelectColumns + ` FROM comments c JOIN users u ON u.id = c.author_id
		WHERE c.parent_id = ANY($1) AND c.deleted_at IS NULL
		ORDER BY c.created_at ASC`
	replyRows, err := r.pool.Query(ctx, replyQuery, topIDs)
	if err != nil {
		return nil, 0, err
	}
	defer replyRows.Close()

	repliesByParent := make(map[uuid.UUID][]models.Comment)
	for replyRows.Next() {
		c, err := r.scanComment(replyRows)
		if err != nil {
			return nil, 0, err
		}
		repliesByParent[*c.ParentID] = append(repliesByParent[*c.ParentID], *c)
	}
	if err := replyRows.Err(); err != nil {
		return nil, 0, err
	}

	for i := range top {
		top[i].Replies = repliesByParent[top[i].ID]
	}
	return top, total, nil
}
