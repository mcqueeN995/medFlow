package repository

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/medflow/backend/internal/models"
)

type TextbookChunkRepo struct {
	pool *pgxpool.Pool
}

func NewTextbookChunkRepo(pool *pgxpool.Pool) *TextbookChunkRepo {
	return &TextbookChunkRepo{pool: pool}
}

// pgvector не зарегистрирован как тип в pgtype.Map (тот же приём, что и для
// thread_tag[] в thread_repo.go) - эмбеддинг гоняется как текстовый литерал
// "[0.1,0.2,...]" с явным ::vector кастом на запись и парсится обратно на
// чтение.
func vectorToText(v []float32) string {
	parts := make([]string, len(v))
	for i, f := range v {
		parts[i] = strconv.FormatFloat(float64(f), 'f', -1, 32)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func textToVector(s string) ([]float32, error) {
	s = strings.TrimPrefix(strings.TrimSuffix(s, "]"), "[")
	if s == "" {
		return nil, nil
	}
	parts := strings.Split(s, ",")
	out := make([]float32, len(parts))
	for i, p := range parts {
		f, err := strconv.ParseFloat(p, 32)
		if err != nil {
			return nil, err
		}
		out[i] = float32(f)
	}
	return out, nil
}

// CreateBatch вставляет все чанки одной транзакцией.
func (r *TextbookChunkRepo) CreateBatch(ctx context.Context, chunks []models.TextbookChunk) error {
	if len(chunks) == 0 {
		return nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, c := range chunks {
		if _, err := tx.Exec(ctx, `
			INSERT INTO textbook_chunks (textbook_id, task_id, chunk_index, content, page_number, embedding)
			VALUES ($1, $2, $3, $4, $5, $6::vector)
		`, c.TextbookID, c.TaskID, c.ChunkIndex, c.Content, c.PageNumber, vectorToText(c.Embedding)); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *TextbookChunkRepo) ExistsForTextbook(ctx context.Context, textbookID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM textbook_chunks WHERE textbook_id = $1)`, textbookID).Scan(&exists)
	return exists, err
}

// SearchNearest ищет topK ближайших по косинусному расстоянию (оператор
// pgvector <=>) чанков в рамках одного источника: ровно один из
// textbookID/taskID должен быть не-nil.
func (r *TextbookChunkRepo) SearchNearest(ctx context.Context, textbookID, taskID *uuid.UUID, embedding []float32, topK int) ([]models.TextbookChunk, error) {
	var where string
	var sourceArg any
	switch {
	case textbookID != nil:
		where = "textbook_id = $2"
		sourceArg = *textbookID
	case taskID != nil:
		where = "task_id = $2"
		sourceArg = *taskID
	default:
		return nil, fmt.Errorf("textbook_chunk_repo: SearchNearest requires textbookID or taskID")
	}

	query := fmt.Sprintf(`
		SELECT id, textbook_id, task_id, chunk_index, content, page_number, embedding, created_at
		FROM textbook_chunks
		WHERE %s
		ORDER BY embedding <=> $1::vector
		LIMIT $3
	`, where)

	rows, err := r.pool.Query(ctx, query, vectorToText(embedding), sourceArg, topK)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.TextbookChunk
	for rows.Next() {
		var c models.TextbookChunk
		var embeddingText string
		if err := rows.Scan(&c.ID, &c.TextbookID, &c.TaskID, &c.ChunkIndex, &c.Content, &c.PageNumber, &embeddingText, &c.CreatedAt); err != nil {
			return nil, err
		}
		vec, err := textToVector(embeddingText)
		if err != nil {
			return nil, err
		}
		c.Embedding = vec
		out = append(out, c)
	}
	return out, rows.Err()
}
