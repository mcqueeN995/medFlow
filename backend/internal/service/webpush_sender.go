package service

import (
	"context"
	"fmt"
	"net/http"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/medflow/backend/internal/config"
	"github.com/medflow/backend/internal/models"
)

// ErrPushGone - push-сервис (браузера) ответил 404/410: подписка протухла
// (пользователь отозвал разрешение, переустановил браузер и т.п.) и должна
// быть удалена. PushService.Notify реагирует на эту ошибку чисткой БД.
var ErrPushGone = fmt.Errorf("push subscription gone")

// WebPushSender - реализация PushSender поверх github.com/SherClockHolmes/webpush-go.
type WebPushSender struct{}

func NewWebPushSender() *WebPushSender {
	return &WebPushSender{}
}

func (s *WebPushSender) Send(ctx context.Context, sub models.PushSubscription, vapid config.VAPIDConfig, payload []byte) error {
	resp, err := webpush.SendNotificationWithContext(ctx, payload, &webpush.Subscription{
		Endpoint: sub.Endpoint,
		Keys:     webpush.Keys{P256dh: sub.P256dh, Auth: sub.Auth},
	}, &webpush.Options{
		Subscriber:      vapid.Subject,
		VAPIDPublicKey:  vapid.PublicKey,
		VAPIDPrivateKey: vapid.PrivateKey,
		TTL:             60,
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
		return ErrPushGone
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("push service responded with status %d", resp.StatusCode)
	}
	return nil
}
