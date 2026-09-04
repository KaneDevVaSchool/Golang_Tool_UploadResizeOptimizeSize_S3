package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"s3-upload-tool/internal/database"
	"s3-upload-tool/internal/models"
)

// CommentRepository quản lý artwork_comments - bình luận ẩn danh, người xem
// tự nhập display_name khi comment.
type CommentRepository interface {
	Create(ctx context.Context, comment *models.ArtworkComment) (*models.ArtworkComment, error)
	// ListByArtwork trả comment mới nhất trước; includeHidden chỉ dùng ở
	// trang quản trị (moderation), public luôn truyền false.
	ListByArtwork(ctx context.Context, artworkID int64, includeHidden bool) ([]*models.ArtworkComment, error)
	SetHidden(ctx context.Context, id int64, hidden bool) error
	CountByArtwork(ctx context.Context, artworkID int64) (int64, error)
	// CountByArtworkBatch trả map artworkID -> số comment KHÔNG ẩn, dùng cho
	// trang danh sách nhiều tác phẩm để tránh N+1 query.
	CountByArtworkBatch(ctx context.Context, artworkIDs []int64) (map[int64]int64, error)
}

type commentRepository struct {
	db *database.DB
}

func NewCommentRepository(db *database.DB) CommentRepository {
	return &commentRepository{db: db}
}

func (r *commentRepository) Create(ctx context.Context, comment *models.ArtworkComment) (*models.ArtworkComment, error) {
	now := time.Now()
	query := `
		INSERT INTO artwork_comments (artwork_id, display_name, content, visitor_token, ip_address, is_hidden, created_at)
		VALUES (?, ?, ?, ?, ?, 0, ?)
	`
	result, err := r.db.ExecContext(ctx, query, comment.ArtworkID, comment.DisplayName, comment.Content, comment.VisitorToken, comment.IPAddress, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create comment: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}
	comment.ID = id
	comment.CreatedAt = now
	return comment, nil
}

func (r *commentRepository) ListByArtwork(ctx context.Context, artworkID int64, includeHidden bool) ([]*models.ArtworkComment, error) {
	query := `
		SELECT id, artwork_id, display_name, content, visitor_token, ip_address, is_hidden, created_at
		FROM artwork_comments
		WHERE artwork_id = ?
	`
	if !includeHidden {
		query += ` AND is_hidden = 0`
	}
	query += ` ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, artworkID)
	if err != nil {
		return nil, fmt.Errorf("failed to list comments: %w", err)
	}
	defer rows.Close()

	var comments []*models.ArtworkComment
	for rows.Next() {
		c := &models.ArtworkComment{}
		err := rows.Scan(&c.ID, &c.ArtworkID, &c.DisplayName, &c.Content, &c.VisitorToken, &c.IPAddress, &c.IsHidden, &c.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan comment: %w", err)
		}
		comments = append(comments, c)
	}
	return comments, rows.Err()
}

func (r *commentRepository) SetHidden(ctx context.Context, id int64, hidden bool) error {
	result, err := r.db.ExecContext(ctx, `UPDATE artwork_comments SET is_hidden = ? WHERE id = ?`, hidden, id)
	if err != nil {
		return fmt.Errorf("failed to set comment hidden state: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("comment not found: id=%d", id)
	}
	return nil
}

func (r *commentRepository) CountByArtwork(ctx context.Context, artworkID int64) (int64, error) {
	var count int64
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM artwork_comments WHERE artwork_id = ? AND is_hidden = 0`, artworkID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count comments: %w", err)
	}
	return count, nil
}

func (r *commentRepository) CountByArtworkBatch(ctx context.Context, artworkIDs []int64) (map[int64]int64, error) {
	result := make(map[int64]int64)
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
		SELECT artwork_id, COUNT(*)
		FROM artwork_comments
		WHERE artwork_id IN (%s) AND is_hidden = 0
		GROUP BY artwork_id
	`, strings.Join(placeholders, ","))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to batch count comments: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var artworkID int64
		var count int64
		if err := rows.Scan(&artworkID, &count); err != nil {
			return nil, fmt.Errorf("failed to scan comment count row: %w", err)
		}
		result[artworkID] = count
	}
	return result, rows.Err()
}
