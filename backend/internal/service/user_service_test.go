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
	svc := NewUserService(userRepo, &mockTokenRepository{}, &mockAuditLogRepository{}, &mockLoginChangeRepository{}, &mockEmailSender{})

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
	svc := NewUserService(userRepo, &mockTokenRepository{}, &mockAuditLogRepository{}, &mockLoginChangeRepository{}, &mockEmailSender{})

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
	svc := NewUserService(userRepo, &mockTokenRepository{}, &mockAuditLogRepository{}, &mockLoginChangeRepository{}, &mockEmailSender{})

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
	svc := NewUserService(userRepo, &mockTokenRepository{}, &mockAuditLogRepository{}, &mockLoginChangeRepository{}, &mockEmailSender{})

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
	svc := NewUserService(userRepo, &mockTokenRepository{}, &mockAuditLogRepository{}, &mockLoginChangeRepository{}, &mockEmailSender{})

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
	svc := NewUserService(userRepo, tokenRepo, &mockAuditLogRepository{}, &mockLoginChangeRepository{}, &mockEmailSender{})

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
	svc := NewUserService(userRepo, &mockTokenRepository{}, &mockAuditLogRepository{}, &mockLoginChangeRepository{}, &mockEmailSender{})

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
	svc := NewUserService(userRepo, &mockTokenRepository{}, &mockAuditLogRepository{}, &mockLoginChangeRepository{}, &mockEmailSender{})

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
	svc := NewUserService(userRepo, &mockTokenRepository{}, auditRepo, &mockLoginChangeRepository{}, &mockEmailSender{})

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
	svc := NewUserService(userRepo, &mockTokenRepository{}, auditRepo, &mockLoginChangeRepository{}, &mockEmailSender{})

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
	svc := NewUserService(userRepo, &mockTokenRepository{}, auditRepo, &mockLoginChangeRepository{}, &mockEmailSender{})

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
	svc := NewUserService(userRepo, &mockTokenRepository{}, &mockAuditLogRepository{}, &mockLoginChangeRepository{}, &mockEmailSender{})

	_, err := svc.AdminBan(context.Background(), uuid.New(), uuid.New(), "reason")
	if !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("AdminBan() error = %v, want ErrUserNotFound", err)
	}
}

func ptrTime() *time.Time {
	now := time.Now()
	return &now
}

func TestUserService_RequestLoginChange_WrongPassword(t *testing.T) {
	userID := uuid.New()
	userRepo := &mockUserRepository{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.User, error) {
			return &models.User{ID: userID, Email: "u@medflow.local", PasswordHash: hashPasswordForTest("password123")}, nil
		},
	}
	svc := NewUserService(userRepo, &mockTokenRepository{}, &mockAuditLogRepository{}, &mockLoginChangeRepository{}, &mockEmailSender{})

	err := svc.RequestLoginChange(context.Background(), userID, dto.RequestLoginChangeRequest{NewLogin: "new_login", CurrentPassword: "wrong"})
	if !errors.Is(err, ErrInvalidCreds) {
		t.Fatalf("RequestLoginChange() error = %v, want ErrInvalidCreds", err)
	}
}

func TestUserService_RequestLoginChange_LoginTaken(t *testing.T) {
	userID := uuid.New()
	userRepo := &mockUserRepository{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.User, error) {
			return &models.User{ID: userID, Email: "u@medflow.local", PasswordHash: hashPasswordForTest("password123")}, nil
		},
		findByLoginFn: func(ctx context.Context, login string) (*models.User, error) {
			return &models.User{ID: uuid.New()}, nil // занят другим пользователем
		},
	}
	svc := NewUserService(userRepo, &mockTokenRepository{}, &mockAuditLogRepository{}, &mockLoginChangeRepository{}, &mockEmailSender{})

	err := svc.RequestLoginChange(context.Background(), userID, dto.RequestLoginChangeRequest{NewLogin: "taken_login", CurrentPassword: "password123"})
	if !errors.Is(err, ErrLoginExists) {
		t.Fatalf("RequestLoginChange() error = %v, want ErrLoginExists", err)
	}
}

