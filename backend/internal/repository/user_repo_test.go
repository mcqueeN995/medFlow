package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/medflow/backend/internal/models"
)

func TestUserRepo_Create_Success(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewUserRepo(pool)
	ctx := context.Background()

	uniqueEmail := "test_create_" + uuid.New().String() + "@medflow.local"
	uniqueNick := "test_create_" + uuid.New().String()

	user := &models.User{
		ID:           uuid.New(),
		Email:        uniqueEmail,
		PasswordHash: "hashed_password",
		Nickname:     uniqueNick,
		Role:         models.RoleUser,
	}

	err := repo.Create(ctx, user)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if user.CreatedAt.IsZero() {
		t.Error("CreatedAt was not set")
	}
	if user.UpdatedAt.IsZero() {
		t.Error("UpdatedAt was not set")
	}

	_, _ = pool.Exec(ctx, "DELETE FROM users WHERE id = $1", user.ID)
}

func TestUserRepo_Create_DuplicateEmail(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewUserRepo(pool)
	ctx := context.Background()

	uniqueEmail := "test_dup_" + uuid.New().String() + "@medflow.local"
	uniqueNick := "test_dup_" + uuid.New().String()

	user1 := &models.User{
		ID: uuid.New(), Email: uniqueEmail, PasswordHash: "hash",
		Nickname: uniqueNick, Role: models.RoleUser,
	}
	_ = repo.Create(ctx, user1)
	defer func() { _, _ = pool.Exec(ctx, "DELETE FROM users WHERE id = $1", user1.ID) }()

	user2 := &models.User{
		ID: uuid.New(), Email: uniqueEmail, PasswordHash: "hash",
		Nickname: "another_nick", Role: models.RoleUser,
	}

	err := repo.Create(ctx, user2)
	if err != models.ErrEmailAlreadyExists {
		t.Errorf("Create() error = %v, want %v", err, models.ErrEmailAlreadyExists)
	}
}

func TestUserRepo_FindByEmail_Success(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewUserRepo(pool)
	ctx := context.Background()

	uniqueEmail := "test_find_" + uuid.New().String() + "@medflow.local"
	uniqueNick := "test_find_" + uuid.New().String()

	user := &models.User{
		ID: uuid.New(), Email: uniqueEmail, PasswordHash: "hash",
		Nickname: uniqueNick, Role: models.RoleAdmin,
	}
	_ = repo.Create(ctx, user)
	defer func() { _, _ = pool.Exec(ctx, "DELETE FROM users WHERE id = $1", user.ID) }()

	found, err := repo.FindByEmail(ctx, uniqueEmail)
	if err != nil {
		t.Fatalf("FindByEmail() error = %v", err)
	}

	if found.ID != user.ID {
		t.Errorf("FindByEmail() ID = %v, want %v", found.ID, user.ID)
	}
	if found.Role != models.RoleAdmin {
		t.Errorf("FindByEmail() Role = %v, want %v", found.Role, models.RoleAdmin)
	}
}

func TestUserRepo_FindByEmail_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewUserRepo(pool)
	ctx := context.Background()

	_, err := repo.FindByEmail(ctx, "nonexistent@medflow.local")
	if err != models.ErrUserNotFound {
		t.Errorf("FindByEmail() error = %v, want %v", err, models.ErrUserNotFound)
	}
}

func TestUserRepo_FindByNickname_Success(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewUserRepo(pool)
	ctx := context.Background()

	uniqueEmail := "test_nick_" + uuid.New().String() + "@medflow.local"
	uniqueNick := "test_nick_" + uuid.New().String()

	user := &models.User{
		ID: uuid.New(), Email: uniqueEmail, PasswordHash: "hash",
		Nickname: uniqueNick, Role: models.RoleUser,
	}
	_ = repo.Create(ctx, user)
	defer func() { _, _ = pool.Exec(ctx, "DELETE FROM users WHERE id = $1", user.ID) }()

	found, err := repo.FindByNickname(ctx, uniqueNick)
	if err != nil {
		t.Fatalf("FindByNickname() error = %v", err)
	}

	if found.Nickname != uniqueNick {
		t.Errorf("FindByNickname() Nickname = %v, want %v", found.Nickname, uniqueNick)
	}
}
