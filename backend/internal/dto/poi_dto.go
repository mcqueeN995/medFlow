package dto

import (
	"time"

	"github.com/medflow/backend/internal/models"
)

type POI struct {
	ID                 string         `json:"id"`
	Name               string         `json:"name"`
	Type               models.PoiType `json:"type"`
	Latitude           float64        `json:"latitude"`
	Longitude          float64        `json:"longitude"`
	Address            *string        `json:"address,omitempty"`
	Description        *string        `json:"description,omitempty"`
	PhotoURL           *string        `json:"photo_url,omitempty"`
	Rating             float64        `json:"rating"`
	Tags               []string       `json:"tags"`
	DistanceMeters     *int           `json:"distance_meters"`
	WalkingTimeSeconds *int           `json:"walking_time_seconds"`
	CreatedAt          time.Time      `json:"created_at"`
}

func ToPOI(p *models.POI) POI {
	tags := p.Tags
	if tags == nil {
		tags = []string{}
	}
	return POI{
		ID: p.ID.String(), Name: p.Name, Type: p.Type, Latitude: p.Latitude, Longitude: p.Longitude,
		Address: p.Address, Description: p.Description, PhotoURL: p.PhotoURL, Rating: p.Rating, Tags: tags,
		DistanceMeters: p.DistanceMeters, WalkingTimeSeconds: p.WalkingTimeSeconds, CreatedAt: p.CreatedAt,
	}
}

type CreatePOIRequest struct {
	Name        string         `json:"name" binding:"required,max=255"`
	Type        models.PoiType `json:"type" binding:"required,oneof=coworking cafe library canteen park other"`
	Latitude    float64        `json:"latitude" binding:"required"`
	Longitude   float64        `json:"longitude" binding:"required"`
	Address     *string        `json:"address,omitempty"`
	Description *string        `json:"description,omitempty"`
	PhotoURL    *string        `json:"photo_url,omitempty" binding:"omitempty,uri"`
	Tags        []string       `json:"tags,omitempty"`
}

type UpdatePOIRequest struct {
	Name        *string         `json:"name,omitempty" binding:"omitempty,max=255"`
	Type        *models.PoiType `json:"type,omitempty" binding:"omitempty,oneof=coworking cafe library canteen park other"`
	Latitude    *float64        `json:"latitude,omitempty"`
	Longitude   *float64        `json:"longitude,omitempty"`
	Address     *string         `json:"address,omitempty"`
	Description *string         `json:"description,omitempty"`
	PhotoURL    *string         `json:"photo_url,omitempty" binding:"omitempty,uri"`
	Tags        []string        `json:"tags,omitempty"`
}
