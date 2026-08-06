package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/medflow/backend/internal/dto"
	"github.com/medflow/backend/internal/models"
)

var (
	ErrTextbookNotFound   = errors.New("textbook not found")
	ErrNotDownloadable    = errors.New("textbook is not available for download")
	ErrNoSourceLink       = errors.New("textbook has no source link")
	ErrPDFFileRequired    = errors.New("pdf_file_id is required for storage_type A")
	ErrSourceURLRequired  = errors.New("source_url is required for storage_type B")
	ErrPDFUploadNotFound  = errors.New("uploaded pdf not found or expired")
	ErrPDFUploadWrongType = errors.New("referenced upload is not a pdf")
)

// downloadURLTTL - на сколько живёт presigned-ссылка на скачивание PDF.
const downloadURLTTL = 15 * time.Minute

type LibraryService struct {
	textbookRepo TextbookRepository
	uploadRepo   UploadRepository
	storage      ObjectStorage
	auditLogRepo AuditLogRepository
}

func NewLibraryService(textbookRepo TextbookRepository, uploadRepo UploadRepository, storage ObjectStorage, auditLogRepo AuditLogRepository) *LibraryService {
	return &LibraryService{textbookRepo: textbookRepo, uploadRepo: uploadRepo, storage: storage, auditLogRepo: auditLogRepo}
}

func (s *LibraryService) ListTextbooks(ctx context.Context, f models.TextbookListFilter) (*dto.Pagination, []dto.TextbookListItem, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Limit < 1 || f.Limit > 100 {
		f.Limit = 20
	}

	textbooks, total, err := s.textbookRepo.List(ctx, f)
	if err != nil {
		return nil, nil, err
	}

	items := make([]dto.TextbookListItem, len(textbooks))
	for i := range textbooks {
		items[i] = dto.ToTextbookListItem(&textbooks[i])
	}

	pagination := dto.NewPagination(f.Page, f.Limit, total)
	return &pagination, items, nil
}

func (s *LibraryService) GetTextbook(ctx context.Context, id uuid.UUID) (*dto.Textbook, error) {
	t, err := s.textbookRepo.FindByID(ctx, id)
	if err != nil {
		return nil, s.mapErr(err)
	}
	out := dto.ToTextbook(t)
	return &out, nil
}

// Download возвращает presigned-ссылку на PDF в S3. Доступно только для
// storage_type A - для B у medFlow нет прав раздавать сам файл, см. Source.
func (s *LibraryService) Download(ctx context.Context, id uuid.UUID) (string, error) {
	t, err := s.textbookRepo.FindByID(ctx, id)
	if err != nil {
		return "", s.mapErr(err)
	}
	if t.StorageType != models.TextbookStorageA || t.PDFS3Key == nil {
		return "", ErrNotDownloadable
	}
	return s.storage.PresignedGetURL(ctx, *t.PDFS3Key, downloadURLTTL)
}

// Source возвращает внешнюю ссылку на источник. Доступно только для
// storage_type B.
func (s *LibraryService) Source(ctx context.Context, id uuid.UUID) (string, error) {
	t, err := s.textbookRepo.FindByID(ctx, id)
	if err != nil {
		return "", s.mapErr(err)
	}
	if t.StorageType != models.TextbookStorageB || t.SourceURL == nil {
		return "", ErrNoSourceLink
	}
	return *t.SourceURL, nil
}

// ==================== ADMIN ====================

func (s *LibraryService) AdminListTextbooks(ctx context.Context, f models.AdminTextbookListFilter) (*dto.Pagination, []dto.Textbook, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Limit < 1 || f.Limit > 100 {
		f.Limit = 20
	}

	textbooks, total, err := s.textbookRepo.AdminList(ctx, f)
	if err != nil {
		return nil, nil, err
	}

	items := make([]dto.Textbook, len(textbooks))
	for i := range textbooks {
		items[i] = dto.ToAdminTextbook(&textbooks[i])
	}

	pagination := dto.NewPagination(f.Page, f.Limit, total)
	return &pagination, items, nil
}

