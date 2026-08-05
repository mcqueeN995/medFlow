package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/medflow/backend/internal/models"
)

type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	FindByNickname(ctx context.Context, nickname string) (*models.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	Update(ctx context.Context, id uuid.UUID, nickname string, university *models.University, course *int, faculty *string) (*models.User, error)
	SoftDelete(ctx context.Context, id uuid.UUID) error
	FindPublicByID(ctx context.Context, id uuid.UUID) (*models.PublicUser, error)
}

type TokenRepository interface {
	Save(ctx context.Context, token *models.RefreshToken) error
	FindByHash(ctx context.Context, hash string) (*models.RefreshToken, error)
	DeleteByID(ctx context.Context, id uuid.UUID) error
	DeleteByUserID(ctx context.Context, userID uuid.UUID) error
	DeleteExpired(ctx context.Context) (int64, error)
}

type ThreadRepository interface {
	Create(ctx context.Context, authorID uuid.UUID, title, content string, tags []models.ThreadTag) (*models.Thread, error)
	FindByID(ctx context.Context, id uuid.UUID) (*models.Thread, error)
	IncrementViews(ctx context.Context, id uuid.UUID) error
	Update(ctx context.Context, id uuid.UUID, title, content string, tags []models.ThreadTag) (*models.Thread, error)
	SoftDelete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, f models.ThreadListFilter) ([]models.Thread, int, error)
}

type CommentRepository interface {
	Create(ctx context.Context, threadID, authorID uuid.UUID, parentID *uuid.UUID, depth int, content string) (*models.Comment, error)
	FindByID(ctx context.Context, id uuid.UUID) (*models.Comment, error)
	Update(ctx context.Context, id uuid.UUID, content string) (*models.Comment, error)
	SoftDelete(ctx context.Context, id, threadID uuid.UUID) error
	ListByThread(ctx context.Context, threadID uuid.UUID, page, limit int) ([]models.Comment, int, error)
}

type ReactionRepository interface {
	Upsert(ctx context.Context, userID uuid.UUID, targetType models.ReactionTargetType, targetID uuid.UUID, emoji string) (*models.Reaction, error)
	Delete(ctx context.Context, userID uuid.UUID, targetType models.ReactionTargetType, targetID uuid.UUID) error
}

type ReportRepository interface {
	Create(ctx context.Context, reporterID uuid.UUID, targetType string, targetID uuid.UUID, reason string) (*models.Report, error)
}
