package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/medflow/backend/internal/dto"
	"github.com/medflow/backend/internal/models"
	"github.com/medflow/backend/internal/pkg/password"
)

func TestUserService_Me_Success(t *testing.T) {
	userID := uuid.New()
	userRepo := &mockUserRepository{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.User, error) {
			return &models.User{ID: userID, Nickname: "nik", Role: models.RoleUser}, nil
		},
	}
	svc := NewUserService(userRepo, &mockTokenRepository{})

	profile, err := svc.Me(context.Background(), userID)
	if err != nil {
		t.Fatalf("Me() error = %v", err)
	}
	if profile.Nickname != "nik" {
		t.Errorf("Nickname = %q, want nik", profile.Nickname)
	}
}

func TestUserService_Me_NotFound(t *testing.T) {
	userRepo := &mockUserRepository{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.User, error) {
			return nil, models.ErrUserNotFound
		},
	}
	svc := NewUserService(userRepo, &mockTokenRepository{})

	_, err := svc.Me(context.Background(), uuid.New())
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("Me() error = %v, want ErrUserNotFound", err)
	}
}

func TestUserService_UpdateProfile_PartialFieldsPreserved(t *testing.T) {
	userID := uuid.New()
	faculty := "Педиатрический"
	userRepo := &mockUserRepository{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.User, error) {
			return &models.User{ID: userID, Nickname: "old_nick", Faculty: &faculty}, nil
		},
		updateFn: func(ctx context.Context, id uuid.UUID, nickname string, university *models.University, course *int, faculty *string) (*models.User, error) {
			if nickname != "new_nick" {
				t.Errorf("nickname = %q, want new_nick", nickname)
			}
			if faculty == nil || *faculty != "Педиатрический" {
				t.Errorf("faculty should be preserved when not in request, got %v", faculty)
			}
			return &models.User{ID: id, Nickname: nickname, Faculty: faculty}, nil
		},
	}
	svc := NewUserService(userRepo, &mockTokenRepository{})

	newNick := "new_nick"
	profile, err := svc.UpdateProfile(context.Background(), userID, dto.UpdateProfileRequest{Nickname: &newNick})
	if err != nil {
		t.Fatalf("UpdateProfile() error = %v", err)
	}
	if profile.Nickname != "new_nick" {
		t.Errorf("Nickname = %q, want new_nick", profile.Nickname)
	}
}

func TestUserService_UpdateProfile_NicknameTaken(t *testing.T) {
	userID := uuid.New()
	userRepo := &mockUserRepository{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.User, error) {
			return &models.User{ID: userID, Nickname: "old_nick"}, nil
		},
		updateFn: func(ctx context.Context, id uuid.UUID, nickname string, university *models.University, course *int, faculty *string) (*models.User, error) {
			return nil, models.ErrNicknameExists
		},
	}
	svc := NewUserService(userRepo, &mockTokenRepository{})

	taken := "taken_nick"
	_, err := svc.UpdateProfile(context.Background(), userID, dto.UpdateProfileRequest{Nickname: &taken})
	if !errors.Is(err, ErrNicknameExists) {
		t.Fatalf("UpdateProfile() error = %v, want ErrNicknameExists", err)
	}
}

func TestUserService_DeleteAccount_WrongPassword(t *testing.T) {
	userID := uuid.New()
	hash, err := password.Hash("correct-password")
	if err != nil {
		t.Fatalf("password.Hash() error = %v", err)
	}
	userRepo := &mockUserRepository{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.User, error) {
			return &models.User{ID: userID, PasswordHash: hash}, nil
		},
	}
	svc := NewUserService(userRepo, &mockTokenRepository{})

	err = svc.DeleteAccount(context.Background(), userID, "wrong-password")
	if !errors.Is(err, ErrInvalidCreds) {
		t.Fatalf("DeleteAccount() error = %v, want ErrInvalidCreds", err)
	}
}

func TestUserService_DeleteAccount_Success_RevokesTokens(t *testing.T) {
	userID := uuid.New()
	hash, err := password.Hash("correct-password")
	if err != nil {
		t.Fatalf("password.Hash() error = %v", err)
	}
	softDeleteCalled := false
	revokeCalled := false
	userRepo := &mockUserRepository{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.User, error) {
			return &models.User{ID: userID, PasswordHash: hash}, nil
		},
		softDeleteFn: func(ctx context.Context, id uuid.UUID) error {
			softDeleteCalled = true
			return nil
		},
	}
	tokenRepo := &mockTokenRepository{
		deleteByUserIDFn: func(ctx context.Context, id uuid.UUID) error {
			revokeCalled = true
			return nil
		},
	}
	svc := NewUserService(userRepo, tokenRepo)

	if err := svc.DeleteAccount(context.Background(), userID, "correct-password"); err != nil {
		t.Fatalf("DeleteAccount() error = %v", err)
	}
	if !softDeleteCalled {
		t.Error("expected SoftDelete to be called")
	}
	if !revokeCalled {
		t.Error("expected refresh tokens to be revoked")
	}
}

func TestUserService_PublicProfile_Success(t *testing.T) {
	userID := uuid.New()
	userRepo := &mockUserRepository{
		findPublicByIDFn: func(ctx context.Context, id uuid.UUID) (*models.PublicUser, error) {
			return &models.PublicUser{ID: userID, Nickname: "nik", ThreadsCount: 3}, nil
		},
	}
	svc := NewUserService(userRepo, &mockTokenRepository{})

	profile, err := svc.PublicProfile(context.Background(), userID)
	if err != nil {
		t.Fatalf("PublicProfile() error = %v", err)
	}
	if profile.ThreadsCount != 3 {
		t.Errorf("ThreadsCount = %d, want 3", profile.ThreadsCount)
	}
}
