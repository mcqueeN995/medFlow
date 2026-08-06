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

type TextbookRepo struct {
	pool *pgxpool.Pool
}

func NewTextbookRepo(pool *pgxpool.Pool) *TextbookRepo {
	return &TextbookRepo{pool: pool}
}

const textbookSelectColumns = `
	t.id, t.title, t.authors, t.isbn, t.year, t.pages, t.description, t.subject, t.course, t.department,
	t.storage_type, t.license_type, t.copyright_holder, t.hidden_at, t.deleted_at, t.created_at, t.updated_at,
	tf.pdf_s3_key, tl.source_url
`

func (r *TextbookRepo) scanTextbook(row pgx.Row) (*models.Textbook, error) {
	var t models.Textbook
	var storageType string
	var licenseType *string
	err := row.Scan(
		&t.ID, &t.Title, &t.Authors, &t.ISBN, &t.Year, &t.Pages, &t.Description, &t.Subject, &t.Course, &t.Department,
		&storageType, &licenseType, &t.CopyrightHolder, &t.HiddenAt, &t.DeletedAt, &t.CreatedAt, &t.UpdatedAt,
		&t.PDFS3Key, &t.SourceURL,
	)
	if err != nil {
		return nil, err
	}
	t.StorageType = models.TextbookStorageType(storageType)
	if licenseType != nil {
		lt := models.TextbookLicenseType(*licenseType)
		t.LicenseType = &lt
	}
	return &t, nil
}

// Create вставляет запись учебника и (в зависимости от storage_type) связанную
// строку в textbook_files (A, PDF в S3) либо textbook_links (B, внешняя
// ссылка) - одной транзакцией, чтобы не оставлять "голый" учебник без файла/
// ссылки при сбое посередине.
func (r *TextbookRepo) Create(ctx context.Context, t *models.Textbook) (*models.Textbook, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var id uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO textbooks (title, authors, isbn, year, pages, description, subject, course, department,
			storage_type, license_type, copyright_holder)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::textbook_storage_type, $11::textbook_license_type, $12)
		RETURNING id
	`, t.Title, t.Authors, t.ISBN, t.Year, t.Pages, t.Description, t.Subject, t.Course, t.Department,
		string(t.StorageType), licenseString(t.LicenseType), t.CopyrightHolder).Scan(&id)
	if err != nil {
		return nil, err
	}

	if t.StorageType == models.TextbookStorageA && t.PDFS3Key != nil {
		if _, err := tx.Exec(ctx, `INSERT INTO textbook_files (textbook_id, pdf_s3_key) VALUES ($1, $2)`, id, *t.PDFS3Key); err != nil {
			return nil, err
		}
	}
	if t.StorageType == models.TextbookStorageB && t.SourceURL != nil {
		if _, err := tx.Exec(ctx, `INSERT INTO textbook_links (textbook_id, source_url) VALUES ($1, $2)`, id, *t.SourceURL); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.FindByID(ctx, id)
}

func (r *TextbookRepo) FindByID(ctx context.Context, id uuid.UUID) (*models.Textbook, error) {
	query := `SELECT ` + textbookSelectColumns + `
		FROM textbooks t
		LEFT JOIN textbook_files tf ON tf.textbook_id = t.id
		LEFT JOIN textbook_links tl ON tl.textbook_id = t.id
		WHERE t.id = $1 AND t.deleted_at IS NULL`
	t, err := r.scanTextbook(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrTextbookNotFound
		}
		return nil, err
	}
	return t, nil
}

// AdminFindByID - то же самое, но без фильтра deleted_at IS NULL, чтобы
// админ мог видеть/редактировать скрытые и мягко удалённые учебники.
func (r *TextbookRepo) AdminFindByID(ctx context.Context, id uuid.UUID) (*models.Textbook, error) {
	query := `SELECT ` + textbookSelectColumns + `
		FROM textbooks t
		LEFT JOIN textbook_files tf ON tf.textbook_id = t.id
		LEFT JOIN textbook_links tl ON tl.textbook_id = t.id
		WHERE t.id = $1`
	t, err := r.scanTextbook(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrTextbookNotFound
		}
		return nil, err
	}
	return t, nil
}

func (r *TextbookRepo) Update(ctx context.Context, id uuid.UUID, t *models.Textbook) (*models.Textbook, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	cmd, err := tx.Exec(ctx, `
		UPDATE textbooks SET
			title = $2, authors = $3, isbn = $4, year = $5, pages = $6, description = $7,
			subject = $8, course = $9, department = $10, storage_type = $11::textbook_storage_type,
			license_type = $12::textbook_license_type, copyright_holder = $13, updated_at = now()
		WHERE id = $1
	`, id, t.Title, t.Authors, t.ISBN, t.Year, t.Pages, t.Description, t.Subject, t.Course, t.Department,
		string(t.StorageType), licenseString(t.LicenseType), t.CopyrightHolder)
	if err != nil {
		return nil, err
	}
	if cmd.RowsAffected() == 0 {
		return nil, models.ErrTextbookNotFound
	}

	if t.StorageType == models.TextbookStorageA && t.PDFS3Key != nil {
		if _, err := tx.Exec(ctx, `
			INSERT INTO textbook_files (textbook_id, pdf_s3_key) VALUES ($1, $2)
			ON CONFLICT (textbook_id) DO UPDATE SET pdf_s3_key = EXCLUDED.pdf_s3_key, updated_at = now()
		`, id, *t.PDFS3Key); err != nil {
			return nil, err
		}
	}
	if t.StorageType == models.TextbookStorageB && t.SourceURL != nil {
		if _, err := tx.Exec(ctx, `
			INSERT INTO textbook_links (textbook_id, source_url) VALUES ($1, $2)
			ON CONFLICT (textbook_id) DO UPDATE SET source_url = EXCLUDED.source_url, updated_at = now()
		`, id, *t.SourceURL); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.AdminFindByID(ctx, id)
}

func (r *TextbookRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	cmd, err := r.pool.Exec(ctx, `UPDATE textbooks SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return models.ErrTextbookNotFound
	}
	return nil
}

