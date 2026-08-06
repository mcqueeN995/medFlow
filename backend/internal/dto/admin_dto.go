package dto

import (
	"time"

	"github.com/medflow/backend/internal/models"
)

// AdminUser - UserProfile + поля, видимые только админу (ban_reason/banned_by).
type AdminUser struct {
	ID              string             `json:"id"`
	Email           string             `json:"email"`
	Nickname        string             `json:"nickname"`
	Role            models.UserRole    `json:"role"`
	University      *models.University `json:"university,omitempty"`
	Course          *int               `json:"course,omitempty"`
	Faculty         *string            `json:"faculty,omitempty"`
	EmailVerifiedAt *time.Time         `json:"email_verified_at,omitempty"`
	BannedAt        *time.Time         `json:"banned_at,omitempty"`
	BanReason       *string            `json:"ban_reason,omitempty"`
	BannedBy        *string            `json:"banned_by,omitempty"`
	CreatedAt       time.Time          `json:"created_at"`
}

func ToAdminUser(u *models.User) AdminUser {
	var bannedBy *string
	if u.BannedBy != nil {
		s := u.BannedBy.String()
		bannedBy = &s
	}
	return AdminUser{
		ID: u.ID.String(), Email: u.Email, Nickname: u.Nickname, Role: u.Role,
		University: u.University, Course: u.Course, Faculty: u.Faculty,
		EmailVerifiedAt: u.EmailVerifiedAt, BannedAt: u.BannedAt, BanReason: u.BanReason, BannedBy: bannedBy,
		CreatedAt: u.CreatedAt,
	}
}

type ChangeRoleRequest struct {
	Role models.UserRole `json:"role" binding:"required,oneof=guest user moderator admin"`
}

type BanRequest struct {
	Reason string `json:"reason" binding:"required,max=2000"`
}

// HideRequest переиспользуется для скрытия и треда, и комментария (тот же
// смысл, что и dto.ReportRequest, переиспользуемый под оба таргета жалоб).
type HideRequest struct {
	Reason string `json:"reason" binding:"required,max=2000"`
}

type ReviewReportRequest struct {
	Status         models.ReportStatus `json:"status" binding:"required,oneof=reviewed dismissed"`
	ResolutionNote *string             `json:"resolution_note,omitempty" binding:"omitempty,max=2000"`
}

type AdminStats struct {
	UsersTotal       int `json:"users_total"`
	UsersBanned      int `json:"users_banned"`
	ThreadsTotal     int `json:"threads_total"`
	CardTasksTotal   int `json:"card_tasks_total"`
	CardTasksPending int `json:"card_tasks_pending"`
	ActiveSessions   int `json:"active_sessions"`
}

func ToAdminStats(s models.AdminStats) AdminStats {
	return AdminStats{
		UsersTotal: s.UsersTotal, UsersBanned: s.UsersBanned, ThreadsTotal: s.ThreadsTotal,
		CardTasksTotal: s.CardTasksTotal, CardTasksPending: s.CardTasksPending, ActiveSessions: s.ActiveSessions,
	}
}

type AuditLog struct {
	ID            string                 `json:"id"`
	ActorID       string                 `json:"actor_id"`
	ActorNickname string                 `json:"actor_nickname"`
	Action        models.AuditAction     `json:"action"`
	TargetType    *string                `json:"target_type,omitempty"`
	TargetID      *string                `json:"target_id,omitempty"`
	Reason        *string                `json:"reason,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	IPAddress     *string                `json:"ip_address,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
}

func ToAuditLog(a *models.AuditLog) AuditLog {
	var targetID *string
	if a.TargetID != nil {
		s := a.TargetID.String()
		targetID = &s
	}
	return AuditLog{
		ID: a.ID.String(), ActorID: a.ActorID.String(), ActorNickname: a.ActorNickname, Action: a.Action,
		TargetType: a.TargetType, TargetID: targetID, Reason: a.Reason, IPAddress: a.IPAddress, CreatedAt: a.CreatedAt,
	}
}
