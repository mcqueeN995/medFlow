package service

import (
	"context"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/medflow/backend/internal/config"
	"github.com/medflow/backend/internal/models"
)

// это моки чтобы тестировать ТОЛЬКО service слой БЕЗ repository

// mockUserRepository ручной мок services.UserRepository
type mockUserRepository struct {
	createFn         func(ctx context.Context, user *models.User) error
	findByEmailFn    func(ctx context.Context, email string) (*models.User, error)
	findByLoginFn    func(ctx context.Context, login string) (*models.User, error)
	findByNicknameFn func(ctx context.Context, nickname string) (*models.User, error)
	findByIDFn       func(ctx context.Context, id uuid.UUID) (*models.User, error)
	updateFn         func(ctx context.Context, id uuid.UUID, nickname string, university *models.University, course *int, faculty *string) (*models.User, error)
	updateLoginFn    func(ctx context.Context, id uuid.UUID, login string) (*models.User, error)
	updatePasswordFn func(ctx context.Context, id uuid.UUID, passwordHash string) error
	softDeleteFn     func(ctx context.Context, id uuid.UUID) error
	findPublicByIDFn func(ctx context.Context, id uuid.UUID) (*models.PublicUser, error)
	adminListFn      func(ctx context.Context, f models.AdminUserListFilter) ([]models.User, int, error)
	changeRoleFn     func(ctx context.Context, id uuid.UUID, role models.UserRole) (*models.User, error)
	banFn            func(ctx context.Context, id, bannedBy uuid.UUID, reason string) (*models.User, error)
	unbanFn          func(ctx context.Context, id uuid.UUID) (*models.User, error)
}

func (m *mockUserRepository) Create(ctx context.Context, user *models.User) error {
	if m.createFn != nil {
		return m.createFn(ctx, user)
	}
	return nil
}

func (m *mockUserRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	if m.findByEmailFn != nil {
		return m.findByEmailFn(ctx, email)
	}
	return nil, models.ErrUserNotFound
}

func (m *mockUserRepository) FindByLogin(ctx context.Context, login string) (*models.User, error) {
	if m.findByLoginFn != nil {
		return m.findByLoginFn(ctx, login)
	}
	return nil, models.ErrUserNotFound
}

func (m *mockUserRepository) UpdateLogin(ctx context.Context, id uuid.UUID, login string) (*models.User, error) {
	if m.updateLoginFn != nil {
		return m.updateLoginFn(ctx, id, login)
	}
	return nil, models.ErrUserNotFound
}

func (m *mockUserRepository) FindByNickname(ctx context.Context, nickname string) (*models.User, error) {
	if m.findByNicknameFn != nil {
		return m.findByNicknameFn(ctx, nickname)
	}
	return nil, models.ErrUserNotFound
}

func (m *mockUserRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	return nil, models.ErrUserNotFound
}

func (m *mockUserRepository) Update(ctx context.Context, id uuid.UUID, nickname string, university *models.University, course *int, faculty *string) (*models.User, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, id, nickname, university, course, faculty)
	}
	return nil, models.ErrUserNotFound
}

func (m *mockUserRepository) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	if m.updatePasswordFn != nil {
		return m.updatePasswordFn(ctx, id, passwordHash)
	}
	return nil
}

func (m *mockUserRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	if m.softDeleteFn != nil {
		return m.softDeleteFn(ctx, id)
	}
	return nil
}

func (m *mockUserRepository) FindPublicByID(ctx context.Context, id uuid.UUID) (*models.PublicUser, error) {
	if m.findPublicByIDFn != nil {
		return m.findPublicByIDFn(ctx, id)
	}
	return nil, models.ErrUserNotFound
}

func (m *mockUserRepository) AdminList(ctx context.Context, f models.AdminUserListFilter) ([]models.User, int, error) {
	if m.adminListFn != nil {
		return m.adminListFn(ctx, f)
	}
	return nil, 0, nil
}

func (m *mockUserRepository) ChangeRole(ctx context.Context, id uuid.UUID, role models.UserRole) (*models.User, error) {
	if m.changeRoleFn != nil {
		return m.changeRoleFn(ctx, id, role)
	}
	return nil, models.ErrUserNotFound
}

func (m *mockUserRepository) Ban(ctx context.Context, id, bannedBy uuid.UUID, reason string) (*models.User, error) {
	if m.banFn != nil {
		return m.banFn(ctx, id, bannedBy, reason)
	}
	return nil, models.ErrUserNotFound
}

func (m *mockUserRepository) Unban(ctx context.Context, id uuid.UUID) (*models.User, error) {
	if m.unbanFn != nil {
		return m.unbanFn(ctx, id)
	}
	return nil, models.ErrUserNotFound
}

// mockTokenRepository ручной мок services.TokenRepository
type mockTokenRepository struct {
	saveFn           func(ctx context.Context, token *models.RefreshToken) error
	findByHashFn     func(ctx context.Context, hash string) (*models.RefreshToken, error)
	deleteByIDFn     func(ctx context.Context, id uuid.UUID) error
	deleteByUserIDFn func(ctx context.Context, userID uuid.UUID) error
	deleteExpiredFn  func(ctx context.Context) (int64, error)
}

func (m *mockTokenRepository) Save(ctx context.Context, token *models.RefreshToken) error {
	if m.saveFn != nil {
		return m.saveFn(ctx, token)
	}
	return nil
}

func (m *mockTokenRepository) FindByHash(ctx context.Context, hash string) (*models.RefreshToken, error) {
	if m.findByHashFn != nil {
		return m.findByHashFn(ctx, hash)
	}
	return nil, models.ErrTokenNotFound
}

