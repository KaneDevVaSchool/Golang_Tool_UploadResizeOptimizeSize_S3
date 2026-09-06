package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"s3-upload-tool/internal/config"
	"s3-upload-tool/internal/database"
	"s3-upload-tool/internal/models"
	"s3-upload-tool/internal/repository"
	"s3-upload-tool/internal/service"
)

type seeder struct {
	ctx          context.Context
	cfg          *config.Config
	db           *database.DB
	s3Repo       repository.S3Repository
	schoolRepo   repository.SchoolRepository
	gradeRepo    repository.GradeLevelRepository
	topicRepo    repository.TopicCategoryRepository
	awardRepo    repository.AwardRepository
	reactionRepo repository.ReactionRepository
	commentRepo  repository.CommentRepository
	artworkSvc   service.ArtworkService
}

// wipe xoá sạch dữ liệu nghiệp vụ (KHÔNG đụng admin_users/admin_sessions/
// grade_levels) theo đúng thứ tự phụ thuộc khoá ngoại trong
// docs/detail_design/01-database.md: tương tác -> artwork_awards -> artworks
// -> students -> awards/topic_categories.
//
// S3 key được thu thập TRƯỚC khi xoá DB row (không phải quét bucket) - chỉ
// xoá đúng object mà artworks đang tham chiếu, không đụng object khác có thể
// đang chờ xử lý (uploads pending) hay ngoài phạm vi seed.
func (s *seeder) wipe() error {
	keys, err := s.collectArtworkS3Keys()
	if err != nil {
		return fmt.Errorf("không liệt kê được s3_key trước khi xoá: %w", err)
	}

	// Thứ tự xoá: bảng phụ thuộc trước, bảng bị phụ thuộc sau.
	stmts := []string{
		`DELETE FROM artwork_views`,
		`DELETE FROM artwork_comments`,
		`DELETE FROM artwork_reactions`,
		`DELETE FROM artwork_awards`,
		`DELETE FROM artworks`,
		`DELETE FROM students`,
		`DELETE FROM awards`,
		`DELETE FROM topic_categories`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.ExecContext(s.ctx, stmt); err != nil {
			return fmt.Errorf("lỗi khi chạy %q: %w", stmt, err)
		}
	}
	log.Printf("Đã xoá dữ liệu nghiệp vụ cũ (bao gồm %d ảnh trên S3 sắp xoá)", len(keys))

	for _, key := range keys {
		if key == "" {
			continue
		}
		if err := s.s3Repo.Delete(s.ctx, s.cfg.AWS.BucketName, key); err != nil {
			// Không chặn seed vì lỗi dọn rác S3: DB đã sạch, object mồ côi còn
			// lại chỉ tốn dung lượng chứ không gây sai dữ liệu - cùng nguyên
			// tắc "lỗi khâu phụ trợ không hỏng thao tác chính" áp dụng ở
			// buildVariants (internal/service/artwork_service.go).
			log.Printf("Cảnh báo: không xoá được object S3 %s: %v", key, err)
		}
	}

	return nil
}

// collectArtworkS3Keys đọc s3_key của artworks + toàn bộ key trong cột
// variants (JSON) trước khi xoá bảng - đây là cách duy nhất biết chính xác
// những object nào seed cũ đã tạo ra trên S3.
func (s *seeder) collectArtworkS3Keys() ([]string, error) {
	rows, err := s.db.QueryContext(s.ctx, `SELECT s3_key, variants FROM artworks`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var s3Key string
		var variants models.ArtworkVariants
		if err := rows.Scan(&s3Key, &variants); err != nil {
			return nil, err
		}
		keys = append(keys, s3Key)
		for _, url := range variants {
			if key := keyFromURL(url); key != "" {
				keys = append(keys, key)
			}
		}
	}
	return keys, rows.Err()
}

