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
	userRepo  UserRepository
	tokenRepo TokenRepository
}

func NewUserService(userRepo UserRepository, tokenRepo TokenRepository) *UserService {
	return &UserService{userRepo: userRepo, tokenRepo: tokenRepo}
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

func (s *UserService) mapErr(err error) error {
	if errors.Is(err, models.ErrUserNotFound) {
		return ErrUserNotFound
	}
	return err
}
