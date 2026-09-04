package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"s3-upload-tool/internal/models"
	"s3-upload-tool/internal/repository"
)

// slugSanitizeRegex giữ lại chữ thường/số/gạch ngang, dùng khi tự sinh slug
// từ tên giải (vd "Giải Nhất" -> "giai-nhat" sau khi FE bỏ dấu, hoặc fallback
// nếu FE gửi slug rỗng).
var slugSanitizeRegex = regexp.MustCompile(`[^a-z0-9-]+`)

// AwardService quản lý cấu hình giải thưởng - mỏng, chủ yếu validate rồi
// ủy quyền cho AwardRepository.
type AwardService interface {
	ListAwards(ctx context.Context, includeInactive bool) ([]*models.Award, error)
	CreateAward(ctx context.Context, award *models.Award) (*models.Award, error)
	UpdateAward(ctx context.Context, award *models.Award) error
	DeleteAward(ctx context.Context, id int64) error
}

type awardService struct {
	repo repository.AwardRepository
}

func NewAwardService(repo repository.AwardRepository) AwardService {
	return &awardService{repo: repo}
}

func (s *awardService) ListAwards(ctx context.Context, includeInactive bool) ([]*models.Award, error) {
	return s.repo.List(ctx, includeInactive)
}

func (s *awardService) CreateAward(ctx context.Context, award *models.Award) (*models.Award, error) {
	if strings.TrimSpace(award.Name) == "" {
		return nil, fmt.Errorf("tên giải thưởng không được để trống")
	}
	if award.Slug == "" {
		award.Slug = slugify(award.Name)
	}
	if award.ColorHex == "" {
		award.ColorHex = "#c49c57" // mặc định màu TRÁCH NHIỆM
	}
	return s.repo.Create(ctx, award)
}

func (s *awardService) UpdateAward(ctx context.Context, award *models.Award) error {
	if strings.TrimSpace(award.Name) == "" {
		return fmt.Errorf("tên giải thưởng không được để trống")
	}
	if award.Slug == "" {
		award.Slug = slugify(award.Name)
	}
	return s.repo.Update(ctx, award)
}

func (s *awardService) DeleteAward(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

// slugify chuyển tên giải thành slug ASCII an toàn cho URL/UNIQUE KEY -
// không xử lý bỏ dấu tiếng Việt đầy đủ (FE nên gửi sẵn slug đã bỏ dấu),
// fallback này chỉ đảm bảo không lỗi UNIQUE KEY khi FE để trống slug.
func slugify(name string) string {
	lower := strings.ToLower(strings.TrimSpace(name))
	lower = strings.ReplaceAll(lower, " ", "-")
	return strings.Trim(slugSanitizeRegex.ReplaceAllString(lower, ""), "-")
}