// keyFromURL lấy lại S3 key từ URL công khai dạng
// https://<bucket>.s3.<region>.amazonaws.com/<key> - variants lưu URL đầy đủ
// chứ không lưu key riêng (xem ArtworkVariants ở models/artwork.go).
func keyFromURL(url string) string {
	const marker = ".amazonaws.com/"
	idx := indexOf(url, marker)
	if idx < 0 {
		return ""
	}
	return url[idx+len(marker):]
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// schoolSeed là 1 dòng trong danh sách 16 cơ sở thật của Hệ thống Trường
// Việt Mỹ, thay cho 5 cơ sở cũ trong migration 004 (đã lỗi thời - xem
// docs/detail_design/01-database.md). Không lưu địa chỉ/hotline: bảng
// schools hiện chỉ có cột name/region/display_order, thêm cột mới nằm
// ngoài phạm vi seed dữ liệu demo này.
type schoolSeed struct {
	Name   string
	Region string
}

var schoolSeeds = []schoolSeed{
	// Sài Gòn (TP.HCM) - 8 cơ sở
	{"Trường Mầm non - Tiểu học - THCS - THPT Việt Mỹ - Bình Thới", models.RegionSaigon},
	{"Trường Mầm non Việt Mỹ - Hoà Bình", models.RegionSaigon},
	{"Trường Tiểu học - THCS Việt Mỹ - Tân Bình", models.RegionSaigon},
	{"Trường Mầm non Việt Mỹ - Vĩnh Hội", models.RegionSaigon},
	{"Trường Mầm non Việt Mỹ - Phú Định", models.RegionSaigon},
	{"Trường Mầm non Việt Mỹ - Hạnh Thông", models.RegionSaigon},
	{"Trường Tiểu học - THCS Việt Mỹ - Phú Định", models.RegionSaigon},
	{"Trường Tiểu học - THCS Việt Mỹ - Thông Tây Hội", models.RegionSaigon},
	// Vũng Tàu - 5 cơ sở
	{"Trường Mầm non Việt Mỹ - Hoàng Diệu", models.RegionVungTau},
	{"Trường Mầm non Việt Mỹ - Chí Linh", models.RegionVungTau},
	{"Trường Tiểu học Việt Anh - Vĩnh Ký", models.RegionVungTau},
	{"Trường Tiểu học Việt Anh - Vĩnh Ký 2", models.RegionVungTau},
	{"Trường TH - THCS - THPT Việt Mỹ Vũng Tàu", models.RegionVungTau},
	// Cần Thơ - 3 cơ sở
	{"Trường Mầm non Việt Mỹ Cần Thơ - Ninh Kiều", models.RegionCanTho},
	{"Trường Mầm non Việt Mỹ Cần Thơ - Hưng Phú", models.RegionCanTho},
	{"Trường Phổ thông Việt Mỹ Cần Thơ - Hưng Phú", models.RegionCanTho},
}

// seedSchools ghi đè bảng schools bằng danh sách 16 cơ sở thật (đã được xác
// nhận với người yêu cầu) thay cho 5 cơ sở cũ trong migration 004. Chạy được
// an toàn ở đây (khác migration) vì luôn thực hiện SAU wipe() - lúc này
// artworks/students không còn tham chiếu FK nào tới schools, DELETE không
// bị RESTRICT chặn.
func (s *seeder) seedSchools() ([]*models.School, error) {
	if _, err := s.db.ExecContext(s.ctx, `DELETE FROM schools`); err != nil {
		return nil, fmt.Errorf("xoá schools cũ thất bại: %w", err)
	}

	now := time.Now()
	var result []*models.School
	for i, sc := range schoolSeeds {
		res, err := s.db.ExecContext(s.ctx,
			`INSERT INTO schools (name, region, display_order, is_active, created_at, updated_at) VALUES (?, ?, ?, 1, ?, ?)`,
			sc.Name, sc.Region, i+1, now, now,
		)
		if err != nil {
			return nil, fmt.Errorf("insert school %q thất bại: %w", sc.Name, err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			return nil, err
		}
		result = append(result, &models.School{
			ID: id, Name: sc.Name, Region: sc.Region, DisplayOrder: i + 1, IsActive: true,
			CreatedAt: now, UpdatedAt: now,
		})
	}
	log.Printf("Đã nạp %d trường (8 saigon, 5 vungtau, 3 cantho)", len(result))
	return result, nil
}

// awardSeed là cấu hình 1 giải mẫu. GradeLabel rỗng = giải dùng chung toàn
// hệ thống (grade_level_id NULL); có giá trị = tìm đúng khối lớp đó.
type awardSeed struct {
	Name      string
	Slug      string
	GradeNum  int // 0 = giải chung, không theo khối
	RankOrder int
	ColorHex  string
}

// RankOrder 0-based theo đúng quy ước trang admin/awards (vị trí kéo-thả
// đầu danh sách = 0) - trang /bang-vang suy ra Nhất/Nhì/Ba bằng rank_order+1
// nên "Giải Nhất" phải là phần tử đầu tiên (0), không phải giải đứng thứ 2.
var awardSeeds = []awardSeed{
	{"Giải Nhất", "nhat", 0, 0, "#d4af37"},
	{"Giải Nhì", "nhi", 0, 1, "#a8a9ad"},
	{"Giải Ba", "ba", 0, 2, "#a97142"},
	{"Giải Đặc biệt", "dac-biet", 0, 3, "#c49c57"},
	{"Giải Khuyến khích", "khuyen-khich", 0, 4, "#725139"},
}

// seedAwards tạo vài giải dùng chung toàn hệ thống (đủ để demo bảng vàng)
// thay vì sinh đủ bộ giải theo từng khối - việc đó là thao tác nghiệp vụ
// admin tự làm qua /admin/awards, không phải việc của dữ liệu demo.
func (s *seeder) seedAwards(grades []*models.GradeLevel) ([]*models.Award, error) {
	_ = grades // giữ tham số cho khả năng mở rộng seed giải theo khối sau này
	var result []*models.Award
	for _, sd := range awardSeeds {
		created, err := s.awardRepo.Create(s.ctx, &models.Award{
			Name:      sd.Name,
			Slug:      sd.Slug,
			RankOrder: sd.RankOrder,
			ColorHex:  sd.ColorHex,
			IsActive:  true,
		})
		if err != nil {
			return nil, fmt.Errorf("tạo giải %q thất bại: %w", sd.Name, err)
		}
		result = append(result, created)
	}
	log.Printf("Đã nạp %d giải thưởng dùng chung", len(result))
	return result, nil
}

type topicSeed struct {
	Name           string
	Slug           string
	ColorHex       string
	EducationLevel *string
}

func edu(level string) *string { return &level }

var topicSeeds = []topicSeed{
	{"Trí tưởng tượng & thế giới thần tiên", "tri-tuong-tuong-the-gioi-than-tien", "#725139", edu(models.EducationLevelPrimary)},
	{"Niềm vui, tình bạn, cảm xúc học đường", "niem-vui-tinh-ban-cam-xuc-hoc-duong", "#3f7d5c", edu(models.EducationLevelPrimary)},
	{"Gia đình & cuộc sống thường ngày", "gia-dinh-cuoc-song-thuong-ngay", "#c49c57", nil},
	{"Việt Mỹ trong mắt em", "viet-my-trong-mat-em", "#4a6fa5", edu(models.EducationLevelSecondary)},
	{"Ước mơ tương lai", "uoc-mo-tuong-lai", "#a8548c", edu(models.EducationLevelSecondary)},
}

func (s *seeder) seedTopicCategories() ([]*models.TopicCategory, error) {
	var result []*models.TopicCategory
	for i, sd := range topicSeeds {
		created, err := s.topicRepo.Create(s.ctx, &models.TopicCategory{
			Name:           sd.Name,
			Slug:           sd.Slug,
			ColorHex:       sd.ColorHex,
			EducationLevel: sd.EducationLevel,
			DisplayOrder:   i + 1,
			IsActive:       true,
		})
		if err != nil {
			return nil, fmt.Errorf("tạo nhóm chủ đề %q thất bại: %w", sd.Name, err)
		}
		result = append(result, created)
	}
	log.Printf("Đã nạp %d nhóm chủ đề sáng tạo", len(result))
	return result, nil
}

// vietnameseGivenNames/vietnameseFamilyNames dựng tên học sinh giả ngẫu
// nhiên - chỉ để demo giao diện, không phải danh sách thật.
var vietnameseFamilyNames = []string{"Nguyễn", "Trần", "Lê", "Phạm", "Hoàng", "Huỳnh", "Phan", "Vũ", "Võ", "Đặng", "Bùi", "Đỗ"}
var vietnameseMiddleGivenNames = []string{"Văn", "Thị", "Hữu", "Minh", "Ngọc", "Thanh", "Gia", "Bảo", "Khánh", "Anh"}
var vietnameseGivenNames = []string{"An", "Bình", "Chi", "Dũng", "Hà", "Khôi", "Lan", "Linh", "Long", "Mai", "Nam", "Nhi", "Phúc", "Quân", "Thảo", "Trang", "Tuấn", "Uyên", "Vy", "Yến"}

func randomStudentName() string {
	return fmt.Sprintf("%s %s %s",
		randomFrom(vietnameseFamilyNames), randomFrom(vietnameseMiddleGivenNames), randomFrom(vietnameseGivenNames))
}

var artworkTitleTemplates = []string{
	"Ước mơ của em", "Ngôi trường thân yêu", "Gia đình hạnh phúc", "Khu vườn cổ tích",
	"Biển đảo quê hương", "Mùa xuân trên phố", "Bạn thân của em", "Chú mèo tinh nghịch",
	"Ngày hội trăng rằm", "Cô giáo của em", "Thế giới trong mơ", "Buổi sáng trên đồng quê",
	"Chuyến du hành vũ trụ", "Lễ hội mùa hè", "Bức tranh gia đình", "Vườn hoa rực rỡ",
	"Chiếc thuyền giấy", "Bầu trời đêm sao", "Khu phố em ở", "Ngày hè rực rỡ",
	"Cánh diều tuổi thơ", "Hồ nước trong xanh", "Rừng cây kỳ diệu", "Con đường đến trường",
	"Chợ hoa ngày Tết", "Đêm hội hoa đăng", "Cầu vồng sau mưa", "Vườn trái cây mùa hè",
	"Ông mặt trời và em", "Ngôi nhà mơ ước", "Chim én báo xuân", "Chuyến đi dã ngoại",
	"Bức tranh mùa thu", "Lớp học của em", "Chú robot thông minh", "Đàn cá tung tăng",
	"Núi rừng hùng vĩ", "Cánh đồng lúa chín", "Bến cảng nhộn nhịp", "Xe đạp của em",
	"Buổi biểu diễn văn nghệ", "Bà và cháu", "Vũ trụ muôn màu", "Khu chợ quê",
	"Chuyến tàu tuổi thơ", "Cây cầu nối bờ vui", "Đàn bướm mùa xuân", "Sân chơi của em",
	"Ngày khai giảng", "Vườn thú kỳ thú", "Chiếc diều no gió", "Ngôi làng nhỏ",
	"Buổi hoàng hôn trên biển", "Đêm Trung thu rộn ràng", "Cầu thang lên mây", "Chú chim non tập bay",
	"Bãi biển đầy nắng", "Khu rừng cổ tích", "Trạm vũ trụ tương lai", "Người bạn bốn chân",
	"Buổi sáng đầu tuần", "Vườn rau của bà", "Cơn mưa mùa hạ", "Đường phố đêm Giáng sinh",
	"Chiếc thuyền ra khơi", "Ngày hội thể thao", "Bức tường đầy màu sắc", "Chuyến thám hiểm rừng xanh",
	"Chợ Tết quê em", "Ngôi sao may mắn", "Khinh khí cầu bay lượn", "Đàn ong chăm chỉ",
	"Mùa gặt vàng", "Con phố nhỏ ngày mưa", "Người lính cứu hoả dũng cảm", "Chuyến xe buýt đến trường",
	"Hội chợ mùa xuân", "Đêm trăng rằm bên sông", "Vườn hoa hướng dương",
}

// shuffledTitles trả về n tên tác phẩm không trùng nhau, lấy ngẫu nhiên
// (không hoàn lại) từ artworkTitleTemplates - tránh nhiều tác phẩm hiện tên
// y hệt nhau trên trang public, khác với cách cũ gắn thêm hậu tố "#số" để
// đảm bảo duy nhất. Yêu cầu n <= len(artworkTitleTemplates); vượt quá thì
// panic sớm lúc chạy seed thay vì âm thầm sinh tên rỗng.
func shuffledTitles(n int) []string {
	if n > len(artworkTitleTemplates) {
		log.Fatalf("cần %d tên tác phẩm nhưng artworkTitleTemplates chỉ có %d - bổ sung thêm mẫu", n, len(artworkTitleTemplates))
	}
	shuffled := make([]string, len(artworkTitleTemplates))
	copy(shuffled, artworkTitleTemplates)
	rand.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
	return shuffled[:n]
}

// seedArtworks upload từng ảnh trong entries lên S3 rồi tạo 1 tác phẩm ứng
// với đúng ảnh đó (không lặp lại ảnh) - mỗi tác phẩm gán ngẫu nhiên 1
// trường + 1 khối lớp cùng cấp học, thỉnh thoảng có nhóm chủ đề, thỉnh
// thoảng có giải, và một phần được đánh dấu is_featured để trang "tiêu
// biểu" có dữ liệu.
func (s *seeder) seedArtworks(entries []imageFile, schools []*models.School, grades []*models.GradeLevel, topics []*models.TopicCategory, awards []*models.Award) ([]*models.Artwork, error) {
	if len(schools) == 0 || len(grades) == 0 {
		return nil, fmt.Errorf("cần ít nhất 1 trường và 1 khối lớp để seed tác phẩm")
	}

	titles := shuffledTitles(len(entries))

	var result []*models.Artwork
	for i, img := range entries {
		grade := randomFrom(grades)
		school := schoolForRegion(schools, grade)

		uploadResp, err := s.uploadOriginal(img)
		if err != nil {
			log.Printf("Bỏ qua ảnh %s: upload S3 thất bại: %v", img.Name, err)
			continue
		}

		width, height := 0, 0

		var topicID *int64
		if len(topics) > 0 && rand.Intn(100) < 70 {
			// Chỉ chọn trong các nhóm khớp cấp học của khối, hoặc nhóm dùng chung.
			candidates := filterTopicsByEducationLevel(topics, grade.EducationLevel)
			if len(candidates) > 0 {
				id := randomFrom(candidates).ID
				topicID = &id
			}
		}

		var awardIDs []int64
		if len(awards) > 0 && rand.Intn(100) < 25 {
			awardIDs = []int64{randomFrom(awards).ID}
		}

		title := titles[i]

		created, err := s.artworkSvc.CreateArtworkFromUpload(s.ctx, service.CreateArtworkRequest{
			Title:           title,
			StudentName:     randomStudentName(),
			SchoolID:        school.ID,
			GradeLevelID:    grade.ID,
			TopicCategoryID: topicID,
			S3Key:           uploadResp.Key,
			S3URL:           uploadResp.URL,
			FileSize:        int64(len(img.Data)),
			Width:           width,
			Height:          height,
			AwardIDs:        awardIDs,
		})
		if err != nil {
			log.Printf("Bỏ qua ảnh %s: tạo artwork thất bại: %v", img.Name, err)
			continue
		}

		// Tiêu biểu: khoảng 15% tác phẩm, đủ để trang "/tac-pham-tieu-bieu" có
		// dữ liệu mà không biến toàn bộ danh sách thành tiêu biểu.
		if rand.Intn(100) < 15 {
			if err := s.artworkSvc.SetFeatured(s.ctx, created.ID, true); err != nil {
				log.Printf("Không đặt tiêu biểu cho artwork %d: %v", created.ID, err)
			} else {
				created.IsFeatured = true
			}
		}

		result = append(result, created)
		log.Printf("[%d/%d] Đã tạo tác phẩm %q (id=%d) từ %s", i+1, len(entries), title, created.ID, img.Name)
	}

	return result, nil
}

// uploadOriginal đẩy ảnh gốc lên S3 qua chính S3Repository - không qua
// UploadService.UploadImage (service đó cần *os.File tạm + luồng validate
// dành cho request HTTP thật), vì seed script đã tự đọc file an toàn từ đĩa.
func (s *seeder) uploadOriginal(img imageFile) (*models.UploadResponse, error) {
	key := buildSeedKey(s.cfg.AWS.BasePath, img.Name)
	contentType := contentTypeFor(img.Data, img.Name)

	url, err := s.s3Repo.Upload(s.ctx, s.cfg.AWS.BucketName, key, bytes.NewReader(img.Data), contentType, s.cfg.AWS.UseACL)
	if err != nil {
		return nil, err
	}
	return &models.UploadResponse{Key: key, URL: url}, nil
}

func buildSeedKey(basePath, filename string) string {
	// Timestamp + tên gốc: đủ để tránh trùng giữa các lần seed liên tiếp mà
	// vẫn dễ nhận ra ảnh nào ứng với ảnh nguồn nào lúc debug.
	name := fmt.Sprintf("seed-%d-%s", time.Now().UnixNano(), filename)
	if basePath == "" {
		return name
	}
	return basePath + "/" + name
}

// schoolForRegion chọn ngẫu nhiên 1 trường bất kỳ trong danh sách - việc gán
// đúng "trường nào thuộc cấp học nào" không áp dụng ở model hiện tại (1
// trường có thể dạy nhiều cấp), nên chọn ngẫu nhiên là hợp lý cho dữ liệu demo.
func schoolForRegion(schools []*models.School, _ *models.GradeLevel) *models.School {
	return randomFrom(schools)
}

func filterTopicsByEducationLevel(topics []*models.TopicCategory, level string) []*models.TopicCategory {
	var out []*models.TopicCategory
	for _, t := range topics {
		if t.EducationLevel == nil || *t.EducationLevel == level {
			out = append(out, t)
		}
	}
	return out
}

var sampleDisplayNames = []string{"Phụ huynh Khối 3", "Cô giáo chủ nhiệm", "Bạn học cùng lớp", "Người xem ẩn danh", "Cựu học sinh Việt Mỹ"}
var sampleComments = []string{
	"Bức tranh đẹp quá, con vẽ rất có hồn!",
	"Màu sắc hài hoà, ý tưởng sáng tạo lắm.",
	"Chúc mừng em đã hoàn thành tác phẩm thật ấn tượng.",
	"Nhìn bức tranh mà nhớ tuổi thơ ghê.",
	"Cố gắng phát huy nhé, con vẽ giỏi lắm!",
}
var reactionTypes = []string{"like", "love", "haha", "wow", "sad", "angry"}

// seedEngagement rải reaction/comment mẫu lên một phần tác phẩm vừa tạo, đủ
// để trang public không trống trơn khi demo tương tác - không phải toàn bộ
// tác phẩm đều có tương tác, giống thực tế.
func (s *seeder) seedEngagement(artworks []*models.Artwork) error {
	reactionCount, commentCount := 0, 0
	for _, a := range artworks {
		if rand.Intn(100) < 60 {
			n := 1 + rand.Intn(3)
			for i := 0; i < n; i++ {
				token := fmt.Sprintf("seed-visitor-%d-%d", a.ID, i)
				rt := randomFrom(reactionTypes)
				if err := s.reactionRepo.Upsert(s.ctx, a.ID, rt, token, "127.0.0.1"); err != nil {
					log.Printf("Không tạo được reaction mẫu cho artwork %d: %v", a.ID, err)
					continue
				}
				reactionCount++
			}
		}
		if rand.Intn(100) < 30 {
			_, err := s.commentRepo.Create(s.ctx, &models.ArtworkComment{
				ArtworkID:    a.ID,
				DisplayName:  randomFrom(sampleDisplayNames),
				Content:      randomFrom(sampleComments),
				VisitorToken: fmt.Sprintf("seed-visitor-comment-%d", a.ID),
				IPAddress:    "127.0.0.1",
			})
			if err != nil {
				log.Printf("Không tạo được comment mẫu cho artwork %d: %v", a.ID, err)
				continue
			}
			commentCount++
		}
	}
	log.Printf("Đã nạp %d reaction và %d comment mẫu", reactionCount, commentCount)
	return nil
}