// List - публичный каталог: скрытые и удалённые учебники исключены.
func (r *TextbookRepo) List(ctx context.Context, f models.TextbookListFilter) ([]models.Textbook, int, error) {
	where := "t.deleted_at IS NULL AND t.hidden_at IS NULL"
	var args []any
	argN := 1
	if f.Query != nil {
		where += fmt.Sprintf(" AND t.title ILIKE $%d", argN)
		args = append(args, "%"+*f.Query+"%")
		argN++
	}
	if f.Subject != nil {
		where += fmt.Sprintf(" AND t.subject = $%d", argN)
		args = append(args, *f.Subject)
		argN++
	}
	if f.Course != nil {
		where += fmt.Sprintf(" AND t.course = $%d", argN)
		args = append(args, *f.Course)
		argN++
	}
	if f.Department != nil {
		where += fmt.Sprintf(" AND t.department = $%d", argN)
		args = append(args, *f.Department)
		argN++
	}
	if f.StorageType != nil {
		where += fmt.Sprintf(" AND t.storage_type = $%d::textbook_storage_type", argN)
		args = append(args, string(*f.StorageType))
		argN++
	}

	var total int
	if err := r.pool.QueryRow(ctx, "SELECT count(*) FROM textbooks t WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	orderBy := "t.created_at DESC"
	switch f.Sort {
	case "title_asc":
		orderBy = "t.title ASC"
	case "title_desc":
		orderBy = "t.title DESC"
	}

	limitArg, offsetArg := argN, argN+1
	query := fmt.Sprintf(`SELECT %s FROM textbooks t
		LEFT JOIN textbook_files tf ON tf.textbook_id = t.id
		LEFT JOIN textbook_links tl ON tl.textbook_id = t.id
		WHERE %s ORDER BY %s LIMIT $%d OFFSET $%d`, textbookSelectColumns, where, orderBy, limitArg, offsetArg)
	dataArgs := append(append([]any{}, args...), f.Limit, (f.Page-1)*f.Limit)

	rows, err := r.pool.Query(ctx, query, dataArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []models.Textbook
	for rows.Next() {
		t, err := r.scanTextbook(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *t)
	}
	return out, total, rows.Err()
}

// AdminList - каталог для админки. По умолчанию (include_hidden=false)
// ведёт себя как публичный список (без скрытых/удалённых); include_hidden=true
// показывает вообще всё, включая скрытые и мягко удалённые учебники - см.
// summary операции в openapi.yaml ("включая скрытые/удалённые").
func (r *TextbookRepo) AdminList(ctx context.Context, f models.AdminTextbookListFilter) ([]models.Textbook, int, error) {
	where := "t.deleted_at IS NULL AND t.hidden_at IS NULL"
	if f.IncludeHidden {
		where = "1=1"
	}
	var args []any
	argN := 1
	if f.StorageType != nil {
		where += fmt.Sprintf(" AND t.storage_type = $%d::textbook_storage_type", argN)
		args = append(args, string(*f.StorageType))
		argN++
	}

	var total int
	if err := r.pool.QueryRow(ctx, "SELECT count(*) FROM textbooks t WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limitArg, offsetArg := argN, argN+1
	query := fmt.Sprintf(`SELECT %s FROM textbooks t
		LEFT JOIN textbook_files tf ON tf.textbook_id = t.id
		LEFT JOIN textbook_links tl ON tl.textbook_id = t.id
		WHERE %s ORDER BY t.created_at DESC LIMIT $%d OFFSET $%d`, textbookSelectColumns, where, limitArg, offsetArg)
	dataArgs := append(append([]any{}, args...), f.Limit, (f.Page-1)*f.Limit)

	rows, err := r.pool.Query(ctx, query, dataArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []models.Textbook
	for rows.Next() {
		t, err := r.scanTextbook(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *t)
	}
	return out, total, rows.Err()
}

func licenseString(l *models.TextbookLicenseType) *string {
	if l == nil {
		return nil
	}
	s := string(*l)
	return &s
}
