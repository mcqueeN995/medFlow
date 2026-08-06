package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/medflow/backend/internal/models"
)

func createTestPOI(t *testing.T, pool *pgxpool.Pool, repo *POIRepo, name string, poiType models.PoiType, tags []string) *models.POI {
	t.Helper()
	created, err := repo.Create(context.Background(), &models.POI{
		Name: name, Type: poiType, Latitude: 55.7325, Longitude: 37.582, Tags: tags,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), "DELETE FROM poi WHERE id = $1", created.ID) })
	return created
}

func TestPOIRepo_Create_And_FindByID(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewPOIRepo(pool)
	ctx := context.Background()

	created := createTestPOI(t, pool, repo, "Coffee 8", models.PoiCafe, []string{"wifi", "розетки"})

	found, err := repo.FindByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if found.Name != "Coffee 8" || found.Type != models.PoiCafe {
		t.Errorf("found = %+v, unexpected", found)
	}
	if len(found.Tags) != 2 {
		t.Errorf("Tags = %v, want 2 tags", found.Tags)
	}
}

func TestPOIRepo_FindByID_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewPOIRepo(pool)

	_, err := repo.FindByID(context.Background(), uuid.New())
	if err != models.ErrPOINotFound {
		t.Fatalf("FindByID() error = %v, want ErrPOINotFound", err)
	}
}

func TestPOIRepo_Update(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewPOIRepo(pool)
	ctx := context.Background()
	created := createTestPOI(t, pool, repo, "old name", models.PoiCafe, nil)

	updated, err := repo.Update(ctx, created.ID, &models.POI{
		Name: "new name", Type: models.PoiLibrary, Latitude: 1, Longitude: 2, Tags: []string{"тихо"},
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Name != "new name" || updated.Type != models.PoiLibrary {
		t.Errorf("Update() = %+v, want name/type updated", updated)
	}
}

func TestPOIRepo_Update_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewPOIRepo(pool)

	_, err := repo.Update(context.Background(), uuid.New(), &models.POI{Name: "x", Type: models.PoiOther})
	if err != models.ErrPOINotFound {
		t.Fatalf("Update() error = %v, want ErrPOINotFound", err)
	}
}

func TestPOIRepo_Delete_IsPhysical(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewPOIRepo(pool)
	ctx := context.Background()
	created := createTestPOI(t, pool, repo, "to delete", models.PoiPark, nil)

	if err := repo.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := repo.FindByID(ctx, created.ID); err != models.ErrPOINotFound {
		t.Fatalf("FindByID() after delete error = %v, want ErrPOINotFound", err)
	}
	if err := repo.Delete(ctx, created.ID); err != models.ErrPOINotFound {
		t.Fatalf("Delete() twice error = %v, want ErrPOINotFound", err)
	}
}

func TestPOIRepo_Delete_CleansUpCampusLinksFirst(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewPOIRepo(pool)
	ctx := context.Background()
	campus := createTestPOI(t, pool, repo, "campus anchor", models.PoiOther, nil)
	poi := createTestPOI(t, pool, repo, "linked poi", models.PoiCafe, nil)

	if _, err := pool.Exec(ctx, `
		INSERT INTO poi_campus_links (poi_id, campus_id, distance_meters, walking_time_seconds) VALUES ($1, $2, 100, 80)
	`, poi.ID, campus.ID); err != nil {
		t.Fatalf("failed to insert poi_campus_links: %v", err)
	}

	// удаление любой из двух сторон связи не должно падать на FK-констрейнте
	if err := repo.Delete(ctx, poi.ID); err != nil {
		t.Fatalf("Delete(poi referenced as poi_id) error = %v", err)
	}
	if err := repo.Delete(ctx, campus.ID); err != nil {
		t.Fatalf("Delete(poi referenced as campus_id) error = %v", err)
	}
}

func TestPOIRepo_List_FiltersByTypeAndTags(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewPOIRepo(pool)
	ctx := context.Background()

	target := createTestPOI(t, pool, repo, "quiet cafe", models.PoiCafe, []string{"wifi", "тихо"})
	createTestPOI(t, pool, repo, "loud cafe", models.PoiCafe, []string{"wifi"})
	createTestPOI(t, pool, repo, "quiet library", models.PoiLibrary, []string{"тихо"})

	cafe := models.PoiCafe
	items, err := repo.List(ctx, models.PoiListFilter{Type: &cafe, Tags: []string{"wifi", "тихо"}})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items) != 1 || items[0].ID != target.ID {
		t.Fatalf("List() = %v, want only %v", items, target.ID)
	}
}

func TestPOIRepo_List_CampusScoped_UsesLinkDistance(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewPOIRepo(pool)
	ctx := context.Background()

	campus := createTestPOI(t, pool, repo, "campus anchor", models.PoiOther, nil)
	nearby := createTestPOI(t, pool, repo, "nearby cafe", models.PoiCafe, nil)
	unrelated := createTestPOI(t, pool, repo, "unrelated cafe", models.PoiCafe, nil)
	_ = unrelated

	var linkID uuid.UUID
	err := pool.QueryRow(ctx, `
		INSERT INTO poi_campus_links (poi_id, campus_id, distance_meters, walking_time_seconds)
		VALUES ($1, $2, 350, 260) RETURNING id
	`, nearby.ID, campus.ID).Scan(&linkID)
	if err != nil {
		t.Fatalf("failed to insert poi_campus_links: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), "DELETE FROM poi_campus_links WHERE id = $1", linkID) })

	items, err := repo.List(ctx, models.PoiListFilter{CampusID: &campus.ID})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items) != 1 || items[0].ID != nearby.ID {
		t.Fatalf("List(campus_id) = %v, want only %v", items, nearby.ID)
	}
	if items[0].DistanceMeters == nil || *items[0].DistanceMeters != 350 {
		t.Errorf("DistanceMeters = %v, want 350", items[0].DistanceMeters)
	}
	if items[0].WalkingTimeSeconds == nil || *items[0].WalkingTimeSeconds != 260 {
		t.Errorf("WalkingTimeSeconds = %v, want 260", items[0].WalkingTimeSeconds)
	}
}

func TestPOIRepo_List_NoFilter_DistanceIsNil(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewPOIRepo(pool)
	ctx := context.Background()
	created := createTestPOI(t, pool, repo, "plain poi", models.PoiOther, nil)

	items, err := repo.List(ctx, models.PoiListFilter{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	var found *models.POI
	for i := range items {
		if items[i].ID == created.ID {
			found = &items[i]
		}
	}
	if found == nil {
		t.Fatal("created POI missing from List()")
	}
	if found.DistanceMeters != nil {
		t.Errorf("DistanceMeters = %v, want nil without campus_id/lat/lon", found.DistanceMeters)
	}
}

func TestPOIRepo_AdminList_Paginates(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewPOIRepo(pool)
	ctx := context.Background()
	createTestPOI(t, pool, repo, "admin poi 1", models.PoiOther, nil)
	createTestPOI(t, pool, repo, "admin poi 2", models.PoiOther, nil)

	items, total, err := repo.AdminList(ctx, models.AdminPoiListFilter{Page: 1, Limit: 1})
	if err != nil {
		t.Fatalf("AdminList() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1 (limit)", len(items))
	}
	if total < 2 {
		t.Errorf("total = %d, want >= 2", total)
	}
}
