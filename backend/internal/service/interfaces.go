package service

import (
	"context"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/medflow/backend/internal/config"
	"github.com/medflow/backend/internal/models"
)

// CardTaskRepository. Явные методы перехода статуса (Mark*) вместо общего
// Update - переходы задачи (pending→processing→done/failed) вызываются из
// разных, узких мест конвейера, и общий "перезаписать всё" метод легко
// перепутать местами вызовов (например, случайно затереть error_message
// при обычном обновлении статуса).
type CardTaskRepository interface {
	Create(ctx context.Context, t *models.CardTask) (*models.CardTask, error)
	FindByID(ctx context.Context, id uuid.UUID) (*models.CardTask, error)
	List(ctx context.Context, f models.CardTaskListFilter) ([]models.CardTask, int, error)
	MarkProcessing(ctx context.Context, id uuid.UUID) error
	MarkDone(ctx context.Context, id uuid.UUID, cardsCount int) error
	MarkFailed(ctx context.Context, id uuid.UUID, errMsg string) error
	FindDoneByCacheKey(ctx context.Context, cacheKey string) (*models.CardTask, error)
	CountActive(ctx context.Context, userID uuid.UUID) (int, error)
	CountPendingBefore(ctx context.Context, createdAt time.Time) (int, error)
	ListCatalogFeed(ctx context.Context, f models.CardCatalogFeedFilter) ([]models.CardCatalogEntry, int, error)
	SetShareToken(ctx context.Context, taskID uuid.UUID, token string) error
	ClearShareToken(ctx context.Context, taskID uuid.UUID) error
	FindByShareToken(ctx context.Context, token string) (*models.CardTask, error)
}

type CardRepository interface {
	CreateBatch(ctx context.Context, cards []models.Card) ([]models.Card, error)
	// CloneForTask копирует карточки sourceTaskID под новый newTaskID -
	// используется при cache-hit, чтобы не звать LLM повторно.
	CloneForTask(ctx context.Context, sourceTaskID, newTaskID uuid.UUID) ([]models.Card, error)
	FindByID(ctx context.Context, id uuid.UUID) (*models.Card, error)
	ListByTask(ctx context.Context, taskID uuid.UUID, page, limit int) ([]models.Card, int, error)
	IncrementReportCount(ctx context.Context, id uuid.UUID) error
}

type CardProgressRepository interface {
	CreateBatchDefault(ctx context.Context, userID uuid.UUID, cardIDs []uuid.UUID) error
	FindByUserAndCard(ctx context.Context, userID, cardID uuid.UUID) (*models.CardProgress, error)
	Update(ctx context.Context, p *models.CardProgress) error
	ListDueForUser(ctx context.Context, userID uuid.UUID, limit int) ([]models.ReviewCard, error)
	// CountDueForUser - общее число просроченных карточек (не ограничено
	// limit'ом батча ListDueForUser) - для поля "count" в ответе /cards/review.
	CountDueForUser(ctx context.Context, userID uuid.UUID) (int, error)
	StatsForUser(ctx context.Context, userID uuid.UUID) (models.CardsStats, error)
	// DistinctReviewDaysForUser - календарные даты (desc) последних
	// повторений пользователя, для расчёта streak_days.
	DistinctReviewDaysForUser(ctx context.Context, userID uuid.UUID, limit int) ([]time.Time, error)
	ListDueFavoritesForUser(ctx context.Context, userID uuid.UUID, limit int) ([]models.ReviewCard, error)
	CountDueFavoritesForUser(ctx context.Context, userID uuid.UUID) (int, error)
}

// CardFavoriteRepository - избранное карточек, независимо от SM-2 прогресса
// (CardProgressRepository). Доступ гейтится в CardService.authorizeCardAccess.
type CardFavoriteRepository interface {
	Add(ctx context.Context, userID, cardID uuid.UUID) error
	Remove(ctx context.Context, userID, cardID uuid.UUID) error
	ListForUser(ctx context.Context, userID uuid.UUID, page, limit int) ([]models.Card, int, error)
	IsFavoritedBatch(ctx context.Context, userID uuid.UUID, cardIDs []uuid.UUID) (map[uuid.UUID]bool, error)
}

// CardRatingRepository - оценки карточек звёздами 1-5, отдельно от
// избранного и от форумных reactions.
type CardRatingRepository interface {
	Upsert(ctx context.Context, userID, cardID uuid.UUID, stars int) error
	Delete(ctx context.Context, userID, cardID uuid.UUID) error
	AggregateForCardsBatch(ctx context.Context, cardIDs []uuid.UUID) (map[uuid.UUID]models.CardRatingAggregate, error)
	MyRatingsBatch(ctx context.Context, userID uuid.UUID, cardIDs []uuid.UUID) (map[uuid.UUID]int, error)
	ListRatedByUser(ctx context.Context, userID uuid.UUID, page, limit int) ([]models.Card, int, error)
}

