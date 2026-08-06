package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/medflow/backend/internal/models"
)

func ptr[T any](v T) *T { return &v }

// deleteTestTextbook - textbook_files/textbook_links ссылаются на textbooks
// через FK без ON DELETE CASCADE, поэтому удалять нужно в этом порядке,
// иначе DELETE FROM textbooks падает с ошибкой FK-констрейнта.
func deleteTestTextbook(pool *pgxpool.Pool, id uuid.UUID) {
	ctx := context.Background()
	_, _ = pool.Exec(ctx, "DELETE FROM textbook_files WHERE textbook_id = $1", id)
	_, _ = pool.Exec(ctx, "DELETE FROM textbook_links WHERE textbook_id = $1", id)
	_, _ = pool.Exec(ctx, "DELETE FROM textbooks WHERE id = $1", id)
}

func createTestTextbookA(t *testing.T, pool *pgxpool.Pool, repo *TextbookRepo, title string) *models.Textbook {
	t.Helper()
	tb, err := repo.Create(context.Background(), &models.Textbook{
		Title:       title,
		StorageType: models.TextbookStorageA,
		PDFS3Key:    ptr("textbooks/" + uuid.NewString() + ".pdf"),
		LicenseType: ptr(models.LicenseCCBY),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	t.Cleanup(func() { deleteTestTextbook(pool, tb.ID) })
	return tb
}

func TestTextbookRepo_Create_And_FindByID_StorageA(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewTextbookRepo(pool)
	ctx := context.Background()

	created, err := repo.Create(ctx, &models.Textbook{
		Title:       "Анатомия человека",
		Authors:     ptr("Сапин М. Р."),
		Subject:     ptr("Анатомия"),
		Course:      ptr(1),
		StorageType: models.TextbookStorageA,
		PDFS3Key:    ptr("textbooks/anatomy.pdf"),
		LicenseType: ptr(models.LicenseCCBYNC),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	t.Cleanup(func() { deleteTestTextbook(pool, created.ID) })

	found, err := repo.FindByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if found.Title != "Анатомия человека" {
		t.Errorf("Title = %q, want %q", found.Title, "Анатомия человека")
	}
	if found.PDFS3Key == nil || *found.PDFS3Key != "textbooks/anatomy.pdf" {
		t.Errorf("PDFS3Key = %v, want textbooks/anatomy.pdf", found.PDFS3Key)
	}
	if found.SourceURL != nil {
		t.Errorf("SourceURL = %v, want nil for storage_type A", *found.SourceURL)
	}
	if found.StorageType != models.TextbookStorageA {
		t.Errorf("StorageType = %v, want A", found.StorageType)
	}
}

func TestTextbookRepo_Create_And_FindByID_StorageB(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewTextbookRepo(pool)
	ctx := context.Background()

	created, err := repo.Create(ctx, &models.Textbook{
		Title:       "Хирургические болезни",
		StorageType: models.TextbookStorageB,
		SourceURL:   ptr("https://example.org/book"),
		LicenseType: ptr(models.LicenseAllRightsReserved),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	t.Cleanup(func() { deleteTestTextbook(pool, created.ID) })

	found, err := repo.FindByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if found.SourceURL == nil || *found.SourceURL != "https://example.org/book" {
		t.Errorf("SourceURL = %v, want https://example.org/book", found.SourceURL)
	}
	if found.PDFS3Key != nil {
		t.Errorf("PDFS3Key = %v, want nil for storage_type B", *found.PDFS3Key)
	}
}

func TestTextbookRepo_FindByID_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewTextbookRepo(pool)

	_, err := repo.FindByID(context.Background(), uuid.New())
	if err != models.ErrTextbookNotFound {
		t.Fatalf("FindByID() error = %v, want ErrTextbookNotFound", err)
	}
}

func TestTextbookRepo_Update_ChangesFieldsAndPDFKey(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewTextbookRepo(pool)
	ctx := context.Background()
	created := createTestTextbookA(t, pool, repo, "old title")

	updated, err := repo.Update(ctx, created.ID, &models.Textbook{
		Title:       "new title",
		StorageType: models.TextbookStorageA,
		PDFS3Key:    ptr("textbooks/new-key.pdf"),
		LicenseType: ptr(models.LicenseCC0),
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Title != "new title" {
		t.Errorf("Title = %q, want new title", updated.Title)
	}
	if updated.PDFS3Key == nil || *updated.PDFS3Key != "textbooks/new-key.pdf" {
		t.Errorf("PDFS3Key = %v, want textbooks/new-key.pdf", updated.PDFS3Key)
	}
}

func TestTextbookRepo_Update_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewTextbookRepo(pool)

	_, err := repo.Update(context.Background(), uuid.New(), &models.Textbook{Title: "x", StorageType: models.TextbookStorageA})
	if err != models.ErrTextbookNotFound {
		t.Fatalf("Update() error = %v, want ErrTextbookNotFound", err)
	}
}

func TestTextbookRepo_SoftDelete_ExcludesFromFindAndList(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewTextbookRepo(pool)
	ctx := context.Background()
	created := createTestTextbookA(t, pool, repo, "to be deleted")

	if err := repo.SoftDelete(ctx, created.ID); err != nil {
		t.Fatalf("SoftDelete() error = %v", err)
	}
	if _, err := repo.FindByID(ctx, created.ID); err != models.ErrTextbookNotFound {
		t.Fatalf("FindByID() after delete error = %v, want ErrTextbookNotFound", err)
	}
	if err := repo.SoftDelete(ctx, created.ID); err != models.ErrTextbookNotFound {
		t.Fatalf("SoftDelete() twice error = %v, want ErrTextbookNotFound", err)
	}

	// AdminFindByID видит удалённый учебник - это отдельный доступ для админки.
	admin, err := repo.AdminFindByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("AdminFindByID() error = %v", err)
	}
	if admin.DeletedAt == nil {
		t.Errorf("AdminFindByID().DeletedAt = nil, want set")
	}
}

func TestTextbookRepo_List_FiltersByStorageTypeAndQuery(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewTextbookRepo(pool)
	ctx := context.Background()

	target := createTestTextbookA(t, pool, repo, "Уникальное название XYZ123")
	createTestTextbookA(t, pool, repo, "Другой учебник")

	q := "XYZ123"
	storageA := models.TextbookStorageA
	items, total, err := repo.List(ctx, models.TextbookListFilter{Query: &q, StorageType: &storageA, Page: 1, Limit: 20})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if total != 1 {
		t.Fatalf("total = %d, want 1", total)
	}
	if len(items) != 1 || items[0].ID != target.ID {
		t.Fatalf("List() = %v, want only %v", items, target.ID)
	}
}

func TestTextbookRepo_List_ExcludesHiddenAndDeleted(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewTextbookRepo(pool)
	ctx := context.Background()

	visible := createTestTextbookA(t, pool, repo, "видимый учебник для списка")
	hidden := createTestTextbookA(t, pool, repo, "скрытый учебник для списка")
	if _, err := pool.Exec(ctx, "UPDATE textbooks SET hidden_at = now() WHERE id = $1", hidden.ID); err != nil {
		t.Fatalf("failed to hide textbook: %v", err)
	}

	items, _, err := repo.List(ctx, models.TextbookListFilter{Page: 1, Limit: 100})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	var sawVisible, sawHidden bool
	for _, it := range items {
		if it.ID == visible.ID {
			sawVisible = true
		}
		if it.ID == hidden.ID {
			sawHidden = true
		}
	}
	if !sawVisible {
		t.Errorf("List() missing visible textbook %v", visible.ID)
	}
	if sawHidden {
		t.Errorf("List() should not include hidden textbook %v", hidden.ID)
	}
}

func TestTextbookRepo_AdminList_IncludeHiddenShowsAll(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewTextbookRepo(pool)
	ctx := context.Background()

	hidden := createTestTextbookA(t, pool, repo, "скрытый для админ-списка")
	if _, err := pool.Exec(ctx, "UPDATE textbooks SET hidden_at = now() WHERE id = $1", hidden.ID); err != nil {
		t.Fatalf("failed to hide textbook: %v", err)
	}

	items, _, err := repo.AdminList(ctx, models.AdminTextbookListFilter{IncludeHidden: true, Page: 1, Limit: 100})
	if err != nil {
		t.Fatalf("AdminList() error = %v", err)
	}
	var found bool
	for _, it := range items {
		if it.ID == hidden.ID {
			found = true
		}
	}
	if !found {
		t.Errorf("AdminList(IncludeHidden=true) missing hidden textbook %v", hidden.ID)
	}
}
