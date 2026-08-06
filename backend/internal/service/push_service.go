package service

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/medflow/backend/internal/config"
	"github.com/medflow/backend/internal/dto"
	"github.com/medflow/backend/internal/models"
)

var ErrPushSubscriptionNotFound = errors.New("push subscription not found")

type PushService struct {
	pushRepo PushRepository
	sender   PushSender
	vapid    config.VAPIDConfig
}

func NewPushService(pushRepo PushRepository, sender PushSender, vapid config.VAPIDConfig) *PushService {
	return &PushService{pushRepo: pushRepo, sender: sender, vapid: vapid}
}

func (s *PushService) Subscribe(ctx context.Context, userID uuid.UUID, req dto.SubscribeRequest) (uuid.UUID, error) {
	sub, err := s.pushRepo.CreateSubscription(ctx, userID, req.Endpoint, req.P256dh, req.Auth)
	if err != nil {
		return uuid.Nil, err
	}
	return sub.ID, nil
}

func (s *PushService) Unsubscribe(ctx context.Context, userID uuid.UUID, endpoint string) error {
	err := s.pushRepo.DeleteSubscriptionByEndpoint(ctx, userID, endpoint)
	if errors.Is(err, models.ErrPushSubscriptionNotFound) {
		return ErrPushSubscriptionNotFound
	}
	return err
}

func (s *PushService) GetPreferences(ctx context.Context, userID uuid.UUID) (dto.PushPreferences, error) {
	prefs, err := s.pushRepo.GetPreferences(ctx, userID)
	if err != nil {
		return dto.PushPreferences{}, err
	}
	return dto.ToPushPreferences(prefs), nil
}

func (s *PushService) UpdatePreferences(ctx context.Context, userID uuid.UUID, req dto.PushPreferences) (dto.PushPreferences, error) {
	existing, err := s.pushRepo.GetPreferences(ctx, userID)
	if err != nil {
		return dto.PushPreferences{}, err
	}
	merged := req.ApplyTo(*existing)
	merged.UserID = userID
	updated, err := s.pushRepo.UpsertPreferences(ctx, merged)
	if err != nil {
		return dto.PushPreferences{}, err
	}
	return dto.ToPushPreferences(updated), nil
}

type pushPayload struct {
	Title   string `json:"title"`
	Message string `json:"message"`
	Kind    string `json:"kind"`
}

// Notify - лучшее старание: если preferences отключают этот вид уведомлений
// - тихий no-op; сбой отправки одной из подписок не прерывает остальные и не
// возвращается вызывающему коду как ошибка (тот же паттерн, что writeAudit) -
// push-уведомление никогда не должно ронять основное действие (создание
// комментария, завершение генерации карточек).
func (s *PushService) Notify(ctx context.Context, userID uuid.UUID, kind models.NotificationKind, title, message string) error {
	prefs, err := s.pushRepo.GetPreferences(ctx, userID)
	if err != nil {
		return err
	}
	if !prefs.Enabled(kind) {
		return nil
	}

	subs, err := s.pushRepo.ListSubscriptionsForUser(ctx, userID)
	if err != nil {
		return err
	}
	if len(subs) == 0 {
		return nil
	}

	payload, err := json.Marshal(pushPayload{Title: title, Message: message, Kind: string(kind)})
	if err != nil {
		return err
	}

	for _, sub := range subs {
		if sendErr := s.sender.Send(ctx, sub, s.vapid, payload); sendErr != nil {
			if errors.Is(sendErr, ErrPushGone) {
				_ = s.pushRepo.DeleteSubscriptionByRawEndpoint(ctx, sub.Endpoint)
			}
			// остальные подписки этого пользователя всё равно стоит попробовать
			continue
		}
	}
	return nil
}
