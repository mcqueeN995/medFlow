package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/google/uuid"
	"github.com/medflow/backend/internal/models"
)

func setupTestUploadService(uploadRepo *mockUploadRepository, storage *mockObjectStorage) *UploadService {
	if uploadRepo == nil {
		uploadRepo = &mockUploadRepository{}
	}
	if storage == nil {
		storage = &mockObjectStorage{}
	}
	return NewUploadService(uploadRepo, storage)
}

func TestUploadService_Upload_PDF_SetsExpiry(t *testing.T) {
	uploaderID := uuid.New()
	var savedUpload *models.Upload
	uploadRepo := &mockUploadRepository{
		createFn: func(ctx context.Context, u *models.Upload) (*models.Upload, error) {
			u.ID = uuid.New()
			savedUpload = u
			return u, nil
		},
	}
	var putKey, putContentType string
	storage := &mockObjectStorage{
		putFn: func(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
			putKey, putContentType = key, contentType
			return nil
		},
	}
	svc := setupTestUploadService(uploadRepo, storage)

	res, err := svc.Upload(context.Background(), uploaderID, "pdf", "conspect.pdf", "application/pdf", 1024, bytes.NewReader([]byte("%PDF-1.4")))
	if err != nil {
		t.Fatalf("Upload() error = %v", err)
	}
	if res.MimeType != "application/pdf" || res.SizeBytes != 1024 {
		t.Errorf("MimeType/SizeBytes = %q/%d, want application/pdf/1024", res.MimeType, res.SizeBytes)
	}
	if res.ExpiresAt == nil {
		t.Fatal("ExpiresAt = nil, want set for pdf uploads (temporary personal material)")
	}
	if savedUpload == nil || savedUpload.UploaderID != uploaderID {
		t.Fatalf("uploadRepo.Create was not called with expected uploader")
	}
	if putContentType != "application/pdf" {
		t.Errorf("storage.Put contentType = %q, want application/pdf", putContentType)
	}
	if putKey == "" {
		t.Errorf("storage.Put key = empty, want a generated key")
	}
}

func TestUploadService_Upload_RejectsWrongMimeForType(t *testing.T) {
	svc := setupTestUploadService(nil, nil)

	_, err := svc.Upload(context.Background(), uuid.New(), "pdf", "photo.jpg", "image/jpeg", 100, bytes.NewReader(nil))
	if !errors.Is(err, ErrInvalidFileType) {
		t.Fatalf("Upload() error = %v, want ErrInvalidFileType", err)
	}
}

func TestUploadService_Upload_RejectsUnknownType(t *testing.T) {
	svc := setupTestUploadService(nil, nil)

	_, err := svc.Upload(context.Background(), uuid.New(), "video", "clip.mp4", "video/mp4", 100, bytes.NewReader(nil))
	if !errors.Is(err, ErrInvalidUploadType) {
		t.Fatalf("Upload() error = %v, want ErrInvalidUploadType", err)
	}
}

func TestUploadService_Upload_AvatarHasNoExpiry(t *testing.T) {
	uploadRepo := &mockUploadRepository{
		createFn: func(ctx context.Context, u *models.Upload) (*models.Upload, error) {
			u.ID = uuid.New()
			return u, nil
		},
	}
	svc := setupTestUploadService(uploadRepo, nil)

	res, err := svc.Upload(context.Background(), uuid.New(), "avatar", "me.png", "image/png", 500, bytes.NewReader(nil))
	if err != nil {
		t.Fatalf("Upload() error = %v", err)
	}
	if res.ExpiresAt != nil {
		t.Errorf("ExpiresAt = %v, want nil for avatar uploads", *res.ExpiresAt)
	}
}
