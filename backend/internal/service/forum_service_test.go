package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/medflow/backend/internal/dto"
	"github.com/medflow/backend/internal/models"
)

func setupTestForumService(threadRepo *mockThreadRepository, commentRepo *mockCommentRepository, reactionRepo *mockReactionRepository, reportRepo *mockReportRepository) *ForumService {
	if threadRepo == nil {
		threadRepo = &mockThreadRepository{}
	}
	if commentRepo == nil {
		commentRepo = &mockCommentRepository{}
	}
	if reactionRepo == nil {
		reactionRepo = &mockReactionRepository{}
	}
	if reportRepo == nil {
		reportRepo = &mockReportRepository{}
	}
	return NewForumService(threadRepo, commentRepo, reactionRepo, reportRepo, &mockAuditLogRepository{}, &mockPushNotifier{})
}

func TestForumService_CreateThread_Success(t *testing.T) {
	authorID := uuid.New()
	threadRepo := &mockThreadRepository{
		createFn: func(ctx context.Context, gotAuthorID uuid.UUID, title, content string, tags []models.ThreadTag) (*models.Thread, error) {
			if gotAuthorID != authorID {
				t.Errorf("authorID = %v, want %v", gotAuthorID, authorID)
			}
			return &models.Thread{
				ID:      uuid.New(),
				Author:  models.PublicUser{ID: authorID, Nickname: "nik"},
				Title:   title,
				Content: content,
				Tags:    tags,
			}, nil
		},
	}
	svc := setupTestForumService(threadRepo, nil, nil, nil)

	thread, err := svc.CreateThread(context.Background(), authorID, dto.CreateThreadRequest{
		Title:   "Как сдать анатомию",
		Content: "Делитесь лайфхаками",
		Tags:    []models.ThreadTag{models.TagStudy},
	})
	if err != nil {
		t.Fatalf("CreateThread() error = %v", err)
	}
	if thread.Title != "Как сдать анатомию" {
		t.Errorf("Title = %q, want %q", thread.Title, "Как сдать анатомию")
	}
}

func TestForumService_UpdateThread_ForbiddenForNonAuthor(t *testing.T) {
	threadID := uuid.New()
	authorID := uuid.New()
	otherUserID := uuid.New()

	threadRepo := &mockThreadRepository{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.Thread, error) {
			return &models.Thread{ID: threadID, Author: models.PublicUser{ID: authorID}, Title: "old"}, nil
		},
	}
	svc := setupTestForumService(threadRepo, nil, nil, nil)

	newTitle := "hacked title"
	_, err := svc.UpdateThread(context.Background(), threadID, otherUserID, dto.UpdateThreadRequest{Title: &newTitle})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("UpdateThread() error = %v, want ErrForbidden", err)
	}
}

func TestForumService_UpdateThread_Success_PartialFieldsPreserved(t *testing.T) {
	threadID := uuid.New()
	authorID := uuid.New()

	threadRepo := &mockThreadRepository{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.Thread, error) {
			return &models.Thread{
				ID:      threadID,
				Author:  models.PublicUser{ID: authorID},
				Title:   "old title",
				Content: "old content",
				Tags:    []models.ThreadTag{models.TagHumor},
			}, nil
		},
		updateFn: func(ctx context.Context, id uuid.UUID, title, content string, tags []models.ThreadTag) (*models.Thread, error) {
			if content != "old content" {
				t.Errorf("content should be preserved when not in request, got %q", content)
			}
			if len(tags) != 1 || tags[0] != models.TagHumor {
				t.Errorf("tags should be preserved when not in request, got %v", tags)
			}
			return &models.Thread{ID: id, Author: models.PublicUser{ID: authorID}, Title: title, Content: content, Tags: tags}, nil
		},
	}
	svc := setupTestForumService(threadRepo, nil, nil, nil)

	newTitle := "new title"
	thread, err := svc.UpdateThread(context.Background(), threadID, authorID, dto.UpdateThreadRequest{Title: &newTitle})
	if err != nil {
		t.Fatalf("UpdateThread() error = %v", err)
	}
	if thread.Title != "new title" {
		t.Errorf("Title = %q, want %q", thread.Title, "new title")
	}
}

