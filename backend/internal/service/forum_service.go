package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/medflow/backend/internal/dto"
	"github.com/medflow/backend/internal/models"
)

var (
	ErrThreadNotFound        = errors.New("thread not found")
	ErrCommentNotFound       = errors.New("comment not found")
	ErrParentCommentNotFound = errors.New("parent comment not found")
	ErrForbidden             = errors.New("not the author")
	ErrReactionNotFound      = errors.New("reaction not found")
)

type ForumService struct {
	threadRepo   ThreadRepository
	commentRepo  CommentRepository
	reactionRepo ReactionRepository
	reportRepo   ReportRepository
	auditLogRepo AuditLogRepository
	pushNotifier PushNotifier
}

func NewForumService(
	threadRepo ThreadRepository,
	commentRepo CommentRepository,
	reactionRepo ReactionRepository,
	reportRepo ReportRepository,
	auditLogRepo AuditLogRepository,
	pushNotifier PushNotifier,
) *ForumService {
	return &ForumService{
		threadRepo:   threadRepo,
		commentRepo:  commentRepo,
		reactionRepo: reactionRepo,
		reportRepo:   reportRepo,
		auditLogRepo: auditLogRepo,
		pushNotifier: pushNotifier,
	}
}

func (s *ForumService) CreateThread(ctx context.Context, authorID uuid.UUID, req dto.CreateThreadRequest) (*dto.Thread, error) {
	thread, err := s.threadRepo.Create(ctx, authorID, req.Title, req.Content, req.Tags)
	if err != nil {
		return nil, err
	}
	out := dto.ToThread(thread)
	return &out, nil
}

func (s *ForumService) ListThreads(ctx context.Context, f models.ThreadListFilter) (*dto.Pagination, []dto.ThreadListItem, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Limit < 1 || f.Limit > 100 {
		f.Limit = 20
	}

	threads, total, err := s.threadRepo.List(ctx, f)
	if err != nil {
		return nil, nil, err
	}

	items := make([]dto.ThreadListItem, len(threads))
	for i := range threads {
		items[i] = dto.ToThreadListItem(&threads[i])
	}

	pagination := dto.NewPagination(f.Page, f.Limit, total)
	return &pagination, items, nil
}

func (s *ForumService) GetThread(ctx context.Context, id, viewerID uuid.UUID) (*dto.Thread, error) {
	if err := s.threadRepo.IncrementViewsIfNotRecentlyViewed(ctx, id, viewerID); err != nil {
		return nil, s.mapThreadErr(err)
	}
	thread, err := s.threadRepo.FindByID(ctx, id)
	if err != nil {
		return nil, s.mapThreadErr(err)
	}
	out := dto.ToThread(thread)
	return &out, nil
}

func (s *ForumService) UpdateThread(ctx context.Context, id, userID uuid.UUID, req dto.UpdateThreadRequest) (*dto.Thread, error) {
	thread, err := s.threadRepo.FindByID(ctx, id)
	if err != nil {
		return nil, s.mapThreadErr(err)
	}
	if thread.Author.ID != userID {
		return nil, ErrForbidden
	}

	title, content, tags := thread.Title, thread.Content, thread.Tags
	if req.Title != nil {
		title = *req.Title
	}
	if req.Content != nil {
		content = *req.Content
	}
	if req.Tags != nil {
		tags = *req.Tags
	}

	updated, err := s.threadRepo.Update(ctx, id, title, content, tags)
	if err != nil {
		return nil, s.mapThreadErr(err)
	}
	out := dto.ToThread(updated)
	return &out, nil
}

func (s *ForumService) DeleteThread(ctx context.Context, id, userID uuid.UUID) error {
	thread, err := s.threadRepo.FindByID(ctx, id)
	if err != nil {
		return s.mapThreadErr(err)
	}
	if thread.Author.ID != userID {
		return ErrForbidden
	}
	return s.mapThreadErr(s.threadRepo.SoftDelete(ctx, id))
}

func (s *ForumService) AddReaction(ctx context.Context, userID uuid.UUID, targetType models.ReactionTargetType, targetID uuid.UUID, emoji string) (*dto.Reaction, error) {
	reaction, err := s.reactionRepo.Upsert(ctx, userID, targetType, targetID, emoji)
	if err != nil {
		return nil, err
	}
	out := dto.ToReaction(reaction)
	return &out, nil
}

func (s *ForumService) RemoveReaction(ctx context.Context, userID uuid.UUID, targetType models.ReactionTargetType, targetID uuid.UUID) error {
	err := s.reactionRepo.Delete(ctx, userID, targetType, targetID)
	if errors.Is(err, models.ErrReactionNotFound) {
		return ErrReactionNotFound
	}
	return err
}

