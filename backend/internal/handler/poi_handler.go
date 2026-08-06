package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/medflow/backend/internal/dto"
	"github.com/medflow/backend/internal/models"
	"github.com/medflow/backend/internal/service"
)

type POIHandler struct {
	poiService *service.POIService
}

func NewPOIHandler(poiService *service.POIService) *POIHandler {
	return &POIHandler{poiService: poiService}
}

func MapPOIServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrPOINotFound):
		RespondWithError(c, http.StatusNotFound, "NOT_FOUND", "poi not found", nil)
	default:
		RespondWithError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", nil)
	}
}

// ListPOI GET /api/v1/map/poi
func (h *POIHandler) ListPOI(c *gin.Context) {
	var q struct {
		CampusID string   `form:"campus_id"`
		Type     string   `form:"type"`
		Lat      *float64 `form:"lat"`
		Lon      *float64 `form:"lon"`
		Radius   *int     `form:"radius"`
		Tags     string   `form:"tags"`
	}
	if err := c.ShouldBindQuery(&q); err != nil {
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid query params", nil)
		return
	}

	filter := models.PoiListFilter{Lat: q.Lat, Lon: q.Lon, Radius: q.Radius}
	if q.CampusID != "" {
		id, err := uuid.Parse(q.CampusID)
		if err != nil {
			RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid campus_id", nil)
			return
		}
		filter.CampusID = &id
	}
	if q.Type != "" {
		t := models.PoiType(q.Type)
		filter.Type = &t
	}
	if q.Tags != "" {
		for _, tag := range strings.Split(q.Tags, ",") {
			if tag != "" {
				filter.Tags = append(filter.Tags, tag)
			}
		}
	}

	items, err := h.poiService.List(c.Request.Context(), filter)
	if err != nil {
		MapPOIServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}

// ==================== ADMIN ====================

// AdminListPOI GET /api/v1/admin/poi
func (h *POIHandler) AdminListPOI(c *gin.Context) {
	var q struct {
		Page  int `form:"page,default=1"`
		Limit int `form:"limit,default=50"`
	}
	if err := c.ShouldBindQuery(&q); err != nil {
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid query params", nil)
		return
	}

	pagination, items, err := h.poiService.AdminList(c.Request.Context(), q.Page, q.Limit)
	if err != nil {
		MapPOIServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"pagination": pagination, "data": items})
}

// AdminCreatePOI POST /api/v1/admin/poi
func (h *POIHandler) AdminCreatePOI(c *gin.Context) {
	actorID, ok := requireUserID(c)
	if !ok {
		return
	}
	var req dto.CreatePOIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}

	poi, err := h.poiService.AdminCreate(c.Request.Context(), actorID, req)
	if err != nil {
		MapPOIServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, poi)
}

// AdminUpdatePOI PATCH /api/v1/admin/poi/:id
func (h *POIHandler) AdminUpdatePOI(c *gin.Context) {
	actorID, ok := requireUserID(c)
	if !ok {
		return
	}
	id, ok := pathUUID(c, "id")
	if !ok {
		return
	}

	var req dto.UpdatePOIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}

	poi, err := h.poiService.AdminUpdate(c.Request.Context(), actorID, id, req)
	if err != nil {
		MapPOIServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, poi)
}

// AdminDeletePOI DELETE /api/v1/admin/poi/:id
func (h *POIHandler) AdminDeletePOI(c *gin.Context) {
	actorID, ok := requireUserID(c)
	if !ok {
		return
	}
	id, ok := pathUUID(c, "id")
	if !ok {
		return
	}
	if err := h.poiService.AdminDelete(c.Request.Context(), actorID, id); err != nil {
		MapPOIServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