func (m *mockTokenRepository) DeleteByID(ctx context.Context, id uuid.UUID) error {
	if m.deleteByIDFn != nil {
		return m.deleteByIDFn(ctx, id)
	}
	return nil
}

func (m *mockTokenRepository) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	if m.deleteByUserIDFn != nil {
		return m.deleteByUserIDFn(ctx, userID)
	}
	return nil
}

func (m *mockTokenRepository) DeleteExpired(ctx context.Context) (int64, error) {
	if m.deleteExpiredFn != nil {
		return m.deleteExpiredFn(ctx)
	}
	return 0, nil
}

// mockThreadRepository ручной мок services.ThreadRepository
type mockThreadRepository struct {
	createFn                            func(ctx context.Context, authorID uuid.UUID, title, content string, tags []models.ThreadTag) (*models.Thread, error)
	findByIDFn                          func(ctx context.Context, id uuid.UUID) (*models.Thread, error)
	incrementViewsIfNotRecentlyViewedFn func(ctx context.Context, threadID, userID uuid.UUID) error
	updateFn                            func(ctx context.Context, id uuid.UUID, title, content string, tags []models.ThreadTag) (*models.Thread, error)
	softDeleteFn                        func(ctx context.Context, id uuid.UUID) error
	listFn                              func(ctx context.Context, f models.ThreadListFilter) ([]models.Thread, int, error)
	hideFn                              func(ctx context.Context, id, hiddenBy uuid.UUID, reason string) (*models.Thread, error)
}

func (m *mockThreadRepository) Create(ctx context.Context, authorID uuid.UUID, title, content string, tags []models.ThreadTag) (*models.Thread, error) {
	if m.createFn != nil {
		return m.createFn(ctx, authorID, title, content, tags)
	}
	return nil, models.ErrThreadNotFound
}

func (m *mockThreadRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Thread, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	return nil, models.ErrThreadNotFound
}

func (m *mockThreadRepository) IncrementViewsIfNotRecentlyViewed(ctx context.Context, threadID, userID uuid.UUID) error {
	if m.incrementViewsIfNotRecentlyViewedFn != nil {
		return m.incrementViewsIfNotRecentlyViewedFn(ctx, threadID, userID)
	}
	return nil
}

func (m *mockThreadRepository) Update(ctx context.Context, id uuid.UUID, title, content string, tags []models.ThreadTag) (*models.Thread, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, id, title, content, tags)
	}
	return nil, models.ErrThreadNotFound
}

func (m *mockThreadRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	if m.softDeleteFn != nil {
		return m.softDeleteFn(ctx, id)
	}
	return nil
}

func (m *mockThreadRepository) List(ctx context.Context, f models.ThreadListFilter) ([]models.Thread, int, error) {
	if m.listFn != nil {
		return m.listFn(ctx, f)
	}
	return nil, 0, nil
}

func (m *mockThreadRepository) Hide(ctx context.Context, id, hiddenBy uuid.UUID, reason string) (*models.Thread, error) {
	if m.hideFn != nil {
		return m.hideFn(ctx, id, hiddenBy, reason)
	}
	return nil, models.ErrThreadNotFound
}

// mockCommentRepository ручной мок services.CommentRepository
type mockCommentRepository struct {
	createFn       func(ctx context.Context, threadID, authorID uuid.UUID, parentID *uuid.UUID, depth int, content string) (*models.Comment, error)
	findByIDFn     func(ctx context.Context, id uuid.UUID) (*models.Comment, error)
	updateFn       func(ctx context.Context, id uuid.UUID, content string) (*models.Comment, error)
	softDeleteFn   func(ctx context.Context, id, threadID uuid.UUID) error
	listByThreadFn func(ctx context.Context, threadID uuid.UUID, page, limit int, sort string) ([]models.Comment, int, error)
	hideFn         func(ctx context.Context, id, hiddenBy uuid.UUID, reason string) (*models.Comment, error)
}

func (m *mockCommentRepository) Create(ctx context.Context, threadID, authorID uuid.UUID, parentID *uuid.UUID, depth int, content string) (*models.Comment, error) {
	if m.createFn != nil {
		return m.createFn(ctx, threadID, authorID, parentID, depth, content)
	}
	return nil, models.ErrCommentNotFound
}

func (m *mockCommentRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Comment, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	return nil, models.ErrCommentNotFound
}

func (m *mockCommentRepository) Update(ctx context.Context, id uuid.UUID, content string) (*models.Comment, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, id, content)
	}
	return nil, models.ErrCommentNotFound
}

func (m *mockCommentRepository) SoftDelete(ctx context.Context, id, threadID uuid.UUID) error {
	if m.softDeleteFn != nil {
		return m.softDeleteFn(ctx, id, threadID)
	}
	return nil
}

func (m *mockCommentRepository) Hide(ctx context.Context, id, hiddenBy uuid.UUID, reason string) (*models.Comment, error) {
	if m.hideFn != nil {
		return m.hideFn(ctx, id, hiddenBy, reason)
	}
	return nil, models.ErrCommentNotFound
}

func (m *mockCommentRepository) ListByThread(ctx context.Context, threadID uuid.UUID, page, limit int, sort string) ([]models.Comment, int, error) {
	if m.listByThreadFn != nil {
		return m.listByThreadFn(ctx, threadID, page, limit, sort)
	}
	return nil, 0, nil
}

