package repository

import (
	"context"
	"database/sql"
	"fmt"

	"s3-upload-tool/internal/database"
	"s3-upload-tool/internal/models"
)

// GradeLevelRepository đọc danh sách 12 khối lớp (seed sẵn ở migration 005) -
// tương tự SchoolRepository, dữ liệu gần như tĩnh, chỉ cần đọc.
type GradeLevelRepository interface {
	List(ctx context.Context) ([]*models.GradeLevel, error)
	ListByEducationLevel(ctx context.Context, educationLevel string) ([]*models.GradeLevel, error)
	GetByID(ctx context.Context, id int64) (*models.GradeLevel, error)
}

type gradeLevelRepository struct {
	db *database.DB
}

func NewGradeLevelRepository(db *database.DB) GradeLevelRepository {
	return &gradeLevelRepository{db: db}
}

const gradeLevelSelectColumns = `id, education_level, grade_number, label, display_order, created_at, updated_at`

func scanGradeLevel(scanner interface{ Scan(dest ...any) error }) (*models.GradeLevel, error) {
	g := &models.GradeLevel{}
	err := scanner.Scan(&g.ID, &g.EducationLevel, &g.GradeNumber, &g.Label, &g.DisplayOrder, &g.CreatedAt, &g.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return g, nil
}

func (r *gradeLevelRepository) List(ctx context.Context) ([]*models.GradeLevel, error) {
	query := fmt.Sprintf(`SELECT %s FROM grade_levels ORDER BY display_order`, gradeLevelSelectColumns)
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list grade levels: %w", err)
	}
	defer rows.Close()

	var levels []*models.GradeLevel
	for rows.Next() {
		g, err := scanGradeLevel(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan grade level: %w", err)
		}
		levels = append(levels, g)
	}
	return levels, rows.Err()
}

func (r *gradeLevelRepository) ListByEducationLevel(ctx context.Context, educationLevel string) ([]*models.GradeLevel, error) {
	query := fmt.Sprintf(`SELECT %s FROM grade_levels WHERE education_level = ? ORDER BY display_order`, gradeLevelSelectColumns)
	rows, err := r.db.QueryContext(ctx, query, educationLevel)
	if err != nil {
		return nil, fmt.Errorf("failed to list grade levels by education level: %w", err)
	}
	defer rows.Close()

	var levels []*models.GradeLevel
	for rows.Next() {
		g, err := scanGradeLevel(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan grade level: %w", err)
		}
		levels = append(levels, g)
	}
	return levels, rows.Err()
}

func (r *gradeLevelRepository) GetByID(ctx context.Context, id int64) (*models.GradeLevel, error) {
	query := fmt.Sprintf(`SELECT %s FROM grade_levels WHERE id = ?`, gradeLevelSelectColumns)
	g, err := scanGradeLevel(r.db.QueryRowContext(ctx, query, id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get grade level by id: %w", err)
	}
	return g, nil
}
