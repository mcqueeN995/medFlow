package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/medflow/backend/internal/models"
)

func TestUploadRepo_Create_And_FindByID(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewUploadRepo(pool)
	ctx := context.Background()
	uploader := createTestForumUser(t, pool)
	expiresAt := time.Now().Add(24 * time.Hour).Truncate(time.Second)

	created, err := repo.Create(ctx, &models.Upload{
		UploaderID: uploader.ID,
		UploadType: "pdf",
		S3Key:      "uploads/pdf/" + uuid.NewString() + ".pdf",
		MimeType:   "application/pdf",
		SizeBytes:  12345,
		ExpiresAt:  &expiresAt,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), "DELETE FROM uploads WHERE id = $1", created.ID) })

	if created.ID == uuid.Nil {
		t.Fatal("Create() did not set ID")
	}

	found, err := repo.FindByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("FindByID() error = %v", err)
	}
	if found.UploaderID != uploader.ID {
		t.Errorf("UploaderID = %v, want %v", found.UploaderID, uploader.ID)
	}
	if found.MimeType != "application/pdf" || found.SizeBytes != 12345 {
		t.Errorf("MimeType/SizeBytes = %q/%d, want application/pdf/12345", found.MimeType, found.SizeBytes)
	}
	if found.ExpiresAt == nil || !found.ExpiresAt.Equal(expiresAt) {
		t.Errorf("ExpiresAt = %v, want %v", found.ExpiresAt, expiresAt)
	}
}

func TestUploadRepo_FindByID_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	repo := NewUploadRepo(pool)

	_, err := repo.FindByID(context.Background(), uuid.New())
	if err != models.ErrUploadNotFound {
		t.Fatalf("FindByID() error = %v, want ErrUploadNotFound", err)
	}
}