// mockReactionRepository ручной мок services.ReactionRepository
type mockReactionRepository struct {
	upsertFn        func(ctx context.Context, userID uuid.UUID, targetType models.ReactionTargetType, targetID uuid.UUID, emoji string) (*models.Reaction, error)
	deleteFn        func(ctx context.Context, userID uuid.UUID, targetType models.ReactionTargetType, targetID uuid.UUID) error
	upsertVoteFn    func(ctx context.Context, userID uuid.UUID, targetType models.ReactionTargetType, targetID uuid.UUID, direction string) (*models.Reaction, error)
	deleteVoteFn    func(ctx context.Context, userID uuid.UUID, targetType models.ReactionTargetType, targetID uuid.UUID) error
	voteSummariesFn func(ctx context.Context, targetType models.ReactionTargetType, targetIDs []uuid.UUID, viewerID uuid.UUID) (map[uuid.UUID]models.VoteSummary, error)
}

func (m *mockReactionRepository) Upsert(ctx context.Context, userID uuid.UUID, targetType models.ReactionTargetType, targetID uuid.UUID, emoji string) (*models.Reaction, error) {
	if m.upsertFn != nil {
		return m.upsertFn(ctx, userID, targetType, targetID, emoji)
	}
	return nil, models.ErrReactionNotFound
}

func (m *mockReactionRepository) Delete(ctx context.Context, userID uuid.UUID, targetType models.ReactionTargetType, targetID uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, userID, targetType, targetID)
	}
	return nil
}

func (m *mockReactionRepository) UpsertVote(ctx context.Context, userID uuid.UUID, targetType models.ReactionTargetType, targetID uuid.UUID, direction string) (*models.Reaction, error) {
	if m.upsertVoteFn != nil {
		return m.upsertVoteFn(ctx, userID, targetType, targetID, direction)
	}
	return &models.Reaction{UserID: userID, TargetType: targetType, TargetID: targetID, Emoji: direction, Kind: models.ReactionKindVote}, nil
}

func (m *mockReactionRepository) DeleteVote(ctx context.Context, userID uuid.UUID, targetType models.ReactionTargetType, targetID uuid.UUID) error {
	if m.deleteVoteFn != nil {
		return m.deleteVoteFn(ctx, userID, targetType, targetID)
	}
	return nil
}

func (m *mockReactionRepository) VoteSummaries(ctx context.Context, targetType models.ReactionTargetType, targetIDs []uuid.UUID, viewerID uuid.UUID) (map[uuid.UUID]models.VoteSummary, error) {
	if m.voteSummariesFn != nil {
		return m.voteSummariesFn(ctx, targetType, targetIDs, viewerID)
	}
	return map[uuid.UUID]models.VoteSummary{}, nil
}

// mockReportRepository ручной мок services.ReportRepository
type mockReportRepository struct {
	createFn   func(ctx context.Context, reporterID uuid.UUID, targetType string, targetID uuid.UUID, reason string) (*models.Report, error)
	findByIDFn func(ctx context.Context, id uuid.UUID) (*models.Report, error)
	listFn     func(ctx context.Context, f models.ReportListFilter) ([]models.Report, int, error)
	reviewFn   func(ctx context.Context, id, reviewedBy uuid.UUID, status models.ReportStatus, note *string) (*models.Report, error)
}

func (m *mockReportRepository) Create(ctx context.Context, reporterID uuid.UUID, targetType string, targetID uuid.UUID, reason string) (*models.Report, error) {
	if m.createFn != nil {
		return m.createFn(ctx, reporterID, targetType, targetID, reason)
	}
	return nil, nil
}

func (m *mockReportRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Report, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	return nil, models.ErrReportNotFound
}

func (m *mockReportRepository) List(ctx context.Context, f models.ReportListFilter) ([]models.Report, int, error) {
	if m.listFn != nil {
		return m.listFn(ctx, f)
	}
	return nil, 0, nil
}

func (m *mockReportRepository) Review(ctx context.Context, id, reviewedBy uuid.UUID, status models.ReportStatus, note *string) (*models.Report, error) {
	if m.reviewFn != nil {
		return m.reviewFn(ctx, id, reviewedBy, status, note)
	}
	return nil, models.ErrReportNotFound
}

// mockAuditLogRepository ручной мок services.AuditLogRepository
type mockAuditLogRepository struct {
	createFn func(ctx context.Context, entry *models.AuditLog) error
	listFn   func(ctx context.Context, f models.AuditLogListFilter) ([]models.AuditLog, int, error)
}

func (m *mockAuditLogRepository) Create(ctx context.Context, entry *models.AuditLog) error {
	if m.createFn != nil {
		return m.createFn(ctx, entry)
	}
	entry.ID = uuid.New()
	return nil
}

func (m *mockAuditLogRepository) List(ctx context.Context, f models.AuditLogListFilter) ([]models.AuditLog, int, error) {
	if m.listFn != nil {
		return m.listFn(ctx, f)
	}
	return nil, 0, nil
}

// mockAdminStatsRepository ручной мок services.AdminStatsRepository
type mockAdminStatsRepository struct {
	statsFn func(ctx context.Context) (models.AdminStats, error)
}

func (m *mockAdminStatsRepository) Stats(ctx context.Context) (models.AdminStats, error) {
	if m.statsFn != nil {
		return m.statsFn(ctx)
	}
	return models.AdminStats{}, nil
}

// mockTextbookRepository ручной мок services.TextbookRepository
type mockTextbookRepository struct {
	createFn        func(ctx context.Context, t *models.Textbook) (*models.Textbook, error)
	findByIDFn      func(ctx context.Context, id uuid.UUID) (*models.Textbook, error)
	adminFindByIDFn func(ctx context.Context, id uuid.UUID) (*models.Textbook, error)
	updateFn        func(ctx context.Context, id uuid.UUID, t *models.Textbook) (*models.Textbook, error)
	softDeleteFn    func(ctx context.Context, id uuid.UUID) error
	listFn          func(ctx context.Context, f models.TextbookListFilter) ([]models.Textbook, int, error)
	adminListFn     func(ctx context.Context, f models.AdminTextbookListFilter) ([]models.Textbook, int, error)
}

