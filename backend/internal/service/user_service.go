package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/medflow/backend/internal/dto"
	"github.com/medflow/backend/internal/models"
	"github.com/medflow/backend/internal/pkg/password"
)

var ErrUserNotFound = errors.New("user not found")

type UserService struct {
	userRepo     UserRepository
	tokenRepo    TokenRepository
	auditLogRepo AuditLogRepository
}

func NewUserService(userRepo UserRepository, tokenRepo TokenRepository, auditLogRepo AuditLogRepository) *UserService {
	return &UserService{userRepo: userRepo, tokenRepo: tokenRepo, auditLogRepo: auditLogRepo}
}

func (s *UserService) Me(ctx context.Context, userID uuid.UUID) (*dto.UserProfile, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, s.mapErr(err)
	}
	profile := dto.ToUserProfile(user)
	return &profile, nil
}

func (s *UserService) UpdateProfile(ctx context.Context, userID uuid.UUID, req dto.UpdateProfileRequest) (*dto.UserProfile, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, s.mapErr(err)
	}

	nickname, university, course, faculty := user.Nickname, user.University, user.Course, user.Faculty
	if req.Nickname != nil {
		nickname = *req.Nickname
	}
	if req.University != nil {
		university = req.University
	}
	if req.Course != nil {
		course = req.Course
	}
	if req.Faculty != nil {
		faculty = req.Faculty
	}

	updated, err := s.userRepo.Update(ctx, userID, nickname, university, course, faculty)
	if err != nil {
		if errors.Is(err, models.ErrNicknameExists) {
			return nil, ErrNicknameExists
		}
		return nil, s.mapErr(err)
	}
	profile := dto.ToUserProfile(updated)
	return &profile, nil
}

// DeleteAccount - soft delete + отзыв всех refresh-токенов, чтобы уже
// выданные access-токены не пережили удаление дольше своего короткого TTL
// и пользователя нельзя было "разлогинить обратно" через /auth/refresh.
func (s *UserService) DeleteAccount(ctx context.Context, userID uuid.UUID, currentPassword string) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return s.mapErr(err)
	}

	valid, err := password.Compare(currentPassword, user.PasswordHash)
	if err != nil || !valid {
		return ErrInvalidCreds
	}

	if err := s.userRepo.SoftDelete(ctx, userID); err != nil {
		return s.mapErr(err)
	}
	return s.tokenRepo.DeleteByUserID(ctx, userID)
}

func (s *UserService) PublicProfile(ctx context.Context, userID uuid.UUID) (*dto.PublicUser, error) {
	pu, err := s.userRepo.FindPublicByID(ctx, userID)
	if err != nil {
		return nil, s.mapErr(err)
	}
	out := dto.ToPublicUser(*pu)
	return &out, nil
}

// ==================== ADMIN ====================

func (s *UserService) AdminList(ctx context.Context, f models.AdminUserListFilter) (*dto.Pagination, []dto.AdminUser, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Limit < 1 || f.Limit > 100 {
		f.Limit = 20
	}

	users, total, err := s.userRepo.AdminList(ctx, f)
	if err != nil {
		return nil, nil, err
	}
	items := make([]dto.AdminUser, len(users))
	for i := range users {
		items[i] = dto.ToAdminUser(&users[i])
	}
	pagination := dto.NewPagination(f.Page, f.Limit, total)
	return &pagination, items, nil
}

func (s *UserService) AdminChangeRole(ctx context.Context, actorID, id uuid.UUID, role models.UserRole) (*dto.AdminUser, error) {
	updated, err := s.userRepo.ChangeRole(ctx, id, role)
	if err != nil {
		return nil, s.mapErr(err)
	}
	s.writeAudit(ctx, actorID, models.AuditUserRoleChange, "user", id, nil)
	out := dto.ToAdminUser(updated)
	return &out, nil
}

func (s *UserService) AdminBan(ctx context.Context, actorID, id uuid.UUID, reason string) (*dto.AdminUser, error) {
	banned, err := s.userRepo.Ban(ctx, id, actorID, reason)
	if err != nil {
		return nil, s.mapErr(err)
	}
	s.writeAudit(ctx, actorID, models.AuditUserBan, "user", id, &reason)
	out := dto.ToAdminUser(banned)
	return &out, nil
}

func (s *UserService) AdminUnban(ctx context.Context, actorID, id uuid.UUID) (*dto.AdminUser, error) {
	unbanned, err := s.userRepo.Unban(ctx, id)
	if err != nil {
		return nil, s.mapErr(err)
	}
	s.writeAudit(ctx, actorID, models.AuditUserUnban, "user", id, nil)
	out := dto.ToAdminUser(unbanned)
	return &out, nil
}

// writeAudit - лучшее старание: сбой записи аудит-лога не должен проваливать
// уже выполненное административное действие (бан/смена роли и т.п. уже
// применены в БД к моменту вызова).
func (s *UserService) writeAudit(ctx context.Context, actorID uuid.UUID, action models.AuditAction, targetType string, targetID uuid.UUID, reason *string) {
	_ = s.auditLogRepo.Create(ctx, &models.AuditLog{ActorID: actorID, Action: action, TargetType: &targetType, TargetID: &targetID, Reason: reason})
}

func (s *UserService) mapErr(err error) error {
	if errors.Is(err, models.ErrUserNotFound) {
		return ErrUserNotFound
	}
	return err
}
