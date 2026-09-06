package repository

import (
	"context"
	"fmt"
	"time"

	"s3-upload-tool/internal/database"
)

// ArtworkViewRepository ghi artwork_views và tăng artworks.view_count.
type ArtworkViewRepository interface {
	// RecordView ghi một lượt xem (INSERT artwork_views + view_count + 1 trong
	// một transaction). counted luôn true khi thành công.
	RecordView(ctx context.Context, artworkID int64, visitorToken string) (counted bool, err error)
}

type artworkViewRepository struct {
	db *database.DB
}

func NewArtworkViewRepository(db *database.DB) ArtworkViewRepository {
	return &artworkViewRepository{db: db}
}

func (r *artworkViewRepository) RecordView(ctx context.Context, artworkID int64, visitorToken string) (bool, error) {
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