func (m *mockTextbookRepository) Create(ctx context.Context, t *models.Textbook) (*models.Textbook, error) {
	if m.createFn != nil {
		return m.createFn(ctx, t)
	}
	return nil, models.ErrTextbookNotFound
}

func (m *mockTextbookRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Textbook, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	return nil, models.ErrTextbookNotFound
}

func (m *mockTextbookRepository) AdminFindByID(ctx context.Context, id uuid.UUID) (*models.Textbook, error) {
	if m.adminFindByIDFn != nil {
		return m.adminFindByIDFn(ctx, id)
	}
	return nil, models.ErrTextbookNotFound
}

func (m *mockTextbookRepository) Update(ctx context.Context, id uuid.UUID, t *models.Textbook) (*models.Textbook, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, id, t)
	}
	return nil, models.ErrTextbookNotFound
}

func (m *mockTextbookRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	if m.softDeleteFn != nil {
		return m.softDeleteFn(ctx, id)
	}
	return nil
}

func (m *mockTextbookRepository) List(ctx context.Context, f models.TextbookListFilter) ([]models.Textbook, int, error) {
	if m.listFn != nil {
		return m.listFn(ctx, f)
	}
	return nil, 0, nil
}

func (m *mockTextbookRepository) AdminList(ctx context.Context, f models.AdminTextbookListFilter) ([]models.Textbook, int, error) {
	if m.adminListFn != nil {
		return m.adminListFn(ctx, f)
	}
	return nil, 0, nil
}

// mockUploadRepository ручной мок services.UploadRepository
type mockUploadRepository struct {
	createFn   func(ctx context.Context, u *models.Upload) (*models.Upload, error)
	findByIDFn func(ctx context.Context, id uuid.UUID) (*models.Upload, error)
}

func (m *mockUploadRepository) Create(ctx context.Context, u *models.Upload) (*models.Upload, error) {
	if m.createFn != nil {
		return m.createFn(ctx, u)
	}
	return nil, models.ErrUploadNotFound
}

func (m *mockUploadRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Upload, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	return nil, models.ErrUploadNotFound
}

// mockObjectStorage ручной мок services.ObjectStorage
type mockObjectStorage struct {
	putFn             func(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error
	presignedGetURLFn func(ctx context.Context, key string, expiry time.Duration) (string, error)
	getFn             func(ctx context.Context, key string) (io.ReadCloser, error)
	removeFn          func(ctx context.Context, key string) error
}

func (m *mockObjectStorage) Put(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	if m.putFn != nil {
		return m.putFn(ctx, key, reader, size, contentType)
	}
	return nil
}

func (m *mockObjectStorage) PresignedGetURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	if m.presignedGetURLFn != nil {
		return m.presignedGetURLFn(ctx, key, expiry)
	}
	return "https://s3.example.org/" + key, nil
}

func (m *mockObjectStorage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	if m.getFn != nil {
		return m.getFn(ctx, key)
	}
	return io.NopCloser(strings.NewReader("")), nil
}

func (m *mockObjectStorage) Remove(ctx context.Context, key string) error {
	if m.removeFn != nil {
		return m.removeFn(ctx, key)
	}
	return nil
}

// mockCardTaskRepository ручной мок services.CardTaskRepository
type mockCardTaskRepository struct {
	createFn             func(ctx context.Context, t *models.CardTask) (*models.CardTask, error)
	findByIDFn           func(ctx context.Context, id uuid.UUID) (*models.CardTask, error)
	listFn               func(ctx context.Context, f models.CardTaskListFilter) ([]models.CardTask, int, error)
	markProcessingFn     func(ctx context.Context, id uuid.UUID) error
	markDoneFn           func(ctx context.Context, id uuid.UUID, cardsCount int) error
	markFailedFn         func(ctx context.Context, id uuid.UUID, errMsg string) error
	findDoneByCacheKeyFn func(ctx context.Context, cacheKey string) (*models.CardTask, error)
	countActiveFn        func(ctx context.Context, userID uuid.UUID) (int, error)
	countPendingBeforeFn func(ctx context.Context, createdAt time.Time) (int, error)
	listCatalogFeedFn    func(ctx context.Context, f models.CardCatalogFeedFilter) ([]models.CardCatalogEntry, int, error)
	setShareTokenFn      func(ctx context.Context, taskID uuid.UUID, token string) error
	clearShareTokenFn    func(ctx context.Context, taskID uuid.UUID) error
	findByShareTokenFn   func(ctx context.Context, token string) (*models.CardTask, error)
}

func (m *mockCardTaskRepository) Create(ctx context.Context, t *models.CardTask) (*models.CardTask, error) {
	if m.createFn != nil {
		return m.createFn(ctx, t)
	}
	cp := *t
	cp.ID = uuid.New()
	cp.Status = models.CardTaskPending
	return &cp, nil
}

func (m *mockCardTaskRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.CardTask, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	return nil, models.ErrCardTaskNotFound
}

func (m *mockCardTaskRepository) List(ctx context.Context, f models.CardTaskListFilter) ([]models.CardTask, int, error) {
	if m.listFn != nil {
		return m.listFn(ctx, f)
	}
	return nil, 0, nil
}

func (m *mockCardTaskRepository) MarkProcessing(ctx context.Context, id uuid.UUID) error {
	if m.markProcessingFn != nil {
		return m.markProcessingFn(ctx, id)
	}
	return nil
}

