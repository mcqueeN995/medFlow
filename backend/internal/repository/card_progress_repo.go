package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/medflow/backend/internal/models"
)

type CardProgressRepo struct {
	pool *pgxpool.Pool
}

func NewCardProgressRepo(pool *pgxpool.Pool) *CardProgressRepo {
	return &CardProgressRepo{pool: pool}
}

// CreateBatchDefault заводит card_progress со стандартным SM-2 стартом
// (ease_factor=2.5, interval_days=0, repetitions=0, next_review_at=now())
// сразу при генерации карточек - так свежесгенерированные карточки сразу
// доступны в /cards/review, без отдельного "первого показа".
func (r *CardProgressRepo) CreateBatchDefault(ctx context.Context, userID uuid.UUID, cardIDs []uuid.UUID) error {
	if len(cardIDs) == 0 {
		return nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, cardID := range cardIDs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO card_progress (user_id, card_id, ease_factor, interval_days, repetitions, next_review_at)
			VALUES ($1, $2, 2.5, 0, 0, now())
			ON CONFLICT (user_id, card_id) DO NOTHING
		`, userID, cardID); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *CardProgressRepo) scan(row pgx.Row) (*models.CardProgress, error) {
	var p models.CardProgress
	err := row.Scan(&p.ID, &p.UserID, &p.CardID, &p.EaseFactor, &p.IntervalDays, &p.Repetitions,
		&p.NextReviewAt, &p.LastReviewAt, &p.LastGrade, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *CardProgressRepo) FindByUserAndCard(ctx context.Context, userID, cardID uuid.UUID) (*models.CardProgress, error) {
	query := `
		SELECT id, user_id, card_id, ease_factor, interval_days, repetitions, next_review_at, last_review_at, last_grade, created_at, updated_at
		FROM card_progress WHERE user_id = $1 AND card_id = $2
	`
	p, err := r.scan(r.pool.QueryRow(ctx, query, userID, cardID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrCardProgressNotFound
		}
		return nil, err
	}
	return p, nil
}

func (r *CardProgressRepo) Update(ctx context.Context, p *models.CardProgress) error {
	cmd, err := r.pool.Exec(ctx, `
		UPDATE card_progress SET
			ease_factor = $2, interval_days = $3, repetitions = $4, next_review_at = $5,
			last_review_at = $6, last_grade = $7, updated_at = now()
		WHERE id = $1
	`, p.ID, p.EaseFactor, p.IntervalDays, p.Repetitions, p.NextReviewAt, p.LastReviewAt, p.LastGrade)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return models.ErrCardProgressNotFound
	}
	return nil
}

func (r *CardProgressRepo) CountDueForUser(ctx context.Context, userID uuid.UUID) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `
		SELECT count(*) FROM card_progress WHERE user_id = $1 AND next_review_at <= now()
	`, userID).Scan(&n)
	return n, err
}

// ListDueForUser отдаёт карточки, у которых next_review_at уже наступил,
// отсортированные по next_review_at (сначала самые просроченные).
func (r *CardProgressRepo) ListDueForUser(ctx context.Context, userID uuid.UUID, limit int) ([]models.ReviewCard, error) {
	query := `
		SELECT c.id, c.task_id, c.textbook_id, c.chapter, c.topic, c.subtopic, c.question, c.answer,
			c.page_approx, c.source_reference, c.difficulty, c.report_count, c.created_at,
			cp.id, cp.ease_factor, cp.interval_days, cp.repetitions, cp.next_review_at, cp.last_review_at, cp.last_grade, cp.created_at, cp.updated_at
		FROM cards c
		JOIN card_progress cp ON cp.card_id = c.id
		WHERE cp.user_id = $1 AND cp.next_review_at <= now()
		ORDER BY cp.next_review_at ASC
		LIMIT $2`

	rows, err := r.pool.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.ReviewCard
	for rows.Next() {
		var rc models.ReviewCard
		var difficulty string
		err := rows.Scan(
			&rc.ID, &rc.TaskID, &rc.TextbookID, &rc.Chapter, &rc.Topic, &rc.Subtopic, &rc.Question, &rc.Answer,
			&rc.PageApprox, &rc.SourceReference, &difficulty, &rc.ReportCount, &rc.CreatedAt,
			&rc.Progress.ID, &rc.Progress.EaseFactor, &rc.Progress.IntervalDays, &rc.Progress.Repetitions,
			&rc.Progress.NextReviewAt, &rc.Progress.LastReviewAt, &rc.Progress.LastGrade, &rc.Progress.CreatedAt, &rc.Progress.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		rc.Difficulty = models.CardDifficulty(difficulty)
		rc.Progress.UserID = userID
		rc.Progress.CardID = rc.ID
		out = append(out, rc)
	}
	return out, rows.Err()
}

// StatsForUser считает агрегаты одним запросом: обучено (repetitions>0),
// due today (next_review_at до конца сегодняшнего календарного дня),
// средний ease_factor, разбивка по сложности карточки.
func (r *CardProgressRepo) StatsForUser(ctx context.Context, userID uuid.UUID) (models.CardsStats, error) {
	var stats models.CardsStats
	err := r.pool.QueryRow(ctx, `
		SELECT
			count(*) FILTER (WHERE cp.repetitions > 0),
			count(*) FILTER (WHERE cp.next_review_at <= date_trunc('day', now()) + interval '1 day'),
			coalesce(avg(cp.ease_factor), 2.5)
		FROM card_progress cp WHERE cp.user_id = $1
	`, userID).Scan(&stats.TotalCardsLearned, &stats.DueToday, &stats.AvgEaseFactor)
	if err != nil {
		return stats, err
	}

	stats.ByDifficulty = map[models.CardDifficulty]int{
		models.DifficultyEasy: 0, models.DifficultyMedium: 0, models.DifficultyHard: 0,
	}
	rows, err := r.pool.Query(ctx, `
		SELECT c.difficulty, count(*)
		FROM card_progress cp JOIN cards c ON c.id = cp.card_id
		WHERE cp.user_id = $1
		GROUP BY c.difficulty
	`, userID)
	if err != nil {
		return stats, err
	}
	defer rows.Close()
	for rows.Next() {
		var difficulty string
		var n int
		if err := rows.Scan(&difficulty, &n); err != nil {
			return stats, err
		}
		stats.ByDifficulty[models.CardDifficulty(difficulty)] = n
	}
	return stats, rows.Err()
}

// DistinctReviewDaysForUser - календарные даты (desc), в которые пользователь
// хоть раз оценил карточку - основа для расчёта streak_days в CardService.
func (r *CardProgressRepo) DistinctReviewDaysForUser(ctx context.Context, userID uuid.UUID, limit int) ([]time.Time, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT date_trunc('day', last_review_at) AS d
		FROM card_progress
		WHERE user_id = $1 AND last_review_at IS NOT NULL
		ORDER BY d DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []time.Time
	for rows.Next() {
		var d time.Time
		if err := rows.Scan(&d); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
