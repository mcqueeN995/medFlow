package models

import (
	"time"

	"github.com/google/uuid"
)

// NotificationKind - виды push-уведомлений, зеркалит enum NotificationType
// в openapi.yaml. Используется и как ключ PushPreferences, и как тип
// уведомления, передаваемый в PushService.Notify.
type NotificationKind string

const (
	NotificationThreadReply      NotificationKind = "thread_reply"
	NotificationCommentReply     NotificationKind = "comment_reply"
	NotificationReaction         NotificationKind = "reaction"
	NotificationCardTaskDone     NotificationKind = "card_task_done"
	NotificationCardTaskFailed   NotificationKind = "card_task_failed"
	NotificationModerationAction NotificationKind = "moderation_action"
	NotificationSystem           NotificationKind = "system"
)

type PushSubscription struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Endpoint  string
	P256dh    string
	Auth      string
	CreatedAt time.Time
}

// PushPreferences - per-user переключатели по каждому NotificationKind.
// Строка создаётся лениво с дефолтами true при первом Subscribe/GetPreferences.
type PushPreferences struct {
	UserID           uuid.UUID
	ThreadReply      bool
	CommentReply     bool
	Reaction         bool
	CardTaskDone     bool
	CardTaskFailed   bool
	ModerationAction bool
	System           bool
	UpdatedAt        time.Time
}

// Enabled - проверяет флаг preferences для конкретного вида уведомления.
func (p PushPreferences) Enabled(kind NotificationKind) bool {
	switch kind {
	case NotificationThreadReply:
		return p.ThreadReply
	case NotificationCommentReply:
		return p.CommentReply
	case NotificationReaction:
		return p.Reaction
	case NotificationCardTaskDone:
		return p.CardTaskDone
	case NotificationCardTaskFailed:
		return p.CardTaskFailed
	case NotificationModerationAction:
		return p.ModerationAction
	case NotificationSystem:
		return p.System
	default:
		return false
	}
}

func DefaultPushPreferences(userID uuid.UUID) PushPreferences {
	return PushPreferences{
		UserID:           userID,
		ThreadReply:      true,
		CommentReply:     true,
		Reaction:         true,
		CardTaskDone:     true,
		CardTaskFailed:   true,
		ModerationAction: true,
		System:           true,
	}
}
