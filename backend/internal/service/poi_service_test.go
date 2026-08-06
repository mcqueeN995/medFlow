package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/medflow/backend/internal/dto"
	"github.com/medflow/backend/internal/models"
)

func TestPOIService_List_ComputesDistanceAndSortsByProximity(t *testing.T) {
	origin := struct{ lat, lon float64 }{55.7325, 37.582}
	near := models.POI{ID: uuid.New(), Name: "near", Latitude: 55.7326, Longitude: 37.5821} // ~15м
	far := models.POI{ID: uuid.New(), Name: "far", Latitude: 55.75, Longitude: 37.6}        // несколько км

	repo := &mockPOIRepository{
		listFn: func(ctx context.Context, f models.PoiListFilter) ([]models.POI, error) {
			return []models.POI{far, near}, nil // намеренно в "неправильном" порядке
		},
	}
	svc := NewPOIService(repo, &mockAuditLogRepository{})

	items, err := svc.List(context.Background(), models.PoiListFilter{Lat: &origin.lat, Lon: &origin.lon})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if items[0].ID != near.ID.String() {
		t.Errorf("items[0] = %q, want the nearer POI %q first", items[0].ID, near.ID)
	}
	if items[0].DistanceMeters == nil || *items[0].DistanceMeters >= *items[1].DistanceMeters {
		t.Errorf("distances not ascending: %v, %v", items[0].DistanceMeters, items[1].DistanceMeters)
	}
}

func TestPOIService_List_FiltersByRadius(t *testing.T) {
	origin := struct{ lat, lon float64 }{55.7325, 37.582}
	near := models.POI{ID: uuid.New(), Latitude: 55.7326, Longitude: 37.5821}
	far := models.POI{ID: uuid.New(), Latitude: 55.9, Longitude: 37.9}

	repo := &mockPOIRepository{
		listFn: func(ctx context.Context, f models.PoiListFilter) ([]models.POI, error) {
			return []models.POI{near, far}, nil
		},
	}
	svc := NewPOIService(repo, &mockAuditLogRepository{})

	radius := 500
	items, err := svc.List(context.Background(), models.PoiListFilter{Lat: &origin.lat, Lon: &origin.lon, Radius: &radius})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items) != 1 || items[0].ID != near.ID.String() {
		t.Fatalf("List(radius=500) = %v, want only the near POI", items)
	}
}

func TestPOIService_List_CampusID_KeepsRepoDistanceUntouched(t *testing.T) {
	poi := models.POI{ID: uuid.New(), DistanceMeters: intPtrPOI(350), WalkingTimeSeconds: intPtrPOI(260)}
	campusID := uuid.New()

	repo := &mockPOIRepository{
		listFn: func(ctx context.Context, f models.PoiListFilter) ([]models.POI, error) {
			if f.CampusID == nil || *f.CampusID != campusID {
				t.Errorf("campus_id not passed through to repo: %v", f.CampusID)
			}
			return []models.POI{poi}, nil
		},
	}
	svc := NewPOIService(repo, &mockAuditLogRepository{})

	items, err := svc.List(context.Background(), models.PoiListFilter{CampusID: &campusID})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items) != 1 || items[0].DistanceMeters == nil || *items[0].DistanceMeters != 350 {
		t.Fatalf("List(campus_id) = %v, want distance untouched from repo (350)", items)
	}
}

