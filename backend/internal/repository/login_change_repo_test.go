package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/medflow/backend/internal/models"
)

func TestLoginChangeRepo_Save_And_FindByCodeHash(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewLoginChangeRepo(pool)
	userRepo := NewUserRepo(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, ctx)
	defer func() { _, _ = pool.Exec(ctx, "DELETE FROM users WHERE id = $1", user.ID) }()

	req := &models.LoginChangeRequest{
		ID:        uuid.New(),
		UserID:    user.ID,
		NewLogin:  "new_login_" + uuid.New().String(),
		CodeHash:  "code_hash_" + uuid.New().String(),
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}

	if err := repo.Save(ctx, req); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	defer func() { _ = repo.DeleteByID(ctx, req.ID) }()

	found, err := repo.FindByCodeHash(ctx, req.CodeHash)
	if err != nil {
		t.Fatalf("FindByCodeHash() error = %v", err)
	}
	if found.UserID != user.ID || found.NewLogin != req.NewLogin {
		t.Errorf("FindByCodeHash() = %+v, want matching UserID/NewLogin for %+v", found, req)
	}
}

func TestLoginChangeRepo_FindByCodeHash_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewLoginChangeRepo(pool)

	_, err := repo.FindByCodeHash(context.Background(), "nonexistent_hash")
	if err != models.ErrLoginChangeRequestNotFound {
		t.Errorf("FindByCodeHash() error = %v, want %v", err, models.ErrLoginChangeRequestNotFound)
	}
}

func TestLoginChangeRepo_DeleteByUserID_RemovesPriorRequests(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewLoginChangeRepo(pool)
	userRepo := NewUserRepo(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, ctx)
	defer func() { _, _ = pool.Exec(ctx, "DELETE FROM users WHERE id = $1", user.ID) }()

	req := &models.LoginChangeRequest{
		ID: uuid.New(), UserID: user.ID, NewLogin: "x", CodeHash: "hash_" + uuid.New().String(),
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}
	if err := repo.Save(ctx, req); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if err := repo.DeleteByUserID(ctx, user.ID); err != nil {
		t.Fatalf("DeleteByUserID() error = %v", err)
	}

	if _, err := repo.FindByCodeHash(ctx, req.CodeHash); err != models.ErrLoginChangeRequestNotFound {
		t.Errorf("FindByCodeHash() after DeleteByUserID error = %v, want %v", err, models.ErrLoginChangeRequestNotFound)
	}
}
