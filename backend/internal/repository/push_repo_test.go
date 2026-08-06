package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/medflow/backend/internal/models"
)

func TestPushRepo_CreateSubscription_UpsertsByEndpoint(t *testing.T) {
	pool := setupTestDB(t)
	userRepo := NewUserRepo(pool)
	repo := NewPushRepo(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, ctx)
	defer func() { _, _ = pool.Exec(ctx, "DELETE FROM users WHERE id = $1", user.ID) }()

	endpoint := "https://push.example/" + uuid.New().String()
	created, err := repo.CreateSubscription(ctx, user.ID, endpoint, "p256dh-1", "auth-1")
	if err != nil {
		t.Fatalf("CreateSubscription() error = %v", err)
	}
	defer func() { _, _ = pool.Exec(ctx, "DELETE FROM push_subscriptions WHERE id = $1", created.ID) }()

	// повторная подписка того же endpoint должна обновить ключи, а не конфликтовать
	updated, err := repo.CreateSubscription(ctx, user.ID, endpoint, "p256dh-2", "auth-2")
	if err != nil {
		t.Fatalf("CreateSubscription() (upsert) error = %v", err)
	}
	if updated.P256dh != "p256dh-2" || updated.Auth != "auth-2" {
		t.Errorf("upsert did not update keys: %+v", updated)
	}

	subs, err := repo.ListSubscriptionsForUser(ctx, user.ID)
	if err != nil {
		t.Fatalf("ListSubscriptionsForUser() error = %v", err)
	}
	if len(subs) != 1 {
		t.Fatalf("len(subs) = %d, want 1 (upsert must not create a duplicate row)", len(subs))
	}
}

func TestPushRepo_DeleteSubscriptionByEndpoint_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewPushRepo(pool)

	err := repo.DeleteSubscriptionByEndpoint(context.Background(), uuid.New(), "https://push.example/missing")
	if err != models.ErrPushSubscriptionNotFound {
		t.Fatalf("DeleteSubscriptionByEndpoint() error = %v, want ErrPushSubscriptionNotFound", err)
	}
}

func TestPushRepo_GetPreferences_CreatesDefaultsLazily(t *testing.T) {
	pool := setupTestDB(t)
	userRepo := NewUserRepo(pool)
	repo := NewPushRepo(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, ctx)
	defer func() { _, _ = pool.Exec(ctx, "DELETE FROM users WHERE id = $1", user.ID) }()

	prefs, err := repo.GetPreferences(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetPreferences() error = %v", err)
	}
	if !prefs.ThreadReply || !prefs.System {
		t.Errorf("GetPreferences() = %+v, want all-true defaults", prefs)
	}

	// повторный вызов не должен упасть на конфликте уникальности user_id
	again, err := repo.GetPreferences(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetPreferences() (second call) error = %v", err)
	}
	if again.UserID != prefs.UserID {
		t.Errorf("GetPreferences() second call returned different row")
	}
}

func TestPushRepo_UpsertPreferences_PersistsChanges(t *testing.T) {
	pool := setupTestDB(t)
	userRepo := NewUserRepo(pool)
	repo := NewPushRepo(pool)
	ctx := context.Background()

	user := createTestUser(t, userRepo, ctx)
	defer func() { _, _ = pool.Exec(ctx, "DELETE FROM users WHERE id = $1", user.ID) }()

	prefs := models.DefaultPushPreferences(user.ID)
	prefs.ThreadReply = false
	prefs.Reaction = false

	updated, err := repo.UpsertPreferences(ctx, prefs)
	if err != nil {
		t.Fatalf("UpsertPreferences() error = %v", err)
	}
	if updated.ThreadReply || updated.Reaction {
		t.Errorf("UpsertPreferences() = %+v, want thread_reply/reaction false", updated)
	}
	if !updated.CommentReply {
		t.Errorf("UpsertPreferences() CommentReply = false, want true (untouched field)")
	}

	reread, err := repo.GetPreferences(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetPreferences() error = %v", err)
	}
	if reread.ThreadReply {
		t.Errorf("GetPreferences() after upsert = %+v, want thread_reply persisted as false", reread)
	}
}
