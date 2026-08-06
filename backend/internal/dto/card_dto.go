package dto

import (
	"time"

	"github.com/medflow/backend/internal/models"
)

// CardDisclaimer сопровождает каждую ИИ-сгенерированную карточку - контент
// не проверен человеком, источник может быть законспектирован с ошибками.
const CardDisclaimer = "Сгенерировано ИИ. Проверьте по источнику перед использованием."

// CardTask. CardsCount отдаётся как 0, пока задача не завершена - см.
// комментарий у models.CardTask.CardsCount: та же колонка БД хранит сначала
// запрошенное количество (для воркера), потом фактическое сгенерированное,
// и наружу до завершения задачи её показывать не нужно (соответствует
// поведению фронтенд-моков, см. frontend/src/mocks/handlers/cards.ts).
type CardTask struct {
	ID                   string                `json:"id"`
	Status               models.CardTaskStatus `json:"status"`
	PositionInQueue      *int                  `json:"position_in_queue"`
	EstimatedWaitSeconds *int                  `json:"estimated_wait_seconds"`
	CardsCount           int                   `json:"cards_count"`
	ErrorMessage         *string               `json:"error_message"`
	StartedAt            *time.Time            `json:"started_at"`
	FinishedAt           *time.Time            `json:"finished_at"`
	CreatedAt            time.Time             `json:"created_at"`
}

func ToCardTask(t *models.CardTask) CardTask {
	// cards_count раскрывается только для done - у pending/processing он ещё
	// не известен, а у failed хранит "протухший" запрошенный (не фактический)
	// счётчик, т.к. MarkFailed его не трогает (см. models.CardTask).
	cardsCount := 0
	if t.Status == models.CardTaskDone && t.CardsCount != nil {
		cardsCount = *t.CardsCount
	}
	return CardTask{
		ID: t.ID.String(), Status: t.Status, PositionInQueue: t.PositionInQueue, EstimatedWaitSeconds: t.EstimatedWaitSeconds,
		CardsCount: cardsCount, ErrorMessage: t.ErrorMessage, StartedAt: t.StartedAt, FinishedAt: t.FinishedAt, CreatedAt: t.CreatedAt,
	}
}

type Card struct {
	ID              string                `json:"id"`
	Chapter         *string               `json:"chapter,omitempty"`
	Topic           *string               `json:"topic,omitempty"`
	Subtopic        *string               `json:"subtopic,omitempty"`
	Question        string                `json:"question"`
	Answer          string                `json:"answer"`
	PageApprox      *int                  `json:"page_approx,omitempty"`
	SourceReference *string               `json:"source_reference,omitempty"`
	Difficulty      models.CardDifficulty `json:"difficulty"`
	Disclaimer      string                `json:"disclaimer"`
	CreatedAt       time.Time             `json:"created_at"`
}

func ToCard(c *models.Card) Card {
	return Card{
		ID: c.ID.String(), Chapter: c.Chapter, Topic: c.Topic, Subtopic: c.Subtopic,
		Question: c.Question, Answer: c.Answer, PageApprox: c.PageApprox, SourceReference: c.SourceReference,
		Difficulty: c.Difficulty, Disclaimer: CardDisclaimer, CreatedAt: c.CreatedAt,
	}
}

type CardProgressInfo struct {
	NextReviewAt time.Time `json:"next_review_at"`
	IntervalDays int       `json:"interval_days"`
	EaseFactor   float64   `json:"ease_factor"`
	Repetitions  int       `json:"repetitions"`
}

func ToCardProgressInfo(p *models.CardProgress) CardProgressInfo {
	return CardProgressInfo{
		NextReviewAt: p.NextReviewAt, IntervalDays: p.IntervalDays, EaseFactor: p.EaseFactor, Repetitions: p.Repetitions,
	}
}

type ReviewCard struct {
	CardID     string           `json:"card_id"`
	Question   string           `json:"question"`
	Answer     string           `json:"answer"`
	Disclaimer string           `json:"disclaimer"`
	Progress   CardProgressInfo `json:"progress"`
}

func ToReviewCard(rc *models.ReviewCard) ReviewCard {
	return ReviewCard{
		CardID: rc.ID.String(), Question: rc.Question, Answer: rc.Answer,
		Disclaimer: CardDisclaimer, Progress: ToCardProgressInfo(&rc.Progress),
	}
}

type CardsStatsByDifficulty struct {
	Easy   int `json:"easy"`
	Medium int `json:"medium"`
	Hard   int `json:"hard"`
}

type CardsStats struct {
	TotalCardsLearned int                    `json:"total_cards_learned"`
	DueToday          int                    `json:"due_today"`
	StreakDays        int                    `json:"streak_days"`
	AvgEaseFactor     float64                `json:"avg_ease_factor"`
	ByDifficulty      CardsStatsByDifficulty `json:"by_difficulty"`
}

func ToCardsStats(s models.CardsStats) CardsStats {
	return CardsStats{
		TotalCardsLearned: s.TotalCardsLearned,
		DueToday:          s.DueToday,
		StreakDays:        s.StreakDays,
		AvgEaseFactor:     s.AvgEaseFactor,
		ByDifficulty: CardsStatsByDifficulty{
			Easy:   s.ByDifficulty[models.DifficultyEasy],
			Medium: s.ByDifficulty[models.DifficultyMedium],
			Hard:   s.ByDifficulty[models.DifficultyHard],
		},
	}
}

type CreateCardTaskRequest struct {
	FileID     string                `json:"file_id" binding:"required,uuid"`
	TextbookID *string               `json:"textbook_id,omitempty" binding:"omitempty,uuid"`
	Topic      string                `json:"topic" binding:"required,max=255"`
	Difficulty models.CardDifficulty `json:"difficulty,omitempty" binding:"omitempty,oneof=easy medium hard"`
	CardsCount int                   `json:"cards_count,omitempty" binding:"omitempty,min=1,max=100"`
}

// RateCardRequest.Grade сознательно без binding:"required" - 0 ("не помню")
// валидная оценка, а required у Gin считает нулевое значение int отсутствующим.
type RateCardRequest struct {
	Grade int `json:"grade" binding:"min=0,max=3"`
}