type TextbookChunkRepository interface {
	CreateBatch(ctx context.Context, chunks []models.TextbookChunk) error
	ExistsForTextbook(ctx context.Context, textbookID uuid.UUID) (bool, error)
	// SearchNearest ищет topK ближайших по эмбеддингу чанков в рамках ОДНОГО
	// источника - передаётся либо textbookID, либо taskID (см. models.TextbookChunk).
	SearchNearest(ctx context.Context, textbookID, taskID *uuid.UUID, embedding []float32, topK int) ([]models.TextbookChunk, error)
}

// TaskEnqueuer - минимальный интерфейс над очередью задач (см.
// internal/pkg/queue), достаточный для юнит-тестов CardService без Redis.
type TaskEnqueuer interface {
	Enqueue(typename string, payload []byte) error
}

type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	FindByLogin(ctx context.Context, login string) (*models.User, error)
	FindByNickname(ctx context.Context, nickname string) (*models.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	Update(ctx context.Context, id uuid.UUID, nickname string, university *models.University, course *int, faculty *string) (*models.User, error)
	UpdateLogin(ctx context.Context, id uuid.UUID, login string) (*models.User, error)
	UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
	FindPublicByID(ctx context.Context, id uuid.UUID) (*models.PublicUser, error)
	AdminList(ctx context.Context, f models.AdminUserListFilter) ([]models.User, int, error)
	ChangeRole(ctx context.Context, id uuid.UUID, role models.UserRole) (*models.User, error)
	Ban(ctx context.Context, id, bannedBy uuid.UUID, reason string) (*models.User, error)
	Unban(ctx context.Context, id uuid.UUID) (*models.User, error)
}

type TokenRepository interface {
	Save(ctx context.Context, token *models.RefreshToken) error
	FindByHash(ctx context.Context, hash string) (*models.RefreshToken, error)
	DeleteByID(ctx context.Context, id uuid.UUID) error
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
	DeleteExpired(ctx context.Context) (int64, error)
}

type LoginChangeRepository interface {
	Save(ctx context.Context, req *models.LoginChangeRequest) error
	FindByCodeHash(ctx context.Context, codeHash string) (*models.LoginChangeRequest, error)
	DeleteByID(ctx context.Context, id uuid.UUID) error
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
}

type PasswordResetRepository interface {
	Save(ctx context.Context, req *models.PasswordResetRequest) error
	FindByCodeHash(ctx context.Context, codeHash string) (*models.PasswordResetRequest, error)
	DeleteByID(ctx context.Context, id uuid.UUID) error
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
}

type ThreadRepository interface {
	Create(ctx context.Context, authorID uuid.UUID, title, content string, tags []models.ThreadTag) (*models.Thread, error)
	FindByID(ctx context.Context, id uuid.UUID) (*models.Thread, error)
	IncrementViewsIfNotRecentlyViewed(ctx context.Context, threadID, userID uuid.UUID) error
	Update(ctx context.Context, id uuid.UUID, title, content string, tags []models.ThreadTag) (*models.Thread, error)
	SoftDelete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, f models.ThreadListFilter) ([]models.Thread, int, error)
	Hide(ctx context.Context, id, hiddenBy uuid.UUID, reason string) (*models.Thread, error)
}

type CommentRepository interface {
	Create(ctx context.Context, threadID, authorID uuid.UUID, parentID, replyToID *uuid.UUID, depth int, content string) (*models.Comment, error)
	FindByID(ctx context.Context, id uuid.UUID) (*models.Comment, error)
	Update(ctx context.Context, id uuid.UUID, content string) (*models.Comment, error)
	SoftDelete(ctx context.Context, id uuid.UUID) error
	ListByThread(ctx context.Context, threadID uuid.UUID, page, limit int, sort string) ([]models.Comment, int, error)
	Hide(ctx context.Context, id, hiddenBy uuid.UUID, reason string) (*models.Comment, error)
}

type ReactionRepository interface {
	Upsert(ctx context.Context, userID uuid.UUID, targetType models.ReactionTargetType, targetID uuid.UUID, emoji string) (*models.Reaction, error)
	Delete(ctx context.Context, userID uuid.UUID, targetType models.ReactionTargetType, targetID uuid.UUID) error
	UpsertVote(ctx context.Context, userID uuid.UUID, targetType models.ReactionTargetType, targetID uuid.UUID, direction string) (*models.Reaction, error)
	DeleteVote(ctx context.Context, userID uuid.UUID, targetType models.ReactionTargetType, targetID uuid.UUID) error
	VoteSummaries(ctx context.Context, targetType models.ReactionTargetType, targetIDs []uuid.UUID, viewerID uuid.UUID) (map[uuid.UUID]models.VoteSummary, error)
}

