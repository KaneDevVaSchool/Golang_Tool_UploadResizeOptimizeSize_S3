package repository

import (
	"context"
	"fmt"
	"strings"
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
	// ListByIDs lấy nhiều học sinh trong MỘT truy vấn, trả map theo id.
	//
	// Có mặt để thay cho vòng lặp gọi GetByID từng dòng khi ghép metadata cho
	// một trang tác phẩm (ArtworkService.enrichArtworks): một trang 36 tranh
	// nghĩa là 36 lượt khứ hồi tới DB, trong khi awards/reactions/comments
	// cạnh đó vốn đã gộp sẵn thành một truy vấn.
	ListByIDs(ctx context.Context, ids []int64) (map[int64]*models.Student, error)
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

const studentSelectColumns = `id, full_name, school_id, grade_level_id, class_name, created_at, updated_at`

func scanStudent(scanner interface{ Scan(dest ...any) error }) (*models.Student, error) {
	s := &models.Student{}
	err := scanner.Scan(&s.ID, &s.FullName, &s.SchoolID, &s.GradeLevelID, &s.ClassName, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (r *studentRepository) GetByID(ctx context.Context, id int64) (*models.Student, error) {
	query := `SELECT ` + studentSelectColumns + ` FROM students WHERE id = ?`
	s, err := scanStudent(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		return nil, fmt.Errorf("failed to get student by id: %w", err)
	}
	return s, nil
}

func (r *studentRepository) ListByIDs(ctx context.Context, ids []int64) (map[int64]*models.Student, error) {
	result := make(map[int64]*models.Student, len(ids))
	if len(ids) == 0 {
		return result, nil
	}

	// Gộp id trùng: nhiều tác phẩm cùng một học sinh vẫn chỉ cần lấy một lần.
	placeholders := make([]string, 0, len(ids))
	args := make([]any, 0, len(ids))
	seen := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		placeholders = append(placeholders, "?")
		args = append(args, id)
	}

	query := fmt.Sprintf(
		`SELECT %s FROM students WHERE id IN (%s)`,
		studentSelectColumns, strings.Join(placeholders, ","),
	)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list students by ids: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		s, err := scanStudent(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan student row: %w", err)
		}
		result[s.ID] = s
	}
	return result, rows.Err()
}