func TestForumService_DeleteThread_NotFound(t *testing.T) {
	threadRepo := &mockThreadRepository{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.Thread, error) {
			return nil, models.ErrThreadNotFound
		},
	}
	svc := setupTestForumService(threadRepo, nil, nil, nil)

	err := svc.DeleteThread(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, ErrThreadNotFound) {
		t.Fatalf("DeleteThread() error = %v, want ErrThreadNotFound", err)
	}
}

func TestForumService_CreateComment_TopLevel(t *testing.T) {
	threadID := uuid.New()
	authorID := uuid.New()

	threadRepo := &mockThreadRepository{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.Thread, error) {
			return &models.Thread{ID: threadID}, nil
		},
	}
	commentRepo := &mockCommentRepository{
		createFn: func(ctx context.Context, gotThreadID, gotAuthorID uuid.UUID, parentID *uuid.UUID, depth int, content string) (*models.Comment, error) {
			if parentID != nil {
				t.Errorf("parentID = %v, want nil for a top-level comment", *parentID)
			}
			if depth != 0 {
				t.Errorf("depth = %d, want 0", depth)
			}
			return &models.Comment{ID: uuid.New(), ThreadID: gotThreadID, Author: models.PublicUser{ID: gotAuthorID}, Content: content, Depth: depth}, nil
		},
	}
	svc := setupTestForumService(threadRepo, commentRepo, nil, nil)

	_, err := svc.CreateComment(context.Background(), threadID, authorID, dto.CreateCommentRequest{Content: "первый!"})
	if err != nil {
		t.Fatalf("CreateComment() error = %v", err)
	}
}

func TestForumService_CreateComment_ReplyToTopLevel_BecomesDepth1(t *testing.T) {
	threadID := uuid.New()
	parentID := uuid.New()
	authorID := uuid.New()
	parentIDStr := parentID.String()

	threadRepo := &mockThreadRepository{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.Thread, error) {
			return &models.Thread{ID: threadID}, nil
		},
	}
	commentRepo := &mockCommentRepository{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.Comment, error) {
			return &models.Comment{ID: parentID, ThreadID: threadID, Depth: 0, ParentID: nil}, nil
		},
		createFn: func(ctx context.Context, gotThreadID, gotAuthorID uuid.UUID, gotParentID *uuid.UUID, depth int, content string) (*models.Comment, error) {
			if gotParentID == nil || *gotParentID != parentID {
				t.Errorf("parentID = %v, want %v", gotParentID, parentID)
			}
			if depth != 1 {
				t.Errorf("depth = %d, want 1", depth)
			}
			return &models.Comment{ID: uuid.New(), ThreadID: gotThreadID, ParentID: gotParentID, Depth: depth, Author: models.PublicUser{ID: gotAuthorID}, Content: content}, nil
		},
	}
	svc := setupTestForumService(threadRepo, commentRepo, nil, nil)

	_, err := svc.CreateComment(context.Background(), threadID, authorID, dto.CreateCommentRequest{Content: "ответ", ParentID: &parentIDStr})
	if err != nil {
		t.Fatalf("CreateComment() error = %v", err)
	}
}

func TestForumService_CreateComment_ReplyToReply_FlattensToGrandparent(t *testing.T) {
	threadID := uuid.New()
	grandparentID := uuid.New()
	replyID := uuid.New()
	authorID := uuid.New()
	replyIDStr := replyID.String()

	threadRepo := &mockThreadRepository{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.Thread, error) {
			return &models.Thread{ID: threadID}, nil
		},
	}
	commentRepo := &mockCommentRepository{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.Comment, error) {
			// reply уже сам depth=1, его родитель - grandparent (depth=0)
			return &models.Comment{ID: replyID, ThreadID: threadID, Depth: 1, ParentID: &grandparentID}, nil
		},
		createFn: func(ctx context.Context, gotThreadID, gotAuthorID uuid.UUID, gotParentID *uuid.UUID, depth int, content string) (*models.Comment, error) {
			if gotParentID == nil || *gotParentID != grandparentID {
				t.Errorf("parentID = %v, want grandparent %v (flattened)", gotParentID, grandparentID)
			}
			if depth != 1 {
				t.Errorf("depth = %d, want 1 (tree capped at 2 levels)", depth)
			}
			return &models.Comment{ID: uuid.New(), ThreadID: gotThreadID, ParentID: gotParentID, Depth: depth, Author: models.PublicUser{ID: gotAuthorID}, Content: content}, nil
		},
	}
	svc := setupTestForumService(threadRepo, commentRepo, nil, nil)

	_, err := svc.CreateComment(context.Background(), threadID, authorID, dto.CreateCommentRequest{Content: "ответ на ответ", ParentID: &replyIDStr})
	if err != nil {
		t.Fatalf("CreateComment() error = %v", err)
	}
}

