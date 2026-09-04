package repository

import (
	"context"
	"fmt"
	"time"

	"s3-upload-tool/internal/database"
)

// viewDedupeWindow là khoảng thời gian không đếm trùng view cho cùng 1
// visitor_token trên cùng 1 tác phẩm.
const viewDedupeWindow = 24 * time.Hour

// ArtworkViewRepository quản lý artwork_views - chống đếm trùng lượt xem
// trong viewDedupeWindow trước khi tăng artworks.view_count.
type ArtworkViewRepository interface {
	// RecordView kiểm tra visitor đã xem trong viewDedupeWindow chưa; nếu
	// chưa thì ghi view mới + tăng artworks.view_count (2 thao tác trong 1
	// transaction ngắn), trả counted=true. Nếu đã xem gần đây, không ghi gì
	// và trả counted=false - caller không cần tăng view_count phía response.
	RecordView(ctx context.Context, artworkID int64, visitorToken string) (counted bool, err error)
}

type artworkViewRepository struct {
	db *database.DB
}

func NewArtworkViewRepository(db *database.DB) ArtworkViewRepository {
	return &artworkViewRepository{db: db}
}

func (r *artworkViewRepository) RecordView(ctx context.Context, artworkID int64, visitorToken string) (bool, error) {
	cutoff := time.Now().Add(-viewDedupeWindow)

	var exists int
	checkQuery := `
		SELECT 1 FROM artwork_views
		WHERE artwork_id = ? AND visitor_token = ? AND viewed_at > ?
		LIMIT 1
	`
	err := r.db.QueryRowContext(ctx, checkQuery, artworkID, visitorToken, cutoff).Scan(&exists)
	if err == nil {
		// Đã xem trong window - không đếm thêm.
		return false, nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("failed to begin transaction: %w", err)
	}

	now := time.Now()
	if _, err := tx.ExecContext(ctx, `INSERT INTO artwork_views (artwork_id, visitor_token, viewed_at) VALUES (?, ?, ?)`, artworkID, visitorToken, now); err != nil {
		tx.Rollback()
		return false, fmt.Errorf("failed to insert artwork view: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `UPDATE artworks SET view_count = view_count + 1 WHERE id = ?`, artworkID); err != nil {
		tx.Rollback()
		return false, fmt.Errorf("failed to increment view_count: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("failed to commit view transaction: %w", err)
	}

	return true, nil
}
