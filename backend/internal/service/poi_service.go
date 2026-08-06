package service

import (
	"context"
	"errors"
	"math"
	"sort"

	"github.com/google/uuid"
	"github.com/medflow/backend/internal/dto"
	"github.com/medflow/backend/internal/models"
)

var ErrPOINotFound = errors.New("poi not found")

// Формула Haversine и средняя скорость пешехода - точный порт
// frontend/src/lib/geo.ts (там прямо сказано в комментарии, что бэкенд
// считает так же - оценка расстояния/времени обязана совпадать на клиенте
// и сервере).
const (
	earthRadiusMeters      = 6371000.0
	averageWalkingSpeedMPS = 1.35
)

type POIService struct {
	poiRepo      POIRepository
	auditLogRepo AuditLogRepository
}

func NewPOIService(poiRepo POIRepository, auditLogRepo AuditLogRepository) *POIService {
	return &POIService{poiRepo: poiRepo, auditLogRepo: auditLogRepo}
}

// List. Если задан campus_id, расстояние/время берутся из сохранённой связи
// poi_campus_links (см. POIRepository.List). Иначе, если заданы lat/lon,
// расстояние считается на лету по Haversine, список фильтруется по radius
// (если задан) и сортируется по возрастанию расстояния - как в
// frontend-моке.
func (s *POIService) List(ctx context.Context, f models.PoiListFilter) ([]dto.POI, error) {
	pois, err := s.poiRepo.List(ctx, f)
	if err != nil {
		return nil, err
	}

	if f.CampusID == nil && f.Lat != nil && f.Lon != nil {
		filtered := pois[:0]
		for _, p := range pois {
			meters := haversineMeters(*f.Lat, *f.Lon, p.Latitude, p.Longitude)
			d := int(math.Round(meters))
			w := int(math.Round(meters / averageWalkingSpeedMPS))
			p.DistanceMeters = &d
			p.WalkingTimeSeconds = &w
			if f.Radius != nil && d > *f.Radius {
				continue
			}
			filtered = append(filtered, p)
		}
		pois = filtered
		sort.Slice(pois, func(i, j int) bool { return *pois[i].DistanceMeters < *pois[j].DistanceMeters })
	}

	items := make([]dto.POI, len(pois))
	for i := range pois {
		items[i] = dto.ToPOI(&pois[i])
	}
	return items, nil
}

func haversineMeters(lat1, lon1, lat2, lon2 float64) float64 {
	toRad := func(deg float64) float64 { return deg * math.Pi / 180 }
	dLat := toRad(lat2 - lat1)
	dLon := toRad(lon2 - lon1)
	sinLat := math.Sin(dLat / 2)
	sinLon := math.Sin(dLon / 2)
	h := sinLat*sinLat + math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*sinLon*sinLon
	return 2 * earthRadiusMeters * math.Asin(math.Sqrt(h))
}

// ==================== ADMIN ====================

func (s *POIService) AdminList(ctx context.Context, page, limit int) (*dto.Pagination, []dto.POI, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}

	pois, total, err := s.poiRepo.AdminList(ctx, models.AdminPoiListFilter{Page: page, Limit: limit})
	if err != nil {
		return nil, nil, err
	}
	items := make([]dto.POI, len(pois))
	for i := range pois {
		items[i] = dto.ToPOI(&pois[i])
	}
	pagination := dto.NewPagination(page, limit, total)
	return &pagination, items, nil
}

func (s *POIService) AdminCreate(ctx context.Context, actorID uuid.UUID, req dto.CreatePOIRequest) (*dto.POI, error) {
	p := &models.POI{
		Name: req.Name, Type: req.Type, Latitude: req.Latitude, Longitude: req.Longitude,
		Address: req.Address, Description: req.Description, PhotoURL: req.PhotoURL, Tags: req.Tags,
	}
	created, err := s.poiRepo.Create(ctx, p)
	if err != nil {
		return nil, err
	}
	s.writeAudit(ctx, actorID, models.AuditPOICreate, created.ID)
	out := dto.ToPOI(created)
	return &out, nil
}

func (s *POIService) AdminUpdate(ctx context.Context, actorID, id uuid.UUID, req dto.UpdatePOIRequest) (*dto.POI, error) {
	existing, err := s.poiRepo.FindByID(ctx, id)
	if err != nil {
		return nil, s.mapErr(err)
	}

	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.Type != nil {
		existing.Type = *req.Type
	}
	if req.Latitude != nil {
		existing.Latitude = *req.Latitude
	}
	if req.Longitude != nil {
		existing.Longitude = *req.Longitude
	}
	if req.Address != nil {
		existing.Address = req.Address
	}
	if req.Description != nil {
		existing.Description = req.Description
	}
	if req.PhotoURL != nil {
		existing.PhotoURL = req.PhotoURL
	}
	if req.Tags != nil {
		existing.Tags = req.Tags
	}

	updated, err := s.poiRepo.Update(ctx, id, existing)
	if err != nil {
		return nil, s.mapErr(err)
	}
	s.writeAudit(ctx, actorID, models.AuditPOIUpdate, id)
	out := dto.ToPOI(updated)
	return &out, nil
}

func (s *POIService) AdminDelete(ctx context.Context, actorID, id uuid.UUID) error {
	if err := s.mapErr(s.poiRepo.Delete(ctx, id)); err != nil {
		return err
	}
	s.writeAudit(ctx, actorID, models.AuditPOIDelete, id)
	return nil
}

// writeAudit - лучшее старание: сбой записи аудит-лога не должен проваливать
// уже выполненное действие над точкой интереса.
func (s *POIService) writeAudit(ctx context.Context, actorID uuid.UUID, action models.AuditAction, targetID uuid.UUID) {
	targetType := "poi"
	_ = s.auditLogRepo.Create(ctx, &models.AuditLog{ActorID: actorID, Action: action, TargetType: &targetType, TargetID: &targetID})
}

func (s *POIService) mapErr(err error) error {
	if errors.Is(err, models.ErrPOINotFound) {
		return ErrPOINotFound
	}
	return err
}
