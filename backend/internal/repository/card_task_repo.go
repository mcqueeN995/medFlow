package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/medflow/backend/internal/models"
)

type CardTaskRepo struct {
	pool *pgxpool.Pool
}

func NewCardTaskRepo(pool *pgxpool.Pool) *CardTaskRepo {
	return &CardTaskRepo{pool: pool}
}

// position_in_queue/estimated_wait_seconds - колонки в схеме БД, но
// намеренно не заполняются: значение осмысленно только "на текущий момент",
// поэтому CardService вычисляет их на лету при чтении (см.
// CardTaskRepository.CountPendingBefore), а не хранит устаревающий снимок.
const cardTaskSelectColumns = `
	id, user_id, textbook_id, source_type, topic, difficulty, cards_count, cache_key,
	temp_s3_key, temp_file_name, temp_file_size, status, error_message, started_at, finished_at, created_at, share_token
`

func (r *CardTaskRepo) scan(row pgx.Row) (*models.CardTask, error) {
	var t models.CardTask
	var sourceType, status string
	var difficulty *string
	err := row.Scan(
		&t.ID, &t.UserID, &t.TextbookID, &sourceType, &t.Topic, &difficulty, &t.CardsCount, &t.CacheKey,
		&t.TempS3Key, &t.TempFileName, &t.TempFileSize, &status, &t.ErrorMessage, &t.StartedAt, &t.FinishedAt, &t.CreatedAt, &t.ShareToken,
	)
	if err != nil {
		return nil, err
	}
	t.SourceType = models.CardTaskSourceType(sourceType)
	t.Status = models.CardTaskStatus(status)
	if difficulty != nil {
		t.Difficulty = models.CardDifficulty(*difficulty)
	} else {
		t.Difficulty = models.DifficultyMedium
	}
	return &t, nil
}

func (r *CardTaskRepo) Create(ctx context.Context, t *models.CardTask) (*models.CardTask, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `
		INSERT INTO card_tasks (user_id, textbook_id, source_type, topic, difficulty, cards_count, cache_key,
			temp_s3_key, temp_file_name, temp_file_size, status)
		VALUES ($1, $2, $3::card_task_source_type, $4, $5::card_difficulty, $6, $7, $8, $9, $10, 'pending')
		RETURNING id
	`, t.UserID, t.TextbookID, string(t.SourceType), t.Topic, string(t.Difficulty), t.CardsCount, t.CacheKey,
		t.TempS3Key, t.TempFileName, t.TempFileSize).Scan(&id)
	if err != nil {
		return nil, err
	}
	return r.FindByID(ctx, id)
}

func (r *CardTaskRepo) FindByID(ctx context.Context, id uuid.UUID) (*models.CardTask, error) {
	query := `SELECT ` + cardTaskSelectColumns + ` FROM card_tasks WHERE id = $1`
	t, err := r.scan(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrCardTaskNotFound
		}
		return nil, err
	}
	return t, nil
}

func (r *CardTaskRepo) List(ctx context.Context, f models.CardTaskListFilter) ([]models.CardTask, int, error) {
	where := "user_id = $1"
	args := []any{f.UserID}
	if f.Status != nil {
		where += " AND status = $2::card_task_status"
		args = append(args, string(*f.Status))
	}

	var total int
	if err := r.pool.QueryRow(ctx, "SELECT count(*) FROM card_tasks WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limitArg, offsetArg := len(args)+1, len(args)+2
	query := fmt.Sprintf(`SELECT %s FROM card_tasks WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		cardTaskSelectColumns, where, limitArg, offsetArg)
	dataArgs := append(append([]any{}, args...), f.Limit, (f.Page-1)*f.Limit)

	rows, err := r.pool.Query(ctx, query, dataArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []models.CardTask
	for rows.Next() {
		t, err := r.scan(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *t)
	}
	return out, total, rows.Err()
}

func (r *CardTaskRepo) MarkProcessing(ctx context.Context, id uuid.UUID) error {
	cmd, err := r.pool.Exec(ctx, `UPDATE card_tasks SET status = 'processing', started_at = now() WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return models.ErrCardTaskNotFound
	}
	return nil
}

func (r *CardTaskRepo) MarkDone(ctx context.Context, id uuid.UUID, cardsCount int) error {
	cmd, err := r.pool.Exec(ctx, `
		UPDATE card_tasks SET status = 'done', cards_count = $2, finished_at = now() WHERE id = $1
	`, id, cardsCount)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return models.ErrCardTaskNotFound
	}
	return nil
}

func (r *CardTaskRepo) MarkFailed(ctx context.Context, id uuid.UUID, errMsg string) error {
	cmd, err := r.pool.Exec(ctx, `
		UPDATE card_tasks SET status = 'failed', error_message = $2, finished_at = now() WHERE id = $1
	`, id, errMsg)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return models.ErrCardTaskNotFound
	}
	return nil
}

