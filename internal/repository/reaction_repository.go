package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"s3-upload-tool/internal/database"
)

// ReactionRepository quản lý artwork_reactions - cảm xúc ẩn danh (6 loại),
// định danh qua visitor_token (không phải xác thực danh tính thật).
type ReactionRepository interface {
	// Upsert thêm 1 reaction; nếu visitor đã react cùng loại cho cùng tác
	// phẩm thì không lỗi (idempotent nhờ UNIQUE KEY + INSERT IGNORE).
	Upsert(ctx context.Context, artworkID int64, reactionType, visitorToken, ipAddress string) error
	Remove(ctx context.Context, artworkID int64, reactionType, visitorToken string) error
	// CountByArtwork trả về map reaction_type -> số lượng cho 1 tác phẩm.
	CountByArtwork(ctx context.Context, artworkID int64) (map[string]int64, error)
	// CountByArtworkBatch trả map artworkID -> map[reactionType]count, dùng
	// cho trang danh sách nhiều tác phẩm để tránh N+1 query.
	CountByArtworkBatch(ctx context.Context, artworkIDs []int64) (map[int64]map[string]int64, error)
}

type reactionRepository struct {
	db *database.DB
}

func NewReactionRepository(db *database.DB) ReactionRepository {
	return &reactionRepository{db: db}
}

func (r *reactionRepository) Upsert(ctx context.Context, artworkID int64, reactionType, visitorToken, ipAddress string) error {
	query := `
		INSERT IGNORE INTO artwork_reactions (artwork_id, reaction_type, visitor_token, ip_address, created_at)
		VALUES (?, ?, ?, ?, ?)
	`
	_, err := r.db.ExecContext(ctx, query, artworkID, reactionType, visitorToken, ipAddress, time.Now())
	if err != nil {
		return fmt.Errorf("failed to upsert reaction: %w", err)
	}
	return nil
}

func (r *reactionRepository) Remove(ctx context.Context, artworkID int64, reactionType, visitorToken string) error {
	query := `DELETE FROM artwork_reactions WHERE artwork_id = ? AND reaction_type = ? AND visitor_token = ?`
	_, err := r.db.ExecContext(ctx, query, artworkID, reactionType, visitorToken)
	if err != nil {
		return fmt.Errorf("failed to remove reaction: %w", err)
	}
	return nil
}

func (r *reactionRepository) CountByArtwork(ctx context.Context, artworkID int64) (map[string]int64, error) {
	query := `SELECT reaction_type, COUNT(*) FROM artwork_reactions WHERE artwork_id = ? GROUP BY reaction_type`
	rows, err := r.db.QueryContext(ctx, query, artworkID)
	if err != nil {
		return nil, fmt.Errorf("failed to count reactions: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int64)
	for rows.Next() {
		var reactionType string
		var count int64
		if err := rows.Scan(&reactionType, &count); err != nil {
			return nil, fmt.Errorf("failed to scan reaction count: %w", err)
		}
		counts[reactionType] = count
	}
	return counts, rows.Err()
}

func (r *reactionRepository) CountByArtworkBatch(ctx context.Context, artworkIDs []int64) (map[int64]map[string]int64, error) {
	result := make(map[int64]map[string]int64)
	if len(artworkIDs) == 0 {
		return result, nil
	}

	placeholders := make([]string, len(artworkIDs))
	args := make([]any, len(artworkIDs))
	for i, id := range artworkIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT artwork_id, reaction_type, COUNT(*)
		FROM artwork_reactions
		WHERE artwork_id IN (%s)
		GROUP BY artwork_id, reaction_type
	`, strings.Join(placeholders, ","))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to batch count reactions: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var artworkID int64
		var reactionType string
		var count int64
		if err := rows.Scan(&artworkID, &reactionType, &count); err != nil {
			return nil, fmt.Errorf("failed to scan reaction count row: %w", err)
		}
		if result[artworkID] == nil {
			result[artworkID] = make(map[string]int64)
		}
		result[artworkID][reactionType] = count
	}
	return result, rows.Err()
}