func TestForumService_CreateComment_ParentFromDifferentThread(t *testing.T) {
	threadID := uuid.New()
	otherThreadID := uuid.New()
	parentID := uuid.New()
	parentIDStr := parentID.String()

	threadRepo := &mockThreadRepository{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.Thread, error) {
			return &models.Thread{ID: threadID}, nil
		},
	}
	commentRepo := &mockCommentRepository{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.Comment, error) {
			return &models.Comment{ID: parentID, ThreadID: otherThreadID, Depth: 0}, nil
		},
	}
	svc := setupTestForumService(threadRepo, commentRepo, nil, nil)

	_, err := svc.CreateComment(context.Background(), threadID, uuid.New(), dto.CreateCommentRequest{Content: "x", ParentID: &parentIDStr})
	if !errors.Is(err, ErrParentCommentNotFound) {
		t.Fatalf("CreateComment() error = %v, want ErrParentCommentNotFound", err)
	}
}

func TestForumService_CreateComment_TopLevel_NotifiesThreadAuthor(t *testing.T) {
	threadID := uuid.New()
	threadAuthorID := uuid.New()
	commentAuthorID := uuid.New()

	threadRepo := &mockThreadRepository{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.Thread, error) {
			return &models.Thread{ID: threadID, Title: "тема", Author: models.PublicUser{ID: threadAuthorID}}, nil
		},
	}
	commentRepo := &mockCommentRepository{
		createFn: func(ctx context.Context, gotThreadID, gotAuthorID uuid.UUID, parentID *uuid.UUID, depth int, content string) (*models.Comment, error) {
			return &models.Comment{ID: uuid.New(), ThreadID: gotThreadID, Author: models.PublicUser{ID: gotAuthorID}, Content: content}, nil
		},
	}
	notifier := &mockPushNotifier{}
	svc := NewForumService(threadRepo, commentRepo, &mockReactionRepository{}, &mockReportRepository{}, &mockAuditLogRepository{}, notifier)

	_, err := svc.CreateComment(context.Background(), threadID, commentAuthorID, dto.CreateCommentRequest{Content: "первый!"})
	if err != nil {
		t.Fatalf("CreateComment() error = %v", err)
	}
	if len(notifier.calls) != 1 || notifier.calls[0] != models.NotificationThreadReply {
		t.Fatalf("notifier.calls = %v, want [thread_reply]", notifier.calls)
	}
}

func TestForumService_CreateComment_Reply_NotifiesParentAuthorNotSelf(t *testing.T) {
	threadID := uuid.New()
	parentID := uuid.New()
	parentIDStr := parentID.String()
	authorID := uuid.New() // отвечает сам себе на свой же комментарий

	threadRepo := &mockThreadRepository{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.Thread, error) {
			return &models.Thread{ID: threadID, Author: models.PublicUser{ID: uuid.New()}}, nil
		},
	}
	commentRepo := &mockCommentRepository{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.Comment, error) {
			return &models.Comment{ID: parentID, ThreadID: threadID, Depth: 0, Author: models.PublicUser{ID: authorID}}, nil
		},
		createFn: func(ctx context.Context, gotThreadID, gotAuthorID uuid.UUID, parentID *uuid.UUID, depth int, content string) (*models.Comment, error) {
			return &models.Comment{ID: uuid.New(), ThreadID: gotThreadID, ParentID: parentID, Depth: depth, Author: models.PublicUser{ID: gotAuthorID}, Content: content}, nil
		},
	}
	notifier := &mockPushNotifier{}
	svc := NewForumService(threadRepo, commentRepo, &mockReactionRepository{}, &mockReportRepository{}, &mockAuditLogRepository{}, notifier)

	_, err := svc.CreateComment(context.Background(), threadID, authorID, dto.CreateCommentRequest{Content: "ответ самому себе", ParentID: &parentIDStr})
	if err != nil {
		t.Fatalf("CreateComment() error = %v", err)
	}
	if len(notifier.calls) != 0 {
		t.Fatalf("notifier.calls = %v, want no notification for self-reply", notifier.calls)
	}
}