func (m *mockCardTaskRepository) MarkDone(ctx context.Context, id uuid.UUID, cardsCount int) error {
	if m.markDoneFn != nil {
		return m.markDoneFn(ctx, id, cardsCount)
	}
	return nil
}

func (m *mockCardTaskRepository) MarkFailed(ctx context.Context, id uuid.UUID, errMsg string) error {
	if m.markFailedFn != nil {
		return m.markFailedFn(ctx, id, errMsg)
	}
	return nil
}

func (m *mockCardTaskRepository) FindDoneByCacheKey(ctx context.Context, cacheKey string) (*models.CardTask, error) {
	if m.findDoneByCacheKeyFn != nil {
		return m.findDoneByCacheKeyFn(ctx, cacheKey)
	}
	return nil, models.ErrCardTaskNotFound
}

func (m *mockCardTaskRepository) CountActive(ctx context.Context, userID uuid.UUID) (int, error) {
	if m.countActiveFn != nil {
		return m.countActiveFn(ctx, userID)
	}
	return 0, nil
}

func (m *mockCardTaskRepository) CountPendingBefore(ctx context.Context, createdAt time.Time) (int, error) {
	if m.countPendingBeforeFn != nil {
		return m.countPendingBeforeFn(ctx, createdAt)
	}
	return 0, nil
}

func (m *mockCardTaskRepository) ListCatalogFeed(ctx context.Context, f models.CardCatalogFeedFilter) ([]models.CardCatalogEntry, int, error) {
	if m.listCatalogFeedFn != nil {
		return m.listCatalogFeedFn(ctx, f)
	}
	return nil, 0, nil
}

func (m *mockCardTaskRepository) SetShareToken(ctx context.Context, taskID uuid.UUID, token string) error {
	if m.setShareTokenFn != nil {
		return m.setShareTokenFn(ctx, taskID, token)
	}
	return nil
}

func (m *mockCardTaskRepository) ClearShareToken(ctx context.Context, taskID uuid.UUID) error {
	if m.clearShareTokenFn != nil {
		return m.clearShareTokenFn(ctx, taskID)
	}
	return nil
}

func (m *mockCardTaskRepository) FindByShareToken(ctx context.Context, token string) (*models.CardTask, error) {
	if m.findByShareTokenFn != nil {
		return m.findByShareTokenFn(ctx, token)
	}
	return nil, models.ErrCardTaskNotFound
}

// mockCardRepository ручной мок services.CardRepository
type mockCardRepository struct {
	createBatchFn          func(ctx context.Context, cards []models.Card) ([]models.Card, error)
	cloneForTaskFn         func(ctx context.Context, sourceTaskID, newTaskID uuid.UUID) ([]models.Card, error)
	findByIDFn             func(ctx context.Context, id uuid.UUID) (*models.Card, error)
	listByTaskFn           func(ctx context.Context, taskID uuid.UUID, page, limit int) ([]models.Card, int, error)
	incrementReportCountFn func(ctx context.Context, id uuid.UUID) error
}

func (m *mockCardRepository) CreateBatch(ctx context.Context, cards []models.Card) ([]models.Card, error) {
	if m.createBatchFn != nil {
		return m.createBatchFn(ctx, cards)
	}
	out := make([]models.Card, len(cards))
	for i, c := range cards {
		c.ID = uuid.New()
		out[i] = c
	}
	return out, nil
}

func (m *mockCardRepository) CloneForTask(ctx context.Context, sourceTaskID, newTaskID uuid.UUID) ([]models.Card, error) {
	if m.cloneForTaskFn != nil {
		return m.cloneForTaskFn(ctx, sourceTaskID, newTaskID)
	}
	return nil, nil
}

func (m *mockCardRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Card, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	return nil, models.ErrCardNotFound
}

func (m *mockCardRepository) ListByTask(ctx context.Context, taskID uuid.UUID, page, limit int) ([]models.Card, int, error) {
	if m.listByTaskFn != nil {
		return m.listByTaskFn(ctx, taskID, page, limit)
	}
	return nil, 0, nil
}

func (m *mockCardRepository) IncrementReportCount(ctx context.Context, id uuid.UUID) error {
	if m.incrementReportCountFn != nil {
		return m.incrementReportCountFn(ctx, id)
	}
	return nil
}

// mockCardProgressRepository ручной мок services.CardProgressRepository
type mockCardProgressRepository struct {
	createBatchDefaultFn        func(ctx context.Context, userID uuid.UUID, cardIDs []uuid.UUID) error
	findByUserAndCardFn         func(ctx context.Context, userID, cardID uuid.UUID) (*models.CardProgress, error)
	updateFn                    func(ctx context.Context, p *models.CardProgress) error
	listDueForUserFn            func(ctx context.Context, userID uuid.UUID, limit int) ([]models.ReviewCard, error)
	countDueForUserFn           func(ctx context.Context, userID uuid.UUID) (int, error)
	statsForUserFn              func(ctx context.Context, userID uuid.UUID) (models.CardsStats, error)
	distinctReviewDaysForUserFn func(ctx context.Context, userID uuid.UUID, limit int) ([]time.Time, error)
	listDueFavoritesForUserFn   func(ctx context.Context, userID uuid.UUID, limit int) ([]models.ReviewCard, error)
	countDueFavoritesForUserFn  func(ctx context.Context, userID uuid.UUID) (int, error)
}

func (m *mockCardProgressRepository) CreateBatchDefault(ctx context.Context, userID uuid.UUID, cardIDs []uuid.UUID) error {
	if m.createBatchDefaultFn != nil {
		return m.createBatchDefaultFn(ctx, userID, cardIDs)
	}
	return nil
}

