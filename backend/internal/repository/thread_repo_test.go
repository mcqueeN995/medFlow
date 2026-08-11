package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/medflow/backend/internal/models"
)

func createTestThread(t *testing.T, pool *pgxpool.Pool, repo *ThreadRepo, authorID uuid.UUID, title string, tags []models.ThreadTag) *models.Thread {
	t.Helper()
	thread, err := repo.Create(context.Background(), authorID, title, "content body", tags)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM threads WHERE id = $1", thread.ID)
	})
	return thread
}

func TestThreadRepo_Create_And_FindByID(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewThreadRepo(pool)
	ctx := context.Background()
	author := createTestForumUser(t, pool)

	created := createTestThread(t, pool, repo, author.ID, "Как сдать анатомию", []models.ThreadTag{models.TagStudy, models.TagHelp})

	found, err := repo.FindByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if found.Title != "Как сдать анатомию" {
		t.Errorf("Title = %q, want %q", found.Title, "Как сдать анатомию")
	}
	if len(found.Tags) != 2 {
		t.Fatalf("Tags = %v, want 2 tags", found.Tags)
	}
	if found.Author.ID != author.ID || found.Author.Nickname != author.Nickname {
		t.Errorf("Author = %+v, want ID=%v Nickname=%v", found.Author, author.ID, author.Nickname)
	}
	if found.Author.ThreadsCount < 1 {
		t.Errorf("Author.ThreadsCount = %d, want >= 1", found.Author.ThreadsCount)
	}
}

func TestThreadRepo_FindByID_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewThreadRepo(pool)

	_, err := repo.FindByID(context.Background(), uuid.New())
	if err != models.ErrThreadNotFound {
		t.Fatalf("FindByID() error = %v, want ErrThreadNotFound", err)
	}
}

