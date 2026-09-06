package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"s3-upload-tool/internal/database"
	"s3-upload-tool/internal/models"
)

// TopicCategoryRepository quản lý danh mục nhóm chủ đề sáng tạo - cùng mẫu
// CRUD với AwardRepository (admin tự tạo/sửa/xoá qua trang quản lý).
type TopicCategoryRepository interface {
	List(ctx context.Context, includeInactive bool) ([]*models.TopicCategory, error)
	GetByID(ctx context.Context, id int64) (*models.TopicCategory, error)
	Create(ctx context.Context, category *models.TopicCategory) (*models.TopicCategory, error)
	Update(ctx context.Context, category *models.TopicCategory) error
	Delete(ctx context.Context, id int64) error
}

type topicCategoryRepository struct {
	db *database.DB
}

func NewTopicCategoryRepository(db *database.DB) TopicCategoryRepository {
	return &topicCategoryRepository{db: db}
}

const topicCategorySelectColumns = `id, name, slug, education_level, display_order, is_active, created_at, updated_at`

func scanTopicCategory(scanner interface{ Scan(dest ...any) error }) (*models.TopicCategory, error) {
	c := &models.TopicCategory{}
	var educationLevel sql.NullString
	err := scanner.Scan(&c.ID, &c.Name, &c.Slug, &educationLevel, &c.DisplayOrder, &c.IsActive, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if educationLevel.Valid {
		c.EducationLevel = &educationLevel.String
	}
	return c, nil
}

func (r *topicCategoryRepository) List(ctx context.Context, includeInactive bool) ([]*models.TopicCategory, error) {
	query := fmt.Sprintf(`SELECT %s FROM topic_categories`, topicCategorySelectColumns)
	if !includeInactive {
		query += ` WHERE is_active = 1`
	}
	query += ` ORDER BY display_order`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list topic categories: %w", err)
	}
	defer rows.Close()

	var categories []*models.TopicCategory
	for rows.Next() {
		c, err := scanTopicCategory(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan topic category: %w", err)
		}
		categories = append(categories, c)
	}
	return categories, rows.Err()
}

func (r *topicCategoryRepository) GetByID(ctx context.Context, id int64) (*models.TopicCategory, error) {
	query := fmt.Sprintf(`SELECT %s FROM topic_categories WHERE id = ?`, topicCategorySelectColumns)
	c, err := scanTopicCategory(r.db.QueryRowContext(ctx, query, id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get topic category by id: %w", err)
	}
	return c, nil
}

func (r *topicCategoryRepository) Create(ctx context.Context, category *models.TopicCategory) (*models.TopicCategory, error) {
	now := time.Now()
	query := `
		INSERT INTO topic_categories (name, slug, education_level, display_order, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	result, err := r.db.ExecContext(ctx, query, category.Name, category.Slug, category.EducationLevel, category.DisplayOrder, category.IsActive, now, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create topic category: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}
	category.ID = id
	category.CreatedAt = now
	category.UpdatedAt = now
	return category, nil
}

func (r *topicCategoryRepository) Update(ctx context.Context, category *models.TopicCategory) error {
	query := `
		UPDATE topic_categories
		SET name = ?, slug = ?, education_level = ?, display_order = ?, is_active = ?, updated_at = ?
		WHERE id = ?
	`
	now := time.Now()
	result, err := r.db.ExecContext(ctx, query, category.Name, category.Slug, category.EducationLevel, category.DisplayOrder, category.IsActive, now, category.ID)
	if err != nil {
		return fmt.Errorf("failed to update topic category: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("topic category not found: id=%d", category.ID)
	}
	category.UpdatedAt = now
	return nil
}

func (r *topicCategoryRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM topic_categories WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete topic category: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("topic category not found: id=%d", id)
	}
	return nil
}