func TestUserService_RequestLoginChange_Success_SendsCodeToCurrentEmail(t *testing.T) {
	userID := uuid.New()
	userRepo := &mockUserRepository{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.User, error) {
			return &models.User{ID: userID, Email: "owner@medflow.local", PasswordHash: hashPasswordForTest("password123")}, nil
		},
	}
	var savedReq *models.LoginChangeRequest
	loginChangeRepo := &mockLoginChangeRepository{
		saveFn: func(ctx context.Context, req *models.LoginChangeRequest) error {
			savedReq = req
			return nil
		},
	}
	emailSender := &mockEmailSender{}
	svc := NewUserService(userRepo, &mockTokenRepository{}, &mockAuditLogRepository{}, loginChangeRepo, emailSender)

	err := svc.RequestLoginChange(context.Background(), userID, dto.RequestLoginChangeRequest{NewLogin: "new_login", CurrentPassword: "password123"})
	if err != nil {
		t.Fatalf("RequestLoginChange() error = %v", err)
	}
	if savedReq == nil || savedReq.NewLogin != "new_login" || savedReq.UserID != userID {
		t.Fatalf("saved login change request = %+v, want NewLogin=new_login UserID=%v", savedReq, userID)
	}
	if len(emailSender.sent) != 1 || emailSender.sent[0] != "owner@medflow.local" {
		t.Fatalf("email sent to = %v, want [owner@medflow.local] (текущий, не новый, email)", emailSender.sent)
	}
}

func TestUserService_ConfirmLoginChange_InvalidCode(t *testing.T) {
	userID := uuid.New()
	svc := NewUserService(&mockUserRepository{}, &mockTokenRepository{}, &mockAuditLogRepository{}, &mockLoginChangeRepository{}, &mockEmailSender{})

	_, err := svc.ConfirmLoginChange(context.Background(), userID, "000000")
	if !errors.Is(err, ErrLoginChangeCodeInvalid) {
		t.Fatalf("ConfirmLoginChange() error = %v, want ErrLoginChangeCodeInvalid", err)
	}
}

func TestUserService_ConfirmLoginChange_CodeBelongsToAnotherUser(t *testing.T) {
	userID := uuid.New()
	loginChangeRepo := &mockLoginChangeRepository{
		findByCodeHashFn: func(ctx context.Context, codeHash string) (*models.LoginChangeRequest, error) {
			return &models.LoginChangeRequest{ID: uuid.New(), UserID: uuid.New(), NewLogin: "x", ExpiresAt: time.Now().Add(time.Minute)}, nil
		},
	}
	svc := NewUserService(&mockUserRepository{}, &mockTokenRepository{}, &mockAuditLogRepository{}, loginChangeRepo, &mockEmailSender{})

	_, err := svc.ConfirmLoginChange(context.Background(), userID, "123456")
	if !errors.Is(err, ErrLoginChangeCodeInvalid) {
		t.Fatalf("ConfirmLoginChange() error = %v, want ErrLoginChangeCodeInvalid (чужой запрос)", err)
	}
}

func TestUserService_ConfirmLoginChange_Expired(t *testing.T) {
	userID := uuid.New()
	reqID := uuid.New()
	var deletedID uuid.UUID
	loginChangeRepo := &mockLoginChangeRepository{
		findByCodeHashFn: func(ctx context.Context, codeHash string) (*models.LoginChangeRequest, error) {
			return &models.LoginChangeRequest{ID: reqID, UserID: userID, NewLogin: "x", ExpiresAt: time.Now().Add(-time.Minute)}, nil
		},
		deleteByIDFn: func(ctx context.Context, id uuid.UUID) error {
			deletedID = id
			return nil
		},
	}
	svc := NewUserService(&mockUserRepository{}, &mockTokenRepository{}, &mockAuditLogRepository{}, loginChangeRepo, &mockEmailSender{})

	_, err := svc.ConfirmLoginChange(context.Background(), userID, "123456")
	if !errors.Is(err, ErrLoginChangeCodeInvalid) {
		t.Fatalf("ConfirmLoginChange() error = %v, want ErrLoginChangeCodeInvalid", err)
	}
	if deletedID != reqID {
		t.Errorf("expired request was not cleaned up: deletedID = %v, want %v", deletedID, reqID)
	}
}

func TestUserService_ConfirmLoginChange_Success(t *testing.T) {
	userID := uuid.New()
	reqID := uuid.New()
	loginChangeRepo := &mockLoginChangeRepository{
		findByCodeHashFn: func(ctx context.Context, codeHash string) (*models.LoginChangeRequest, error) {
			return &models.LoginChangeRequest{ID: reqID, UserID: userID, NewLogin: "new_login", ExpiresAt: time.Now().Add(time.Minute)}, nil
		},
	}
	userRepo := &mockUserRepository{
		updateLoginFn: func(ctx context.Context, id uuid.UUID, login string) (*models.User, error) {
			return &models.User{ID: id, Login: login}, nil
		},
	}
	svc := NewUserService(userRepo, &mockTokenRepository{}, &mockAuditLogRepository{}, loginChangeRepo, &mockEmailSender{})

	profile, err := svc.ConfirmLoginChange(context.Background(), userID, "123456")
	if err != nil {
		t.Fatalf("ConfirmLoginChange() error = %v", err)
	}
	if profile.Login != "new_login" {
		t.Errorf("Login = %q, want new_login", profile.Login)
	}
}
