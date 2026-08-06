package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/medflow/backend/internal/models"
)

type CardRepo struct {
	pool *pgxpool.Pool
}

func NewCardRepo(pool *pgxpool.Pool) *CardRepo {
	return &CardRepo{pool: pool}
}

const cardSelectColumns = `
	id, task_id, textbook_id, chapter, topic, subtopic, question, answer,
	page_approx, source_reference, difficulty, report_count, created_at
`

func (r *CardRepo) scan(row pgx.Row) (*models.Card, error) {
	var c models.Card
	var difficulty string
	err := row.Scan(
		&c.ID, &c.TaskID, &c.TextbookID, &c.Chapter, &c.Topic, &c.Subtopic, &c.Question, &c.Answer,
		&c.PageApprox, &c.SourceReference, &difficulty, &c.ReportCount, &c.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	c.Difficulty = models.CardDifficulty(difficulty)
	return &c, nil
}

// CreateBatch вставляет все карточки задачи одной транзакцией - частично
// сохранённая задача (часть карточек есть, часть нет) не должна пережить
// сбой посередине.
func (r *CardRepo) CreateBatch(ctx context.Context, cards []models.Card) ([]models.Card, error) {
	if len(cards) == 0 {
		return nil, nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	out := make([]models.Card, 0, len(cards))
	for _, c := range cards {
		var id uuid.UUID
		var createdAt = c.CreatedAt
		err := tx.QueryRow(ctx, `
			INSERT INTO cards (task_id, textbook_id, chapter, topic, subtopic, question, answer,
				page_approx, source_reference, difficulty)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::card_difficulty)
			RETURNING id, created_at
		`, c.TaskID, c.TextbookID, c.Chapter, c.Topic, c.Subtopic, c.Question, c.Answer,
			c.PageApprox, c.SourceReference, string(c.Difficulty)).Scan(&id, &createdAt)
		if err != nil {
			return nil, err
		}
		c.ID = id
		c.CreatedAt = createdAt
		out = append(out, c)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return out, nil
}

// CloneForTask копирует все карточки sourceTaskID под newTaskID (cache-hit:
// избегаем повторного вызова LLM для уже сгенерированного учебник+тема+
// сложность+количество).
func (r *CardRepo) CloneForTask(ctx context.Context, sourceTaskID, newTaskID uuid.UUID) ([]models.Card, error) {
	query := `
		INSERT INTO cards (task_id, textbook_id, chapter, topic, subtopic, question, answer,
			page_approx, source_reference, difficulty)
		SELECT $2, textbook_id, chapter, topic, subtopic, question, answer, page_approx, source_reference, difficulty
		FROM cards WHERE task_id = $1
		RETURNING ` + cardSelectColumns

	rows, err := r.pool.Query(ctx, query, sourceTaskID, newTaskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Card
	for rows.Next() {
		c, err := r.scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

func (r *CardRepo) FindByID(ctx context.Context, id uuid.UUID) (*models.Card, error) {
	query := `SELECT ` + cardSelectColumns + ` FROM cards WHERE id = $1`
	c, err := r.scan(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrCardNotFound
		}
		return nil, err
	}
	return c, nil
}

func (r *CardRepo) ListByTask(ctx context.Context, taskID uuid.UUID, page, limit int) ([]models.Card, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM cards WHERE task_id = $1`, taskID).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `SELECT ` + cardSelectColumns + ` FROM cards WHERE task_id = $1 ORDER BY created_at ASC LIMIT $2 OFFSET $3`
	rows, err := r.pool.Query(ctx, query, taskID, limit, (page-1)*limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []models.Card
	for rows.Next() {
		c, err := r.scan(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *c)
	}
	return out, total, rows.Err()
}

func (r *CardRepo) IncrementReportCount(ctx context.Context, id uuid.UUID) error {
	cmd, err := r.pool.Exec(ctx, `UPDATE cards SET report_count = report_count + 1 WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return models.ErrCardNotFound
	}
	return nil
}
