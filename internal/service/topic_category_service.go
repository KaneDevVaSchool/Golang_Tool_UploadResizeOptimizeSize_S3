package service

import (
	"context"
	"fmt"
	"strings"

	"s3-upload-tool/internal/models"
	"s3-upload-tool/internal/repository"
)

// TopicCategoryService quản lý danh mục nhóm chủ đề sáng tạo - mỏng, chủ
// yếu validate rồi ủy quyền cho TopicCategoryRepository (cùng mẫu AwardService).
type TopicCategoryService interface {
	ListTopicCategories(ctx context.Context, includeInactive bool) ([]*models.TopicCategory, error)
	CreateTopicCategory(ctx context.Context, category *models.TopicCategory) (*models.TopicCategory, error)
	UpdateTopicCategory(ctx context.Context, category *models.TopicCategory) error
	DeleteTopicCategory(ctx context.Context, id int64) error
}

type topicCategoryService struct {
	repo repository.TopicCategoryRepository
}

func NewTopicCategoryService(repo repository.TopicCategoryRepository) TopicCategoryService {
	return &topicCategoryService{repo: repo}
}

func (s *topicCategoryService) ListTopicCategories(ctx context.Context, includeInactive bool) ([]*models.TopicCategory, error) {
	return s.repo.List(ctx, includeInactive)
}

func (s *topicCategoryService) CreateTopicCategory(ctx context.Context, category *models.TopicCategory) (*models.TopicCategory, error) {
	if strings.TrimSpace(category.Name) == "" {
		return nil, fmt.Errorf("tên nhóm chủ đề không được để trống")
	}
	if err := validateEducationLevel(category.EducationLevel); err != nil {
		return nil, err
	}
	if category.Slug == "" {
		category.Slug = slugify(category.Name)
	}
	return s.repo.Create(ctx, category)
}

func (s *topicCategoryService) UpdateTopicCategory(ctx context.Context, category *models.TopicCategory) error {
	if strings.TrimSpace(category.Name) == "" {
		return fmt.Errorf("tên nhóm chủ đề không được để trống")
	}
	if err := validateEducationLevel(category.EducationLevel); err != nil {
		return err
	}
	if category.Slug == "" {
		category.Slug = slugify(category.Name)
	}
	return s.repo.Update(ctx, category)
}

func (s *topicCategoryService) DeleteTopicCategory(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

// validateEducationLevel chặn giá trị rác ở tầng service - không tin cấp học
// gửi từ frontend chỉ vì ENUM ở DB sẽ tự chặn (lỗi ENUM trả 500 khó hiểu hơn
// nhiều so với validate rõ ràng ở đây).
func validateEducationLevel(level *string) error {
	if level == nil {
		return nil
	}
	if *level != models.EducationLevelPrimary && *level != models.EducationLevelSecondary {
		return fmt.Errorf("cấp học không hợp lệ: %s", *level)
	}
	return nil
}