func TestForumService_UpdateComment_ForbiddenForNonAuthor(t *testing.T) {
	commentID := uuid.New()
	authorID := uuid.New()

	commentRepo := &mockCommentRepository{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.Comment, error) {
			return &models.Comment{ID: commentID, Author: models.PublicUser{ID: authorID}}, nil
		},
	}
	svc := setupTestForumService(nil, commentRepo, nil, nil)

	_, err := svc.UpdateComment(context.Background(), commentID, uuid.New(), "новый текст")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("UpdateComment() error = %v, want ErrForbidden", err)
	}
}

func TestForumService_RemoveReaction_NotFound(t *testing.T) {
	reactionRepo := &mockReactionRepository{
		deleteFn: func(ctx context.Context, userID uuid.UUID, targetType models.ReactionTargetType, targetID uuid.UUID) error {
			return models.ErrReactionNotFound
		},
	}
	svc := setupTestForumService(nil, nil, reactionRepo, nil)

	err := svc.RemoveReaction(context.Background(), uuid.New(), models.ReactionTargetThread, uuid.New())
	if !errors.Is(err, ErrReactionNotFound) {
		t.Fatalf("RemoveReaction() error = %v, want ErrReactionNotFound", err)
	}
}

func TestForumService_ListThreads_ClampsPageAndLimit(t *testing.T) {
	var gotFilter models.ThreadListFilter
	threadRepo := &mockThreadRepository{
		listFn: func(ctx context.Context, f models.ThreadListFilter) ([]models.Thread, int, error) {
			gotFilter = f
			return nil, 0, nil
		},
	}
	svc := setupTestForumService(threadRepo, nil, nil, nil)

	pagination, _, err := svc.ListThreads(context.Background(), models.ThreadListFilter{Page: 0, Limit: 0})
	if err != nil {
		t.Fatalf("ListThreads() error = %v", err)
	}
	if gotFilter.Page != 1 {
		t.Errorf("Page = %d, want 1", gotFilter.Page)
	}
	if gotFilter.Limit != 20 {
		t.Errorf("Limit = %d, want 20", gotFilter.Limit)
	}
	if pagination.Page != 1 || pagination.Limit != 20 {
		t.Errorf("pagination = %+v, want Page=1 Limit=20", pagination)
	}
}

func TestForumService_AdminHideThread_WritesAuditLog(t *testing.T) {
	actorID, threadID := uuid.New(), uuid.New()
	var hiddenBy uuid.UUID
	var hiddenReason string
	threadRepo := &mockThreadRepository{
		hideFn: func(ctx context.Context, id, by uuid.UUID, reason string) (*models.Thread, error) {
			hiddenBy, hiddenReason = by, reason
			return &models.Thread{ID: id, HiddenAt: nil}, nil
		},
	}
	var auditEntry *models.AuditLog
	auditRepo := &mockAuditLogRepository{
		createFn: func(ctx context.Context, entry *models.AuditLog) error { auditEntry = entry; return nil },
	}
	svc := NewForumService(threadRepo, &mockCommentRepository{}, &mockReactionRepository{}, &mockReportRepository{}, auditRepo, &mockPushNotifier{})

	if _, err := svc.AdminHideThread(context.Background(), actorID, threadID, "спам"); err != nil {
		t.Fatalf("AdminHideThread() error = %v", err)
	}
	if hiddenBy != actorID || hiddenReason != "спам" {
		t.Errorf("Hide() called with by=%v reason=%q", hiddenBy, hiddenReason)
	}
	if auditEntry == nil || auditEntry.Action != models.AuditThreadHide || auditEntry.ActorID != actorID {
		t.Fatalf("audit log entry = %+v, want AuditThreadHide by %v", auditEntry, actorID)
	}
}