func (s *LibraryService) AdminCreateTextbook(ctx context.Context, actorID uuid.UUID, req dto.CreateTextbookRequest) (*dto.Textbook, error) {
	t := &models.Textbook{
		Title: req.Title, Authors: req.Authors, ISBN: req.ISBN, Year: req.Year, Pages: req.Pages,
		Description: req.Description, Subject: req.Subject, Course: req.Course, Department: req.Department,
		StorageType: req.StorageType, LicenseType: req.LicenseType, CopyrightHolder: req.CopyrightHolder,
	}

	switch req.StorageType {
	case models.TextbookStorageA:
		s3Key, err := s.resolvePDFUpload(ctx, req.PDFFileID)
		if err != nil {
			return nil, err
		}
		t.PDFS3Key = &s3Key
	case models.TextbookStorageB:
		if req.SourceURL == nil || *req.SourceURL == "" {
			return nil, ErrSourceURLRequired
		}
		t.SourceURL = req.SourceURL
	}

	created, err := s.textbookRepo.Create(ctx, t)
	if err != nil {
		return nil, err
	}
	s.writeAudit(ctx, actorID, models.AuditTextbookCreate, created.ID, nil)
	out := dto.ToAdminTextbook(created)
	return &out, nil
}

func (s *LibraryService) AdminUpdateTextbook(ctx context.Context, actorID, id uuid.UUID, req dto.UpdateTextbookRequest) (*dto.Textbook, error) {
	existing, err := s.textbookRepo.AdminFindByID(ctx, id)
	if err != nil {
		return nil, s.mapErr(err)
	}

	if req.Title != nil {
		existing.Title = *req.Title
	}
	if req.Authors != nil {
		existing.Authors = req.Authors
	}
	if req.ISBN != nil {
		existing.ISBN = req.ISBN
	}
	if req.Year != nil {
		existing.Year = req.Year
	}
	if req.Pages != nil {
		existing.Pages = req.Pages
	}
	if req.Description != nil {
		existing.Description = req.Description
	}
	if req.Subject != nil {
		existing.Subject = req.Subject
	}
	if req.Course != nil {
		existing.Course = req.Course
	}
	if req.Department != nil {
		existing.Department = req.Department
	}
	if req.StorageType != nil {
		existing.StorageType = *req.StorageType
	}
	if req.LicenseType != nil {
		existing.LicenseType = req.LicenseType
	}
	if req.CopyrightHolder != nil {
		existing.CopyrightHolder = req.CopyrightHolder
	}
	if req.SourceURL != nil {
		existing.SourceURL = req.SourceURL
	}
	if req.PDFFileID != nil {
		s3Key, err := s.resolvePDFUpload(ctx, req.PDFFileID)
		if err != nil {
			return nil, err
		}
		existing.PDFS3Key = &s3Key
	}

	updated, err := s.textbookRepo.Update(ctx, id, existing)
	if err != nil {
		return nil, s.mapErr(err)
	}
	s.writeAudit(ctx, actorID, models.AuditTextbookUpdate, id, nil)
	out := dto.ToAdminTextbook(updated)
	return &out, nil
}

func (s *LibraryService) AdminDeleteTextbook(ctx context.Context, actorID, id uuid.UUID) error {
	if err := s.mapErr(s.textbookRepo.SoftDelete(ctx, id)); err != nil {
		return err
	}
	s.writeAudit(ctx, actorID, models.AuditTextbookDelete, id, nil)
	return nil
}

// writeAudit - лучшее старание: сбой записи аудит-лога не должен проваливать
// уже выполненное действие над каталогом.
func (s *LibraryService) writeAudit(ctx context.Context, actorID uuid.UUID, action models.AuditAction, targetID uuid.UUID, reason *string) {
	targetType := "textbook"
	_ = s.auditLogRepo.Create(ctx, &models.AuditLog{ActorID: actorID, Action: action, TargetType: &targetType, TargetID: &targetID, Reason: reason})
}

// resolvePDFUpload проверяет, что pdf_file_id ссылается на существующую PDF-
// загрузку (см. POST /upload), и возвращает её S3-ключ.
func (s *LibraryService) resolvePDFUpload(ctx context.Context, fileID *string) (string, error) {
	if fileID == nil || *fileID == "" {
		return "", ErrPDFFileRequired
	}
	id, err := uuid.Parse(*fileID)
	if err != nil {
		return "", ErrPDFUploadNotFound
	}
	upload, err := s.uploadRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, models.ErrUploadNotFound) {
			return "", ErrPDFUploadNotFound
		}
		return "", err
	}
	if upload.UploadType != "pdf" {
		return "", ErrPDFUploadWrongType
	}
	return upload.S3Key, nil
}

func (s *LibraryService) mapErr(err error) error {
	if errors.Is(err, models.ErrTextbookNotFound) {
		return ErrTextbookNotFound
	}
	return err
}
