package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/medflow/backend/internal/models"
)

// это моки чтобы тестировать ТОЛЬКО service слой БЕЗ repository

// mockUserRepository ручной мок services.UserRepository
type mockUserRepository struct {
	createFn         func(ctx context.Context, user *models.User) error
	findByEmailFn    func(ctx context.Context, email string) (*models.User, error)
	findByNicknameFn func(ctx context.Context, nickname string) (*models.User, error)
	findByIDFn       func(ctx context.Context, id uuid.UUID) (*models.User, error)
	updateFn         func(ctx context.Context, id uuid.UUID, nickname string, university *models.University, course *int, faculty *string) (*models.User, error)
	softDeleteFn     func(ctx context.Context, id uuid.UUID) error
	findPublicByIDFn func(ctx context.Context, id uuid.UUID) (*models.PublicUser, error)
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
	createFn         func(ctx context.Context, authorID uuid.UUID, title, content string, tags []models.ThreadTag) (*models.Thread, error)
	findByIDFn       func(ctx context.Context, id uuid.UUID) (*models.Thread, error)
	incrementViewsFn func(ctx context.Context, id uuid.UUID) error
	updateFn         func(ctx context.Context, id uuid.UUID, title, content string, tags []models.ThreadTag) (*models.Thread, error)
	softDeleteFn     func(ctx context.Context, id uuid.UUID) error
	listFn           func(ctx context.Context, f models.ThreadListFilter) ([]models.Thread, int, error)
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

func (m *mockThreadRepository) IncrementViews(ctx context.Context, id uuid.UUID) error {
	if m.incrementViewsFn != nil {
		return m.incrementViewsFn(ctx, id)
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

// mockCommentRepository ручной мок services.CommentRepository
type mockCommentRepository struct {
	createFn       func(ctx context.Context, threadID, authorID uuid.UUID, parentID *uuid.UUID, depth int, content string) (*models.Comment, error)
	findByIDFn     func(ctx context.Context, id uuid.UUID) (*models.Comment, error)
	updateFn       func(ctx context.Context, id uuid.UUID, content string) (*models.Comment, error)
	softDeleteFn   func(ctx context.Context, id, threadID uuid.UUID) error
	listByThreadFn func(ctx context.Context, threadID uuid.UUID, page, limit int) ([]models.Comment, int, error)
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

func (m *mockCommentRepository) ListByThread(ctx context.Context, threadID uuid.UUID, page, limit int) ([]models.Comment, int, error) {
	if m.listByThreadFn != nil {
		return m.listByThreadFn(ctx, threadID, page, limit)
	}
	return nil, 0, nil
}

// mockReactionRepository ручной мок services.ReactionRepository
type mockReactionRepository struct {
	upsertFn func(ctx context.Context, userID uuid.UUID, targetType models.ReactionTargetType, targetID uuid.UUID, emoji string) (*models.Reaction, error)
	deleteFn func(ctx context.Context, userID uuid.UUID, targetType models.ReactionTargetType, targetID uuid.UUID) error
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

// mockReportRepository ручной мок services.ReportRepository
type mockReportRepository struct {
	createFn func(ctx context.Context, reporterID uuid.UUID, targetType string, targetID uuid.UUID, reason string) (*models.Report, error)
}

func (m *mockReportRepository) Create(ctx context.Context, reporterID uuid.UUID, targetType string, targetID uuid.UUID, reason string) (*models.Report, error) {
	if m.createFn != nil {
		return m.createFn(ctx, reporterID, targetType, targetID, reason)
	}
	return nil, nil
}
