package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/medflow/backend/internal/config"
	"github.com/medflow/backend/internal/dto"
	"github.com/medflow/backend/internal/models"
)

func TestPushService_Notify_SkipsWhenPreferenceDisabled(t *testing.T) {
	userID := uuid.New()
	repo := &mockPushRepository{
		getPreferencesFn: func(ctx context.Context, id uuid.UUID) (*models.PushPreferences, error) {
			prefs := models.DefaultPushPreferences(id)
			prefs.ThreadReply = false
			return &prefs, nil
		},
		listSubscriptionsForUserFn: func(ctx context.Context, id uuid.UUID) ([]models.PushSubscription, error) {
			t.Fatal("ListSubscriptionsForUser must not be called when preference is disabled")
			return nil, nil
		},
	}
	sender := &mockPushSender{}
	svc := NewPushService(repo, sender, config.VAPIDConfig{})

	if err := svc.Notify(context.Background(), userID, models.NotificationThreadReply, "t", "m"); err != nil {
		t.Fatalf("Notify() error = %v", err)
	}
	if len(sender.sent) != 0 {
		t.Errorf("sender.sent = %d sends, want 0", len(sender.sent))
	}
}

func TestPushService_Notify_SendsToAllSubscriptions(t *testing.T) {
	userID := uuid.New()
	subs := []models.PushSubscription{
		{ID: uuid.New(), UserID: userID, Endpoint: "https://push.example/a"},
		{ID: uuid.New(), UserID: userID, Endpoint: "https://push.example/b"},
	}
	repo := &mockPushRepository{
		listSubscriptionsForUserFn: func(ctx context.Context, id uuid.UUID) ([]models.PushSubscription, error) {
			return subs, nil
		},
	}
	sender := &mockPushSender{}
	svc := NewPushService(repo, sender, config.VAPIDConfig{})

	if err := svc.Notify(context.Background(), userID, models.NotificationCardTaskDone, "t", "m"); err != nil {
		t.Fatalf("Notify() error = %v", err)
	}
	if len(sender.sent) != 2 {
		t.Fatalf("sender.sent = %d sends, want 2", len(sender.sent))
	}
}

func TestPushService_Notify_PrunesGoneSubscription(t *testing.T) {
	userID := uuid.New()
	goneSub := models.PushSubscription{ID: uuid.New(), UserID: userID, Endpoint: "https://push.example/gone"}
	var deletedEndpoint string
	repo := &mockPushRepository{
		listSubscriptionsForUserFn: func(ctx context.Context, id uuid.UUID) ([]models.PushSubscription, error) {
			return []models.PushSubscription{goneSub}, nil
		},
		deleteSubscriptionByRawEndpointFn: func(ctx context.Context, endpoint string) error {
			deletedEndpoint = endpoint
			return nil
		},
	}
	sender := &mockPushSender{
		sendFn: func(ctx context.Context, sub models.PushSubscription, vapid config.VAPIDConfig, payload []byte) error {
			return ErrPushGone
		},
	}
	svc := NewPushService(repo, sender, config.VAPIDConfig{})

	if err := svc.Notify(context.Background(), userID, models.NotificationCardTaskDone, "t", "m"); err != nil {
		t.Fatalf("Notify() error = %v", err)
	}
	if deletedEndpoint != goneSub.Endpoint {
		t.Errorf("deletedEndpoint = %q, want %q", deletedEndpoint, goneSub.Endpoint)
	}
}

func TestPushService_UpdatePreferences_MergesPartialUpdate(t *testing.T) {
	userID := uuid.New()
	existing := models.DefaultPushPreferences(userID)
	repo := &mockPushRepository{
		getPreferencesFn: func(ctx context.Context, id uuid.UUID) (*models.PushPreferences, error) {
			return &existing, nil
		},
		upsertPreferencesFn: func(ctx context.Context, p models.PushPreferences) (*models.PushPreferences, error) {
			return &p, nil
		},
	}
	svc := NewPushService(repo, &mockPushSender{}, config.VAPIDConfig{})

	disabled := false
	got, err := svc.UpdatePreferences(context.Background(), userID, dto.PushPreferences{ThreadReply: &disabled})
	if err != nil {
		t.Fatalf("UpdatePreferences() error = %v", err)
	}
	if got.ThreadReply == nil || *got.ThreadReply != false {
		t.Errorf("ThreadReply = %v, want false", got.ThreadReply)
	}
	if got.Reaction == nil || *got.Reaction != true {
		t.Errorf("Reaction = %v, want unchanged true", got.Reaction)
	}
}

func TestPushService_Unsubscribe_MapsNotFound(t *testing.T) {
	repo := &mockPushRepository{
		deleteSubscriptionByEndpointFn: func(ctx context.Context, userID uuid.UUID, endpoint string) error {
			return models.ErrPushSubscriptionNotFound
		},
	}
	svc := NewPushService(repo, &mockPushSender{}, config.VAPIDConfig{})

	err := svc.Unsubscribe(context.Background(), uuid.New(), "https://push.example/x")
	if !errors.Is(err, ErrPushSubscriptionNotFound) {
		t.Fatalf("Unsubscribe() error = %v, want ErrPushSubscriptionNotFound", err)
	}
}
