package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/medflow/backend/internal/dto"
	"github.com/medflow/backend/internal/models"
	"github.com/medflow/backend/internal/service"
)

type AdminHandler struct {
	adminService *service.AdminService
}

func NewAdminHandler(adminService *service.AdminService) *AdminHandler {
	return &AdminHandler{adminService: adminService}
}

func MapAdminServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrReportNotFound):
		RespondWithError(c, http.StatusNotFound, "NOT_FOUND", "report not found", nil)
	default:
		RespondWithError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", nil)
	}
}

// AdminListReports GET /api/v1/admin/reports
func (h *AdminHandler) AdminListReports(c *gin.Context) {
	var q struct {
		Status     string `form:"status"`
		TargetType string `form:"target_type"`
		Page       int    `form:"page,default=1"`
		Limit      int    `form:"limit,default=20"`
	}
	if err := c.ShouldBindQuery(&q); err != nil {
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid query params", nil)
		return
	}

	filter := models.ReportListFilter{Page: q.Page, Limit: q.Limit}
	if q.Status != "" {
		status := models.ReportStatus(q.Status)
		filter.Status = &status
	}
	if q.TargetType != "" {
		filter.TargetType = &q.TargetType
	}

	pagination, items, err := h.adminService.ListReports(c.Request.Context(), filter)
	if err != nil {
		MapAdminServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"pagination": pagination, "data": items})
}

// AdminReviewReport PATCH /api/v1/admin/reports/:id
func (h *AdminHandler) AdminReviewReport(c *gin.Context) {
	actorID, ok := requireUserID(c)
	if !ok {
		return
	}
	id, ok := pathUUID(c, "id")
	if !ok {
		return
	}

	var req dto.ReviewReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}

	report, err := h.adminService.ReviewReport(c.Request.Context(), actorID, id, req.Status, req.ResolutionNote)
	if err != nil {
		MapAdminServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, report)
}

// AdminStats GET /api/v1/admin/stats
func (h *AdminHandler) AdminStats(c *gin.Context) {
	stats, err := h.adminService.Stats(c.Request.Context())
	if err != nil {
		MapAdminServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, stats)
}

// AdminAuditLogs GET /api/v1/admin/audit-logs
func (h *AdminHandler) AdminAuditLogs(c *gin.Context) {
	var q struct {
		ActorID    string `form:"actor_id"`
		Action     string `form:"action"`
		TargetType string `form:"target_type"`
		From       string `form:"from"`
		To         string `form:"to"`
		Page       int    `form:"page,default=1"`
		Limit      int    `form:"limit,default=50"`
	}
	if err := c.ShouldBindQuery(&q); err != nil {
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid query params", nil)
		return
	}

	filter := models.AuditLogListFilter{Page: q.Page, Limit: q.Limit}
	if q.ActorID != "" {
		id, err := uuid.Parse(q.ActorID)
		if err != nil {
			RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid actor_id", nil)
			return
		}
		filter.ActorID = &id
	}
	if q.Action != "" {
		action := models.AuditAction(q.Action)
		filter.Action = &action
	}
	if q.TargetType != "" {
		filter.TargetType = &q.TargetType
	}
	if q.From != "" {
		from, err := time.Parse(time.RFC3339, q.From)
		if err != nil {
			RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid from", nil)
			return
		}
		filter.From = &from
	}
	if q.To != "" {
		to, err := time.Parse(time.RFC3339, q.To)
		if err != nil {
			RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid to", nil)
			return
		}
		filter.To = &to
	}

	pagination, items, err := h.adminService.AuditLogs(c.Request.Context(), filter)
	if err != nil {
		MapAdminServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"pagination": pagination, "data": items})
}
