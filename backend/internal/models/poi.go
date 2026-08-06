package models

import (
	"time"

	"github.com/google/uuid"
)

type PoiType string

const (
	PoiCoworking PoiType = "coworking"
	PoiCafe      PoiType = "cafe"
	PoiLibrary   PoiType = "library"
	PoiCanteen   PoiType = "canteen"
	PoiPark      PoiType = "park"
	PoiOther     PoiType = "other"
)

// POI. DistanceMeters/WalkingTimeSeconds заполняются только при запросе с
// lat/lon либо campus_id (см. POIService.List) - таблица poi не хранит
// расстояние, это всегда производная величина относительно точки отсчёта.
// Таблица не имеет hidden_at/deleted_at - удаление POI в этом модуле
// физическое (см. POIRepository.Delete), в отличие от Thread/Comment/Textbook.
type POI struct {
	ID          uuid.UUID
	Name        string
	Type        PoiType
	Latitude    float64
	Longitude   float64
	Address     *string
	Description *string
	PhotoURL    *string
	Rating      float64
	Tags        []string
	CreatedAt   time.Time
	UpdatedAt   time.Time

	DistanceMeters     *int
	WalkingTimeSeconds *int
}

type PoiListFilter struct {
	CampusID *uuid.UUID
	Type     *PoiType
	Lat, Lon *float64
	Radius   *int
	Tags     []string
}

type AdminPoiListFilter struct {
	Page, Limit int
}
