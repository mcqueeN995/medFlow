package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/medflow/backend/internal/dto"
	"github.com/medflow/backend/internal/models"
)

var (
	ErrInvalidUploadType = errors.New("invalid upload type")
	ErrInvalidFileType   = errors.New("invalid file type for this upload type")
)

// pdfUploadTTL - срок жизни временной PDF-загрузки пользователя (личные
// материалы для ИИ-карточек, см. LibraryUploadPage на фронтенде). Фоновой
// очистки истёкших файлов пока нет - файл физически остаётся в S3 до
// появления воркер-задачи очистки в Этапе 3 (это намеренный видимый пробел,
// не забытая часть, см. models.Upload).
const pdfUploadTTL = 24 * time.Hour

// urlPresignTTL - на сколько живёт presigned-ссылка, возвращаемая сразу
// после загрузки в UploadResponse.url.
const urlPresignTTL = time.Hour

type UploadService struct {
	uploadRepo UploadRepository
	storage    ObjectStorage
}

func NewUploadService(uploadRepo UploadRepository, storage ObjectStorage) *UploadService {
	return &UploadService{uploadRepo: uploadRepo, storage: storage}
}

func (s *UploadService) Upload(ctx context.Context, uploaderID uuid.UUID, uploadType, filename, contentType string, size int64, reader io.Reader) (*dto.UploadResponse, error) {
	if err := validateUploadType(uploadType, contentType); err != nil {
		return nil, err
	}

	key := fmt.Sprintf("uploads/%s/%s%s", uploadType, uuid.New(), filepath.Ext(filename))
	if err := s.storage.Put(ctx, key, reader, size, contentType); err != nil {
		return nil, err
	}

	u := &models.Upload{
		UploaderID: uploaderID,
		UploadType: uploadType,
		S3Key:      key,
		MimeType:   contentType,
		SizeBytes:  size,
	}
	if uploadType == "pdf" {
		expiresAt := time.Now().Add(pdfUploadTTL)
		u.ExpiresAt = &expiresAt
	}

	created, err := s.uploadRepo.Create(ctx, u)
	if err != nil {
		return nil, err
	}

	url, err := s.storage.PresignedGetURL(ctx, key, urlPresignTTL)
	if err != nil {
		return nil, err
	}

	return &dto.UploadResponse{
		FileID:    created.ID.String(),
		URL:       url,
		SizeBytes: created.SizeBytes,
		MimeType:  created.MimeType,
		ExpiresAt: created.ExpiresAt,
	}, nil
}

func validateUploadType(uploadType, contentType string) error {
	switch uploadType {
	case "pdf":
		if contentType != "application/pdf" {
			return ErrInvalidFileType
		}
	case "image", "avatar":
		if !strings.HasPrefix(contentType, "image/") {
			return ErrInvalidFileType
		}
	default:
		return ErrInvalidUploadType
	}
	return nil
}
