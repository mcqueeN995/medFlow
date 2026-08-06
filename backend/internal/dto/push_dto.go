package dto

import "github.com/medflow/backend/internal/models"

type SubscribeRequest struct {
	Endpoint string `json:"endpoint" binding:"required,uri"`
	P256dh   string `json:"p256dh" binding:"required"`
	Auth     string `json:"auth" binding:"required"`
}

// PushPreferences - зеркалит схему PushPreferences в openapi.yaml. Указатели,
// а не bool - PATCH позволяет прислать только часть полей (остальные не
// трогаются), у самого ответа поля всегда заполнены.
type PushPreferences struct {
	ThreadReply      *bool `json:"thread_reply,omitempty"`
	CommentReply     *bool `json:"comment_reply,omitempty"`
	Reaction         *bool `json:"reaction,omitempty"`
	CardTaskDone     *bool `json:"card_task_done,omitempty"`
	CardTaskFailed   *bool `json:"card_task_failed,omitempty"`
	ModerationAction *bool `json:"moderation_action,omitempty"`
	System           *bool `json:"system,omitempty"`
}

func ToPushPreferences(p *models.PushPreferences) PushPreferences {
	return PushPreferences{
		ThreadReply: &p.ThreadReply, CommentReply: &p.CommentReply, Reaction: &p.Reaction,
		CardTaskDone: &p.CardTaskDone, CardTaskFailed: &p.CardTaskFailed,
		ModerationAction: &p.ModerationAction, System: &p.System,
	}
}

// ApplyTo - накладывает присланные (не-nil) поля на существующую модель, не
// трогая остальные - тот же паттерн частичного PATCH, что в UpdatePOIRequest.
func (p PushPreferences) ApplyTo(existing models.PushPreferences) models.PushPreferences {
	if p.ThreadReply != nil {
		existing.ThreadReply = *p.ThreadReply
	}
	if p.CommentReply != nil {
		existing.CommentReply = *p.CommentReply
	}
	if p.Reaction != nil {
		existing.Reaction = *p.Reaction
	}
	if p.CardTaskDone != nil {
		existing.CardTaskDone = *p.CardTaskDone
	}
	if p.CardTaskFailed != nil {
		existing.CardTaskFailed = *p.CardTaskFailed
	}
	if p.ModerationAction != nil {
		existing.ModerationAction = *p.ModerationAction
	}
	if p.System != nil {
		existing.System = *p.System
	}
	return existing
}
