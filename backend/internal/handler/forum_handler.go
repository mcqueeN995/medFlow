package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/medflow/backend/internal/dto"
	"github.com/medflow/backend/internal/models"
	"github.com/medflow/backend/internal/service"
)

type ForumHandler struct {
	forumService *service.ForumService
}

func NewForumHandler(forumService *service.ForumService) *ForumHandler {
	return &ForumHandler{forumService: forumService}
}

func MapForumServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrThreadNotFound), errors.Is(err, service.ErrCommentNotFound):
		RespondWithError(c, http.StatusNotFound, "NOT_FOUND", "not found", nil)
	case errors.Is(err, service.ErrParentCommentNotFound):
		RespondWithError(c, http.StatusBadRequest, "INVALID_PARENT", "parent comment not found in this thread", nil)
	case errors.Is(err, service.ErrForbidden):
		RespondWithError(c, http.StatusForbidden, "FORBIDDEN", "you are not the author", nil)
	case errors.Is(err, service.ErrReactionNotFound):
		RespondWithError(c, http.StatusNotFound, "NOT_FOUND", "reaction not found", nil)
	default:
		RespondWithError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", nil)
	}
}

func requireUserID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(GetUserID(c))
	if err != nil {
		RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required", nil)
		return uuid.UUID{}, false
	}
	return id, true
}

func pathUUID(c *gin.Context, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(name))
	if err != nil {
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid id", nil)
		return uuid.UUID{}, false
	}
	return id, true
}

// ==================== THREADS ====================

// CreateThread POST /api/v1/threads
func (h *ForumHandler) CreateThread(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	var req dto.CreateThreadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}

	thread, err := h.forumService.CreateThread(c.Request.Context(), userID, req)
	if err != nil {
		MapForumServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, thread)
}

// ListThreads GET /api/v1/threads
func (h *ForumHandler) ListThreads(c *gin.Context) {
	var q struct {
		Tag      string `form:"tag"`
		AuthorID string `form:"author_id"`
		Q        string `form:"q"`
		Sort     string `form:"sort"`
		Page     int    `form:"page,default=1"`
		Limit    int    `form:"limit,default=20"`
	}
	if err := c.ShouldBindQuery(&q); err != nil {
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid query params", nil)
		return
	}

	filter := models.ThreadListFilter{Sort: q.Sort, Page: q.Page, Limit: q.Limit}
	if q.Tag != "" {
		tag := models.ThreadTag(q.Tag)
		filter.Tag = &tag
	}
	if q.AuthorID != "" {
		authorID, err := uuid.Parse(q.AuthorID)
		if err != nil {
			RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid author_id", nil)
			return
		}
		filter.AuthorID = &authorID
	}
	if q.Q != "" {
		filter.Q = &q.Q
	}

	pagination, items, err := h.forumService.ListThreads(c.Request.Context(), filter)
	if err != nil {
		MapForumServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"pagination": pagination, "data": items})
}

// GetThread GET /api/v1/threads/:id
func (h *ForumHandler) GetThread(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	id, ok := pathUUID(c, "id")
	if !ok {
		return
	}
	thread, err := h.forumService.GetThread(c.Request.Context(), id, userID)
	if err != nil {
		MapForumServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, thread)
}

// UpdateThread PATCH /api/v1/threads/:id
func (h *ForumHandler) UpdateThread(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	id, ok := pathUUID(c, "id")
	if !ok {
		return
	}

	var req dto.UpdateThreadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}

	thread, err := h.forumService.UpdateThread(c.Request.Context(), id, userID, req)
	if err != nil {
		MapForumServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, thread)
}

// DeleteThread DELETE /api/v1/threads/:id
func (h *ForumHandler) DeleteThread(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	id, ok := pathUUID(c, "id")
	if !ok {
		return
	}
	if err := h.forumService.DeleteThread(c.Request.Context(), id, userID); err != nil {
		MapForumServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// AddThreadReaction POST /api/v1/threads/:id/reactions
func (h *ForumHandler) AddThreadReaction(c *gin.Context) {
	h.addReaction(c, "id", models.ReactionTargetThread)
}

// RemoveThreadReaction DELETE /api/v1/threads/:id/reactions
func (h *ForumHandler) RemoveThreadReaction(c *gin.Context) {
	h.removeReaction(c, "id", models.ReactionTargetThread)
}

// ReportThread POST /api/v1/threads/:id/report
func (h *ForumHandler) ReportThread(c *gin.Context) {
	h.report(c, "id", "thread")
}

// ListComments GET /api/v1/threads/:id/comments
func (h *ForumHandler) ListComments(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	threadID, ok := pathUUID(c, "id")
	if !ok {
		return
	}

	var q struct {
		Page  int    `form:"page,default=1"`
		Limit int    `form:"limit,default=50"`
		Sort  string `form:"sort,default=new"`
	}
	if err := c.ShouldBindQuery(&q); err != nil {
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid query params", nil)
		return
	}

	pagination, items, err := h.forumService.ListComments(c.Request.Context(), threadID, userID, q.Page, q.Limit, q.Sort)
	if err != nil {
		MapForumServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"pagination": pagination, "data": items})
}

// ==================== COMMENTS ====================

// CreateComment POST /api/v1/threads/:id/comments
func (h *ForumHandler) CreateComment(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	threadID, ok := pathUUID(c, "id")
	if !ok {
		return
	}

	var req dto.CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}

	comment, err := h.forumService.CreateComment(c.Request.Context(), threadID, userID, req)
	if err != nil {
		MapForumServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, comment)
}

