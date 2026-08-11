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
	uniqueLogin := "test_create_" + uuid.New().String()
	uniqueNick := "test_create_" + uuid.New().String()

	user := &models.User{
		ID:           uuid.New(),
		Email:        uniqueEmail,
		Login:        uniqueLogin,
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
		ID: uuid.New(), Email: uniqueEmail, Login: "login_" + uuid.New().String(), PasswordHash: "hash",
		Nickname: uniqueNick, Role: models.RoleUser,
	}
	_ = repo.Create(ctx, user1)
	defer func() { _, _ = pool.Exec(ctx, "DELETE FROM users WHERE id = $1", user1.ID) }()

	user2 := &models.User{
		ID: uuid.New(), Email: uniqueEmail, Login: "login_" + uuid.New().String(), PasswordHash: "hash",
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
	uniqueLogin := "test_find_" + uuid.New().String()
	uniqueNick := "test_find_" + uuid.New().String()

	user := &models.User{
		ID: uuid.New(), Email: uniqueEmail, Login: uniqueLogin, PasswordHash: "hash",
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
	uniqueLogin := "test_nick_" + uuid.New().String()
	uniqueNick := "test_nick_" + uuid.New().String()

	user := &models.User{
		ID: uuid.New(), Email: uniqueEmail, Login: uniqueLogin, PasswordHash: "hash",
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

func TestUserRepo_FindByLogin_Success(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewUserRepo(pool)
	ctx := context.Background()

	uniqueEmail := "test_login_" + uuid.New().String() + "@medflow.local"
	uniqueLogin := "test_login_" + uuid.New().String()
	uniqueNick := "test_login_" + uuid.New().String()

	user := &models.User{
		ID: uuid.New(), Email: uniqueEmail, Login: uniqueLogin, PasswordHash: "hash",
		Nickname: uniqueNick, Role: models.RoleUser,
	}
	_ = repo.Create(ctx, user)
	defer func() { _, _ = pool.Exec(ctx, "DELETE FROM users WHERE id = $1", user.ID) }()

	found, err := repo.FindByLogin(ctx, uniqueLogin)
	if err != nil {
		t.Fatalf("FindByLogin() error = %v", err)
	}
	if found.ID != user.ID {
		t.Errorf("FindByLogin() ID = %v, want %v", found.ID, user.ID)
	}
}

func TestUserRepo_FindByLogin_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewUserRepo(pool)

	_, err := repo.FindByLogin(context.Background(), "nonexistent_login")
	if err != models.ErrUserNotFound {
		t.Errorf("FindByLogin() error = %v, want %v", err, models.ErrUserNotFound)
	}
}

func TestUserRepo_UpdateLogin_Success(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewUserRepo(pool)
	ctx := context.Background()
	user := createTestUser(t, repo, ctx)
	defer func() { _, _ = pool.Exec(ctx, "DELETE FROM users WHERE id = $1", user.ID) }()

	newLogin := "updated_login_" + uuid.New().String()
	updated, err := repo.UpdateLogin(ctx, user.ID, newLogin)
	if err != nil {
		t.Fatalf("UpdateLogin() error = %v", err)
	}
	if updated.Login != newLogin {
		t.Errorf("Login = %q, want %q", updated.Login, newLogin)
	}
}

func TestUserRepo_UpdateLogin_DuplicateLogin(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewUserRepo(pool)
	ctx := context.Background()
	user1 := createTestUser(t, repo, ctx)
	defer func() { _, _ = pool.Exec(ctx, "DELETE FROM users WHERE id = $1", user1.ID) }()
	user2 := createTestUser(t, repo, ctx)
	defer func() { _, _ = pool.Exec(ctx, "DELETE FROM users WHERE id = $1", user2.ID) }()

	_, err := repo.UpdateLogin(ctx, user2.ID, user1.Login)
	if err != models.ErrLoginExists {
		t.Fatalf("UpdateLogin() error = %v, want ErrLoginExists", err)
	}
}

func TestUserRepo_UpdatePassword_Success(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewUserRepo(pool)
	ctx := context.Background()
	user := createTestUser(t, repo, ctx)
	defer func() { _, _ = pool.Exec(ctx, "DELETE FROM users WHERE id = $1", user.ID) }()

	newHash := "new_hash_" + uuid.New().String()
	if err := repo.UpdatePassword(ctx, user.ID, newHash); err != nil {
		t.Fatalf("UpdatePassword() error = %v", err)
	}

	found, err := repo.FindByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if found.PasswordHash != newHash {
		t.Errorf("PasswordHash = %q, want %q", found.PasswordHash, newHash)
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

func TestUserRepo_AdminList_FiltersByRoleAndBanned(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewUserRepo(pool)
	ctx := context.Background()
	admin := createTestUser(t, repo, ctx)
	defer func() { _, _ = pool.Exec(ctx, "DELETE FROM users WHERE id = $1", admin.ID) }()
	if _, err := repo.ChangeRole(ctx, admin.ID, models.RoleAdmin); err != nil {
		t.Fatalf("ChangeRole() error = %v", err)
	}
	regular := createTestUser(t, repo, ctx)
	defer func() { _, _ = pool.Exec(ctx, "DELETE FROM users WHERE id = $1", regular.ID) }()

	role := models.RoleAdmin
	items, _, err := repo.AdminList(ctx, models.AdminUserListFilter{Role: &role, Page: 1, Limit: 100})
	if err != nil {
		t.Fatalf("AdminList() error = %v", err)
	}
	// Не полагаемся на точный total/len - таблица users общая для всей сессии
	// тестов (и может уже содержать других admin, напр. постоянный тестовый
	// аккаунт admin@medflow.local) - проверяем только, что наш свежесозданный
	// admin присутствует в выдаче, а обычный пользователь - нет.
	found, foundRegular := false, false
	for _, u := range items {
		if u.ID == admin.ID {
			found = true
		}
		if u.ID == regular.ID {
			foundRegular = true
		}
	}
	if !found {
		t.Fatalf("AdminList(role=admin) = %v, want it to include %v", items, admin.ID)
	}
	if foundRegular {
		t.Fatalf("AdminList(role=admin) = %v, must not include non-admin user %v", items, regular.ID)
	}
}

func TestUserRepo_ChangeRole(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewUserRepo(pool)
	ctx := context.Background()
	user := createTestUser(t, repo, ctx)
	defer func() { _, _ = pool.Exec(ctx, "DELETE FROM users WHERE id = $1", user.ID) }()

	updated, err := repo.ChangeRole(ctx, user.ID, models.RoleModerator)
	if err != nil {
		t.Fatalf("ChangeRole() error = %v", err)
	}
	if updated.Role != models.RoleModerator {
		t.Errorf("Role = %v, want moderator", updated.Role)
	}
}

func TestUserRepo_ChangeRole_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewUserRepo(pool)

	_, err := repo.ChangeRole(context.Background(), uuid.New(), models.RoleModerator)
	if err != models.ErrUserNotFound {
		t.Fatalf("ChangeRole() error = %v, want ErrUserNotFound", err)
	}
}

func TestUserRepo_Ban_And_Unban(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewUserRepo(pool)
	ctx := context.Background()
	user := createTestUser(t, repo, ctx)
	defer func() { _, _ = pool.Exec(ctx, "DELETE FROM users WHERE id = $1", user.ID) }()
	admin := createTestUser(t, repo, ctx)
	defer func() { _, _ = pool.Exec(ctx, "DELETE FROM users WHERE id = $1", admin.ID) }()

	banned, err := repo.Ban(ctx, user.ID, admin.ID, "нарушение правил")
	if err != nil {
		t.Fatalf("Ban() error = %v", err)
	}
	if banned.BannedAt == nil || banned.BanReason == nil || *banned.BanReason != "нарушение правил" {
		t.Fatalf("Ban() result = %+v, want banned with reason", banned)
	}
	if banned.BannedBy == nil || *banned.BannedBy != admin.ID {
		t.Errorf("BannedBy = %v, want %v", banned.BannedBy, admin.ID)
	}

	unbanned, err := repo.Unban(ctx, user.ID)
	if err != nil {
		t.Fatalf("Unban() error = %v", err)
	}
	if unbanned.BannedAt != nil || unbanned.BanReason != nil || unbanned.BannedBy != nil {
		t.Fatalf("Unban() result = %+v, want all ban fields cleared", unbanned)
	}
}

func TestUserRepo_Ban_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewUserRepo(pool)

	_, err := repo.Ban(context.Background(), uuid.New(), uuid.New(), "reason")
	if err != models.ErrUserNotFound {
		t.Fatalf("Ban() error = %v, want ErrUserNotFound", err)
	}
}
