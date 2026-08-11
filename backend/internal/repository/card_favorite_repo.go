package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/medflow/backend/internal/models"
)

type CardFavoriteRepo struct {
	pool *pgxpool.Pool
}

func NewCardFavoriteRepo(pool *pgxpool.Pool) *CardFavoriteRepo {
	return &CardFavoriteRepo{pool: pool}
}

// Add - идемпотентно: повторное добавление уже избранной карточки не ошибка.
func (r *CardFavoriteRepo) Add(ctx context.Context, userID, cardID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO card_favorites (user_id, card_id) VALUES ($1, $2)
		ON CONFLICT (user_id, card_id) DO NOTHING
	`, userID, cardID)
	return err
}

func (r *CardFavoriteRepo) Remove(ctx context.Context, userID, cardID uuid.UUID) error {
	cmd, err := r.pool.Exec(ctx, `DELETE FROM card_favorites WHERE user_id = $1 AND card_id = $2`, userID, cardID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return models.ErrCardFavoriteNotFound
	}
	return nil
}

func (r *CardFavoriteRepo) ListForUser(ctx context.Context, userID uuid.UUID, page, limit int) ([]models.Card, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM card_favorites WHERE user_id = $1`, userID).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT ` + cardSelectColumnsAliased + `
		FROM card_favorites f JOIN cards c ON c.id = f.card_id
		WHERE f.user_id = $1
		ORDER BY f.created_at DESC
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

// IsFavoritedBatch - батч вместо N+1 при рендере списка карточек (см.
// CardService.enrichCards, тот же принцип, что и VoteSummaries в форуме).
func (r *CardFavoriteRepo) IsFavoritedBatch(ctx context.Context, userID uuid.UUID, cardIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	out := make(map[uuid.UUID]bool, len(cardIDs))
	if len(cardIDs) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT card_id FROM card_favorites WHERE user_id = $1 AND card_id = ANY($2)
	`, userID, cardIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var cardID uuid.UUID
		if err := rows.Scan(&cardID); err != nil {
			return nil, err
		}
		out[cardID] = true
	}
	return out, rows.Err()
}
