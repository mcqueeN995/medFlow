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

func TestUserRepo_Update_ChangesEditableFields(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewUserRepo(pool)
	ctx := context.Background()
	user := createTestUser(t, repo, ctx)
	defer func() { _, _ = pool.Exec(ctx, "DELETE FROM users WHERE id = $1", user.ID) }()

	newNick := "updated_" + uuid.New().String()[:8]
	uni := models.UniPirogov
	course := 4
	faculty := "Педиатрический"

	updated, err := repo.Update(ctx, user.ID, newNick, &uni, &course, &faculty)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Nickname != newNick {
		t.Errorf("Nickname = %q, want %q", updated.Nickname, newNick)
	}
	if updated.University == nil || *updated.University != models.UniPirogov {
		t.Errorf("University = %v, want %v", updated.University, models.UniPirogov)
	}
	if updated.Course == nil || *updated.Course != 4 {
		t.Errorf("Course = %v, want 4", updated.Course)
	}
}

func TestUserRepo_Update_DuplicateNickname(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewUserRepo(pool)
	ctx := context.Background()
	user1 := createTestUser(t, repo, ctx)
	defer func() { _, _ = pool.Exec(ctx, "DELETE FROM users WHERE id = $1", user1.ID) }()
	user2 := createTestUser(t, repo, ctx)
	defer func() { _, _ = pool.Exec(ctx, "DELETE FROM users WHERE id = $1", user2.ID) }()

	_, err := repo.Update(ctx, user2.ID, user1.Nickname, nil, nil, nil)
	if err != models.ErrNicknameExists {
		t.Fatalf("Update() error = %v, want ErrNicknameExists", err)
	}
}

func TestUserRepo_Update_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewUserRepo(pool)

	_, err := repo.Update(context.Background(), uuid.New(), "whoever", nil, nil, nil)
	if err != models.ErrUserNotFound {
		t.Fatalf("Update() error = %v, want ErrUserNotFound", err)
	}
}

func TestUserRepo_SoftDelete_ExcludesFromFind(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewUserRepo(pool)
	ctx := context.Background()
	user := createTestUser(t, repo, ctx)
	defer func() { _, _ = pool.Exec(ctx, "DELETE FROM users WHERE id = $1", user.ID) }()

	if err := repo.SoftDelete(ctx, user.ID); err != nil {
		t.Fatalf("SoftDelete() error = %v", err)
	}

	if _, err := repo.FindByID(ctx, user.ID); err != models.ErrUserNotFound {
		t.Fatalf("FindByID() after delete error = %v, want ErrUserNotFound", err)
	}
	if _, err := repo.FindByEmail(ctx, user.Email); err != models.ErrUserNotFound {
		t.Fatalf("FindByEmail() after delete error = %v, want ErrUserNotFound", err)
	}
	if err := repo.SoftDelete(ctx, user.ID); err != models.ErrUserNotFound {
		t.Fatalf("SoftDelete() twice error = %v, want ErrUserNotFound", err)
	}
}

func TestUserRepo_FindPublicByID(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewUserRepo(pool)
	threadRepo := NewThreadRepo(pool)
	ctx := context.Background()
	user := createTestUser(t, repo, ctx)
	defer func() { _, _ = pool.Exec(ctx, "DELETE FROM users WHERE id = $1", user.ID) }()
	thread := createTestThread(t, pool, threadRepo, user.ID, "thread by this user", nil)
	_ = thread

	pu, err := repo.FindPublicByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("FindPublicByID() error = %v", err)
	}
	if pu.Nickname != user.Nickname {
		t.Errorf("Nickname = %q, want %q", pu.Nickname, user.Nickname)
	}
	if pu.ThreadsCount != 1 {
		t.Errorf("ThreadsCount = %d, want 1", pu.ThreadsCount)
	}
}

func TestUserRepo_FindPublicByID_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewUserRepo(pool)

	_, err := repo.FindPublicByID(context.Background(), uuid.New())
	if err != models.ErrUserNotFound {
		t.Fatalf("FindPublicByID() error = %v, want ErrUserNotFound", err)
	}
}
