package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/medflow/backend/internal/dto"
	"github.com/medflow/backend/internal/models"
)

func setupTestLibraryService(textbookRepo *mockTextbookRepository, uploadRepo *mockUploadRepository, storage *mockObjectStorage) *LibraryService {
	if textbookRepo == nil {
		textbookRepo = &mockTextbookRepository{}
	}
	if uploadRepo == nil {
		uploadRepo = &mockUploadRepository{}
	}
	if storage == nil {
		storage = &mockObjectStorage{}
	}
	return NewLibraryService(textbookRepo, uploadRepo, storage, &mockAuditLogRepository{})
}

func TestLibraryService_GetTextbook_ReducesFieldsForStorageB(t *testing.T) {
	id := uuid.New()
	authors := "Some Author"
	textbookRepo := &mockTextbookRepository{
		findByIDFn: func(ctx context.Context, gotID uuid.UUID) (*models.Textbook, error) {
			return &models.Textbook{ID: id, Title: "B book", Authors: &authors, StorageType: models.TextbookStorageB}, nil
		},
	}
	svc := setupTestLibraryService(textbookRepo, nil, nil)

	out, err := svc.GetTextbook(context.Background(), id)
	if err != nil {
		t.Fatalf("GetTextbook() error = %v", err)
	}
	if out.Authors != nil {
		t.Errorf("Authors = %v, want nil (category B must not expose authors)", *out.Authors)
	}
	if out.Title != "B book" {
		t.Errorf("Title = %q, want %q", out.Title, "B book")
	}
}

func TestLibraryService_GetTextbook_NotFound(t *testing.T) {
	textbookRepo := &mockTextbookRepository{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.Textbook, error) {
			return nil, models.ErrTextbookNotFound
		},
	}
	svc := setupTestLibraryService(textbookRepo, nil, nil)

	_, err := svc.GetTextbook(context.Background(), uuid.New())
	if !errors.Is(err, ErrTextbookNotFound) {
		t.Fatalf("GetTextbook() error = %v, want ErrTextbookNotFound", err)
	}
}

func TestLibraryService_Download_Success(t *testing.T) {
	id := uuid.New()
	key := "textbooks/anatomy.pdf"
	textbookRepo := &mockTextbookRepository{
		findByIDFn: func(ctx context.Context, gotID uuid.UUID) (*models.Textbook, error) {
			return &models.Textbook{ID: id, StorageType: models.TextbookStorageA, PDFS3Key: &key}, nil
		},
	}
	storage := &mockObjectStorage{
		presignedGetURLFn: func(ctx context.Context, gotKey string, expiry time.Duration) (string, error) {
			if gotKey != key {
				t.Errorf("key = %q, want %q", gotKey, key)
			}
			return "https://s3.example.org/" + gotKey + "?sig=abc", nil
		},
	}
	svc := setupTestLibraryService(textbookRepo, nil, storage)

	url, err := svc.Download(context.Background(), id)
	if err != nil {
		t.Fatalf("Download() error = %v", err)
	}
	if url != "https://s3.example.org/"+key+"?sig=abc" {
		t.Errorf("url = %q, unexpected", url)
	}
}

func TestLibraryService_Download_ForbiddenForStorageB(t *testing.T) {
	id := uuid.New()
	textbookRepo := &mockTextbookRepository{
		findByIDFn: func(ctx context.Context, gotID uuid.UUID) (*models.Textbook, error) {
			return &models.Textbook{ID: id, StorageType: models.TextbookStorageB}, nil
		},
	}
	svc := setupTestLibraryService(textbookRepo, nil, nil)

	_, err := svc.Download(context.Background(), id)
	if !errors.Is(err, ErrNotDownloadable) {
		t.Fatalf("Download() error = %v, want ErrNotDownloadable", err)
	}
}

func TestLibraryService_Source_Success(t *testing.T) {
	id := uuid.New()
	sourceURL := "https://example.org/book"
	textbookRepo := &mockTextbookRepository{
		findByIDFn: func(ctx context.Context, gotID uuid.UUID) (*models.Textbook, error) {
			return &models.Textbook{ID: id, StorageType: models.TextbookStorageB, SourceURL: &sourceURL}, nil
		},
	}
	svc := setupTestLibraryService(textbookRepo, nil, nil)

	url, err := svc.Source(context.Background(), id)
	if err != nil {
		t.Fatalf("Source() error = %v", err)
	}
	if url != sourceURL {
		t.Errorf("url = %q, want %q", url, sourceURL)
	}
}

func TestLibraryService_Source_NotFoundForStorageA(t *testing.T) {
	id := uuid.New()
	textbookRepo := &mockTextbookRepository{
		findByIDFn: func(ctx context.Context, gotID uuid.UUID) (*models.Textbook, error) {
			return &models.Textbook{ID: id, StorageType: models.TextbookStorageA}, nil
		},
	}
	svc := setupTestLibraryService(textbookRepo, nil, nil)

	_, err := svc.Source(context.Background(), id)
	if !errors.Is(err, ErrNoSourceLink) {
		t.Fatalf("Source() error = %v, want ErrNoSourceLink", err)
	}
}