func TestPOIService_List_NoLocationFilter_NoDistance(t *testing.T) {
	poi := models.POI{ID: uuid.New()}
	repo := &mockPOIRepository{
		listFn: func(ctx context.Context, f models.PoiListFilter) ([]models.POI, error) { return []models.POI{poi}, nil },
	}
	svc := NewPOIService(repo, &mockAuditLogRepository{})

	items, err := svc.List(context.Background(), models.PoiListFilter{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if items[0].DistanceMeters != nil {
		t.Errorf("DistanceMeters = %v, want nil without lat/lon/campus_id", items[0].DistanceMeters)
	}
}

func TestPOIService_AdminCreate(t *testing.T) {
	var created *models.POI
	repo := &mockPOIRepository{
		createFn: func(ctx context.Context, p *models.POI) (*models.POI, error) {
			cp := *p
			cp.ID = uuid.New()
			created = &cp
			return &cp, nil
		},
	}
	svc := NewPOIService(repo, &mockAuditLogRepository{})

	out, err := svc.AdminCreate(context.Background(), uuid.New(), dto.CreatePOIRequest{
		Name: "Coffee 8", Type: models.PoiCafe, Latitude: 55.73, Longitude: 37.58,
	})
	if err != nil {
		t.Fatalf("AdminCreate() error = %v", err)
	}
	if out.Name != "Coffee 8" || created.Type != models.PoiCafe {
		t.Errorf("AdminCreate() = %+v, unexpected", out)
	}
}

func TestPOIService_AdminUpdate_PartialFieldsPreserved(t *testing.T) {
	id := uuid.New()
	repo := &mockPOIRepository{
		findByIDFn: func(ctx context.Context, gotID uuid.UUID) (*models.POI, error) {
			return &models.POI{ID: id, Name: "old", Type: models.PoiCafe, Latitude: 1, Longitude: 2}, nil
		},
		updateFn: func(ctx context.Context, gotID uuid.UUID, p *models.POI) (*models.POI, error) {
			return p, nil
		},
	}
	svc := NewPOIService(repo, &mockAuditLogRepository{})

	newName := "new name"
	out, err := svc.AdminUpdate(context.Background(), uuid.New(), id, dto.UpdatePOIRequest{Name: &newName})
	if err != nil {
		t.Fatalf("AdminUpdate() error = %v", err)
	}
	if out.Name != "new name" || out.Type != models.PoiCafe {
		t.Errorf("AdminUpdate() = %+v, want name updated and type preserved", out)
	}
}

func TestPOIService_AdminUpdate_NotFound(t *testing.T) {
	repo := &mockPOIRepository{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.POI, error) { return nil, models.ErrPOINotFound },
	}
	svc := NewPOIService(repo, &mockAuditLogRepository{})

	_, err := svc.AdminUpdate(context.Background(), uuid.New(), uuid.New(), dto.UpdatePOIRequest{})
	if !errors.Is(err, ErrPOINotFound) {
		t.Fatalf("AdminUpdate() error = %v, want ErrPOINotFound", err)
	}
}

func TestPOIService_AdminDelete_NotFound(t *testing.T) {
	repo := &mockPOIRepository{
		deleteFn: func(ctx context.Context, id uuid.UUID) error { return models.ErrPOINotFound },
	}
	svc := NewPOIService(repo, &mockAuditLogRepository{})

	err := svc.AdminDelete(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, ErrPOINotFound) {
		t.Fatalf("AdminDelete() error = %v, want ErrPOINotFound", err)
	}
}

func TestPOIService_AdminCreate_WritesAuditLog(t *testing.T) {
	actorID := uuid.New()
	repo := &mockPOIRepository{
		createFn: func(ctx context.Context, p *models.POI) (*models.POI, error) {
			p.ID = uuid.New()
			return p, nil
		},
	}
	var auditEntry *models.AuditLog
	auditRepo := &mockAuditLogRepository{
		createFn: func(ctx context.Context, entry *models.AuditLog) error { auditEntry = entry; return nil },
	}
	svc := NewPOIService(repo, auditRepo)

	_, err := svc.AdminCreate(context.Background(), actorID, dto.CreatePOIRequest{
		Name: "Coffee 8", Type: models.PoiCafe, Latitude: 55.73, Longitude: 37.58,
	})
	if err != nil {
		t.Fatalf("AdminCreate() error = %v", err)
	}
	if auditEntry == nil || auditEntry.Action != models.AuditPOICreate || auditEntry.ActorID != actorID {
		t.Fatalf("audit log entry = %+v, want AuditPOICreate by %v", auditEntry, actorID)
	}
}

func intPtrPOI(n int) *int { return &n }
