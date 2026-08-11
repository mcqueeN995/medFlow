package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/medflow/backend/internal/models"
)

func createTestUser(t *testing.T, repo *UserRepo, ctx context.Context) *models.User {
	t.Helper()

	user := &models.User{
		ID:           uuid.New(),
		Email:        "test_user_" + uuid.New().String() + "@medflow.local",
		Login:        "test_login_" + uuid.New().String(),
		PasswordHash: "test_hash",
		Nickname:     "test_nick_" + uuid.New().String(),
		Role:         models.RoleUser,
	}

	err := repo.Create(ctx, user)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	return user
}

func TestTokenRepo_Save_Success(t *testing.T) {
	pool := setupTestDB(t)
	tokenRepo := NewTokenRepo(pool)
	userRepo := NewUserRepo(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, ctx)
	defer func() { _, _ = pool.Exec(ctx, "DELETE FROM users WHERE id = $1", user.ID) }()

	token := &models.RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: "test_hash_" + uuid.New().String(),
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	}

	err := tokenRepo.Save(ctx, token)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if token.CreatedAt.IsZero() {
		t.Error("CreatedAt was not set")
	}
}

func TestTokenRepo_Save_DuplicateHash(t *testing.T) {
	pool := setupTestDB(t)
	tokenRepo := NewTokenRepo(pool)
	userRepo := NewUserRepo(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, ctx)
	defer func() { _, _ = pool.Exec(ctx, "DELETE FROM users WHERE id = $1", user.ID) }()

	uniqueHash := "test_dup_" + uuid.New().String()

	token1 := &models.RefreshToken{
		ID: uuid.New(), UserID: user.ID, TokenHash: uniqueHash,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	_ = tokenRepo.Save(ctx, token1)

	token2 := &models.RefreshToken{
		ID: uuid.New(), UserID: user.ID, TokenHash: uniqueHash,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	err := tokenRepo.Save(ctx, token2)
	if err != models.ErrTokenHashExists {
		t.Errorf("Save() error = %v, want %v", err, models.ErrTokenHashExists)
	}
}

func TestTokenRepo_FindByHash_Success(t *testing.T) {
	pool := setupTestDB(t)
	tokenRepo := NewTokenRepo(pool)
	userRepo := NewUserRepo(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, ctx)
	defer func() { _, _ = pool.Exec(ctx, "DELETE FROM users WHERE id = $1", user.ID) }()

	uniqueHash := "test_find_" + uuid.New().String()

	token := &models.RefreshToken{
		ID: uuid.New(), UserID: user.ID, TokenHash: uniqueHash,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	_ = tokenRepo.Save(ctx, token)

	found, err := tokenRepo.FindByHash(ctx, uniqueHash)
	if err != nil {
		t.Fatalf("FindByHash() error = %v", err)
	}

	if found.ID != token.ID {
		t.Errorf("FindByHash() ID = %v, want %v", found.ID, token.ID)
	}
	if found.UserID != user.ID {
		t.Errorf("FindByHash() UserID = %v, want %v", found.UserID, user.ID)
	}
}

func TestTokenRepo_FindByHash_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewTokenRepo(pool)
	ctx := context.Background()

	_, err := repo.FindByHash(ctx, "nonexistent_hash")
	if err != models.ErrTokenNotFound {
		t.Errorf("FindByHash() error = %v, want %v", err, models.ErrTokenNotFound)
	}
}

func TestTokenRepo_DeleteByID(t *testing.T) {
	pool := setupTestDB(t)
	tokenRepo := NewTokenRepo(pool)
	userRepo := NewUserRepo(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, ctx)
	defer func() { _, _ = pool.Exec(ctx, "DELETE FROM users WHERE id = $1", user.ID) }()

	token := &models.RefreshToken{
		ID: uuid.New(), UserID: user.ID, TokenHash: "test_delete_" + uuid.New().String(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	_ = tokenRepo.Save(ctx, token)

	err := tokenRepo.DeleteByID(ctx, token.ID)
	if err != nil {
		t.Fatalf("DeleteByID() error = %v", err)
	}

	_, err = tokenRepo.FindByHash(ctx, token.TokenHash)
	if err != models.ErrTokenNotFound {
		t.Errorf("Token should be deleted, but found: %v", err)
	}
}

func TestTokenRepo_DeleteByID_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewTokenRepo(pool)
	ctx := context.Background()

	err := repo.DeleteByID(ctx, uuid.New())
	if err != models.ErrTokenNotFound {
		t.Errorf("DeleteByID() error = %v, want %v", err, models.ErrTokenNotFound)
	}
}

func TestTokenRepo_DeleteByUserID(t *testing.T) {
	pool := setupTestDB(t)
	tokenRepo := NewTokenRepo(pool)
	userRepo := NewUserRepo(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, ctx)
	defer func() { _, _ = pool.Exec(ctx, "DELETE FROM users WHERE id = $1", user.ID) }()

	for i := 0; i < 3; i++ {
		token := &models.RefreshToken{
			ID: uuid.New(), UserID: user.ID, TokenHash: "test_user_" + uuid.New().String(),
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}
		_ = tokenRepo.Save(ctx, token)
	}

	err := tokenRepo.DeleteByUserID(ctx, user.ID)
	if err != nil {
		t.Fatalf("DeleteByUserID() error = %v", err)
	}
}

func TestTokenRepo_DeleteExpired(t *testing.T) {
	pool := setupTestDB(t)
	tokenRepo := NewTokenRepo(pool)
	userRepo := NewUserRepo(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, ctx)
	defer func() { _, _ = pool.Exec(ctx, "DELETE FROM users WHERE id = $1", user.ID) }()

	expiredToken := &models.RefreshToken{
		ID: uuid.New(), UserID: user.ID, TokenHash: "test_expired_" + uuid.New().String(),
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	}
	_ = tokenRepo.Save(ctx, expiredToken)

	validToken := &models.RefreshToken{
		ID: uuid.New(), UserID: user.ID, TokenHash: "test_valid_" + uuid.New().String(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	_ = tokenRepo.Save(ctx, validToken)

	deleted, err := tokenRepo.DeleteExpired(ctx)
	if err != nil {
		t.Fatalf("DeleteExpired() error = %v", err)
	}

	if deleted != 1 {
		t.Errorf("DeleteExpired() deleted = %v, want 1", deleted)
	}
}
