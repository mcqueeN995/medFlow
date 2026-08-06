package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type AuditAction string

const (
	AuditUserBan        AuditAction = "user_ban"
	AuditUserUnban      AuditAction = "user_unban"
	AuditUserRoleChange AuditAction = "user_role_change"
	AuditThreadHide     AuditAction = "thread_hide"
	AuditThreadUnhide   AuditAction = "thread_unhide"
	AuditThreadDelete   AuditAction = "thread_delete"
	AuditCommentHide    AuditAction = "comment_hide"
	AuditCommentUnhide  AuditAction = "comment_unhide"
	AuditCommentDelete  AuditAction = "comment_delete"
	AuditPOICreate      AuditAction = "poi_create"
	AuditPOIUpdate      AuditAction = "poi_update"
	AuditPOIDelete      AuditAction = "poi_delete"
	AuditTextbookCreate AuditAction = "textbook_create"
	AuditTextbookUpdate AuditAction = "textbook_update"
	AuditTextbookDelete AuditAction = "textbook_delete"
)

// AuditLog - запись журнала действий модераторов/админов. ActorNickname
// собирается JOIN'ом в репозитории (не хранится в таблице audit_logs).
type AuditLog struct {
	ID            uuid.UUID
	ActorID       uuid.UUID
	ActorNickname string
	Action        AuditAction
	TargetType    *string
	TargetID      *uuid.UUID
	Reason        *string
	Metadata      json.RawMessage
	IPAddress     *string
	CreatedAt     time.Time
}

type AuditLogListFilter struct {
	ActorID     *uuid.UUID
	Action      *AuditAction
	TargetType  *string
	From, To    *time.Time
	Page, Limit int
}

type AdminUserListFilter struct {
	Role        *UserRole
	Banned      *bool
	University  *University
	Q           *string
	Page, Limit int
}

type AdminStats struct {
	UsersTotal       int
	UsersBanned      int
	ThreadsTotal     int
	CardTasksTotal   int
	CardTasksPending int
	ActiveSessions   int
}
