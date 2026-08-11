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

// Upsert - у пользователя может быть максимум одна эмодзи-реакция на цель
// (uq_reactions_user_target теперь на 4 колонки, включая kind - см. миграцию
// 000016_reactions_kind), поэтому повторный вызов с другим emoji просто
// меняет эмодзи существующей реакции и не трогает likes_count. kind='emoji'
// не пересекается с kind='vote' (см. UpsertVote) - лайк и голос на одной и
// той же цели сосуществуют независимо.
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
		Kind:       models.ReactionKindEmoji,
	}

	var inserted bool
	err = tx.QueryRow(ctx, `
		INSERT INTO reactions (user_id, target_type, target_id, emoji, kind)
		VALUES ($1, $2::reaction_target_type, $3, $4, 'emoji')
		ON CONFLICT (user_id, target_type, target_id, kind)
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
		DELETE FROM reactions WHERE user_id = $1 AND target_type = $2::reaction_target_type AND target_id = $3 AND kind = 'emoji'
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

// UpsertVote - голос up/down, kind='vote'. В отличие от Upsert (эмодзи), не
// трогает никакой счётчик - агрегированный score считается на лету в
// VoteSummaries, а не хранится денормализованно.
func (r *ReactionRepo) UpsertVote(ctx context.Context, userID uuid.UUID, targetType models.ReactionTargetType, targetID uuid.UUID, direction string) (*models.Reaction, error) {
	reaction := &models.Reaction{
		UserID:     userID,
		TargetType: targetType,
		TargetID:   targetID,
		Emoji:      direction,
		Kind:       models.ReactionKindVote,
	}
	err := r.pool.QueryRow(ctx, `
		INSERT INTO reactions (user_id, target_type, target_id, emoji, kind)
		VALUES ($1, $2::reaction_target_type, $3, $4, 'vote')
		ON CONFLICT (user_id, target_type, target_id, kind)
		DO UPDATE SET emoji = EXCLUDED.emoji
		RETURNING id, created_at
	`, userID, string(targetType), targetID, direction).Scan(&reaction.ID, &reaction.CreatedAt)
	if err != nil {
		return nil, err
	}
	return reaction, nil
}

func (r *ReactionRepo) DeleteVote(ctx context.Context, userID uuid.UUID, targetType models.ReactionTargetType, targetID uuid.UUID) error {
	cmd, err := r.pool.Exec(ctx, `
		DELETE FROM reactions WHERE user_id = $1 AND target_type = $2::reaction_target_type AND target_id = $3 AND kind = 'vote'
	`, userID, string(targetType), targetID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return models.ErrReactionNotFound
	}
	return nil
}

// VoteSummaries - батч по списку целей (см. батчинг repliesByParent в
// CommentRepo.ListByThread - тот же принцип: один запрос на список ID вместо
// N+1 при рендере списка комментариев). MAX(CASE WHEN user_id = viewerID ...)
// корректно достаёт "голос текущего юзера", т.к. на пользователя может быть
// максимум один голос на цель (uq_reactions_user_target).
func (r *ReactionRepo) VoteSummaries(ctx context.Context, targetType models.ReactionTargetType, targetIDs []uuid.UUID, viewerID uuid.UUID) (map[uuid.UUID]models.VoteSummary, error) {
	out := make(map[uuid.UUID]models.VoteSummary, len(targetIDs))
	if len(targetIDs) == 0 {
		return out, nil
	}

	rows, err := r.pool.Query(ctx, `
		SELECT target_id,
		       SUM(CASE emoji WHEN 'up' THEN 1 WHEN 'down' THEN -1 ELSE 0 END)::int AS score,
		       MAX(CASE WHEN user_id = $3 THEN emoji END) AS my_vote
		FROM reactions
		WHERE target_type = $1::reaction_target_type AND target_id = ANY($2) AND kind = 'vote'
		GROUP BY target_id
	`, string(targetType), targetIDs, viewerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var targetID uuid.UUID
		var summary models.VoteSummary
		if err := rows.Scan(&targetID, &summary.Score, &summary.MyVote); err != nil {
			return nil, err
		}
		out[targetID] = summary
	}
	return out, rows.Err()
}
