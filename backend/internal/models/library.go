package models

import (
	"time"

	"github.com/google/uuid"
)

type TextbookStorageType string

const (
	TextbookStorageA TextbookStorageType = "A" // хранится в S3, скачивание PDF
	TextbookStorageB TextbookStorageType = "B" // внешняя ссылка на источник
)

type TextbookLicenseType string

const (
	LicenseCCBY              TextbookLicenseType = "cc_by"
	LicenseCCBYNC            TextbookLicenseType = "cc_by_nc"
	LicenseCCBYSA            TextbookLicenseType = "cc_by_sa"
	LicenseCC0               TextbookLicenseType = "cc0"
	LicensePublicDomain      TextbookLicenseType = "public_domain"
	LicenseAllRightsReserved TextbookLicenseType = "all_rights_reserved"
	LicenseCustom            TextbookLicenseType = "custom"
)

type Textbook struct {
	ID              uuid.UUID
	Title           string
	Authors         *string
	ISBN            *string
	Year            *int
	Pages           *int
	Description     *string
	Subject         *string
	Course          *int
	Department      *string
	StorageType     TextbookStorageType
	LicenseType     *TextbookLicenseType
	CopyrightHolder *string
	PDFS3Key        *string // заполнен только для storage_type A
	SourceURL       *string // заполнен только для storage_type B
	HiddenAt        *time.Time
	DeletedAt       *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (t *Textbook) IsHidden() bool {
	return t.HiddenAt != nil
}

type TextbookListFilter struct {
	Query       *string
	Subject     *string
	Course      *int
	Department  *string
	StorageType *TextbookStorageType
	Sort        string
	Page, Limit int
}

type AdminTextbookListFilter struct {
	StorageType   *TextbookStorageType
	IncludeHidden bool
	Page, Limit   int
}

// Upload - метаданные файла, загруженного через POST /upload. ExpiresAt задаёт
// момент, после которого файл считается временным материалом пользователя
// (см. use-case "личные материалы для ИИ-карточек" в LibraryUploadPage) -
// фоновой очистки истёкших загрузок пока нет, это намеренный видимый пробел
// до появления воркер-задачи в Этапе 3 (ИИ-карточки).
type Upload struct {
	ID         uuid.UUID
	UploaderID uuid.UUID
	UploadType string
	S3Key      string
	MimeType   string
	SizeBytes  int64
	ExpiresAt  *time.Time
	CreatedAt  time.Time
}
