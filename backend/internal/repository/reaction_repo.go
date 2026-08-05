package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/medflow/backend/internal/models"
)

type ReactionRepo struct {
	pool *pgxpool.Pool
}

func NewReactionRepo(pool *pgxpool.Pool) *ReactionRepo {
	return &ReactionRepo{pool: pool}
}

func targetTable(targetType models.ReactionTargetType) string {
	if targetType == models.ReactionTargetComment {
		return "comments"
	}
	return "threads"
}

// Upsert - у пользователя может быть максимум одна реакция на цель
// (uq_reactions_user_target не включает emoji), поэтому повторный вызов с
// другим emoji просто меняет эмодзи существующей реакции и не трогает
// likes_count.
func (r *ReactionRepo) Upsert(ctx context.Context, userID uuid.UUID, targetType models.ReactionTargetType, targetID uuid.UUID, emoji string) (*models.Reaction, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	reaction := &models.Reaction{
		UserID:     userID,
		TargetType: targetType,
		TargetID:   targetID,
		Emoji:      emoji,
	}

	var inserted bool
	err = tx.QueryRow(ctx, `
		INSERT INTO reactions (user_id, target_type, target_id, emoji)
		VALUES ($1, $2::reaction_target_type, $3, $4)
		ON CONFLICT (user_id, target_type, target_id)
		DO UPDATE SET emoji = EXCLUDED.emoji
		RETURNING id, created_at, (xmax = 0) AS inserted
	`, userID, string(targetType), targetID, emoji).Scan(&reaction.ID, &reaction.CreatedAt, &inserted)
	if err != nil {
		return nil, err
	}

	if inserted {
		table := targetTable(targetType)
		if _, err := tx.Exec(ctx, fmt.Sprintf("UPDATE %s SET likes_count = likes_count + 1 WHERE id = $1", table), targetID); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return reaction, nil
}

func (r *ReactionRepo) Delete(ctx context.Context, userID uuid.UUID, targetType models.ReactionTargetType, targetID uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	cmd, err := tx.Exec(ctx, `
		DELETE FROM reactions WHERE user_id = $1 AND target_type = $2::reaction_target_type AND target_id = $3
	`, userID, string(targetType), targetID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return models.ErrReactionNotFound
	}

	table := targetTable(targetType)
	if _, err := tx.Exec(ctx, fmt.Sprintf("UPDATE %s SET likes_count = GREATEST(likes_count - 1, 0) WHERE id = $1", table), targetID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
