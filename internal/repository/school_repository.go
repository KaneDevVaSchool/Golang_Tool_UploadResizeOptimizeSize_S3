package repository

import (
	"context"
	"database/sql"
	"fmt"

	"s3-upload-tool/internal/database"
	"s3-upload-tool/internal/models"
)

// SchoolRepository đọc danh sách 5 cơ sở vật lý (schools) - dữ liệu gần như
// tĩnh (seed sẵn ở migration 004), chỉ cần List/GetByID, không cần CRUD đầy
// đủ vì trường học không tạo/xoá qua UI hội thi này.
type SchoolRepository interface {
	List(ctx context.Context) ([]*models.School, error)
	GetByID(ctx context.Context, id int64) (*models.School, error)
}

type schoolRepository struct {
	db *database.DB
}

func NewSchoolRepository(db *database.DB) SchoolRepository {
	return &schoolRepository{db: db}
}

const schoolSelectColumns = `id, name, region, display_order, is_active, created_at, updated_at`

func scanSchool(scanner interface{ Scan(dest ...any) error }) (*models.School, error) {
	s := &models.School{}
	err := scanner.Scan(&s.ID, &s.Name, &s.Region, &s.DisplayOrder, &s.IsActive, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (r *schoolRepository) List(ctx context.Context) ([]*models.School, error) {
	query := fmt.Sprintf(`SELECT %s FROM schools WHERE is_active = 1 ORDER BY display_order`, schoolSelectColumns)
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list schools: %w", err)
	}
	defer rows.Close()

	var schools []*models.School
	for rows.Next() {
		s, err := scanSchool(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan school: %w", err)
		}
		schools = append(schools, s)
	}
	return schools, rows.Err()
}

func (r *schoolRepository) GetByID(ctx context.Context, id int64) (*models.School, error) {
	query := fmt.Sprintf(`SELECT %s FROM schools WHERE id = ?`, schoolSelectColumns)
	s, err := scanSchool(r.db.QueryRowContext(ctx, query, id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get school by id: %w", err)
	}
	return s, nil
}
