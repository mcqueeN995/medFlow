package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/medflow/backend/internal/models"
)

type CardRatingRepo struct {
	pool *pgxpool.Pool
}

func NewCardRatingRepo(pool *pgxpool.Pool) *CardRatingRepo {
	return &CardRatingRepo{pool: pool}
}

// Upsert - повторная оценка той же карточки тем же пользователем меняет
// звёзды, а не создаёт вторую запись (PK(user_id, card_id)).
func (r *CardRatingRepo) Upsert(ctx context.Context, userID, cardID uuid.UUID, stars int) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO card_ratings (user_id, card_id, stars) VALUES ($1, $2, $3)
		ON CONFLICT (user_id, card_id) DO UPDATE SET stars = EXCLUDED.stars, updated_at = now()
	`, userID, cardID, stars)
	return err
}

func (r *CardRatingRepo) Delete(ctx context.Context, userID, cardID uuid.UUID) error {
	cmd, err := r.pool.Exec(ctx, `DELETE FROM card_ratings WHERE user_id = $1 AND card_id = $2`, userID, cardID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return models.ErrCardRatingNotFound
	}
	return nil
}

// AggregateForCardsBatch - средняя оценка + число оценок, батчем на список
// карточек (см. CardService.enrichCards).
func (r *CardRatingRepo) AggregateForCardsBatch(ctx context.Context, cardIDs []uuid.UUID) (map[uuid.UUID]models.CardRatingAggregate, error) {
	out := make(map[uuid.UUID]models.CardRatingAggregate, len(cardIDs))
	if len(cardIDs) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT card_id, avg(stars)::float8, count(*)
		FROM card_ratings WHERE card_id = ANY($1)
		GROUP BY card_id
	`, cardIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var cardID uuid.UUID
		var agg models.CardRatingAggregate
		if err := rows.Scan(&cardID, &agg.AverageStars, &agg.RatingsCount); err != nil {
			return nil, err
		}
		out[cardID] = agg
	}
	return out, rows.Err()
}

// MyRatingsBatch - оценка viewerID (если есть) для каждой карточки из списка.
func (r *CardRatingRepo) MyRatingsBatch(ctx context.Context, userID uuid.UUID, cardIDs []uuid.UUID) (map[uuid.UUID]int, error) {
	out := make(map[uuid.UUID]int, len(cardIDs))
	if len(cardIDs) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT card_id, stars FROM card_ratings WHERE user_id = $1 AND card_id = ANY($2)
	`, userID, cardIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var cardID uuid.UUID
		var stars int
		if err := rows.Scan(&cardID, &stars); err != nil {
			return nil, err
		}
		out[cardID] = stars
	}
	return out, rows.Err()
}

func (r *CardRatingRepo) ListRatedByUser(ctx context.Context, userID uuid.UUID, page, limit int) ([]models.Card, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM card_ratings WHERE user_id = $1`, userID).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT ` + cardSelectColumnsAliased + `
		FROM card_ratings cr JOIN cards c ON c.id = cr.card_id
		WHERE cr.user_id = $1
		ORDER BY cr.updated_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.pool.Query(ctx, query, userID, limit, (page-1)*limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []models.Card
	for rows.Next() {
		c, err := scanCardAliased(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *c)
	}
	return out, total, rows.Err()
}