func TestForumService_AdminDeleteThread_BypassesOwnership(t *testing.T) {
	actorID, threadID := uuid.New(), uuid.New()
	softDeleteCalled := false
	threadRepo := &mockThreadRepository{
		softDeleteFn: func(ctx context.Context, id uuid.UUID) error { softDeleteCalled = true; return nil },
	}
	var auditEntry *models.AuditLog
	auditRepo := &mockAuditLogRepository{
		createFn: func(ctx context.Context, entry *models.AuditLog) error { auditEntry = entry; return nil },
	}
	svc := NewForumService(threadRepo, &mockCommentRepository{}, &mockReactionRepository{}, &mockReportRepository{}, auditRepo, &mockPushNotifier{})

	// В отличие от DeleteThread, AdminDeleteThread не проверяет FindByID/авторство -
	// удаляет напрямую по id, даже если "актёр" не автор.
	if err := svc.AdminDeleteThread(context.Background(), actorID, threadID); err != nil {
		t.Fatalf("AdminDeleteThread() error = %v", err)
	}
	if !softDeleteCalled {
		t.Error("expected SoftDelete to be called")
	}
	if auditEntry == nil || auditEntry.Action != models.AuditThreadDelete {
		t.Fatalf("audit log entry = %+v, want AuditThreadDelete", auditEntry)
	}
}

func TestForumService_AdminHideComment_WritesAuditLog(t *testing.T) {
	actorID, commentID := uuid.New(), uuid.New()
	commentRepo := &mockCommentRepository{
		hideFn: func(ctx context.Context, id, by uuid.UUID, reason string) (*models.Comment, error) {
			return &models.Comment{ID: id}, nil
		},
	}
	var auditEntry *models.AuditLog
	auditRepo := &mockAuditLogRepository{
		createFn: func(ctx context.Context, entry *models.AuditLog) error { auditEntry = entry; return nil },
	}
	svc := NewForumService(&mockThreadRepository{}, commentRepo, &mockReactionRepository{}, &mockReportRepository{}, auditRepo, &mockPushNotifier{})

	if _, err := svc.AdminHideComment(context.Background(), actorID, commentID, "спам"); err != nil {
		t.Fatalf("AdminHideComment() error = %v", err)
	}
	if auditEntry == nil || auditEntry.Action != models.AuditCommentHide {
		t.Fatalf("audit log entry = %+v, want AuditCommentHide", auditEntry)
	}
}

func TestForumService_AdminDeleteComment_BypassesOwnership(t *testing.T) {
	actorID, commentID, threadID := uuid.New(), uuid.New(), uuid.New()
	softDeleteCalled := false
	commentRepo := &mockCommentRepository{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.Comment, error) {
			return &models.Comment{ID: id, ThreadID: threadID, Author: models.PublicUser{ID: uuid.New()}}, nil
		},
		softDeleteFn: func(ctx context.Context, id, gotThreadID uuid.UUID) error { softDeleteCalled = true; return nil },
	}
	var auditEntry *models.AuditLog
	auditRepo := &mockAuditLogRepository{
		createFn: func(ctx context.Context, entry *models.AuditLog) error { auditEntry = entry; return nil },
	}
	svc := NewForumService(&mockThreadRepository{}, commentRepo, &mockReactionRepository{}, &mockReportRepository{}, auditRepo, &mockPushNotifier{})

	if err := svc.AdminDeleteComment(context.Background(), actorID, commentID); err != nil {
		t.Fatalf("AdminDeleteComment() error = %v", err)
	}
	if !softDeleteCalled {
		t.Error("expected SoftDelete to be called even though actor is not the comment author")
	}
	if auditEntry == nil || auditEntry.Action != models.AuditCommentDelete {
		t.Fatalf("audit log entry = %+v, want AuditCommentDelete", auditEntry)
	}
}
