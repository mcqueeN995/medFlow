package dto

import "time"

type UploadResponse struct {
	FileID    string     `json:"file_id"`
	URL       string     `json:"url"`
	SizeBytes int64      `json:"size_bytes"`
	MimeType  string     `json:"mime_type"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}
