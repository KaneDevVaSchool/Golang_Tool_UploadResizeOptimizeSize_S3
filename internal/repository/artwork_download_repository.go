package repository

import (
	"context"
	"fmt"
	"time"

	"s3-upload-tool/internal/database"
	"s3-upload-tool/internal/models"
)

// ArtworkDownloadRepository ghi nhật ký tải ảnh gốc - chỉ INSERT, không có
// nghiệp vụ nào khác nên không cần transaction (khác ArtworkViewRepository
// vốn phải đồng bộ với view_count).
type ArtworkDownloadRepository interface {
	Create(ctx context.Context, download *models.ArtworkDownload) error
}

type artworkDownloadRepository struct {
	db *database.DB
}

func NewArtworkDownloadRepository(db *database.DB) ArtworkDownloadRepository {
	return &artworkDownloadRepository{db: db}
}

func (r *artworkDownloadRepository) Create(ctx context.Context, download *models.ArtworkDownload) error {
	now := time.Now()
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO artwork_downloads (artwork_id, admin_user_id, source, ip_address, user_agent, downloaded_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, download.ArtworkID, download.AdminUserID, download.Source, download.IPAddress, download.UserAgent, now)
	if err != nil {
		return fmt.Errorf("không ghi được nhật ký tải ảnh: %w", err)
	}

	id, err := result.LastInsertId()
	if err == nil {
		download.ID = id
	}
	download.DownloadedAt = now
	return nil
}
