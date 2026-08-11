package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/medflow/backend/internal/models"
)

// cleanupComment регистрирует удаление комментария до удаления его треда -
// иначе FK fk_comments_thread ломает cleanup самого треда (t.Cleanup выполняется
// LIFO, поэтому регистрация "после" createTestThread гарантированно
// отработает раньше её cleanup'а).
func cleanupComment(t *testing.T, pool *pgxpool.Pool, id uuid.UUID) {
	t.Helper()
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM comments WHERE id = $1", id)
	})
}

func TestCommentRepo_Create_IncrementsThreadCommentsCount(t *testing.T) {
	pool := setupTestDB(t)
	threadRepo := NewThreadRepo(pool)
	commentRepo := NewCommentRepo(pool)
	ctx := context.Background()
	author := createTestForumUser(t, pool)
	thread := createTestThread(t, pool, threadRepo, author.ID, "thread with comments", nil)

	comment, err := commentRepo.Create(ctx, thread.ID, author.ID, nil, 0, "первый комментарий")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	cleanupComment(t, pool, comment.ID)

	if comment.Depth != 0 || comment.ParentID != nil {
		t.Errorf("Create() = %+v, want top-level comment", comment)
	}
	if comment.Author.ID != author.ID {
		t.Errorf("Author.ID = %v, want %v", comment.Author.ID, author.ID)
	}

	found, err := threadRepo.FindByID(ctx, thread.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if found.CommentsCount != 1 {
		t.Errorf("thread.CommentsCount = %d, want 1", found.CommentsCount)
	}
}

func TestCommentRepo_SoftDelete_DecrementsThreadCommentsCount(t *testing.T) {
	pool := setupTestDB(t)
	threadRepo := NewThreadRepo(pool)
	commentRepo := NewCommentRepo(pool)
	ctx := context.Background()
	author := createTestForumUser(t, pool)
	thread := createTestThread(t, pool, threadRepo, author.ID, "thread", nil)

	comment, err := commentRepo.Create(ctx, thread.ID, author.ID, nil, 0, "будет удалён")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	cleanupComment(t, pool, comment.ID)

	if err := commentRepo.SoftDelete(ctx, comment.ID, thread.ID); err != nil {
		t.Fatalf("SoftDelete() error = %v", err)
	}

	if _, err := commentRepo.FindByID(ctx, comment.ID); err != models.ErrCommentNotFound {
		t.Fatalf("FindByID() after delete error = %v, want ErrCommentNotFound", err)
	}

	found, err := threadRepo.FindByID(ctx, thread.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if found.CommentsCount != 0 {
		t.Errorf("thread.CommentsCount = %d, want 0", found.CommentsCount)
	}
}

func TestCommentRepo_Update(t *testing.T) {
	pool := setupTestDB(t)
	threadRepo := NewThreadRepo(pool)
	commentRepo := NewCommentRepo(pool)
	ctx := context.Background()
	author := createTestForumUser(t, pool)
	thread := createTestThread(t, pool, threadRepo, author.ID, "thread", nil)
	comment, err := commentRepo.Create(ctx, thread.ID, author.ID, nil, 0, "old content")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	cleanupComment(t, pool, comment.ID)

	updated, err := commentRepo.Update(ctx, comment.ID, "new content")
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Content != "new content" {
		t.Errorf("Content = %q, want %q", updated.Content, "new content")
	}
}

func TestCommentRepo_ListByThread_TopLevelWithReplies(t *testing.T) {
	pool := setupTestDB(t)
	threadRepo := NewThreadRepo(pool)
	commentRepo := NewCommentRepo(pool)
	ctx := context.Background()
	author := createTestForumUser(t, pool)
	thread := createTestThread(t, pool, threadRepo, author.ID, "thread", nil)

	top1, err := commentRepo.Create(ctx, thread.ID, author.ID, nil, 0, "top 1")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	cleanupComment(t, pool, top1.ID)

	top2, err := commentRepo.Create(ctx, thread.ID, author.ID, nil, 0, "top 2")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	cleanupComment(t, pool, top2.ID)

	reply, err := commentRepo.Create(ctx, thread.ID, author.ID, &top1.ID, 1, "reply to top 1")
	if err != nil {
		t.Fatalf("Create() reply error = %v", err)
	}
	cleanupComment(t, pool, reply.ID)

	comments, total, err := commentRepo.ListByThread(ctx, thread.ID, 1, 50, "new")
	if err != nil {
		t.Fatalf("ListByThread() error = %v", err)
	}
	if total != 2 {
		t.Fatalf("total = %d, want 2 top-level comments", total)
	}
	if len(comments) != 2 {
		t.Fatalf("len(comments) = %d, want 2", len(comments))
	}

	byID := map[uuid.UUID][]models.Comment{}
	for _, c := range comments {
		byID[c.ID] = c.Replies
	}
	if len(byID[top1.ID]) != 1 {
		t.Errorf("top1 replies = %d, want 1", len(byID[top1.ID]))
	}
	if len(byID[top2.ID]) != 0 {
		t.Errorf("top2 replies = %d, want 0", len(byID[top2.ID]))
	}
}

func TestCommentRepo_ListByThread_SortBest_OrdersByVoteScore(t *testing.T) {
	pool := setupTestDB(t)
	threadRepo := NewThreadRepo(pool)
	commentRepo := NewCommentRepo(pool)
	reactionRepo := NewReactionRepo(pool)
	ctx := context.Background()
	author := createTestForumUser(t, pool)
	voter1 := createTestForumUser(t, pool)
	voter2 := createTestForumUser(t, pool)
	thread := createTestThread(t, pool, threadRepo, author.ID, "thread", nil)

	low, err := commentRepo.Create(ctx, thread.ID, author.ID, nil, 0, "low score, posted first")
	if err != nil {
		t.Fatalf("Create() [low] error = %v", err)
	}
	cleanupComment(t, pool, low.ID)

	high, err := commentRepo.Create(ctx, thread.ID, author.ID, nil, 0, "high score, posted second")
	if err != nil {
		t.Fatalf("Create() [high] error = %v", err)
	}
	cleanupComment(t, pool, high.ID)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM reactions WHERE target_id IN ($1, $2)", low.ID, high.ID)
	})

	if _, err := reactionRepo.UpsertVote(ctx, voter1.ID, models.ReactionTargetComment, high.ID, "up"); err != nil {
		t.Fatalf("UpsertVote() error = %v", err)
	}
	if _, err := reactionRepo.UpsertVote(ctx, voter2.ID, models.ReactionTargetComment, high.ID, "up"); err != nil {
		t.Fatalf("UpsertVote() error = %v", err)
	}
	if _, err := reactionRepo.UpsertVote(ctx, voter1.ID, models.ReactionTargetComment, low.ID, "down"); err != nil {
		t.Fatalf("UpsertVote() error = %v", err)
	}

	comments, _, err := commentRepo.ListByThread(ctx, thread.ID, 1, 50, "best")
	if err != nil {
		t.Fatalf("ListByThread() error = %v", err)
	}
	if len(comments) != 2 {
		t.Fatalf("len(comments) = %d, want 2", len(comments))
	}
	// "high" опубликован позже "low", но с сортировкой sort=best должен идти
	// первым - score важнее created_at.
	if comments[0].ID != high.ID {
		t.Errorf("comments[0].ID = %v, want high.ID=%v (сортировка по score, не по времени)", comments[0].ID, high.ID)
	}
	if comments[1].ID != low.ID {
		t.Errorf("comments[1].ID = %v, want low.ID=%v", comments[1].ID, low.ID)
	}
}

