package repository

import (
	"context"
	"testing"

	"github.com/medflow/backend/internal/models"
)

func TestReactionRepo_Upsert_NewReaction_IncrementsLikesCount(t *testing.T) {
	pool := setupTestDB(t)
	threadRepo := NewThreadRepo(pool)
	reactionRepo := NewReactionRepo(pool)
	ctx := context.Background()
	author := createTestForumUser(t, pool)
	thread := createTestThread(t, pool, threadRepo, author.ID, "reactable thread", nil)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM reactions WHERE target_id = $1", thread.ID)
	})

	reaction, err := reactionRepo.Upsert(ctx, author.ID, models.ReactionTargetThread, thread.ID, "🔥")
	if err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
	if reaction.Emoji != "🔥" {
		t.Errorf("Emoji = %q, want 🔥", reaction.Emoji)
	}

	found, err := threadRepo.FindByID(ctx, thread.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if found.LikesCount != 1 {
		t.Errorf("LikesCount = %d, want 1", found.LikesCount)
	}
}

func TestReactionRepo_Upsert_SameUserChangesEmoji_DoesNotDoubleLikesCount(t *testing.T) {
	pool := setupTestDB(t)
	threadRepo := NewThreadRepo(pool)
	reactionRepo := NewReactionRepo(pool)
	ctx := context.Background()
	author := createTestForumUser(t, pool)
	thread := createTestThread(t, pool, threadRepo, author.ID, "reactable thread", nil)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM reactions WHERE target_id = $1", thread.ID)
	})

	if _, err := reactionRepo.Upsert(ctx, author.ID, models.ReactionTargetThread, thread.ID, "🔥"); err != nil {
		t.Fatalf("first Upsert() error = %v", err)
	}
	reaction, err := reactionRepo.Upsert(ctx, author.ID, models.ReactionTargetThread, thread.ID, "❤️")
	if err != nil {
		t.Fatalf("second Upsert() error = %v", err)
	}
	if reaction.Emoji != "❤️" {
		t.Errorf("Emoji = %q, want ❤️ (changed)", reaction.Emoji)
	}

	found, err := threadRepo.FindByID(ctx, thread.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	// один пользователь = максимум одна реакция на цель (uq_reactions_user_target
	// не включает emoji), смена эмодзи не должна увеличивать likes_count повторно.
	if found.LikesCount != 1 {
		t.Errorf("LikesCount = %d, want 1 (emoji change is not a new like)", found.LikesCount)
	}
}

func TestReactionRepo_Delete_DecrementsLikesCount(t *testing.T) {
	pool := setupTestDB(t)
	threadRepo := NewThreadRepo(pool)
	reactionRepo := NewReactionRepo(pool)
	ctx := context.Background()
	author := createTestForumUser(t, pool)
	thread := createTestThread(t, pool, threadRepo, author.ID, "reactable thread", nil)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM reactions WHERE target_id = $1", thread.ID)
	})

	if _, err := reactionRepo.Upsert(ctx, author.ID, models.ReactionTargetThread, thread.ID, "🔥"); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}

	if err := reactionRepo.Delete(ctx, author.ID, models.ReactionTargetThread, thread.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	found, err := threadRepo.FindByID(ctx, thread.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if found.LikesCount != 0 {
		t.Errorf("LikesCount = %d, want 0", found.LikesCount)
	}
}

func TestReactionRepo_Delete_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	threadRepo := NewThreadRepo(pool)
	reactionRepo := NewReactionRepo(pool)
	ctx := context.Background()
	author := createTestForumUser(t, pool)
	thread := createTestThread(t, pool, threadRepo, author.ID, "no reaction here", nil)

	err := reactionRepo.Delete(ctx, author.ID, models.ReactionTargetThread, thread.ID)
	if err != models.ErrReactionNotFound {
		t.Fatalf("Delete() error = %v, want ErrReactionNotFound", err)
	}
}
