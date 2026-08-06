package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/medflow/backend/internal/models"
)

func TestAuditLogRepo_Create_And_List(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewAuditLogRepo(pool)
	ctx := context.Background()
	actor := createTestForumUser(t, pool)

	targetID := uuid.New()
	targetType := "user"
	reason := "нарушение правил"
	entry := &models.AuditLog{ActorID: actor.ID, Action: models.AuditUserBan, TargetType: &targetType, TargetID: &targetID, Reason: &reason}
	if err := repo.Create(ctx, entry); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), "DELETE FROM audit_logs WHERE id = $1", entry.ID) })

	if entry.ID == uuid.Nil {
		t.Fatal("Create() did not set ID")
	}

	action := models.AuditUserBan
	items, total, err := repo.List(ctx, models.AuditLogListFilter{ActorID: &actor.ID, Action: &action, Page: 1, Limit: 20})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if total != 1 || len(items) != 1 {
		t.Fatalf("List() = %v (total=%d), want 1", items, total)
	}
	if items[0].ActorNickname != actor.Nickname {
		t.Errorf("ActorNickname = %q, want %q", items[0].ActorNickname, actor.Nickname)
	}
	if items[0].Reason == nil || *items[0].Reason != reason {
		t.Errorf("Reason = %v, want %q", items[0].Reason, reason)
	}
}

func TestAuditLogRepo_List_FiltersByTargetType(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewAuditLogRepo(pool)
	ctx := context.Background()
	actor := createTestForumUser(t, pool)

	userTarget := "user"
	threadTarget := "thread"
	id1 := uuid.New()
	id2 := uuid.New()
	e1 := &models.AuditLog{ActorID: actor.ID, Action: models.AuditUserBan, TargetType: &userTarget, TargetID: &id1}
	e2 := &models.AuditLog{ActorID: actor.ID, Action: models.AuditThreadHide, TargetType: &threadTarget, TargetID: &id2}
	for _, e := range []*models.AuditLog{e1, e2} {
		if err := repo.Create(ctx, e); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		eid := e.ID
		t.Cleanup(func() { _, _ = pool.Exec(context.Background(), "DELETE FROM audit_logs WHERE id = $1", eid) })
	}

	target := "thread"
	items, total, err := repo.List(ctx, models.AuditLogListFilter{TargetType: &target, Page: 1, Limit: 20})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if total != 1 || len(items) != 1 || items[0].ID != e2.ID {
		t.Fatalf("List(target_type=thread) = %v (total=%d), want only %v", items, total, e2.ID)
	}
}