func (s *ForumService) Report(ctx context.Context, reporterID uuid.UUID, targetType string, targetID uuid.UUID, reason string) (*dto.Report, error) {
	report, err := s.reportRepo.Create(ctx, reporterID, targetType, targetID, reason)
	if err != nil {
		return nil, err
	}
	out := dto.ToReport(report)
	return &out, nil
}

func (s *ForumService) CreateComment(ctx context.Context, threadID, authorID uuid.UUID, req dto.CreateCommentRequest) (*dto.Comment, error) {
	thread, err := s.threadRepo.FindByID(ctx, threadID)
	if err != nil {
		return nil, s.mapThreadErr(err)
	}

	var parentID *uuid.UUID
	var parent *models.Comment
	depth := 0
	if req.ParentID != nil && *req.ParentID != "" {
		parsedParentID, err := uuid.Parse(*req.ParentID)
		if err != nil {
			return nil, ErrParentCommentNotFound
		}
		parent, err = s.commentRepo.FindByID(ctx, parsedParentID)
		if err != nil || parent.ThreadID != threadID {
			return nil, ErrParentCommentNotFound
		}

		// дерево ограничено 2 уровнями: ответ на ответ схлопывается к
		// родителю верхнего уровня, а не создаёт depth=2.
		if parent.Depth == 0 {
			parentID = &parent.ID
		} else {
			parentID = parent.ParentID
		}
		depth = 1
	}

	comment, err := s.commentRepo.Create(ctx, threadID, authorID, parentID, depth, req.Content)
	if err != nil {
		return nil, err
	}
	s.notifyReply(ctx, authorID, thread, parent)
	out := dto.ToComment(comment, nil)
	return &out, nil
}

// notifyReply - лучшее старание: push-уведомление не должно ронять уже
// созданный комментарий. Ответ на комментарий уведомляет автора комментария
// (comment_reply), топ-левел ответ в треде - автора треда (thread_reply);
// себя не уведомляем.
func (s *ForumService) notifyReply(ctx context.Context, authorID uuid.UUID, thread *models.Thread, parent *models.Comment) {
	if parent != nil {
		if parent.Author.ID != authorID {
			_ = s.pushNotifier.Notify(ctx, parent.Author.ID, models.NotificationCommentReply, "Новый ответ", thread.Author.Nickname+" ответил(а) на ваш комментарий")
		}
		return
	}
	if thread.Author.ID != authorID {
		_ = s.pushNotifier.Notify(ctx, thread.Author.ID, models.NotificationThreadReply, "Новый ответ в теме", "Кто-то ответил в теме «"+thread.Title+"»")
	}
}

func (s *ForumService) ListComments(ctx context.Context, threadID, viewerID uuid.UUID, page, limit int, sort string) (*dto.Pagination, []dto.CommentTree, error) {
	if _, err := s.threadRepo.FindByID(ctx, threadID); err != nil {
		return nil, nil, s.mapThreadErr(err)
	}
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}
	if sort != "best" {
		sort = "new"
	}

	comments, total, err := s.commentRepo.ListByThread(ctx, threadID, page, limit, sort)
	if err != nil {
		return nil, nil, err
	}

	// Батч голосов для верхнего уровня и реплаев разом - по аналогии с
	// repliesByParent в CommentRepo.ListByThread: один запрос на список ID
	// вместо N+1 при обогащении каждого комментария по отдельности.
	allIDs := make([]uuid.UUID, 0, len(comments)*2)
	for i := range comments {
		allIDs = append(allIDs, comments[i].ID)
		for _, reply := range comments[i].Replies {
			allIDs = append(allIDs, reply.ID)
		}
	}
	summaries, err := s.reactionRepo.VoteSummaries(ctx, models.ReactionTargetComment, allIDs, viewerID)
	if err != nil {
		return nil, nil, err
	}

	items := make([]dto.CommentTree, len(comments))
	for i := range comments {
		items[i] = dto.ToCommentTree(&comments[i], summaries)
	}

	pagination := dto.NewPagination(page, limit, total)
	return &pagination, items, nil
}

// VoteComment - голос up/down. В отличие от AddReaction (эмодзи-лайк),
// сосуществует с ним независимо на одном комментарии (kind='vote' vs
// kind='emoji', см. миграцию 000016_reactions_kind).
func (s *ForumService) VoteComment(ctx context.Context, userID, commentID uuid.UUID, direction string) (*dto.VoteResult, error) {
	if _, err := s.commentRepo.FindByID(ctx, commentID); err != nil {
		return nil, s.mapCommentErr(err)
	}
	if _, err := s.reactionRepo.UpsertVote(ctx, userID, models.ReactionTargetComment, commentID, direction); err != nil {
		return nil, err
	}
	return s.commentVoteResult(ctx, userID, commentID)
}