type ReportRepository interface {
	Create(ctx context.Context, reporterID uuid.UUID, targetType string, targetID uuid.UUID, reason string) (*models.Report, error)
	FindByID(ctx context.Context, id uuid.UUID) (*models.Report, error)
	List(ctx context.Context, f models.ReportListFilter) ([]models.Report, int, error)
	Review(ctx context.Context, id, reviewedBy uuid.UUID, status models.ReportStatus, note *string) (*models.Report, error)
}

// AuditLogRepository - журнал действий модераторов/админов.
type AuditLogRepository interface {
	Create(ctx context.Context, entry *models.AuditLog) error
	List(ctx context.Context, f models.AuditLogListFilter) ([]models.AuditLog, int, error)
}

type AdminStatsRepository interface {
	Stats(ctx context.Context) (models.AdminStats, error)
}

type POIRepository interface {
	Create(ctx context.Context, p *models.POI) (*models.POI, error)
	FindByID(ctx context.Context, id uuid.UUID) (*models.POI, error)
	Update(ctx context.Context, id uuid.UUID, p *models.POI) (*models.POI, error)
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, f models.PoiListFilter) ([]models.POI, error)
	AdminList(ctx context.Context, f models.AdminPoiListFilter) ([]models.POI, int, error)
}

type TextbookRepository interface {
	Create(ctx context.Context, t *models.Textbook) (*models.Textbook, error)
	FindByID(ctx context.Context, id uuid.UUID) (*models.Textbook, error)
	AdminFindByID(ctx context.Context, id uuid.UUID) (*models.Textbook, error)
	Update(ctx context.Context, id uuid.UUID, t *models.Textbook) (*models.Textbook, error)
	SoftDelete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, f models.TextbookListFilter) ([]models.Textbook, int, error)
	AdminList(ctx context.Context, f models.AdminTextbookListFilter) ([]models.Textbook, int, error)
}

type UploadRepository interface {
	Create(ctx context.Context, u *models.Upload) (*models.Upload, error)
	FindByID(ctx context.Context, id uuid.UUID) (*models.Upload, error)
}

// ObjectStorage - минимальный интерфейс над S3-совместимым хранилищем
// (см. internal/pkg/storage), достаточный для юнит-тестов сервисов без
// поднятия реального MinIO.
type ObjectStorage interface {
	Put(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error
	PresignedGetURL(ctx context.Context, key string, expiry time.Duration) (string, error)
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Remove(ctx context.Context, key string) error
}

type PushRepository interface {
	CreateSubscription(ctx context.Context, userID uuid.UUID, endpoint, p256dh, auth string) (*models.PushSubscription, error)
	DeleteSubscriptionByEndpoint(ctx context.Context, userID uuid.UUID, endpoint string) error
	DeleteSubscriptionByRawEndpoint(ctx context.Context, endpoint string) error
	ListSubscriptionsForUser(ctx context.Context, userID uuid.UUID) ([]models.PushSubscription, error)
	GetPreferences(ctx context.Context, userID uuid.UUID) (*models.PushPreferences, error)
	UpsertPreferences(ctx context.Context, p models.PushPreferences) (*models.PushPreferences, error)
}

// PushSender - тонкая обёртка над реальной отправкой Web Push (webpush-go),
// вынесена в интерфейс, чтобы юнит-тесты PushService не били по сети.
// ErrPushGone сигнализирует протухшую подписку (сервис отдал 404/410) -
// PushService по этому сигналу удаляет подписку из БД.
type PushSender interface {
	Send(ctx context.Context, sub models.PushSubscription, vapid config.VAPIDConfig, payload []byte) error
}

// PushNotifier - узкий интерфейс с единственным методом, нужным другим
// сервисам (ForumService, воркеру) для триггера push-уведомлений - не весь
// PushService целиком, по аналогии с тем, как AuditLogRepository используется
// как узкая зависимость вместо полного PushService.
type PushNotifier interface {
	Notify(ctx context.Context, userID uuid.UUID, kind models.NotificationKind, title, message string) error
}

// EmailSender - узкая абстракция над pkg/email.Sender, нужна только чтобы
// UserService можно было тестировать без реального SMTP (см. RequestLoginChange).
type EmailSender interface {
	Send(to, subject, body string) error
}
