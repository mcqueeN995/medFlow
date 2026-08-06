package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/medflow/backend/internal/models"
)

type POIRepo struct {
	pool *pgxpool.Pool
}

func NewPOIRepo(pool *pgxpool.Pool) *POIRepo {
	return &POIRepo{pool: pool}
}

const poiSelectColumns = `
	p.id, p.name, p.type, p.latitude, p.longitude, p.address, p.description, p.photo_url, p.rating, p.tags::text[], p.created_at, p.updated_at
`

func (r *POIRepo) scan(row pgx.Row) (*models.POI, error) {
	var p models.POI
	var poiType string
	var tags []string
	err := row.Scan(&p.ID, &p.Name, &poiType, &p.Latitude, &p.Longitude, &p.Address, &p.Description, &p.PhotoURL, &p.Rating, &tags, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	p.Type = models.PoiType(poiType)
	p.Tags = tags
	return &p, nil
}

func (r *POIRepo) Create(ctx context.Context, p *models.POI) (*models.POI, error) {
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `
		INSERT INTO poi (name, type, latitude, longitude, address, description, photo_url, tags)
		VALUES ($1, $2::poi_type, $3, $4, $5, $6, $7, $8::text[])
		RETURNING id
	`, p.Name, string(p.Type), p.Latitude, p.Longitude, p.Address, p.Description, p.PhotoURL, p.Tags).Scan(&id)
	if err != nil {
		return nil, err
	}
	return r.FindByID(ctx, id)
}

func (r *POIRepo) FindByID(ctx context.Context, id uuid.UUID) (*models.POI, error) {
	query := `SELECT ` + poiSelectColumns + ` FROM poi p WHERE p.id = $1`
	p, err := r.scan(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrPOINotFound
		}
		return nil, err
	}
	return p, nil
}

func (r *POIRepo) Update(ctx context.Context, id uuid.UUID, p *models.POI) (*models.POI, error) {
	cmd, err := r.pool.Exec(ctx, `
		UPDATE poi SET
			name = $2, type = $3::poi_type, latitude = $4, longitude = $5, address = $6,
			description = $7, photo_url = $8, tags = $9::text[], updated_at = now()
		WHERE id = $1
	`, id, p.Name, string(p.Type), p.Latitude, p.Longitude, p.Address, p.Description, p.PhotoURL, p.Tags)
	if err != nil {
		return nil, err
	}
	if cmd.RowsAffected() == 0 {
		return nil, models.ErrPOINotFound
	}
	return r.FindByID(ctx, id)
}

// Delete - физическое удаление: таблица poi не хранит deleted_at/hidden_at.
// poi_campus_links ссылается на poi дважды (poi_id и campus_id - "кампус"
// тоже строка poi, см. схему БД) без ON DELETE CASCADE, поэтому сперва
// чистим все связи, где id участвует любой стороной, иначе DELETE падает на
// FK-констрейнте.
func (r *POIRepo) Delete(ctx context.Context, id uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM poi_campus_links WHERE poi_id = $1 OR campus_id = $1`, id); err != nil {
		return err
	}

	cmd, err := tx.Exec(ctx, `DELETE FROM poi WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return models.ErrPOINotFound
	}

	return tx.Commit(ctx)
}

// List - публичный список для /map/poi. Если задан CampusID, применяется
// INNER JOIN на poi_campus_links: список сужается только до POI, привязанных
// к этому "кампусу" (campus сам моделируется как строка poi, см. схему БД),
// а distance_meters/walking_time_seconds берутся из заранее сохранённой
// связи. Без CampusID эти два поля остаются nil - их считает сервис по
// haversine, если в запросе были lat/lon (см. POIService.List).
func (r *POIRepo) List(ctx context.Context, f models.PoiListFilter) ([]models.POI, error) {
	selectCols := poiSelectColumns + `, NULL::int, NULL::int`
	from := `FROM poi p`
	where := "1=1"
	var args []any
	argN := 1

	if f.CampusID != nil {
		selectCols = poiSelectColumns + `, pcl.distance_meters, pcl.walking_time_seconds`
		from = `FROM poi p JOIN poi_campus_links pcl ON pcl.poi_id = p.id`
		where += fmt.Sprintf(" AND pcl.campus_id = $%d", argN)
		args = append(args, *f.CampusID)
		argN++
	}
	if f.Type != nil {
		where += fmt.Sprintf(" AND p.type = $%d::poi_type", argN)
		args = append(args, string(*f.Type))
		argN++
	}
	if len(f.Tags) > 0 {
		where += fmt.Sprintf(" AND p.tags @> $%d::text[]", argN)
		args = append(args, f.Tags)
		argN++
	}

	query := fmt.Sprintf(`SELECT %s %s WHERE %s ORDER BY p.created_at DESC`, selectCols, from, where)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.POI
	for rows.Next() {
		p, err := r.scanWithDistance(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

func (r *POIRepo) scanWithDistance(row pgx.Row) (*models.POI, error) {
	var p models.POI
	var poiType string
	var tags []string
	err := row.Scan(&p.ID, &p.Name, &poiType, &p.Latitude, &p.Longitude, &p.Address, &p.Description, &p.PhotoURL, &p.Rating, &tags, &p.CreatedAt, &p.UpdatedAt, &p.DistanceMeters, &p.WalkingTimeSeconds)
	if err != nil {
		return nil, err
	}
	p.Type = models.PoiType(poiType)
	p.Tags = tags
	return &p, nil
}

func (r *POIRepo) AdminList(ctx context.Context, f models.AdminPoiListFilter) ([]models.POI, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM poi`).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `SELECT ` + poiSelectColumns + ` FROM poi p ORDER BY p.created_at DESC LIMIT $1 OFFSET $2`
	rows, err := r.pool.Query(ctx, query, f.Limit, (f.Page-1)*f.Limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []models.POI
	for rows.Next() {
		p, err := r.scan(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *p)
	}
	return out, total, rows.Err()
}
