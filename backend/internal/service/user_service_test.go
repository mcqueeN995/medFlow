package service

import (
	"context"
	"errors"
	"testing"
	"time"

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
	svc := NewUserService(userRepo, &mockTokenRepository{}, &mockAuditLogRepository{})

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
	svc := NewUserService(userRepo, &mockTokenRepository{}, &mockAuditLogRepository{})

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
	svc := NewUserService(userRepo, &mockTokenRepository{}, &mockAuditLogRepository{})

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
	svc := NewUserService(userRepo, &mockTokenRepository{}, &mockAuditLogRepository{})

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
	svc := NewUserService(userRepo, &mockTokenRepository{}, &mockAuditLogRepository{})

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
	svc := NewUserService(userRepo, tokenRepo, &mockAuditLogRepository{})

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
	svc := NewUserService(userRepo, &mockTokenRepository{}, &mockAuditLogRepository{})

	profile, err := svc.PublicProfile(context.Background(), userID)
	if err != nil {
		t.Fatalf("PublicProfile() error = %v", err)
	}
	if profile.ThreadsCount != 3 {
		t.Errorf("ThreadsCount = %d, want 3", profile.ThreadsCount)
	}
}

func TestUserService_AdminList(t *testing.T) {
	var gotFilter models.AdminUserListFilter
	userRepo := &mockUserRepository{
		adminListFn: func(ctx context.Context, f models.AdminUserListFilter) ([]models.User, int, error) {
			gotFilter = f
			return []models.User{{ID: uuid.New(), Nickname: "a"}}, 1, nil
		},
	}
	svc := NewUserService(userRepo, &mockTokenRepository{}, &mockAuditLogRepository{})

	role := models.RoleModerator
	pagination, items, err := svc.AdminList(context.Background(), models.AdminUserListFilter{Role: &role})
	if err != nil {
		t.Fatalf("AdminList() error = %v", err)
	}
	if len(items) != 1 || pagination.Total != 1 {
		t.Fatalf("AdminList() = %v (pagination=%+v), want 1 item", items, pagination)
	}
	if gotFilter.Role == nil || *gotFilter.Role != models.RoleModerator {
		t.Errorf("filter.Role = %v, want moderator", gotFilter.Role)
	}
}

func TestUserService_AdminBan_WritesAuditLog(t *testing.T) {
	actorID, targetID := uuid.New(), uuid.New()
	var bannedID, bannedBy uuid.UUID
	var bannedReason string
	userRepo := &mockUserRepository{
		banFn: func(ctx context.Context, id, by uuid.UUID, reason string) (*models.User, error) {
			bannedID, bannedBy, bannedReason = id, by, reason
			return &models.User{ID: id, BannedAt: ptrTime(), BanReason: &reason, BannedBy: &by}, nil
		},
	}
	var auditEntry *models.AuditLog
	auditRepo := &mockAuditLogRepository{
		createFn: func(ctx context.Context, entry *models.AuditLog) error {
			auditEntry = entry
			return nil
		},
	}
	svc := NewUserService(userRepo, &mockTokenRepository{}, auditRepo)

	out, err := svc.AdminBan(context.Background(), actorID, targetID, "нарушение правил")
	if err != nil {
		t.Fatalf("AdminBan() error = %v", err)
	}
	if bannedID != targetID || bannedBy != actorID || bannedReason != "нарушение правил" {
		t.Errorf("Ban() called with id=%v by=%v reason=%q", bannedID, bannedBy, bannedReason)
	}
	if out.BanReason == nil || *out.BanReason != "нарушение правил" {
		t.Errorf("AdminUser.BanReason = %v, unexpected", out.BanReason)
	}
	if auditEntry == nil || auditEntry.Action != models.AuditUserBan || auditEntry.ActorID != actorID {
		t.Fatalf("audit log entry = %+v, want AuditUserBan by %v", auditEntry, actorID)
	}
}

func TestUserService_AdminUnban_WritesAuditLog(t *testing.T) {
	actorID, targetID := uuid.New(), uuid.New()
	userRepo := &mockUserRepository{
		unbanFn: func(ctx context.Context, id uuid.UUID) (*models.User, error) {
			return &models.User{ID: id}, nil
		},
	}
	var auditEntry *models.AuditLog
	auditRepo := &mockAuditLogRepository{
		createFn: func(ctx context.Context, entry *models.AuditLog) error { auditEntry = entry; return nil },
	}
	svc := NewUserService(userRepo, &mockTokenRepository{}, auditRepo)

	if _, err := svc.AdminUnban(context.Background(), actorID, targetID); err != nil {
		t.Fatalf("AdminUnban() error = %v", err)
	}
	if auditEntry == nil || auditEntry.Action != models.AuditUserUnban {
		t.Fatalf("audit log entry = %+v, want AuditUserUnban", auditEntry)
	}
}

func TestUserService_AdminChangeRole_WritesAuditLog(t *testing.T) {
	actorID, targetID := uuid.New(), uuid.New()
	userRepo := &mockUserRepository{
		changeRoleFn: func(ctx context.Context, id uuid.UUID, role models.UserRole) (*models.User, error) {
			return &models.User{ID: id, Role: role}, nil
		},
	}
	var auditEntry *models.AuditLog
	auditRepo := &mockAuditLogRepository{
		createFn: func(ctx context.Context, entry *models.AuditLog) error { auditEntry = entry; return nil },
	}
	svc := NewUserService(userRepo, &mockTokenRepository{}, auditRepo)

	out, err := svc.AdminChangeRole(context.Background(), actorID, targetID, models.RoleModerator)
	if err != nil {
		t.Fatalf("AdminChangeRole() error = %v", err)
	}
	if out.Role != models.RoleModerator {
		t.Errorf("Role = %v, want moderator", out.Role)
	}
	if auditEntry == nil || auditEntry.Action != models.AuditUserRoleChange {
		t.Fatalf("audit log entry = %+v, want AuditUserRoleChange", auditEntry)
	}
}

func TestUserService_AdminBan_NotFound(t *testing.T) {
	userRepo := &mockUserRepository{
		banFn: func(ctx context.Context, id, by uuid.UUID, reason string) (*models.User, error) {
			return nil, models.ErrUserNotFound
		},
	}
	svc := NewUserService(userRepo, &mockTokenRepository{}, &mockAuditLogRepository{})

	_, err := svc.AdminBan(context.Background(), uuid.New(), uuid.New(), "reason")
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("AdminBan() error = %v, want ErrUserNotFound", err)
	}
}

func ptrTime() *time.Time {
	now := time.Now()
	return &now
}