func TestLibraryService_AdminCreateTextbook_StorageA_ResolvesPDFUpload(t *testing.T) {
	uploadID := uuid.New()
	s3Key := "uploads/pdf/abc.pdf"
	uploadRepo := &mockUploadRepository{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.Upload, error) {
			if id != uploadID {
				t.Errorf("upload id = %v, want %v", id, uploadID)
			}
			return &models.Upload{ID: uploadID, UploadType: "pdf", S3Key: s3Key}, nil
		},
	}
	var createdS3Key *string
	textbookRepo := &mockTextbookRepository{
		createFn: func(ctx context.Context, t *models.Textbook) (*models.Textbook, error) {
			createdS3Key = t.PDFS3Key
			t.ID = uuid.New()
			return t, nil
		},
	}
	svc := setupTestLibraryService(textbookRepo, uploadRepo, nil)

	fileID := uploadID.String()
	_, err := svc.AdminCreateTextbook(context.Background(), uuid.New(), dto.CreateTextbookRequest{
		Title: "New book", StorageType: models.TextbookStorageA, PDFFileID: &fileID,
	})
	if err != nil {
		t.Fatalf("AdminCreateTextbook() error = %v", err)
	}
	if createdS3Key == nil || *createdS3Key != s3Key {
		t.Errorf("PDFS3Key = %v, want %v", createdS3Key, s3Key)
	}
}

func TestLibraryService_AdminCreateTextbook_StorageA_RequiresPDFFileID(t *testing.T) {
	svc := setupTestLibraryService(nil, nil, nil)

	_, err := svc.AdminCreateTextbook(context.Background(), uuid.New(), dto.CreateTextbookRequest{
		Title: "New book", StorageType: models.TextbookStorageA,
	})
	if !errors.Is(err, ErrPDFFileRequired) {
		t.Fatalf("AdminCreateTextbook() error = %v, want ErrPDFFileRequired", err)
	}
}

func TestLibraryService_AdminCreateTextbook_StorageA_RejectsNonPDFUpload(t *testing.T) {
	uploadID := uuid.New()
	uploadRepo := &mockUploadRepository{
		findByIDFn: func(ctx context.Context, id uuid.UUID) (*models.Upload, error) {
			return &models.Upload{ID: uploadID, UploadType: "image", S3Key: "uploads/image/x.png"}, nil
		},
	}
	svc := setupTestLibraryService(nil, uploadRepo, nil)

	fileID := uploadID.String()
	_, err := svc.AdminCreateTextbook(context.Background(), uuid.New(), dto.CreateTextbookRequest{
		Title: "New book", StorageType: models.TextbookStorageA, PDFFileID: &fileID,
	})
	if !errors.Is(err, ErrPDFUploadWrongType) {
		t.Fatalf("AdminCreateTextbook() error = %v, want ErrPDFUploadWrongType", err)
	}
}

func TestLibraryService_AdminCreateTextbook_StorageB_RequiresSourceURL(t *testing.T) {
	svc := setupTestLibraryService(nil, nil, nil)

	_, err := svc.AdminCreateTextbook(context.Background(), uuid.New(), dto.CreateTextbookRequest{
		Title: "New book", StorageType: models.TextbookStorageB,
	})
	if !errors.Is(err, ErrSourceURLRequired) {
		t.Fatalf("AdminCreateTextbook() error = %v, want ErrSourceURLRequired", err)
	}
}

func TestLibraryService_AdminDeleteTextbook_NotFound(t *testing.T) {
	textbookRepo := &mockTextbookRepository{
		softDeleteFn: func(ctx context.Context, id uuid.UUID) error {
			return models.ErrTextbookNotFound
		},
	}
	svc := setupTestLibraryService(textbookRepo, nil, nil)

	err := svc.AdminDeleteTextbook(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, ErrTextbookNotFound) {
		t.Fatalf("AdminDeleteTextbook() error = %v, want ErrTextbookNotFound", err)
	}
}

func TestLibraryService_AdminCreateTextbook_WritesAuditLog(t *testing.T) {
	actorID := uuid.New()
	textbookRepo := &mockTextbookRepository{
		createFn: func(ctx context.Context, t *models.Textbook) (*models.Textbook, error) {
			t.ID = uuid.New()
			return t, nil
		},
	}
	var auditEntry *models.AuditLog
	auditRepo := &mockAuditLogRepository{
		createFn: func(ctx context.Context, entry *models.AuditLog) error { auditEntry = entry; return nil },
	}
	svc := NewLibraryService(textbookRepo, &mockUploadRepository{}, &mockObjectStorage{}, auditRepo)

	sourceURL := "https://example.org/book"
	_, err := svc.AdminCreateTextbook(context.Background(), actorID, dto.CreateTextbookRequest{
		Title: "Учебник", StorageType: models.TextbookStorageB, SourceURL: &sourceURL,
	})
	if err != nil {
		t.Fatalf("AdminCreateTextbook() error = %v", err)
	}
	if auditEntry == nil || auditEntry.Action != models.AuditTextbookCreate || auditEntry.ActorID != actorID {
		t.Fatalf("audit log entry = %+v, want AuditTextbookCreate by %v", auditEntry, actorID)
	}
}
