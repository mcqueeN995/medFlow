package dto

import (
	"time"

	"github.com/medflow/backend/internal/models"
)

// Textbook - публичное представление учебника. Для storage_type B контракт
// (см. openapi.yaml, TextbookListItem.description) отдаёт только
// title/storage_type/license_type/created_at: medFlow не имеет прав
// republish-ить сведения о произведении сверх ссылки на легальный источник.
type Textbook struct {
	ID              string                      `json:"id"`
	Title           string                      `json:"title"`
	Authors         *string                     `json:"authors,omitempty"`
	ISBN            *string                     `json:"isbn,omitempty"`
	Year            *int                        `json:"year,omitempty"`
	Pages           *int                        `json:"pages,omitempty"`
	Description     *string                     `json:"description,omitempty"`
	Subject         *string                     `json:"subject,omitempty"`
	Course          *int                        `json:"course,omitempty"`
	Department      *string                     `json:"department,omitempty"`
	StorageType     models.TextbookStorageType  `json:"storage_type"`
	LicenseType     *models.TextbookLicenseType `json:"license_type,omitempty"`
	CopyrightHolder *string                     `json:"copyright_holder,omitempty"`
	HiddenAt        *time.Time                  `json:"hidden_at,omitempty"`
	DeletedAt       *time.Time                  `json:"deleted_at,omitempty"`
	CreatedAt       time.Time                   `json:"created_at"`
}

func ToTextbook(t *models.Textbook) Textbook {
	if t.StorageType == models.TextbookStorageB {
		return Textbook{ID: t.ID.String(), Title: t.Title, StorageType: t.StorageType, LicenseType: t.LicenseType, CreatedAt: t.CreatedAt}
	}
	return Textbook{
		ID: t.ID.String(), Title: t.Title, Authors: t.Authors, ISBN: t.ISBN, Year: t.Year, Pages: t.Pages,
		Description: t.Description, Subject: t.Subject, Course: t.Course, Department: t.Department,
		StorageType: t.StorageType, LicenseType: t.LicenseType, CopyrightHolder: t.CopyrightHolder,
		HiddenAt: t.HiddenAt, DeletedAt: t.DeletedAt, CreatedAt: t.CreatedAt,
	}
}

// ToAdminTextbook - без B-редукции: в админке нужны все поля независимо от
// storage_type, иначе учебник категории B нельзя было бы редактировать.
func ToAdminTextbook(t *models.Textbook) Textbook {
	return Textbook{
		ID: t.ID.String(), Title: t.Title, Authors: t.Authors, ISBN: t.ISBN, Year: t.Year, Pages: t.Pages,
		Description: t.Description, Subject: t.Subject, Course: t.Course, Department: t.Department,
		StorageType: t.StorageType, LicenseType: t.LicenseType, CopyrightHolder: t.CopyrightHolder,
		HiddenAt: t.HiddenAt, DeletedAt: t.DeletedAt, CreatedAt: t.CreatedAt,
	}
}

type TextbookListItem struct {
	ID          string                      `json:"id"`
	Title       string                      `json:"title"`
	Authors     *string                     `json:"authors,omitempty"`
	ISBN        *string                     `json:"isbn,omitempty"`
	Year        *int                        `json:"year,omitempty"`
	Subject     *string                     `json:"subject,omitempty"`
	Course      *int                        `json:"course,omitempty"`
	StorageType models.TextbookStorageType  `json:"storage_type"`
	LicenseType *models.TextbookLicenseType `json:"license_type,omitempty"`
	CreatedAt   time.Time                   `json:"created_at"`
}

func ToTextbookListItem(t *models.Textbook) TextbookListItem {
	if t.StorageType == models.TextbookStorageB {
		return TextbookListItem{ID: t.ID.String(), Title: t.Title, StorageType: t.StorageType, LicenseType: t.LicenseType, CreatedAt: t.CreatedAt}
	}
	return TextbookListItem{
		ID: t.ID.String(), Title: t.Title, Authors: t.Authors, ISBN: t.ISBN, Year: t.Year, Subject: t.Subject,
		Course: t.Course, StorageType: t.StorageType, LicenseType: t.LicenseType, CreatedAt: t.CreatedAt,
	}
}

type CreateTextbookRequest struct {
	Title           string                      `json:"title" binding:"required,max=500"`
	Authors         *string                     `json:"authors,omitempty"`
	ISBN            *string                     `json:"isbn,omitempty" binding:"omitempty,max=20"`
	Year            *int                        `json:"year,omitempty"`
	Pages           *int                        `json:"pages,omitempty"`
	Description     *string                     `json:"description,omitempty"`
	Subject         *string                     `json:"subject,omitempty" binding:"omitempty,max=100"`
	Course          *int                        `json:"course,omitempty"`
	Department      *string                     `json:"department,omitempty" binding:"omitempty,max=100"`
	StorageType     models.TextbookStorageType  `json:"storage_type" binding:"required,oneof=A B"`
	LicenseType     *models.TextbookLicenseType `json:"license_type,omitempty"`
	CopyrightHolder *string                     `json:"copyright_holder,omitempty" binding:"omitempty,max=255"`
	PDFFileID       *string                     `json:"pdf_file_id,omitempty" binding:"omitempty,uuid"`
	SourceURL       *string                     `json:"source_url,omitempty" binding:"omitempty,uri"`
}

type UpdateTextbookRequest struct {
	Title           *string                     `json:"title,omitempty" binding:"omitempty,max=500"`
	Authors         *string                     `json:"authors,omitempty"`
	ISBN            *string                     `json:"isbn,omitempty" binding:"omitempty,max=20"`
	Year            *int                        `json:"year,omitempty"`
	Pages           *int                        `json:"pages,omitempty"`
	Description     *string                     `json:"description,omitempty"`
	Subject         *string                     `json:"subject,omitempty" binding:"omitempty,max=100"`
	Course          *int                        `json:"course,omitempty"`
	Department      *string                     `json:"department,omitempty" binding:"omitempty,max=100"`
	StorageType     *models.TextbookStorageType `json:"storage_type,omitempty" binding:"omitempty,oneof=A B"`
	LicenseType     *models.TextbookLicenseType `json:"license_type,omitempty"`
	CopyrightHolder *string                     `json:"copyright_holder,omitempty" binding:"omitempty,max=255"`
	PDFFileID       *string                     `json:"pdf_file_id,omitempty" binding:"omitempty,uuid"`
	SourceURL       *string                     `json:"source_url,omitempty" binding:"omitempty,uri"`
}