func (m *mockCardProgressRepository) FindByUserAndCard(ctx context.Context, userID, cardID uuid.UUID) (*models.CardProgress, error) {
	if m.findByUserAndCardFn != nil {
		return m.findByUserAndCardFn(ctx, userID, cardID)
	}
	return nil, models.ErrCardProgressNotFound
}

func (m *mockCardProgressRepository) Update(ctx context.Context, p *models.CardProgress) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, p)
	}
	return nil
}

func (m *mockCardProgressRepository) ListDueForUser(ctx context.Context, userID uuid.UUID, limit int) ([]models.ReviewCard, error) {
	if m.listDueForUserFn != nil {
		return m.listDueForUserFn(ctx, userID, limit)
	}
	return nil, nil
}

func (m *mockCardProgressRepository) CountDueForUser(ctx context.Context, userID uuid.UUID) (int, error) {
	if m.countDueForUserFn != nil {
		return m.countDueForUserFn(ctx, userID)
	}
	return 0, nil
}

func (m *mockCardProgressRepository) ListDueFavoritesForUser(ctx context.Context, userID uuid.UUID, limit int) ([]models.ReviewCard, error) {
	if m.listDueFavoritesForUserFn != nil {
		return m.listDueFavoritesForUserFn(ctx, userID, limit)
	}
	return nil, nil
}

func (m *mockCardProgressRepository) CountDueFavoritesForUser(ctx context.Context, userID uuid.UUID) (int, error) {
	if m.countDueFavoritesForUserFn != nil {
		return m.countDueFavoritesForUserFn(ctx, userID)
	}
	return 0, nil
}

func (m *mockCardProgressRepository) StatsForUser(ctx context.Context, userID uuid.UUID) (models.CardsStats, error) {
	if m.statsForUserFn != nil {
		return m.statsForUserFn(ctx, userID)
	}
	return models.CardsStats{ByDifficulty: map[models.CardDifficulty]int{}}, nil
}

func (m *mockCardProgressRepository) DistinctReviewDaysForUser(ctx context.Context, userID uuid.UUID, limit int) ([]time.Time, error) {
	if m.distinctReviewDaysForUserFn != nil {
		return m.distinctReviewDaysForUserFn(ctx, userID, limit)
	}
	return nil, nil
}

// mockCardFavoriteRepository ручной мок services.CardFavoriteRepository
type mockCardFavoriteRepository struct {
	addFn              func(ctx context.Context, userID, cardID uuid.UUID) error
	removeFn           func(ctx context.Context, userID, cardID uuid.UUID) error
	listForUserFn      func(ctx context.Context, userID uuid.UUID, page, limit int) ([]models.Card, int, error)
	isFavoritedBatchFn func(ctx context.Context, userID uuid.UUID, cardIDs []uuid.UUID) (map[uuid.UUID]bool, error)
}

func (m *mockCardFavoriteRepository) Add(ctx context.Context, userID, cardID uuid.UUID) error {
	if m.addFn != nil {
		return m.addFn(ctx, userID, cardID)
	}
	return nil
}

func (m *mockCardFavoriteRepository) Remove(ctx context.Context, userID, cardID uuid.UUID) error {
	if m.removeFn != nil {
		return m.removeFn(ctx, userID, cardID)
	}
	return nil
}

func (m *mockCardFavoriteRepository) ListForUser(ctx context.Context, userID uuid.UUID, page, limit int) ([]models.Card, int, error) {
	if m.listForUserFn != nil {
		return m.listForUserFn(ctx, userID, page, limit)
	}
	return nil, 0, nil
}

func (m *mockCardFavoriteRepository) IsFavoritedBatch(ctx context.Context, userID uuid.UUID, cardIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	if m.isFavoritedBatchFn != nil {
		return m.isFavoritedBatchFn(ctx, userID, cardIDs)
	}
	return map[uuid.UUID]bool{}, nil
}

// mockCardRatingRepository ручной мок services.CardRatingRepository
type mockCardRatingRepository struct {
	upsertFn                 func(ctx context.Context, userID, cardID uuid.UUID, stars int) error
	deleteFn                 func(ctx context.Context, userID, cardID uuid.UUID) error
	aggregateForCardsBatchFn func(ctx context.Context, cardIDs []uuid.UUID) (map[uuid.UUID]models.CardRatingAggregate, error)
	myRatingsBatchFn         func(ctx context.Context, userID uuid.UUID, cardIDs []uuid.UUID) (map[uuid.UUID]int, error)
	listRatedByUserFn        func(ctx context.Context, userID uuid.UUID, page, limit int) ([]models.Card, int, error)
}

func (m *mockCardRatingRepository) Upsert(ctx context.Context, userID, cardID uuid.UUID, stars int) error {
	if m.upsertFn != nil {
		return m.upsertFn(ctx, userID, cardID, stars)
	}
	return nil
}

func (m *mockCardRatingRepository) Delete(ctx context.Context, userID, cardID uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, userID, cardID)
	}
	return nil
}

func (m *mockCardRatingRepository) AggregateForCardsBatch(ctx context.Context, cardIDs []uuid.UUID) (map[uuid.UUID]models.CardRatingAggregate, error) {
	if m.aggregateForCardsBatchFn != nil {
		return m.aggregateForCardsBatchFn(ctx, cardIDs)
	}
	return map[uuid.UUID]models.CardRatingAggregate{}, nil
}

func (m *mockCardRatingRepository) MyRatingsBatch(ctx context.Context, userID uuid.UUID, cardIDs []uuid.UUID) (map[uuid.UUID]int, error) {
	if m.myRatingsBatchFn != nil {
		return m.myRatingsBatchFn(ctx, userID, cardIDs)
	}
	return map[uuid.UUID]int{}, nil
}

