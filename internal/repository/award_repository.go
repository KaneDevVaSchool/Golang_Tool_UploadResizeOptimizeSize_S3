package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"s3-upload-tool/internal/database"
	"s3-upload-tool/internal/models"
)

// AwardRepository quản lý cấu hình giải thưởng (awards) và quan hệ N-N với
// artworks (artwork_awards).
type AwardRepository interface {
	List(ctx context.Context, includeInactive bool) ([]*models.Award, error)
	GetByID(ctx context.Context, id int64) (*models.Award, error)
	Create(ctx context.Context, award *models.Award) (*models.Award, error)
	Update(ctx context.Context, award *models.Award) error
	Delete(ctx context.Context, id int64) error

	AttachToArtwork(ctx context.Context, artworkID, awardID int64) error
	DetachFromArtwork(ctx context.Context, artworkID, awardID int64) error
	ListByArtworkID(ctx context.Context, artworkID int64) ([]*models.Award, error)
	// ListByArtworkIDs trả map artworkID -> []*Award, dùng khi render danh
	// sách nhiều tác phẩm cùng lúc để tránh N+1 query.
	ListByArtworkIDs(ctx context.Context, artworkIDs []int64) (map[int64][]*models.Award, error)
}

type awardRepository struct {
	db *database.DB
}

func NewAwardRepository(db *database.DB) AwardRepository {
	return &awardRepository{db: db}
}

const awardSelectColumns = `id, name, slug, rank_order, color_hex, icon_key, is_active, created_at, updated_at`

func scanAward(scanner interface{ Scan(dest ...any) error }) (*models.Award, error) {
	a := &models.Award{}
	var iconKey sql.NullString
	err := scanner.Scan(&a.ID, &a.Name, &a.Slug, &a.RankOrder, &a.ColorHex, &iconKey, &a.IsActive, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if iconKey.Valid {
		a.IconKey = &iconKey.String
	}
	return a, nil
}

func (r *awardRepository) List(ctx context.Context, includeInactive bool) ([]*models.Award, error) {
	query := fmt.Sprintf(`SELECT %s FROM awards`, awardSelectColumns)
	if !includeInactive {
		query += ` WHERE is_active = 1`
	}
	query += ` ORDER BY rank_order`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list awards: %w", err)
	}
	defer rows.Close()

	var awards []*models.Award
	for rows.Next() {
		a, err := scanAward(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan award: %w", err)
		}
		awards = append(awards, a)
	}
	return awards, rows.Err()
}

func (r *awardRepository) GetByID(ctx context.Context, id int64) (*models.Award, error) {
	query := fmt.Sprintf(`SELECT %s FROM awards WHERE id = ?`, awardSelectColumns)
	a, err := scanAward(r.db.QueryRowContext(ctx, query, id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get award by id: %w", err)
	}
	return a, nil
}

func (r *awardRepository) Create(ctx context.Context, award *models.Award) (*models.Award, error) {
	now := time.Now()
	query := `
		INSERT INTO awards (name, slug, rank_order, color_hex, icon_key, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	result, err := r.db.ExecContext(ctx, query, award.Name, award.Slug, award.RankOrder, award.ColorHex, award.IconKey, award.IsActive, now, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create award: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}
	award.ID = id
	award.CreatedAt = now
	award.UpdatedAt = now
	return award, nil
}

func (r *awardRepository) Update(ctx context.Context, award *models.Award) error {
	query := `
		UPDATE awards
		SET name = ?, slug = ?, rank_order = ?, color_hex = ?, icon_key = ?, is_active = ?, updated_at = ?
		WHERE id = ?
	`
	now := time.Now()
	result, err := r.db.ExecContext(ctx, query, award.Name, award.Slug, award.RankOrder, award.ColorHex, award.IconKey, award.IsActive, now, award.ID)
	if err != nil {
		return fmt.Errorf("failed to update award: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("award not found: id=%d", award.ID)
	}
	award.UpdatedAt = now
	return nil
}

func (r *awardRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM awards WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete award: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("award not found: id=%d", id)
	}
	return nil
}

func (r *awardRepository) AttachToArtwork(ctx context.Context, artworkID, awardID int64) error {
	query := `
		INSERT INTO artwork_awards (artwork_id, award_id, awarded_at, created_at)
		VALUES (?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE awarded_at = VALUES(awarded_at)
	`
	now := time.Now()
	_, err := r.db.ExecContext(ctx, query, artworkID, awardID, now, now)
	if err != nil {
		return fmt.Errorf("failed to attach award to artwork: %w", err)
	}
	return nil
}

func (r *awardRepository) DetachFromArtwork(ctx context.Context, artworkID, awardID int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM artwork_awards WHERE artwork_id = ? AND award_id = ?`, artworkID, awardID)
	if err != nil {
		return fmt.Errorf("failed to detach award from artwork: %w", err)
	}
	return nil
}

func (r *awardRepository) ListByArtworkID(ctx context.Context, artworkID int64) ([]*models.Award, error) {
	query := fmt.Sprintf(`
		SELECT %s FROM awards a
		JOIN artwork_awards aa ON aa.award_id = a.id
		WHERE aa.artwork_id = ?
		ORDER BY a.rank_order
	`, prefixColumns("a", awardSelectColumns))
	rows, err := r.db.QueryContext(ctx, query, artworkID)
	if err != nil {
		return nil, fmt.Errorf("failed to list awards by artwork id: %w", err)
	}
	defer rows.Close()

	var awards []*models.Award
	for rows.Next() {
		a, err := scanAward(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan award: %w", err)
		}
		awards = append(awards, a)
	}
	return awards, rows.Err()
}

func (r *awardRepository) ListByArtworkIDs(ctx context.Context, artworkIDs []int64) (map[int64][]*models.Award, error) {
	result := make(map[int64][]*models.Award)
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
		SELECT aa.artwork_id, %s FROM awards a
		JOIN artwork_awards aa ON aa.award_id = a.id
		WHERE aa.artwork_id IN (%s)
		ORDER BY a.rank_order
	`, prefixColumns("a", awardSelectColumns), strings.Join(placeholders, ","))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list awards by artwork ids: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var artworkID int64
		a := &models.Award{}
		var iconKey sql.NullString
		err := rows.Scan(&artworkID, &a.ID, &a.Name, &a.Slug, &a.RankOrder, &a.ColorHex, &iconKey, &a.IsActive, &a.CreatedAt, &a.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan award row: %w", err)
		}
		if iconKey.Valid {
			a.IconKey = &iconKey.String
		}
		result[artworkID] = append(result[artworkID], a)
	}
	return result, rows.Err()
}

// prefixColumns thêm prefix bảng (vd "a.") vào từng cột trong danh sách
// comma-separated - dùng khi JOIN cần cột rõ ràng thuộc bảng nào. Dùng
// chung cho mọi repository cần JOIN (award/artwork/reaction/comment).
func prefixColumns(prefix, columns string) string {
	parts := strings.Split(columns, ",")
	for i, p := range parts {
		parts[i] = prefix + "." + strings.TrimSpace(p)
	}
	return strings.Join(parts, ", ")
}