func TestCommentRepo_Hide(t *testing.T) {
	pool := setupTestDB(t)
	threadRepo := NewThreadRepo(pool)
	commentRepo := NewCommentRepo(pool)
	ctx := context.Background()
	author := createTestForumUser(t, pool)
	moderator := createTestForumUser(t, pool)
	thread := createTestThread(t, pool, threadRepo, author.ID, "thread with a bad comment", nil)
	comment, err := commentRepo.Create(ctx, thread.ID, author.ID, nil, 0, "spam comment")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	cleanupComment(t, pool, comment.ID)

	hidden, err := commentRepo.Hide(ctx, comment.ID, moderator.ID, "спам")
	if err != nil {
		t.Fatalf("Hide() error = %v", err)
	}
	if hidden.HiddenAt == nil {
		t.Fatal("HiddenAt = nil, want set")
	}
	if hidden.HiddenBy == nil || *hidden.HiddenBy != moderator.ID {
		t.Errorf("HiddenBy = %v, want %v", hidden.HiddenBy, moderator.ID)
	}
	if hidden.HiddenReason == nil || *hidden.HiddenReason != "спам" {
		t.Errorf("HiddenReason = %v, want спам", hidden.HiddenReason)
	}
}

func TestCommentRepo_Hide_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewCommentRepo(pool)

	_, err := repo.Hide(context.Background(), uuid.New(), uuid.New(), "reason")
	if err != models.ErrCommentNotFound {
		t.Fatalf("Hide() error = %v, want ErrCommentNotFound", err)
	}
}