func (m *mockCardRatingRepository) ListRatedByUser(ctx context.Context, userID uuid.UUID, page, limit int) ([]models.Card, int, error) {
	if m.listRatedByUserFn != nil {
		return m.listRatedByUserFn(ctx, userID, page, limit)
	}
	return nil, 0, nil
}

// mockTextbookChunkRepository ручной мок services.TextbookChunkRepository
type mockTextbookChunkRepository struct {
	createBatchFn       func(ctx context.Context, chunks []models.TextbookChunk) error
	existsForTextbookFn func(ctx context.Context, textbookID uuid.UUID) (bool, error)
	searchNearestFn     func(ctx context.Context, textbookID, taskID *uuid.UUID, embedding []float32, topK int) ([]models.TextbookChunk, error)
}

func (m *mockTextbookChunkRepository) CreateBatch(ctx context.Context, chunks []models.TextbookChunk) error {
	if m.createBatchFn != nil {
		return m.createBatchFn(ctx, chunks)
	}
	return nil
}

func (m *mockTextbookChunkRepository) ExistsForTextbook(ctx context.Context, textbookID uuid.UUID) (bool, error) {
	if m.existsForTextbookFn != nil {
		return m.existsForTextbookFn(ctx, textbookID)
	}
	return false, nil
}

func (m *mockTextbookChunkRepository) SearchNearest(ctx context.Context, textbookID, taskID *uuid.UUID, embedding []float32, topK int) ([]models.TextbookChunk, error) {
	if m.searchNearestFn != nil {
		return m.searchNearestFn(ctx, textbookID, taskID, embedding, topK)
	}
	return nil, nil
}

// mockTaskEnqueuer ручной мок services.TaskEnqueuer
type mockTaskEnqueuer struct {
	enqueueFn func(typename string, payload []byte) error
}

func (m *mockTaskEnqueuer) Enqueue(typename string, payload []byte) error {
	if m.enqueueFn != nil {
		return m.enqueueFn(typename, payload)
	}
	return nil
}

// mockLLMProvider ручной мок llm.Provider (internal/pkg/llm)
type mockLLMProvider struct {
	generateFn func(ctx context.Context, prompt string) (string, error)
	embedFn    func(ctx context.Context, text string) ([]float32, error)
}

func (m *mockLLMProvider) Generate(ctx context.Context, prompt string) (string, error) {
	if m.generateFn != nil {
		return m.generateFn(ctx, prompt)
	}
	return "", nil
}

func (m *mockLLMProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	if m.embedFn != nil {
		return m.embedFn(ctx, text)
	}
	return make([]float32, 8), nil
}

// mockPOIRepository ручной мок services.POIRepository
type mockPOIRepository struct {
	createFn    func(ctx context.Context, p *models.POI) (*models.POI, error)
	findByIDFn  func(ctx context.Context, id uuid.UUID) (*models.POI, error)
	updateFn    func(ctx context.Context, id uuid.UUID, p *models.POI) (*models.POI, error)
	deleteFn    func(ctx context.Context, id uuid.UUID) error
	listFn      func(ctx context.Context, f models.PoiListFilter) ([]models.POI, error)
	adminListFn func(ctx context.Context, f models.AdminPoiListFilter) ([]models.POI, int, error)
}

func (m *mockPOIRepository) Create(ctx context.Context, p *models.POI) (*models.POI, error) {
	if m.createFn != nil {
		return m.createFn(ctx, p)
	}
	cp := *p
	cp.ID = uuid.New()
	return &cp, nil
}

func (m *mockPOIRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.POI, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	return nil, models.ErrPOINotFound
}

func (m *mockPOIRepository) Update(ctx context.Context, id uuid.UUID, p *models.POI) (*models.POI, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, id, p)
	}
	return nil, models.ErrPOINotFound
}

func (m *mockPOIRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *mockPOIRepository) List(ctx context.Context, f models.PoiListFilter) ([]models.POI, error) {
	if m.listFn != nil {
		return m.listFn(ctx, f)
	}
	return nil, nil
}

func (m *mockPOIRepository) AdminList(ctx context.Context, f models.AdminPoiListFilter) ([]models.POI, int, error) {
	if m.adminListFn != nil {
		return m.adminListFn(ctx, f)
	}
	return nil, 0, nil
}

// mockPushNotifier ручной мок services.PushNotifier - для ForumService/CardService,
// которым нужен только сам факт триггера, а не полноценный PushService.
type mockPushNotifier struct {
	notifyFn func(ctx context.Context, userID uuid.UUID, kind models.NotificationKind, title, message string) error
	calls    []models.NotificationKind
}

func (m *mockPushNotifier) Notify(ctx context.Context, userID uuid.UUID, kind models.NotificationKind, title, message string) error {
	m.calls = append(m.calls, kind)
	if m.notifyFn != nil {
		return m.notifyFn(ctx, userID, kind, title, message)
	}
	return nil
}

// mockPushRepository ручной мок services.PushRepository
type mockPushRepository struct {
	createSubscriptionFn              func(ctx context.Context, userID uuid.UUID, endpoint, p256dh, auth string) (*models.PushSubscription, error)
	deleteSubscriptionByEndpointFn    func(ctx context.Context, userID uuid.UUID, endpoint string) error
	deleteSubscriptionByRawEndpointFn func(ctx context.Context, endpoint string) error
	listSubscriptionsForUserFn        func(ctx context.Context, userID uuid.UUID) ([]models.PushSubscription, error)
	getPreferencesFn                  func(ctx context.Context, userID uuid.UUID) (*models.PushPreferences, error)
	upsertPreferencesFn               func(ctx context.Context, p models.PushPreferences) (*models.PushPreferences, error)
}