func (r *CardTaskRepo) FindDoneByCacheKey(ctx context.Context, cacheKey string) (*models.CardTask, error) {
	query := `SELECT ` + cardTaskSelectColumns + ` FROM card_tasks WHERE cache_key = $1 AND status = 'done' ORDER BY created_at DESC LIMIT 1`
	t, err := r.scan(r.pool.QueryRow(ctx, query, cacheKey))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrCardTaskNotFound
		}
		return nil, err
	}
	return t, nil
}

func (r *CardTaskRepo) CountActive(ctx context.Context, userID uuid.UUID) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `
		SELECT count(*) FROM card_tasks WHERE user_id = $1 AND status IN ('pending', 'processing')
	`, userID).Scan(&n)
	return n, err
}

func (r *CardTaskRepo) CountPendingBefore(ctx context.Context, createdAt time.Time) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `
		SELECT count(*) FROM card_tasks WHERE status = 'pending' AND created_at < $1
	`, createdAt).Scan(&n)
	return n, err
}

// ListCatalogFeed - лента уже сгенерированных наборов из каталога учебников:
// один канонический task на cache_key (DISTINCT ON, первый дошедший до done -
// клоны других пользователей по тому же cache_key в ленте не дублируются),
// отсортировано по created_at этого канонического task'а, самые новые темы
// первыми.
func (r *CardTaskRepo) ListCatalogFeed(ctx context.Context, f models.CardCatalogFeedFilter) ([]models.CardCatalogEntry, int, error) {
	where := "t.source_type = 'catalog_textbook' AND t.status = 'done'"
	var args []any
	argN := 1
	if f.TextbookID != nil {
		where += fmt.Sprintf(" AND t.textbook_id = $%d", argN)
		args = append(args, *f.TextbookID)
		argN++
	}
	if f.Difficulty != nil {
		where += fmt.Sprintf(" AND t.difficulty = $%d::card_difficulty", argN)
		args = append(args, string(*f.Difficulty))
		argN++
	}
	if f.Q != nil && *f.Q != "" {
		where += fmt.Sprintf(" AND (t.topic ILIKE $%d OR tb.title ILIKE $%d)", argN, argN)
		args = append(args, "%"+*f.Q+"%")
		argN++
	}

	countQuery := fmt.Sprintf(`
		SELECT count(*) FROM (
			SELECT DISTINCT ON (t.cache_key) t.id
			FROM card_tasks t JOIN textbooks tb ON tb.id = t.textbook_id
			WHERE %s
		) sub
	`, where)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limitArg, offsetArg := argN, argN+1
	dataQuery := fmt.Sprintf(`
		SELECT sub.id, sub.textbook_id, sub.textbook_title, sub.topic, sub.difficulty, sub.cards_count, sub.created_at
		FROM (
			SELECT DISTINCT ON (t.cache_key)
				t.id, t.textbook_id, tb.title AS textbook_title, t.topic, t.difficulty, t.cards_count, t.created_at
			FROM card_tasks t JOIN textbooks tb ON tb.id = t.textbook_id
			WHERE %s
			ORDER BY t.cache_key, t.created_at ASC
		) sub
		ORDER BY sub.created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, limitArg, offsetArg)
	dataArgs := append(append([]any{}, args...), f.Limit, (f.Page-1)*f.Limit)

	rows, err := r.pool.Query(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []models.CardCatalogEntry
	for rows.Next() {
		var e models.CardCatalogEntry
		var difficulty string
		var cardsCount *int
		if err := rows.Scan(&e.TaskID, &e.TextbookID, &e.TextbookTitle, &e.Topic, &difficulty, &cardsCount, &e.CreatedAt); err != nil {
			return nil, 0, err
		}
		e.Difficulty = models.CardDifficulty(difficulty)
		if cardsCount != nil {
			e.CardsCount = *cardsCount
		}
		out = append(out, e)
	}
	return out, total, rows.Err()
}

func (r *CardTaskRepo) SetShareToken(ctx context.Context, taskID uuid.UUID, token string) error {
	cmd, err := r.pool.Exec(ctx, `UPDATE card_tasks SET share_token = $2 WHERE id = $1`, taskID, token)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return models.ErrCardTaskNotFound
	}
	return nil
}

func (r *CardTaskRepo) ClearShareToken(ctx context.Context, taskID uuid.UUID) error {
	cmd, err := r.pool.Exec(ctx, `UPDATE card_tasks SET share_token = NULL WHERE id = $1`, taskID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return models.ErrCardTaskNotFound
	}
	return nil
}

func (r *CardTaskRepo) FindByShareToken(ctx context.Context, token string) (*models.CardTask, error) {
	query := `SELECT ` + cardTaskSelectColumns + ` FROM card_tasks WHERE share_token = $1`
	t, err := r.scan(r.pool.QueryRow(ctx, query, token))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrCardTaskNotFound
		}
		return nil, err
	}
	return t, nil
}
