package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
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

// TestReactionRepo_EmojiAndVote_CoexistOnSameComment - регрессионный: до
// миграции 000016_reactions_kind у пользователя могла быть только одна
// реакция (любого рода) на цель. После миграции лайк (kind='emoji') и голос
// (kind='vote') должны сосуществовать независимо на одном и том же
// комментарии, не затирая друг друга, и не задваивая likes_count.
func TestReactionRepo_EmojiAndVote_CoexistOnSameComment(t *testing.T) {
	pool := setupTestDB(t)
	threadRepo := NewThreadRepo(pool)
	commentRepo := NewCommentRepo(pool)
	reactionRepo := NewReactionRepo(pool)
	ctx := context.Background()
	author := createTestForumUser(t, pool)
	thread := createTestThread(t, pool, threadRepo, author.ID, "thread", nil)
	comment, err := commentRepo.Create(ctx, thread.ID, author.ID, nil, 0, "votable comment")
	if err != nil {
		t.Fatalf("Create() comment error = %v", err)
	}
	cleanupComment(t, pool, comment.ID)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM reactions WHERE target_id = $1", comment.ID)
	})

	if _, err := reactionRepo.Upsert(ctx, author.ID, models.ReactionTargetComment, comment.ID, "🔥"); err != nil {
		t.Fatalf("Upsert() emoji error = %v", err)
	}
	if _, err := reactionRepo.UpsertVote(ctx, author.ID, models.ReactionTargetComment, comment.ID, "up"); err != nil {
		t.Fatalf("UpsertVote() error = %v", err)
	}

	found, err := commentRepo.FindByID(ctx, comment.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if found.LikesCount != 1 {
		t.Errorf("LikesCount = %d, want 1 (голос не должен трогать счётчик лайков)", found.LikesCount)
	}

	summaries, err := reactionRepo.VoteSummaries(ctx, models.ReactionTargetComment, []uuid.UUID{comment.ID}, author.ID)
	if err != nil {
		t.Fatalf("VoteSummaries() error = %v", err)
	}
	summary := summaries[comment.ID]
	if summary.Score != 1 || summary.MyVote == nil || *summary.MyVote != "up" {
		t.Errorf("VoteSummaries() = %+v, want Score=1 MyVote=up", summary)
	}

	// Убираем голос - лайк должен остаться нетронутым.
	if err := reactionRepo.DeleteVote(ctx, author.ID, models.ReactionTargetComment, comment.ID); err != nil {
		t.Fatalf("DeleteVote() error = %v", err)
	}
	found, err = commentRepo.FindByID(ctx, comment.ID)
	if err != nil {
		t.Fatalf("FindByID() after DeleteVote error = %v", err)
	}
	if found.LikesCount != 1 {
		t.Errorf("LikesCount = %d after DeleteVote, want 1 (лайк не должен пострадать)", found.LikesCount)
	}
}

func TestReactionRepo_UpsertVote_ChangeDirection_DoesNotDuplicate(t *testing.T) {
	pool := setupTestDB(t)
	threadRepo := NewThreadRepo(pool)
	commentRepo := NewCommentRepo(pool)
	reactionRepo := NewReactionRepo(pool)
	ctx := context.Background()
	author := createTestForumUser(t, pool)
	thread := createTestThread(t, pool, threadRepo, author.ID, "thread", nil)
	comment, err := commentRepo.Create(ctx, thread.ID, author.ID, nil, 0, "votable comment")
	if err != nil {
		t.Fatalf("Create() comment error = %v", err)
	}
	cleanupComment(t, pool, comment.ID)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM reactions WHERE target_id = $1", comment.ID)
	})

	if _, err := reactionRepo.UpsertVote(ctx, author.ID, models.ReactionTargetComment, comment.ID, "up"); err != nil {
		t.Fatalf("UpsertVote() [up] error = %v", err)
	}
	if _, err := reactionRepo.UpsertVote(ctx, author.ID, models.ReactionTargetComment, comment.ID, "down"); err != nil {
		t.Fatalf("UpsertVote() [down] error = %v", err)
	}

	summaries, err := reactionRepo.VoteSummaries(ctx, models.ReactionTargetComment, []uuid.UUID{comment.ID}, author.ID)
	if err != nil {
		t.Fatalf("VoteSummaries() error = %v", err)
	}
	summary := summaries[comment.ID]
	if summary.Score != -1 || summary.MyVote == nil || *summary.MyVote != "down" {
		t.Errorf("VoteSummaries() = %+v, want Score=-1 MyVote=down (флип голоса, не второй голос)", summary)
	}
}

func TestReactionRepo_VoteSummaries_MultipleVoters(t *testing.T) {
	pool := setupTestDB(t)
	threadRepo := NewThreadRepo(pool)
	commentRepo := NewCommentRepo(pool)
	reactionRepo := NewReactionRepo(pool)
	ctx := context.Background()
	author := createTestForumUser(t, pool)
	voter1 := createTestForumUser(t, pool)
	voter2 := createTestForumUser(t, pool)
	thread := createTestThread(t, pool, threadRepo, author.ID, "thread", nil)
	comment, err := commentRepo.Create(ctx, thread.ID, author.ID, nil, 0, "votable comment")
	if err != nil {
		t.Fatalf("Create() comment error = %v", err)
	}
	cleanupComment(t, pool, comment.ID)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM reactions WHERE target_id = $1", comment.ID)
	})

	if _, err := reactionRepo.UpsertVote(ctx, voter1.ID, models.ReactionTargetComment, comment.ID, "up"); err != nil {
		t.Fatalf("UpsertVote() [voter1] error = %v", err)
	}
	if _, err := reactionRepo.UpsertVote(ctx, voter2.ID, models.ReactionTargetComment, comment.ID, "up"); err != nil {
		t.Fatalf("UpsertVote() [voter2] error = %v", err)
	}

	summaries, err := reactionRepo.VoteSummaries(ctx, models.ReactionTargetComment, []uuid.UUID{comment.ID}, voter1.ID)
	if err != nil {
		t.Fatalf("VoteSummaries() error = %v", err)
	}
	summary := summaries[comment.ID]
	if summary.Score != 2 {
		t.Errorf("Score = %d, want 2 (два голоса up)", summary.Score)
	}
	if summary.MyVote == nil || *summary.MyVote != "up" {
		t.Errorf("MyVote = %v, want up (для viewerID=voter1)", summary.MyVote)
	}
}

func TestReactionRepo_DeleteVote_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	threadRepo := NewThreadRepo(pool)
	commentRepo := NewCommentRepo(pool)
	reactionRepo := NewReactionRepo(pool)
	ctx := context.Background()
	author := createTestForumUser(t, pool)
	thread := createTestThread(t, pool, threadRepo, author.ID, "thread", nil)
	comment, err := commentRepo.Create(ctx, thread.ID, author.ID, nil, 0, "no vote here")
	if err != nil {
		t.Fatalf("Create() comment error = %v", err)
	}
	cleanupComment(t, pool, comment.ID)

	err = reactionRepo.DeleteVote(ctx, author.ID, models.ReactionTargetComment, comment.ID)
	if err != models.ErrReactionNotFound {
		t.Fatalf("DeleteVote() error = %v, want ErrReactionNotFound", err)
	}
}
