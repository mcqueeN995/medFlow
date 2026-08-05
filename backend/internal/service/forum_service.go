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
}

func NewForumService(
	threadRepo ThreadRepository,
	commentRepo CommentRepository,
	reactionRepo ReactionRepository,
	reportRepo ReportRepository,
) *ForumService {
	return &ForumService{
		threadRepo:   threadRepo,
		commentRepo:  commentRepo,
		reactionRepo: reactionRepo,
		reportRepo:   reportRepo,
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

func (s *ForumService) GetThread(ctx context.Context, id uuid.UUID) (*dto.Thread, error) {
	if err := s.threadRepo.IncrementViews(ctx, id); err != nil {
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
	if _, err := s.threadRepo.FindByID(ctx, threadID); err != nil {
		return nil, s.mapThreadErr(err)
	}

	var parentID *uuid.UUID
	depth := 0
	if req.ParentID != nil && *req.ParentID != "" {
		parsedParentID, err := uuid.Parse(*req.ParentID)
		if err != nil {
			return nil, ErrParentCommentNotFound
		}
		parent, err := s.commentRepo.FindByID(ctx, parsedParentID)
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
	out := dto.ToComment(comment)
	return &out, nil
}

func (s *ForumService) ListComments(ctx context.Context, threadID uuid.UUID, page, limit int) (*dto.Pagination, []dto.CommentTree, error) {
	if _, err := s.threadRepo.FindByID(ctx, threadID); err != nil {
		return nil, nil, s.mapThreadErr(err)
	}
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}

	comments, total, err := s.commentRepo.ListByThread(ctx, threadID, page, limit)
	if err != nil {
		return nil, nil, err
	}

	items := make([]dto.CommentTree, len(comments))
	for i := range comments {
		items[i] = dto.ToCommentTree(&comments[i])
	}

	pagination := dto.NewPagination(page, limit, total)
	return &pagination, items, nil
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
	out := dto.ToComment(updated)
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