func (s *ForumService) RemoveCommentVote(ctx context.Context, userID, commentID uuid.UUID) (*dto.VoteResult, error) {
	if err := s.reactionRepo.DeleteVote(ctx, userID, models.ReactionTargetComment, commentID); err != nil {
		if errors.Is(err, models.ErrReactionNotFound) {
			return nil, ErrReactionNotFound
		}
		return nil, err
	}
	return s.commentVoteResult(ctx, userID, commentID)
}

func (s *ForumService) commentVoteResult(ctx context.Context, userID, commentID uuid.UUID) (*dto.VoteResult, error) {
	summaries, err := s.reactionRepo.VoteSummaries(ctx, models.ReactionTargetComment, []uuid.UUID{commentID}, userID)
	if err != nil {
		return nil, err
	}
	summary := summaries[commentID]
	return &dto.VoteResult{Score: summary.Score, MyVote: summary.MyVote}, nil
}

func (s *ForumService) UpdateComment(ctx context.Context, id, userID uuid.UUID, content string) (*dto.Comment, error) {
	comment, err := s.commentRepo.FindByID(ctx, id)
	if err != nil {
		return nil, s.mapCommentErr(err)
	}
	if comment.Author.ID != userID {
		return nil, ErrForbidden
	}

	updated, err := s.commentRepo.Update(ctx, id, content)
	if err != nil {
		return nil, s.mapCommentErr(err)
	}
	out := dto.ToComment(updated, nil)
	return &out, nil
}

func (s *ForumService) DeleteComment(ctx context.Context, id, userID uuid.UUID) error {
	comment, err := s.commentRepo.FindByID(ctx, id)
	if err != nil {
		return s.mapCommentErr(err)
	}
	if comment.Author.ID != userID {
		return ErrForbidden
	}
	return s.mapCommentErr(s.commentRepo.SoftDelete(ctx, id, comment.ThreadID))
}

// ==================== ADMIN (moderator+/admin) ====================

func (s *ForumService) AdminHideThread(ctx context.Context, actorID, id uuid.UUID, reason string) (*dto.Thread, error) {
	hidden, err := s.threadRepo.Hide(ctx, id, actorID, reason)
	if err != nil {
		return nil, s.mapThreadErr(err)
	}
	s.writeAudit(ctx, actorID, models.AuditThreadHide, "thread", id, &reason)
	out := dto.ToThread(hidden)
	return &out, nil
}

func (s *ForumService) AdminHideComment(ctx context.Context, actorID, id uuid.UUID, reason string) (*dto.Comment, error) {
	hidden, err := s.commentRepo.Hide(ctx, id, actorID, reason)
	if err != nil {
		return nil, s.mapCommentErr(err)
	}
	s.writeAudit(ctx, actorID, models.AuditCommentHide, "comment", id, &reason)
	out := dto.ToComment(hidden, nil)
	return &out, nil
}

// AdminDeleteThread - в отличие от DeleteThread, без проверки авторства
// (модератор/админ вправе удалить чужой тред).
func (s *ForumService) AdminDeleteThread(ctx context.Context, actorID, id uuid.UUID) error {
	if err := s.mapThreadErr(s.threadRepo.SoftDelete(ctx, id)); err != nil {
		return err
	}
	s.writeAudit(ctx, actorID, models.AuditThreadDelete, "thread", id, nil)
	return nil
}

func (s *ForumService) AdminDeleteComment(ctx context.Context, actorID, id uuid.UUID) error {
	comment, err := s.commentRepo.FindByID(ctx, id)
	if err != nil {
		return s.mapCommentErr(err)
	}
	if err := s.mapCommentErr(s.commentRepo.SoftDelete(ctx, id, comment.ThreadID)); err != nil {
		return err
	}
	s.writeAudit(ctx, actorID, models.AuditCommentDelete, "comment", id, nil)
	return nil
}

// writeAudit - лучшее старание: сбой записи аудит-лога не должен проваливать
// уже выполненное модераторское действие.
func (s *ForumService) writeAudit(ctx context.Context, actorID uuid.UUID, action models.AuditAction, targetType string, targetID uuid.UUID, reason *string) {
	_ = s.auditLogRepo.Create(ctx, &models.AuditLog{ActorID: actorID, Action: action, TargetType: &targetType, TargetID: &targetID, Reason: reason})
}

func (s *ForumService) mapThreadErr(err error) error {
	if errors.Is(err, models.ErrThreadNotFound) {
		return ErrThreadNotFound
	}
	return err
}

func (s *ForumService) mapCommentErr(err error) error {
	if errors.Is(err, models.ErrCommentNotFound) {
		return ErrCommentNotFound
	}
	return err
}