func TestThreadRepo_IncrementViewsIfNotRecentlyViewed_FirstViewCounted(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewThreadRepo(pool)
	ctx := context.Background()
	author := createTestForumUser(t, pool)
	viewer := createTestForumUser(t, pool)
	thread := createTestThread(t, pool, repo, author.ID, "views test", nil)

	if err := repo.IncrementViewsIfNotRecentlyViewed(ctx, thread.ID, viewer.ID); err != nil {
		t.Fatalf("IncrementViewsIfNotRecentlyViewed() error = %v", err)
	}

	found, err := repo.FindByID(ctx, thread.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if found.ViewsCount != 1 {
		t.Errorf("ViewsCount = %d, want 1", found.ViewsCount)
	}
}

func TestThreadRepo_IncrementViewsIfNotRecentlyViewed_SameUserRepeatWithin24h_NotCountedTwice(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewThreadRepo(pool)
	ctx := context.Background()
	author := createTestForumUser(t, pool)
	viewer := createTestForumUser(t, pool)
	thread := createTestThread(t, pool, repo, author.ID, "views test", nil)

	if err := repo.IncrementViewsIfNotRecentlyViewed(ctx, thread.ID, viewer.ID); err != nil {
		t.Fatalf("IncrementViewsIfNotRecentlyViewed() [1st] error = %v", err)
	}
	if err := repo.IncrementViewsIfNotRecentlyViewed(ctx, thread.ID, viewer.ID); err != nil {
		t.Fatalf("IncrementViewsIfNotRecentlyViewed() [2nd] error = %v", err)
	}

	found, err := repo.FindByID(ctx, thread.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if found.ViewsCount != 1 {
		t.Errorf("ViewsCount = %d, want 1 (повторный просмотр в пределах 24ч не должен накручивать счётчик)", found.ViewsCount)
	}
}

func TestThreadRepo_IncrementViewsIfNotRecentlyViewed_DifferentUsers_BothCounted(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewThreadRepo(pool)
	ctx := context.Background()
	author := createTestForumUser(t, pool)
	viewer1 := createTestForumUser(t, pool)
	viewer2 := createTestForumUser(t, pool)
	thread := createTestThread(t, pool, repo, author.ID, "views test", nil)

	if err := repo.IncrementViewsIfNotRecentlyViewed(ctx, thread.ID, viewer1.ID); err != nil {
		t.Fatalf("IncrementViewsIfNotRecentlyViewed() [viewer1] error = %v", err)
	}
	if err := repo.IncrementViewsIfNotRecentlyViewed(ctx, thread.ID, viewer2.ID); err != nil {
		t.Fatalf("IncrementViewsIfNotRecentlyViewed() [viewer2] error = %v", err)
	}

	found, err := repo.FindByID(ctx, thread.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if found.ViewsCount != 2 {
		t.Errorf("ViewsCount = %d, want 2 (разные пользователи считаются независимо)", found.ViewsCount)
	}
}

func TestThreadRepo_IncrementViewsIfNotRecentlyViewed_After24h_CountedAgain(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewThreadRepo(pool)
	ctx := context.Background()
	author := createTestForumUser(t, pool)
	viewer := createTestForumUser(t, pool)
	thread := createTestThread(t, pool, repo, author.ID, "views test", nil)

	if err := repo.IncrementViewsIfNotRecentlyViewed(ctx, thread.ID, viewer.ID); err != nil {
		t.Fatalf("IncrementViewsIfNotRecentlyViewed() [1st] error = %v", err)
	}

	// Имитируем "просмотр 25 часов назад" напрямую в БД - без этого пришлось
	// бы реально ждать 24 часа, чтобы проверить дедуп-окно.
	if _, err := pool.Exec(ctx, `UPDATE thread_views SET last_viewed_at = now() - interval '25 hours' WHERE user_id = $1 AND thread_id = $2`, viewer.ID, thread.ID); err != nil {
		t.Fatalf("backdate thread_views error = %v", err)
	}

	if err := repo.IncrementViewsIfNotRecentlyViewed(ctx, thread.ID, viewer.ID); err != nil {
		t.Fatalf("IncrementViewsIfNotRecentlyViewed() [2nd, after 24h] error = %v", err)
	}

	found, err := repo.FindByID(ctx, thread.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if found.ViewsCount != 2 {
		t.Errorf("ViewsCount = %d, want 2 (просмотр спустя >24ч засчитывается снова)", found.ViewsCount)
	}
}

func TestThreadRepo_IncrementViewsIfNotRecentlyViewed_ThreadNotFound(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewThreadRepo(pool)
	ctx := context.Background()
	viewer := createTestForumUser(t, pool)

	err := repo.IncrementViewsIfNotRecentlyViewed(ctx, uuid.New(), viewer.ID)
	if err != models.ErrThreadNotFound {
		t.Fatalf("IncrementViewsIfNotRecentlyViewed() error = %v, want ErrThreadNotFound", err)
	}
}

func TestThreadRepo_Update(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewThreadRepo(pool)
	ctx := context.Background()
	author := createTestForumUser(t, pool)
	thread := createTestThread(t, pool, repo, author.ID, "old title", []models.ThreadTag{models.TagHumor})

	updated, err := repo.Update(ctx, thread.ID, "new title", "new content", []models.ThreadTag{models.TagNews})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Title != "new title" || updated.Content != "new content" {
		t.Errorf("Update() = %+v, want title/content updated", updated)
	}
	if len(updated.Tags) != 1 || updated.Tags[0] != models.TagNews {
		t.Errorf("Tags = %v, want [news]", updated.Tags)
	}
}

func TestThreadRepo_Update_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewThreadRepo(pool)

	_, err := repo.Update(context.Background(), uuid.New(), "t", "c", nil)
	if err != models.ErrThreadNotFound {
		t.Fatalf("Update() error = %v, want ErrThreadNotFound", err)
	}
}

func TestThreadRepo_SoftDelete_ExcludesFromFindAndList(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewThreadRepo(pool)
	ctx := context.Background()
	author := createTestForumUser(t, pool)
	thread := createTestThread(t, pool, repo, author.ID, "to be deleted", nil)

	if err := repo.SoftDelete(ctx, thread.ID); err != nil {
		t.Fatalf("SoftDelete() error = %v", err)
	}

	if _, err := repo.FindByID(ctx, thread.ID); err != models.ErrThreadNotFound {
		t.Fatalf("FindByID() after delete error = %v, want ErrThreadNotFound", err)
	}

	// повторный SoftDelete на уже удалённом треде - тоже not found
	if err := repo.SoftDelete(ctx, thread.ID); err != models.ErrThreadNotFound {
		t.Fatalf("SoftDelete() twice error = %v, want ErrThreadNotFound", err)
	}
}

func TestThreadRepo_List_FiltersByTagAndAuthor(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewThreadRepo(pool)
	ctx := context.Background()
	author := createTestForumUser(t, pool)
	otherAuthor := createTestForumUser(t, pool)

	studyThread := createTestThread(t, pool, repo, author.ID, "study thread", []models.ThreadTag{models.TagStudy})
	createTestThread(t, pool, repo, author.ID, "humor thread", []models.ThreadTag{models.TagHumor})
	createTestThread(t, pool, repo, otherAuthor.ID, "other author study", []models.ThreadTag{models.TagStudy})

	tag := models.TagStudy
	threads, total, err := repo.List(ctx, models.ThreadListFilter{Tag: &tag, AuthorID: &author.ID, Page: 1, Limit: 20})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if total != 1 {
		t.Fatalf("total = %d, want 1", total)
	}
	if len(threads) != 1 || threads[0].ID != studyThread.ID {
		t.Fatalf("List() = %v, want only %v", threads, studyThread.ID)
	}
}

func TestThreadRepo_List_FiltersByQ_CaseInsensitiveSubstring(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewThreadRepo(pool)
	ctx := context.Background()
	author := createTestForumUser(t, pool)
	suffix := uuid.NewString()[:8]

	matching := createTestThread(t, pool, repo, author.ID, "Kак сдать Анатомию "+suffix, nil)
	createTestThread(t, pool, repo, author.ID, "Продам стетоскоп "+suffix, nil)

	q := "анатом"
	threads, total, err := repo.List(ctx, models.ThreadListFilter{Q: &q, Page: 1, Limit: 20})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if total != 1 || len(threads) != 1 || threads[0].ID != matching.ID {
		t.Fatalf("List(q=%q) = %v (total=%d), want only %v (регистронезависимый поиск по подстроке)", q, threads, total, matching.ID)
	}
}

func TestThreadRepo_List_SortPopular(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewThreadRepo(pool)
	ctx := context.Background()
	author := createTestForumUser(t, pool)

	quiet := createTestThread(t, pool, repo, author.ID, "quiet thread", nil)
	popular := createTestThread(t, pool, repo, author.ID, "popular thread", nil)

	if _, err := pool.Exec(ctx, "UPDATE threads SET likes_count = 50 WHERE id = $1", popular.ID); err != nil {
		t.Fatalf("failed to bump likes_count: %v", err)
	}

	threads, _, err := repo.List(ctx, models.ThreadListFilter{AuthorID: &author.ID, Sort: "popular", Page: 1, Limit: 20})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(threads) < 2 {
		t.Fatalf("expected at least 2 threads, got %d", len(threads))
	}
	if threads[0].ID != popular.ID {
		t.Errorf("first thread = %v, want the popular one %v", threads[0].ID, popular.ID)
	}
	_ = quiet
}

func TestThreadRepo_Hide(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewThreadRepo(pool)
	ctx := context.Background()
	author := createTestForumUser(t, pool)
	moderator := createTestForumUser(t, pool)
	thread := createTestThread(t, pool, repo, author.ID, "spam thread", nil)

	hidden, err := repo.Hide(ctx, thread.ID, moderator.ID, "спам")
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

	// скрытый тред остаётся доступным по прямому FindByID (только List его прячет)
	found, err := repo.FindByID(ctx, thread.ID)
	if err != nil {
		t.Fatalf("FindByID() after hide error = %v", err)
	}
	if found.HiddenAt == nil {
		t.Error("FindByID() should still return the hidden thread with HiddenAt set")
	}
}

func TestThreadRepo_Hide_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewThreadRepo(pool)

	_, err := repo.Hide(context.Background(), uuid.New(), uuid.New(), "reason")
	if err != models.ErrThreadNotFound {
		t.Fatalf("Hide() error = %v, want ErrThreadNotFound", err)
	}
}