func (m *mockPushRepository) CreateSubscription(ctx context.Context, userID uuid.UUID, endpoint, p256dh, auth string) (*models.PushSubscription, error) {
	if m.createSubscriptionFn != nil {
		return m.createSubscriptionFn(ctx, userID, endpoint, p256dh, auth)
	}
	return &models.PushSubscription{ID: uuid.New(), UserID: userID, Endpoint: endpoint, P256dh: p256dh, Auth: auth}, nil
}

func (m *mockPushRepository) DeleteSubscriptionByEndpoint(ctx context.Context, userID uuid.UUID, endpoint string) error {
	if m.deleteSubscriptionByEndpointFn != nil {
		return m.deleteSubscriptionByEndpointFn(ctx, userID, endpoint)
	}
	return nil
}

func (m *mockPushRepository) DeleteSubscriptionByRawEndpoint(ctx context.Context, endpoint string) error {
	if m.deleteSubscriptionByRawEndpointFn != nil {
		return m.deleteSubscriptionByRawEndpointFn(ctx, endpoint)
	}
	return nil
}

func (m *mockPushRepository) ListSubscriptionsForUser(ctx context.Context, userID uuid.UUID) ([]models.PushSubscription, error) {
	if m.listSubscriptionsForUserFn != nil {
		return m.listSubscriptionsForUserFn(ctx, userID)
	}
	return nil, nil
}

func (m *mockPushRepository) GetPreferences(ctx context.Context, userID uuid.UUID) (*models.PushPreferences, error) {
	if m.getPreferencesFn != nil {
		return m.getPreferencesFn(ctx, userID)
	}
	defaults := models.DefaultPushPreferences(userID)
	return &defaults, nil
}

func (m *mockPushRepository) UpsertPreferences(ctx context.Context, p models.PushPreferences) (*models.PushPreferences, error) {
	if m.upsertPreferencesFn != nil {
		return m.upsertPreferencesFn(ctx, p)
	}
	return &p, nil
}

// mockPushSender ручной мок services.PushSender - не бьёт по реальной сети.
type mockPushSender struct {
	sendFn func(ctx context.Context, sub models.PushSubscription, vapid config.VAPIDConfig, payload []byte) error
	sent   []models.PushSubscription
}

func (m *mockPushSender) Send(ctx context.Context, sub models.PushSubscription, vapid config.VAPIDConfig, payload []byte) error {
	m.sent = append(m.sent, sub)
	if m.sendFn != nil {
		return m.sendFn(ctx, sub, vapid, payload)
	}
	return nil
}

// mockLoginChangeRepository ручной мок services.LoginChangeRepository.
type mockLoginChangeRepository struct {
	saveFn           func(ctx context.Context, req *models.LoginChangeRequest) error
	findByCodeHashFn func(ctx context.Context, codeHash string) (*models.LoginChangeRequest, error)
	deleteByIDFn     func(ctx context.Context, id uuid.UUID) error
	deleteByUserIDFn func(ctx context.Context, userID uuid.UUID) error
}

func (m *mockLoginChangeRepository) Save(ctx context.Context, req *models.LoginChangeRequest) error {
	if m.saveFn != nil {
		return m.saveFn(ctx, req)
	}
	return nil
}

func (m *mockLoginChangeRepository) FindByCodeHash(ctx context.Context, codeHash string) (*models.LoginChangeRequest, error) {
	if m.findByCodeHashFn != nil {
		return m.findByCodeHashFn(ctx, codeHash)
	}
	return nil, models.ErrLoginChangeRequestNotFound
}

func (m *mockLoginChangeRepository) DeleteByID(ctx context.Context, id uuid.UUID) error {
	if m.deleteByIDFn != nil {
		return m.deleteByIDFn(ctx, id)
	}
	return nil
}

func (m *mockLoginChangeRepository) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	if m.deleteByUserIDFn != nil {
		return m.deleteByUserIDFn(ctx, userID)
	}
	return nil
}

// mockPasswordResetRepository ручной мок services.PasswordResetRepository.
type mockPasswordResetRepository struct {
	saveFn           func(ctx context.Context, req *models.PasswordResetRequest) error
	findByCodeHashFn func(ctx context.Context, codeHash string) (*models.PasswordResetRequest, error)
	deleteByIDFn     func(ctx context.Context, id uuid.UUID) error
	deleteByUserIDFn func(ctx context.Context, userID uuid.UUID) error
}

func (m *mockPasswordResetRepository) Save(ctx context.Context, req *models.PasswordResetRequest) error {
	if m.saveFn != nil {
		return m.saveFn(ctx, req)
	}
	return nil
}

func (m *mockPasswordResetRepository) FindByCodeHash(ctx context.Context, codeHash string) (*models.PasswordResetRequest, error) {
	if m.findByCodeHashFn != nil {
		return m.findByCodeHashFn(ctx, codeHash)
	}
	return nil, models.ErrPasswordResetRequestNotFound
}

func (m *mockPasswordResetRepository) DeleteByID(ctx context.Context, id uuid.UUID) error {
	if m.deleteByIDFn != nil {
		return m.deleteByIDFn(ctx, id)
	}
	return nil
}

func (m *mockPasswordResetRepository) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	if m.deleteByUserIDFn != nil {
		return m.deleteByUserIDFn(ctx, userID)
	}
	return nil
}

// mockEmailSender ручной мок services.EmailSender - не бьёт по реальному SMTP.
type mockEmailSender struct {
	sendFn func(to, subject, body string) error
	sent   []string
}

func (m *mockEmailSender) Send(to, subject, body string) error {
	m.sent = append(m.sent, to)
	if m.sendFn != nil {
		return m.sendFn(to, subject, body)
	}
	return nil
}
