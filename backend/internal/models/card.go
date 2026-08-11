package models

import (
	"time"

	"github.com/google/uuid"
)

type CardDifficulty string

const (
	DifficultyEasy   CardDifficulty = "easy"
	DifficultyMedium CardDifficulty = "medium"
	DifficultyHard   CardDifficulty = "hard"
)

type CardTaskStatus string

const (
	CardTaskPending    CardTaskStatus = "pending"
	CardTaskProcessing CardTaskStatus = "processing"
	CardTaskDone       CardTaskStatus = "done"
	CardTaskFailed     CardTaskStatus = "failed"
)

type CardTaskSourceType string

const (
	SourceCatalogTextbook CardTaskSourceType = "catalog_textbook"
	SourceUserUpload      CardTaskSourceType = "user_upload"
)

// CardTask. CardsCount хранит запрошенное количество карточек сразу после
// создания задачи (нужно воркеру, чтобы знать, сколько генерировать), а
// после завершения перезаписывается фактическим числом сгенерированных
// карточек - единственная колонка card_tasks.cards_count в схеме БД служит
// обеим целям последовательно в рамках жизненного цикла одной задачи.
// PositionInQueue/EstimatedWaitSeconds не хранятся в БД - вычисляются на
// лету в сервисном слое (см. CardService.GetTask/enrichPosition).
type CardTask struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	TextbookID   *uuid.UUID
	SourceType   CardTaskSourceType
	Topic        *string
	Difficulty   CardDifficulty
	CardsCount   *int
	CacheKey     *string
	TempS3Key    *string
	TempFileName *string
	TempFileSize *int64
	Status       CardTaskStatus
	ErrorMessage *string
	StartedAt    *time.Time
	FinishedAt   *time.Time
	CreatedAt    time.Time

	// ShareToken - см. CardService.ShareTask/UnshareTask/GetSharedTask и
	// миграцию 000017_cards_favorites_ratings_share. nil, пока владелец не
	// включил шеринг.
	ShareToken *string

	PositionInQueue      *int
	EstimatedWaitSeconds *int
}

func (t *CardTask) IsActive() bool {
	return t.Status == CardTaskPending || t.Status == CardTaskProcessing
}

type Card struct {
	ID              uuid.UUID
	TaskID          uuid.UUID
	TextbookID      *uuid.UUID
	Chapter         *string
	Topic           *string
	Subtopic        *string
	Question        string
	Answer          string
	PageApprox      *int
	SourceReference *string
	Difficulty      CardDifficulty
	ReportCount     int
	CreatedAt       time.Time
}

type CardProgress struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	CardID       uuid.UUID
	EaseFactor   float64
	IntervalDays int
	Repetitions  int
	NextReviewAt time.Time
	LastReviewAt *time.Time
	LastGrade    *int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ReviewCard - карточка вместе с прогрессом конкретного пользователя,
// для батча /cards/review.
type ReviewCard struct {
	Card
	Progress CardProgress
}

// TextbookChunk - фрагмент текста учебника/личной PDF-загрузки с эмбеддингом
// для векторного поиска (RAG). Ровно одно из TextbookID/TaskID заполнено:
// TextbookID - для catalog_textbook (переиспользуется всеми задачами по
// этому учебнику), TaskID - для user_upload (одноразовый, только для этой
// задачи).
type TextbookChunk struct {
	ID         uuid.UUID
	TextbookID *uuid.UUID
	TaskID     *uuid.UUID
	ChunkIndex int
	Content    string
	PageNumber *int
	Embedding  []float32
	CreatedAt  time.Time
}

type CardTaskListFilter struct {
	UserID uuid.UUID
	Status *CardTaskStatus
	Page   int
	Limit  int
}

// CardCatalogFeedFilter - лента уже сгенерированных наборов карточек из
// каталога учебников (source_type=catalog_textbook, status=done), см.
// CardTaskRepo.ListCatalogFeed. Личные user_upload-задачи в ленту не
// попадают - у них нет cache_key и они приватны.
type CardCatalogFeedFilter struct {
	Q          *string
	TextbookID *uuid.UUID
	Difficulty *CardDifficulty
	Page       int
	Limit      int
}

// CardCatalogEntry - одна строка ленты каталога: один канонический task на
// cache_key (первый дошедший до done), а не каждый пользовательский клон.
type CardCatalogEntry struct {
	TaskID        uuid.UUID
	TextbookID    uuid.UUID
	TextbookTitle string
	Topic         *string
	Difficulty    CardDifficulty
	CardsCount    int
	CreatedAt     time.Time
}

// CardRatingAggregate - см. CardRatingRepo.AggregateForCardsBatch.
type CardRatingAggregate struct {
	AverageStars float64
	RatingsCount int
}

type CardsStats struct {
	TotalCardsLearned int
	DueToday          int
	StreakDays        int
	AvgEaseFactor     float64
	ByDifficulty      map[CardDifficulty]int
}
