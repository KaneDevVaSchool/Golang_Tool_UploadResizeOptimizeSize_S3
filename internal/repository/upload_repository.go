package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"s3-upload-tool/internal/database"
	"s3-upload-tool/internal/models"
)

// UploadRepository xử lý các thao tác với upload records
type UploadRepository interface {
	CreateUpload(ctx context.Context, tx *database.Tx, record *models.UploadRecord) (*models.UploadRecord, error)
	UpdateUploadStatus(ctx context.Context, tx *database.Tx, id int64, status models.UploadStatus, s3Key, s3URL string, err error) error
	GetUploadByID(ctx context.Context, db *database.DB, id int64) (*models.UploadRecord, error)
	GetUploadsByStatus(ctx context.Context, db *database.DB, status models.UploadStatus, limit int) ([]*models.UploadRecord, error)
	Close() error
}

type uploadRepository struct {
	db              *database.DB
	getByIDStmt     *sql.Stmt
	getByStatusStmt *sql.Stmt
}

// NewUploadRepository tạo upload repository với prepared statements
func NewUploadRepository(db *database.DB) (UploadRepository, error) {
	// ? Chỉ prepare statements cho non-transaction queries
	// Transaction queries không thể dùng prepared statements vì chạy trong transaction context
	getByIDStmt, err := db.Prepare(`
		SELECT id, filename, original_name, file_size, content_type, s3_key, s3_url, status, error, created_at, updated_at
		FROM uploads
		WHERE id = $1
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare getByID statement: %w", err)
	}

	getByStatusStmt, err := db.Prepare(`
		SELECT id, filename, original_name, file_size, content_type, s3_key, s3_url, status, error, created_at, updated_at
		FROM uploads
		WHERE status = $1
		ORDER BY created_at DESC
		LIMIT $2
	`)
	if err != nil {
		getByIDStmt.Close()
		return nil, fmt.Errorf("failed to prepare getByStatus statement: %w", err)
	}

	return &uploadRepository{
		db:              db,
		getByIDStmt:     getByIDStmt,
		getByStatusStmt: getByStatusStmt,
	}, nil
}

// Close đóng prepared statements (nên gọi khi shutdown)
func (r *uploadRepository) Close() error {
	var errs []error
	if r.getByIDStmt != nil {
		if err := r.getByIDStmt.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if r.getByStatusStmt != nil {
		if err := r.getByStatusStmt.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("errors closing statements: %v", errs)
	}
	return nil
}

// CreateUpload creates a new upload record in a transaction
func (r *uploadRepository) CreateUpload(ctx context.Context, tx *database.Tx, record *models.UploadRecord) (*models.UploadRecord, error) {
	query := `
		INSERT INTO uploads (filename, original_name, file_size, content_type, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`

	now := time.Now()
	record.CreatedAt = now
	record.UpdatedAt = now
	record.Status = string(models.UploadStatusPending)

	var id int64
	var createdAt, updatedAt time.Time
	err := tx.QueryRowContext(ctx, query,
		record.Filename,
		record.OriginalName,
		record.FileSize,
		record.ContentType,
		record.Status,
		record.CreatedAt,
		record.UpdatedAt,
	).Scan(&id, &createdAt, &updatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create upload record: %w", err)
	}

	record.ID = id
	record.CreatedAt = createdAt
	record.UpdatedAt = updatedAt

	return record, nil
}

// UpdateUploadStatus cập nhật status và thông tin S3 trong transaction
func (r *uploadRepository) UpdateUploadStatus(ctx context.Context, tx *database.Tx, id int64, status models.UploadStatus, s3Key, s3URL string, uploadErr error) error {
	query := `
		UPDATE uploads
		SET status = $1, s3_key = $2, s3_url = $3, error = $4, updated_at = $5
		WHERE id = $6
	`

	var errMsg *string
	if uploadErr != nil {
		msg := uploadErr.Error()
		errMsg = &msg
	}

	now := time.Now()
	result, err := tx.ExecContext(ctx, query, string(status), s3Key, s3URL, errMsg, now, id)
	if err != nil {
		return fmt.Errorf("failed to update upload status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("upload record not found: id=%d", id)
	}

	return nil
}

// GetUploadByID retrieves an upload record by ID using prepared statement
func (r *uploadRepository) GetUploadByID(ctx context.Context, db *database.DB, id int64) (*models.UploadRecord, error) {
	record := &models.UploadRecord{}
	var errMsg sql.NullString
	err := r.getByIDStmt.QueryRowContext(ctx, id).Scan(
		&record.ID,
		&record.Filename,
		&record.OriginalName,
		&record.FileSize,
		&record.ContentType,
		&record.S3Key,
		&record.S3URL,
		&record.Status,
		&errMsg,
		&record.CreatedAt,
		&record.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("upload record not found: id=%d", id)
		}
		return nil, fmt.Errorf("failed to get upload record: %w", err)
	}

	if errMsg.Valid {
		record.Error = &errMsg.String
	}

	return record, nil
}

// GetUploadsByStatus lấy uploads theo status dùng prepared statement
func (r *uploadRepository) GetUploadsByStatus(ctx context.Context, db *database.DB, status models.UploadStatus, limit int) ([]*models.UploadRecord, error) {
	rows, err := r.getByStatusStmt.QueryContext(ctx, string(status), limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query uploads: %w", err)
	}
	defer rows.Close()

	var records []*models.UploadRecord
	for rows.Next() {
		record := &models.UploadRecord{}
		var errMsg sql.NullString
		err := rows.Scan(
			&record.ID,
			&record.Filename,
			&record.OriginalName,
			&record.FileSize,
			&record.ContentType,
			&record.S3Key,
			&record.S3URL,
			&record.Status,
			&errMsg,
			&record.CreatedAt,
			&record.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan upload record: %w", err)
		}

		if errMsg.Valid {
			record.Error = &errMsg.String
		}

		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating upload records: %w", err)
	}

	return records, nil
}
