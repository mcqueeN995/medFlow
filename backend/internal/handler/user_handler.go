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

type UserHandler struct {
	userService *service.UserService
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

func MapUserServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrUserNotFound):
		RespondWithError(c, http.StatusNotFound, "NOT_FOUND", "user not found", nil)
	case errors.Is(err, service.ErrNicknameExists):
		RespondWithError(c, http.StatusConflict, "NICKNAME_EXISTS", "nickname already taken", nil)
	case errors.Is(err, service.ErrInvalidCreds):
		RespondWithError(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid password", nil)
	default:
		RespondWithError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", nil)
	}
}

// Me GET /api/v1/users/me
func (h *UserHandler) Me(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	profile, err := h.userService.Me(c.Request.Context(), userID)
	if err != nil {
		MapUserServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, profile)
}

// UpdateMe PATCH /api/v1/users/me
func (h *UserHandler) UpdateMe(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	var req dto.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}

	profile, err := h.userService.UpdateProfile(c.Request.Context(), userID, req)
	if err != nil {
		MapUserServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, profile)
}

// DeleteMe DELETE /api/v1/users/me?password=...
func (h *UserHandler) DeleteMe(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	pw := c.Query("password")
	if pw == "" {
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "password is required", nil)
		return
	}

	if err := h.userService.DeleteAccount(c.Request.Context(), userID, pw); err != nil {
		MapUserServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// PublicProfile GET /api/v1/users/:id
func (h *UserHandler) PublicProfile(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid id", nil)
		return
	}
	profile, err := h.userService.PublicProfile(c.Request.Context(), id)
	if err != nil {
		MapUserServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, profile)
}

// ==================== ADMIN ====================

// AdminListUsers GET /api/v1/admin/users
func (h *UserHandler) AdminListUsers(c *gin.Context) {
	var q struct {
		Role       string `form:"role"`
		Banned     *bool  `form:"banned"`
		University string `form:"university"`
		Q          string `form:"q"`
		Page       int    `form:"page,default=1"`
		Limit      int    `form:"limit,default=20"`
	}
	if err := c.ShouldBindQuery(&q); err != nil {
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid query params", nil)
		return
	}

	filter := models.AdminUserListFilter{Banned: q.Banned, Page: q.Page, Limit: q.Limit}
	if q.Role != "" {
		role := models.UserRole(q.Role)
		filter.Role = &role
	}
	if q.University != "" {
		uni := models.University(q.University)
		filter.University = &uni
	}
	if q.Q != "" {
		filter.Q = &q.Q
	}

	pagination, items, err := h.userService.AdminList(c.Request.Context(), filter)
	if err != nil {
		MapUserServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"pagination": pagination, "data": items})
}

// AdminChangeRole PATCH /api/v1/admin/users/:id/role
func (h *UserHandler) AdminChangeRole(c *gin.Context) {
	actorID, ok := requireUserID(c)
	if !ok {
		return
	}
	id, ok := pathUUID(c, "id")
	if !ok {
		return
	}

	var req dto.ChangeRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}

	user, err := h.userService.AdminChangeRole(c.Request.Context(), actorID, id, req.Role)
	if err != nil {
		MapUserServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, user)
}

// AdminBanUser POST /api/v1/admin/users/:id/ban
func (h *UserHandler) AdminBanUser(c *gin.Context) {
	actorID, ok := requireUserID(c)
	if !ok {
		return
	}
	id, ok := pathUUID(c, "id")
	if !ok {
		return
	}

	var req dto.BanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}

	user, err := h.userService.AdminBan(c.Request.Context(), actorID, id, req.Reason)
	if err != nil {
		MapUserServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, user)
}

// AdminUnbanUser DELETE /api/v1/admin/users/:id/ban
func (h *UserHandler) AdminUnbanUser(c *gin.Context) {
	actorID, ok := requireUserID(c)
	if !ok {
		return
	}
	id, ok := pathUUID(c, "id")
	if !ok {
		return
	}

	user, err := h.userService.AdminUnban(c.Request.Context(), actorID, id)
	if err != nil {
		MapUserServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, user)
}
