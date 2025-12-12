package models

import (
	"time"
)

// UploadRecord represents an upload record in the database
type UploadRecord struct {
	ID           int64     `db:"id"`
	Filename     string    `db:"filename"`
	OriginalName string    `db:"original_name"`
	FileSize     int64     `db:"file_size"`
	ContentType  string    `db:"content_type"`
	S3Key        string    `db:"s3_key"`
	S3URL        string    `db:"s3_url"`
	Status       string    `db:"status"` // "pending", "completed", "failed"
	Error        *string   `db:"error"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

// UploadStatus represents upload status
type UploadStatus string

const (
	UploadStatusPending   UploadStatus = "pending"
	UploadStatusCompleted UploadStatus = "completed"
	UploadStatusFailed    UploadStatus = "failed"
)
