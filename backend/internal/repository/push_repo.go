package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/medflow/backend/internal/models"
)

type PushRepo struct {
	pool *pgxpool.Pool
}

func NewPushRepo(pool *pgxpool.Pool) *PushRepo {
	return &PushRepo{pool: pool}
}

const pushSubscriptionColumns = `id, user_id, endpoint, p256dh, auth, created_at`

func scanPushSubscription(row pgx.Row) (*models.PushSubscription, error) {
	var s models.PushSubscription
	err := row.Scan(&s.ID, &s.UserID, &s.Endpoint, &s.P256dh, &s.Auth, &s.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// CreateSubscription - upsert по endpoint: повторная подписка того же
// браузера (например, после переустановки PWA) обновляет ключи и владельца
// вместо конфликта на UNIQUE(endpoint).
func (r *PushRepo) CreateSubscription(ctx context.Context, userID uuid.UUID, endpoint, p256dh, auth string) (*models.PushSubscription, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `
		INSERT INTO push_subscriptions (user_id, endpoint, p256dh, auth)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (endpoint) DO UPDATE SET user_id = $1, p256dh = $3, auth = $4
		RETURNING id
	`, userID, endpoint, p256dh, auth).Scan(&id)
	if err != nil {
		return nil, err
	}
	row := r.pool.QueryRow(ctx, `SELECT `+pushSubscriptionColumns+` FROM push_subscriptions WHERE id = $1`, id)
	return scanPushSubscription(row)
}

func (r *PushRepo) DeleteSubscriptionByEndpoint(ctx context.Context, userID uuid.UUID, endpoint string) error {
	cmd, err := r.pool.Exec(ctx, `DELETE FROM push_subscriptions WHERE user_id = $1 AND endpoint = $2`, userID, endpoint)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return models.ErrPushSubscriptionNotFound
	}
	return nil
}

// DeleteSubscriptionByRawEndpoint - удаляет протухшую подписку без привязки
// к userID (используется PushService при чистке подписок после 404/410 от
// push-сервиса, где под рукой есть только endpoint).
func (r *PushRepo) DeleteSubscriptionByRawEndpoint(ctx context.Context, endpoint string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM push_subscriptions WHERE endpoint = $1`, endpoint)
	return err
}

func (r *PushRepo) ListSubscriptionsForUser(ctx context.Context, userID uuid.UUID) ([]models.PushSubscription, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+pushSubscriptionColumns+` FROM push_subscriptions WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.PushSubscription
	for rows.Next() {
		s, err := scanPushSubscription(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}

const pushPreferencesColumns = `user_id, thread_reply, comment_reply, reaction, card_task_done, card_task_failed, moderation_action, system, updated_at`

func scanPushPreferences(row pgx.Row) (*models.PushPreferences, error) {
	var p models.PushPreferences
	err := row.Scan(&p.UserID, &p.ThreadReply, &p.CommentReply, &p.Reaction, &p.CardTaskDone, &p.CardTaskFailed, &p.ModerationAction, &p.System, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// GetPreferences - ленивое создание: если строки ещё нет (пользователь
// никогда не менял настройки), создаёт её с дефолтами (все true) и
// возвращает - вызывающему коду (PushService.Notify) не нужно отдельно
// обрабатывать models.ErrPushSubscriptionNotFound для preferences.
func (r *PushRepo) GetPreferences(ctx context.Context, userID uuid.UUID) (*models.PushPreferences, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+pushPreferencesColumns+` FROM push_preferences WHERE user_id = $1`, userID)
	p, err := scanPushPreferences(row)
	if err == nil {
		return p, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	defaults := models.DefaultPushPreferences(userID)
	row = r.pool.QueryRow(ctx, `
		INSERT INTO push_preferences (user_id, thread_reply, comment_reply, reaction, card_task_done, card_task_failed, moderation_action, system)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (user_id) DO UPDATE SET user_id = $1
		RETURNING `+pushPreferencesColumns, userID, defaults.ThreadReply, defaults.CommentReply, defaults.Reaction, defaults.CardTaskDone, defaults.CardTaskFailed, defaults.ModerationAction, defaults.System)
	return scanPushPreferences(row)
}

func (r *PushRepo) UpsertPreferences(ctx context.Context, p models.PushPreferences) (*models.PushPreferences, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO push_preferences (user_id, thread_reply, comment_reply, reaction, card_task_done, card_task_failed, moderation_action, system)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (user_id) DO UPDATE SET
			thread_reply = $2, comment_reply = $3, reaction = $4, card_task_done = $5,
			card_task_failed = $6, moderation_action = $7, system = $8, updated_at = now()
		RETURNING `+pushPreferencesColumns,
		p.UserID, p.ThreadReply, p.CommentReply, p.Reaction, p.CardTaskDone, p.CardTaskFailed, p.ModerationAction, p.System)
	return scanPushPreferences(row)
}