// UpdateComment PATCH /api/v1/comments/:id
func (h *ForumHandler) UpdateComment(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	id, ok := pathUUID(c, "id")
	if !ok {
		return
	}

	var req dto.UpdateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}

	comment, err := h.forumService.UpdateComment(c.Request.Context(), id, userID, req.Content)
	if err != nil {
		MapForumServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, comment)
}

// DeleteComment DELETE /api/v1/comments/:id
func (h *ForumHandler) DeleteComment(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	id, ok := pathUUID(c, "id")
	if !ok {
		return
	}
	if err := h.forumService.DeleteComment(c.Request.Context(), id, userID); err != nil {
		MapForumServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// AddCommentReaction POST /api/v1/comments/:id/reactions
func (h *ForumHandler) AddCommentReaction(c *gin.Context) {
	h.addReaction(c, "id", models.ReactionTargetComment)
}

// RemoveCommentReaction DELETE /api/v1/comments/:id/reactions
func (h *ForumHandler) RemoveCommentReaction(c *gin.Context) {
	h.removeReaction(c, "id", models.ReactionTargetComment)
}

// VoteComment POST /api/v1/comments/:id/vote
func (h *ForumHandler) VoteComment(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	id, ok := pathUUID(c, "id")
	if !ok {
		return
	}

	var req dto.VoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}

	result, err := h.forumService.VoteComment(c.Request.Context(), userID, id, req.Direction)
	if err != nil {
		MapForumServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// RemoveCommentVote DELETE /api/v1/comments/:id/vote
func (h *ForumHandler) RemoveCommentVote(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	id, ok := pathUUID(c, "id")
	if !ok {
		return
	}

	result, err := h.forumService.RemoveCommentVote(c.Request.Context(), userID, id)
	if err != nil {
		MapForumServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

// ReportComment POST /api/v1/comments/:id/report
func (h *ForumHandler) ReportComment(c *gin.Context) {
	h.report(c, "id", "comment")
}

// ==================== SHARED ====================

func (h *ForumHandler) addReaction(c *gin.Context, param string, targetType models.ReactionTargetType) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	targetID, ok := pathUUID(c, param)
	if !ok {
		return
	}

	var req dto.ReactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}

	reaction, err := h.forumService.AddReaction(c.Request.Context(), userID, targetType, targetID, req.Emoji)
	if err != nil {
		MapForumServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, reaction)
}

func (h *ForumHandler) removeReaction(c *gin.Context, param string, targetType models.ReactionTargetType) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	targetID, ok := pathUUID(c, param)
	if !ok {
		return
	}

	if err := h.forumService.RemoveReaction(c.Request.Context(), userID, targetType, targetID); err != nil {
		MapForumServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *ForumHandler) report(c *gin.Context, param, targetType string) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	targetID, ok := pathUUID(c, param)
	if !ok {
		return
	}

	var req dto.ReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}

	report, err := h.forumService.Report(c.Request.Context(), userID, targetType, targetID, req.Reason)
	if err != nil {
		MapForumServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, report)
}

// ==================== ADMIN (moderator+/admin) ====================

// AdminHideThread POST /api/v1/admin/threads/:id/hide
func (h *ForumHandler) AdminHideThread(c *gin.Context) {
	actorID, ok := requireUserID(c)
	if !ok {
		return
	}
	id, ok := pathUUID(c, "id")
	if !ok {
		return
	}
	var req dto.HideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}

	thread, err := h.forumService.AdminHideThread(c.Request.Context(), actorID, id, req.Reason)
	if err != nil {
		MapForumServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, thread)
}

// AdminHideComment POST /api/v1/admin/comments/:id/hide
func (h *ForumHandler) AdminHideComment(c *gin.Context) {
	actorID, ok := requireUserID(c)
	if !ok {
		return
	}
	id, ok := pathUUID(c, "id")
	if !ok {
		return
	}
	var req dto.HideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}

	comment, err := h.forumService.AdminHideComment(c.Request.Context(), actorID, id, req.Reason)
	if err != nil {
		MapForumServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, comment)
}

// AdminDeleteThread DELETE /api/v1/admin/threads/:id
func (h *ForumHandler) AdminDeleteThread(c *gin.Context) {
	actorID, ok := requireUserID(c)
	if !ok {
		return
	}
	id, ok := pathUUID(c, "id")
	if !ok {
		return
	}
	if err := h.forumService.AdminDeleteThread(c.Request.Context(), actorID, id); err != nil {
		MapForumServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// AdminDeleteComment DELETE /api/v1/admin/comments/:id
func (h *ForumHandler) AdminDeleteComment(c *gin.Context) {
	actorID, ok := requireUserID(c)
	if !ok {
		return
	}
	id, ok := pathUUID(c, "id")
	if !ok {
		return
	}
	if err := h.forumService.AdminDeleteComment(c.Request.Context(), actorID, id); err != nil {
		MapForumServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
