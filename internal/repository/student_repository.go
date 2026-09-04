package repository

import (
	"context"
	"fmt"
	"time"

	"s3-upload-tool/internal/database"
	"s3-upload-tool/internal/models"
)

// StudentRepository quản lý học sinh sáng tác. Không có mã định danh học
// sinh chính thức nên Create luôn tạo record mới - không dedupe theo tên,
// tránh gộp nhầm 2 học sinh trùng tên khác lớp/trường.
type StudentRepository interface {
	// Create nhận *database.Tx vì luôn được gọi trong cùng transaction với
	// artworks.Create (ArtworkService.CreateArtworkFromUpload) - theo đúng
	// pattern UploadRepository.CreateUpload.
	Create(ctx context.Context, tx *database.Tx, student *models.Student) (*models.Student, error)
	GetByID(ctx context.Context, id int64) (*models.Student, error)
}

type studentRepository struct {
	db *database.DB
}

func NewStudentRepository(db *database.DB) StudentRepository {
	return &studentRepository{db: db}
}

func (r *studentRepository) Create(ctx context.Context, tx *database.Tx, student *models.Student) (*models.Student, error) {
	now := time.Now()
	query := `
		INSERT INTO students (full_name, school_id, grade_level_id, class_name, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	result, err := tx.ExecContext(ctx, query, student.FullName, student.SchoolID, student.GradeLevelID, student.ClassName, now, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create student: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	student.ID = id
	student.CreatedAt = now
	student.UpdatedAt = now
	return student, nil
}

func (r *studentRepository) GetByID(ctx context.Context, id int64) (*models.Student, error) {
	query := `SELECT id, full_name, school_id, grade_level_id, class_name, created_at, updated_at FROM students WHERE id = ?`
	s := &models.Student{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(&s.ID, &s.FullName, &s.SchoolID, &s.GradeLevelID, &s.ClassName, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get student by id: %w", err)
	}
	return s, nil
}
