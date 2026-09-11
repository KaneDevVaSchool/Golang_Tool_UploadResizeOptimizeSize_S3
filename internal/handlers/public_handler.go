package handlers

import (
	"encoding/json"
	"encoding/xml"
	"html"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"s3-upload-tool/internal/middleware"
	"s3-upload-tool/internal/models"
	"s3-upload-tool/internal/repository"
	"s3-upload-tool/internal/service"
)

// maxCommentContentLength giới hạn độ dài bình luận - khớp cột
// artwork_comments.content VARCHAR(1000) trong migration 011.
const maxCommentContentLength = 1000

// maxDisplayNameLength giới hạn độ dài tên hiển thị ẩn danh - khớp cột
// artwork_comments.display_name VARCHAR(100) trong migration 011.
const maxDisplayNameLength = 100

var artworkShareTemplate = template.Must(template.New("artwork-share").Parse(`<!doctype html>
<html lang="vi">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <title>{{.Title}}</title>
  <meta name="description" content="{{.Description}}">
  <meta property="og:locale" content="vi_VN">
  <meta property="og:type" content="article">
  <meta property="og:site_name" content="Khu vườn nghệ thuật VA Schools">
  <meta property="og:title" content="{{.Title}}">
  <meta property="og:description" content="{{.Description}}">
  <meta property="og:url" content="{{.ShareURL}}">
  <meta property="og:image" content="{{.ImageURL}}">
  <meta property="og:image:secure_url" content="{{.ImageURL}}">
  <meta property="og:image:alt" content="{{.ImageAlt}}">
  {{if .ImageWidth}}<meta property="og:image:width" content="{{.ImageWidth}}">{{end}}
  {{if .ImageHeight}}<meta property="og:image:height" content="{{.ImageHeight}}">{{end}}
  <meta name="twitter:card" content="summary_large_image">
  <meta name="twitter:title" content="{{.Title}}">
  <meta name="twitter:description" content="{{.Description}}">
  <meta name="twitter:image" content="{{.ImageURL}}">
  <meta name="twitter:image:alt" content="{{.ImageAlt}}">
  <meta http-equiv="refresh" content="0;url={{.AppURL}}">
  <script type="application/ld+json">{{.BreadcrumbJSON}}</script>
</head>
<body>
  <p>Đang mở tác phẩm “{{.ImageAlt}}”… <a href="{{.AppURL}}">Xem tác phẩm</a></p>
</body>
</html>`))

type artworkSharePageData struct {
	Title       string
	Description string
	ShareURL    string
	AppURL      string
	ImageURL    string
	ImageAlt    string
	// ImageWidth/ImageHeight rỗng ("") khi tác phẩm cũ chưa có kích thước lưu
	// sẵn - template bỏ qua 2 thẻ og:image:width/height trong trường hợp đó
	// (chỉ thiếu tối ưu hiển thị preview, không phải lỗi).
	ImageWidth  string
	ImageHeight string
	// BreadcrumbJSON là JSON-LD BreadcrumbList đã escape qua encoding/json,
	// ép kiểu template.JS để html/template không escape thêm lần nữa trong
	// context <script> - an toàn vì encoding/json mặc định đã escape <, >, &
	// thành < v.v. nên tên tác phẩm chứa "</script>" không thoát được.
	BreadcrumbJSON template.JS
}

// PublicHandler expose API không cần đăng nhập cho trang public (danh sách
// tác phẩm, billboard vinh danh, reaction/comment/view ẩn danh). Ghi dữ
// liệu (reaction/comment) được bảo vệ bằng rate-limit riêng ở container.go
// (nghiêm hơn API chung), không dùng session/API key vì đây là tương tác
// công khai theo đúng yêu cầu "ẩn danh tự do".
//
// Không có endpoint tải ảnh gốc cho khách xem ẩn danh (đã gỡ có chủ đích) -
// xem docs/plan/03-risks.md. Khu quản trị vẫn tải được qua
// ArtworkHandler.HandleDownload, dùng cho việc quản lý tác phẩm/giải.
type PublicHandler struct {
	BaseHandler
	artworkService service.ArtworkService
	reactionRepo   repository.ReactionRepository
	commentRepo    repository.CommentRepository
	viewRepo       repository.ArtworkViewRepository
	awardRepo      repository.AwardRepository
}

func NewPublicHandler(
	artworkService service.ArtworkService,
	reactionRepo repository.ReactionRepository,
	commentRepo repository.CommentRepository,
	viewRepo repository.ArtworkViewRepository,
	awardRepo repository.AwardRepository,
) *PublicHandler {
	return &PublicHandler{
		artworkService: artworkService,
		reactionRepo:   reactionRepo,
		commentRepo:    commentRepo,
		viewRepo:       viewRepo,
		awardRepo:      awardRepo,
	}
}

// HandleListArtworks GET /api/v1/public/artworks?search=&region=&grade_level_id=&education_level=&topic_category_id=&page=&page_size=
// Luôn ép is_published=true - không public/trả về draft.
func (h *PublicHandler) HandleListArtworks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.SendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Chỉ hỗ trợ GET")
		return
	}

	published := true
	filter := models.ArtworkFilter{
		Search:         sanitizePublicSearch(r.URL.Query().Get("search")),
		EducationLevel: r.URL.Query().Get("education_level"),
		IsPublished:    &published,
		Page:           parseIntOrDefault(r.URL.Query().Get("page"), 1),
		PageSize:       parseIntOrDefault(r.URL.Query().Get("page_size"), 24),
	}
	if v, err := strconv.ParseInt(r.URL.Query().Get("grade_level_id"), 10, 64); err == nil && v > 0 {
		filter.GradeLevelID = &v
	}
	if v := strings.TrimSpace(r.URL.Query().Get("region")); models.IsKnownRegion(v) {
		filter.Region = &v
	}
	if v, err := strconv.ParseInt(r.URL.Query().Get("school_id"), 10, 64); err == nil && v > 0 {
		filter.SchoolID = &v
	}
	if v, err := strconv.ParseInt(r.URL.Query().Get("topic_category_id"), 10, 64); err == nil && v > 0 {
		filter.TopicCategoryID = &v
	}

	result, err := h.artworkService.ListArtworks(r.Context(), filter)
	if err != nil {
		h.SendError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Không tải được danh sách tác phẩm")
		return
	}

	h.SendSuccess(w, map[string]any{
		"items":       result.Items,
		"total_count": result.TotalCount,
		"page":        result.Page,
		"page_size":   result.PageSize,
	})
}

// HandleListFeatured GET /api/v1/public/artworks/featured?region=saigon|cantho|vungtau
// (region rỗng = tất cả khu vực). Dùng cho Section 3 trang public.
func (h *PublicHandler) HandleListFeatured(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.SendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Chỉ hỗ trợ GET")
		return
	}

	published := true
	featured := true
	filter := models.ArtworkFilter{
		IsPublished: &published,
		IsFeatured:  &featured,
		Page:        1,
		PageSize:    100,
	}
	if v := strings.TrimSpace(r.URL.Query().Get("region")); models.IsKnownRegion(v) {
		filter.Region = &v
	}

	result, err := h.artworkService.ListArtworks(r.Context(), filter)
	if err != nil {
		h.SendError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Không tải được tác phẩm tiêu biểu")
		return
	}

	h.SendSuccess(w, map[string]any{"items": result.Items, "total_count": result.TotalCount})
}

// HandleGetArtwork GET /api/v1/public/artworks/{id} - kèm ghi nhận 1 lượt
// xem (visitor_token trong query param, mỗi request hợp lệ +1).
func (h *PublicHandler) HandleGetArtwork(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		h.SendError(w, http.StatusBadRequest, "INVALID_ID", "ID tác phẩm không hợp lệ")
		return
	}

	artwork, err := h.artworkService.GetArtwork(r.Context(), id)
	if err != nil {
		h.SendError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Không tải được tác phẩm")
		return
	}
	if artwork == nil || !artwork.IsPublished {
		h.SendError(w, http.StatusNotFound, "NOT_FOUND", "Không tìm thấy tác phẩm")
		return
	}

	visitorToken := strings.TrimSpace(r.URL.Query().Get("visitor_token"))
	if visitorToken != "" && h.viewRepo != nil {
		if _, err := h.viewRepo.RecordView(r.Context(), id, visitorToken); err != nil {
			log.Printf("[public] RecordView artwork=%d: %v", id, err)
		}
		// Luôn đọc lại view_count từ DB sau khi ghi (tránh lệch bộ nhớ / race).
		if refreshed, err := h.artworkService.GetArtwork(r.Context(), id); err == nil && refreshed != nil {
			artwork = refreshed
		}
	}

	h.SendSuccess(w, artwork)
}

func artworkDownloadFileName(title, ext string) string {
	safe := strings.TrimSpace(title)
	safe = strings.Map(func(r rune) rune {
		switch r {
		case '<', '>', ':', '"', '/', '\\', '|', '?', '*':
			return '-'
		default:
			if r < 0x20 {
				return '-'
			}
			return r
		}
	}, safe)
	if safe == "" {
		safe = "tac-pham-vas"
	}
	return safe + ext
}

// logArtworkDownload ghi nhật ký một lượt tải ảnh gốc. Hiện chỉ
// ArtworkHandler.HandleDownload (admin) còn gọi hàm này - endpoint tải công
// khai đã bị gỡ (xem docs/plan/03-risks.md), nhưng models.ArtworkDownloadSource*
// vẫn giữ cả hai giá trị "admin"/"public" vì bảng artwork_downloads còn lưu
// lịch sử các lượt tải "public" từ trước khi gỡ.
// Đây là thao tác phụ trợ: lỗi ghi log chỉ log cảnh báo, KHÔNG được chặn
// việc trả ảnh về cho người tải (ảnh đã sẵn sàng ở phía gọi).
func logArtworkDownload(r *http.Request, svc service.ArtworkService, artworkID int64, source string) {
	download := &models.ArtworkDownload{
		ArtworkID: artworkID,
		Source:    source,
		IPAddress: middleware.GetClientIP(r),
		UserAgent: r.UserAgent(),
	}
	if admin := middleware.GetAdminUser(r.Context()); admin != nil {
		download.AdminUserID = &admin.ID
	}
	if err := svc.LogDownload(r.Context(), download); err != nil {
		log.Printf("[handlers] ghi nhật ký tải artwork %d thất bại: %v", artworkID, err)
	}
}

// HandleArtworkSharePage trả HTML có Open Graph ngay từ server để crawler
// Facebook đọc được ảnh preview. Trình duyệt người dùng được chuyển tiếp về
// SPA và mở đúng tác phẩm qua query ?tranh=<id>.
func (h *PublicHandler) HandleArtworkSharePage(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}

	artwork, err := h.artworkService.GetArtwork(r.Context(), id)
	if err != nil || artwork == nil || !artwork.IsPublished {
		http.NotFound(w, r)
		return
	}

	origin := resolveRequestOrigin(r)
	appURL := origin.ResolveReference(&url.URL{
		Path:     "/tac-pham-tieu-bieu",
		RawQuery: url.Values{"tranh": {strconv.FormatInt(id, 10)}}.Encode(),
	})
	shareURL := origin.ResolveReference(r.URL)
	imageURL, parseErr := url.Parse(artwork.S3URL)
	if parseErr != nil || !imageURL.IsAbs() {
		imageURL = origin.ResolveReference(&url.URL{Path: artwork.S3URL})
	}
	title := "“" + artwork.Title + "” — " + artwork.StudentName
	homeURL := origin.ResolveReference(&url.URL{Path: "/"})
	featuredURL := origin.ResolveReference(&url.URL{Path: "/tac-pham-tieu-bieu"})

	var imageWidth, imageHeight string
	if artwork.Width != nil {
		imageWidth = strconv.Itoa(*artwork.Width)
	}
	if artwork.Height != nil {
		imageHeight = strconv.Itoa(*artwork.Height)
	}

	breadcrumbJSON, err := json.Marshal(map[string]any{
		"@context": "https://schema.org",
		"@type":    "BreadcrumbList",
		"itemListElement": []map[string]any{
			{"@type": "ListItem", "position": 1, "name": "Trang chủ", "item": homeURL.String()},
			{"@type": "ListItem", "position": 2, "name": "Tác phẩm tiêu biểu", "item": featuredURL.String()},
			{"@type": "ListItem", "position": 3, "name": artwork.Title, "item": shareURL.String()},
		},
	})
	if err != nil {
		log.Printf("trang chia sẻ tác phẩm %d: không dựng được breadcrumb JSON-LD: %v", id, err)
		breadcrumbJSON = []byte("{}")
	}

	data := artworkSharePageData{
		Title:          title,
		Description:    "Ngắm tác phẩm “" + artwork.Title + "” của " + artwork.StudentName + " tại " + artwork.SchoolName + " trong Khu vườn nghệ thuật VA Schools.",
		ShareURL:       shareURL.String(),
		AppURL:         appURL.String(),
		ImageURL:       imageURL.String(),
		ImageAlt:       artwork.Title,
		ImageWidth:     imageWidth,
		ImageHeight:    imageHeight,
		BreadcrumbJSON: template.JS(breadcrumbJSON),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	if err := artworkShareTemplate.Execute(w, data); err != nil {
		http.Error(w, "Không thể mở trang chia sẻ", http.StatusInternalServerError)
	}
}

// resolveRequestOrigin suy ra scheme+host thật của request để dựng URL tuyệt
// đối - dùng chung cho trang chia sẻ, sitemap.xml và robots.txt. Đọc
// X-Forwarded-Proto vì sau reverse proxy (Nginx) r.TLS luôn nil; không thêm
// biến môi trường APP_URL/SITE_URL vì domain luôn suy ra được từ request.
func resolveRequestOrigin(r *http.Request) *url.URL {
	scheme := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto"))
	if scheme != "http" && scheme != "https" {
		scheme = "http"
		if r.TLS != nil {
			scheme = "https"
		}
	}
	return &url.URL{Scheme: scheme, Host: r.Host}
}

// sitemapURLEntry là 1 phần tử <url> trong sitemap.xml theo chuẩn
// http://www.sitemaps.org/schemas/sitemap/0.9. LastMod/ChangeFreq/Priority
// dùng omitempty vì spec cho phép thiếu các field này.
type sitemapURLEntry struct {
	XMLName    xml.Name `xml:"url"`
	Loc        string   `xml:"loc"`
	LastMod    string   `xml:"lastmod,omitempty"`
	ChangeFreq string   `xml:"changefreq,omitempty"`
	Priority   string   `xml:"priority,omitempty"`
}

type sitemapURLSet struct {
	XMLName xml.Name          `xml:"urlset"`
	Xmlns   string            `xml:"xmlns,attr"`
	URLs    []sitemapURLEntry `xml:"url"`
}

// HandleSitemap GET /sitemap.xml - liệt kê 4 trang public cố định + mọi tác
// phẩm đã publish (trỏ về /chia-se/tac-pham/{id}, không phải ?tranh={id} trên
// SPA, để Google index được nội dung không phụ thuộc JS và tránh trùng lặp
// nội dung giữa các biến thể query param). Không cần Search Console
// API/IndexNow - Google tự crawl lại theo lịch khi thấy <lastmod> mới.
func (h *PublicHandler) HandleSitemap(w http.ResponseWriter, r *http.Request) {
	origin := resolveRequestOrigin(r)

	staticEntries := []struct {
		path       string
		priority   string
		changefreq string
	}{
		{"/", "1.0", "daily"},
		{"/tac-pham-tieu-bieu", "0.8", "daily"},
		{"/phong-trien-lam", "0.8", "daily"},
		{"/bang-vang", "0.7", "weekly"},
	}

	urlSet := sitemapURLSet{Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9"}
	for _, e := range staticEntries {
		loc := origin.ResolveReference(&url.URL{Path: e.path})
		urlSet.URLs = append(urlSet.URLs, sitemapURLEntry{
			Loc:        loc.String(),
			Priority:   e.priority,
			ChangeFreq: e.changefreq,
		})
	}

	// Lỗi lấy danh sách tác phẩm là lỗi phụ trợ: sitemap vẫn trả về hữu ích
	// với 4 URL tĩnh thay vì lỗi 500 toàn bộ - cùng tinh thần buildVariants.
	artworks, err := h.artworkService.ListPublishedForSitemap(r.Context())
	if err != nil {
		log.Printf("sitemap.xml: không lấy được danh sách tác phẩm, chỉ trả URL tĩnh: %v", err)
	}
	for _, a := range artworks {
		loc := origin.ResolveReference(&url.URL{Path: "/chia-se/tac-pham/" + strconv.FormatInt(a.ID, 10)})
		urlSet.URLs = append(urlSet.URLs, sitemapURLEntry{
			Loc:        loc.String(),
			LastMod:    a.UpdatedAt.Format(time.RFC3339),
			Priority:   "0.6",
			ChangeFreq: "monthly",
		})
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=900")
	w.Write([]byte(xml.Header))
	if err := xml.NewEncoder(w).Encode(urlSet); err != nil {
		log.Printf("sitemap.xml: lỗi ghi response: %v", err)
	}
}

// HandleRobotsTxt GET /robots.txt - cho phép crawl toàn bộ trang public, chặn
// khu quản trị/API/OAuth (vệ sinh index, không phải bảo mật), trỏ tới
// sitemap.xml bằng URL tuyệt đối suy từ request.
func (h *PublicHandler) HandleRobotsTxt(w http.ResponseWriter, r *http.Request) {
	origin := resolveRequestOrigin(r)
	sitemapURL := origin.ResolveReference(&url.URL{Path: "/sitemap.xml"})

	body := "User-agent: *\n" +
		"Allow: /\n" +
		"Disallow: /admin/\n" +
		"Disallow: /api/\n" +
		"Disallow: /auth/\n" +
		"\n" +
		"Sitemap: " + sitemapURL.String() + "\n"

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Write([]byte(body))
}

type reactionRequestBody struct {
	ReactionType string `json:"reaction_type"`
	VisitorToken string `json:"visitor_token"`
}

// HandleAddReaction POST /api/v1/public/artworks/{id}/reactions
func (h *PublicHandler) HandleAddReaction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.SendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Chỉ hỗ trợ POST")
		return
	}
	artworkID, ok := parsePathID(r, "id")
	if !ok {
		h.SendError(w, http.StatusBadRequest, "INVALID_ID", "ID tác phẩm không hợp lệ")
		return
	}

	var body reactionRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.SendError(w, http.StatusBadRequest, "INVALID_BODY", "Dữ liệu gửi lên không hợp lệ")
		return
	}
	if !models.ValidReactionTypes[body.ReactionType] {
		h.SendError(w, http.StatusBadRequest, "INVALID_REACTION_TYPE", "Loại cảm xúc không hợp lệ")
		return
	}
	visitorToken := strings.TrimSpace(body.VisitorToken)
	if visitorToken == "" {
		h.SendError(w, http.StatusBadRequest, "MISSING_VISITOR_TOKEN", "Thiếu định danh trình duyệt")
		return
	}

	ip := middleware.GetClientIP(r)
	if err := h.reactionRepo.Upsert(r.Context(), artworkID, body.ReactionType, visitorToken, ip); err != nil {
		h.SendError(w, http.StatusInternalServerError, "REACTION_FAILED", "Không ghi nhận được cảm xúc")
		return
	}

	counts, err := h.reactionRepo.CountByArtwork(r.Context(), artworkID)
	if err != nil {
		h.SendError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Không tải được số lượt cảm xúc")
		return
	}
	h.SendSuccess(w, map[string]any{"reaction_counts": counts})
}

// HandleRemoveReaction DELETE /api/v1/public/artworks/{id}/reactions/{type}
func (h *PublicHandler) HandleRemoveReaction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		h.SendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Chỉ hỗ trợ DELETE")
		return
	}
	artworkID, ok := parsePathID(r, "id")
	if !ok {
		h.SendError(w, http.StatusBadRequest, "INVALID_ID", "ID tác phẩm không hợp lệ")
		return
	}
	reactionType := r.PathValue("type")
	if !models.ValidReactionTypes[reactionType] {
		h.SendError(w, http.StatusBadRequest, "INVALID_REACTION_TYPE", "Loại cảm xúc không hợp lệ")
		return
	}
	visitorToken := strings.TrimSpace(r.URL.Query().Get("visitor_token"))
	if visitorToken == "" {
		h.SendError(w, http.StatusBadRequest, "MISSING_VISITOR_TOKEN", "Thiếu định danh trình duyệt")
		return
	}

	if err := h.reactionRepo.Remove(r.Context(), artworkID, reactionType, visitorToken); err != nil {
		h.SendError(w, http.StatusInternalServerError, "REACTION_FAILED", "Không gỡ được cảm xúc")
		return
	}

	counts, err := h.reactionRepo.CountByArtwork(r.Context(), artworkID)
	if err != nil {
		h.SendError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Không tải được số lượt cảm xúc")
		return
	}
	h.SendSuccess(w, map[string]any{"reaction_counts": counts})
}

// HandleListComments GET /api/v1/public/artworks/{id}/comments - không bao
// giờ trả comment is_hidden=1 (chỉ trang quản trị mới xem được, nếu cần
// thêm sau).
func (h *PublicHandler) HandleListComments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.SendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Chỉ hỗ trợ GET")
		return
	}
	artworkID, ok := parsePathID(r, "id")
	if !ok {
		h.SendError(w, http.StatusBadRequest, "INVALID_ID", "ID tác phẩm không hợp lệ")
		return
	}

	comments, err := h.commentRepo.ListByArtwork(r.Context(), artworkID, false)
	if err != nil {
		h.SendError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Không tải được bình luận")
		return
	}
	visitorToken := strings.TrimSpace(r.URL.Query().Get("visitor_token"))
	publicComments := make([]publicCommentResponse, 0, len(comments))
	for _, comment := range comments {
		publicComments = append(publicComments, toPublicComment(comment, visitorToken))
	}
	h.SendSuccess(w, publicComments)
}

type commentRequestBody struct {
	DisplayName  string `json:"display_name"`
	Content      string `json:"content"`
	VisitorToken string `json:"visitor_token"`
}

type publicCommentResponse struct {
	ID          int64     `json:"id"`
	ArtworkID   int64     `json:"artwork_id"`
	DisplayName string    `json:"display_name"`
	Content     string    `json:"content"`
	CreatedAt   time.Time `json:"created_at"`
	CanDelete   bool      `json:"can_delete"`
}

func toPublicComment(comment *models.ArtworkComment, visitorToken string) publicCommentResponse {
	return publicCommentResponse{
		ID:        comment.ID,
		ArtworkID: comment.ArtworkID,
		// Dữ liệu cũ từng được HTML-escape trước khi lưu khiến React hiển thị
		// nguyên chuỗi &amp;/&quot;. Unescape khi trả JSON; React vẫn tự escape
		// lúc render nên không mở lại lỗ hổng XSS.
		DisplayName: html.UnescapeString(comment.DisplayName),
		Content:     html.UnescapeString(comment.Content),
		CreatedAt:   comment.CreatedAt,
		CanDelete:   visitorToken != "" && visitorToken == comment.VisitorToken,
	}
}

// HandleCreateComment POST /api/v1/public/artworks/{id}/comments - ẩn danh,
// chỉ yêu cầu display_name tự nhập (không xác thực danh tính thật). Lưu text
// nguyên bản; JSON encoder và React escape ở đúng output context.
func (h *PublicHandler) HandleCreateComment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.SendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Chỉ hỗ trợ POST")
		return
	}
	artworkID, ok := parsePathID(r, "id")
	if !ok {
		h.SendError(w, http.StatusBadRequest, "INVALID_ID", "ID tác phẩm không hợp lệ")
		return
	}

	var body commentRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.SendError(w, http.StatusBadRequest, "INVALID_BODY", "Dữ liệu gửi lên không hợp lệ")
		return
	}

	displayName := strings.TrimSpace(body.DisplayName)
	content := strings.TrimSpace(body.Content)
	visitorToken := strings.TrimSpace(body.VisitorToken)

	if displayName == "" {
		h.SendError(w, http.StatusBadRequest, "MISSING_DISPLAY_NAME", "Vui lòng nhập tên hiển thị")
		return
	}
	if len([]rune(displayName)) > maxDisplayNameLength {
		h.SendError(w, http.StatusBadRequest, "DISPLAY_NAME_TOO_LONG", "Tên hiển thị quá dài")
		return
	}
	if content == "" {
		h.SendError(w, http.StatusBadRequest, "MISSING_CONTENT", "Vui lòng nhập nội dung bình luận")
		return
	}
	if len([]rune(content)) > maxCommentContentLength {
		h.SendError(w, http.StatusBadRequest, "CONTENT_TOO_LONG", "Bình luận quá dài")
		return
	}
	if visitorToken == "" {
		h.SendError(w, http.StatusBadRequest, "MISSING_VISITOR_TOKEN", "Thiếu định danh trình duyệt")
		return
	}

	comment := &models.ArtworkComment{
		ArtworkID: artworkID,
		// Lưu text nguyên bản. JSON + React chịu trách nhiệm encode/escape ở
		// đúng output context, tránh double-escape làm lỗi dấu và ký tự.
		DisplayName:  displayName,
		Content:      content,
		VisitorToken: visitorToken,
		IPAddress:    middleware.GetClientIP(r),
	}

	created, err := h.commentRepo.Create(r.Context(), comment)
	if err != nil {
		h.SendError(w, http.StatusInternalServerError, "COMMENT_FAILED", "Không gửi được bình luận")
		return
	}
	h.SendSuccess(w, toPublicComment(created, visitorToken))
}

// HandleDeleteComment chỉ cho trình duyệt đã tạo bình luận xoá bằng đúng
// visitor_token. Không trả thông tin phân biệt "không tồn tại" và
// "không thuộc sở hữu" để tránh dò quyền sở hữu bình luận.
func (h *PublicHandler) HandleDeleteComment(w http.ResponseWriter, r *http.Request) {
	artworkID, ok := parsePathID(r, "id")
	if !ok {
		h.SendError(w, http.StatusBadRequest, "INVALID_ID", "ID tác phẩm không hợp lệ")
		return
	}
	commentID, ok := parsePathID(r, "commentID")
	if !ok {
		h.SendError(w, http.StatusBadRequest, "INVALID_COMMENT_ID", "ID bình luận không hợp lệ")
		return
	}
	visitorToken := strings.TrimSpace(r.URL.Query().Get("visitor_token"))
	if visitorToken == "" {
		h.SendError(w, http.StatusBadRequest, "MISSING_VISITOR_TOKEN", "Thiếu định danh trình duyệt")
		return
	}

	deleted, err := h.commentRepo.DeleteOwned(r.Context(), commentID, artworkID, visitorToken)
	if err != nil {
		h.SendError(w, http.StatusInternalServerError, "COMMENT_DELETE_FAILED", "Không xoá được bình luận")
		return
	}
	if !deleted {
		h.SendError(w, http.StatusNotFound, "COMMENT_NOT_FOUND", "Không tìm thấy bình luận có thể xoá")
		return
	}
	h.SendSuccess(w, map[string]bool{"deleted": true})
}

// billboardEntry là 1 tác phẩm đạt giải kèm thông tin giải - dùng cho
// Section 5 (billboard vinh danh) trang public.
type billboardEntry struct {
	*models.ArtworkWithMeta
	Award models.Award `json:"award"`
}

// HandleBillboard GET /api/v1/public/billboard - danh sách tác phẩm có
// giải, sắp theo rank_order (Nhất trước, Khuyến khích sau).
func (h *PublicHandler) HandleBillboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.SendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Chỉ hỗ trợ GET")
		return
	}

	awards, err := h.awardRepo.List(r.Context(), false)
	if err != nil {
		h.SendError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Không tải được danh sách giải thưởng")
		return
	}

	// Lấy MỘT lần mọi tác phẩm có giải, thay vì lặp từng giải rồi gọi
	// ListArtworks cho mỗi giải: mỗi lượt gọi đó vốn đã kéo theo cả loạt truy
	// vấn ghép metadata, nên với 4 giải là gấp bốn toàn bộ chi phí đó.
	// ListArtworks đã trả kèm Awards của từng tác phẩm nên đủ dữ liệu để tự
	// nhóm lại bên dưới.
	published := true
	hasAward := true
	result, err := h.artworkService.ListArtworks(r.Context(), models.ArtworkFilter{
		IsPublished: &published,
		HasAward:    &hasAward,
		Page:        1,
		PageSize:    maxBillboardArtworks,
	})
	if err != nil {
		h.SendError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Không tải được bảng vinh danh")
		return
	}

	byAward := make(map[int64][]*models.ArtworkWithMeta, len(awards))
	for _, item := range result.Items {
		for _, aw := range item.Awards {
			byAward[aw.ID] = append(byAward[aw.ID], item)
		}
	}

	// Duyệt theo danh sách giải (awardRepo.List đã sắp theo rank_order) để giữ
	// nguyên thứ tự cũ: Nhất trước, Khuyến khích sau. Một tác phẩm mang nhiều
	// giải vẫn xuất hiện một lần cho mỗi giải, đúng như trước.
	entries := make([]billboardEntry, 0, len(result.Items))
	for _, award := range awards {
		for _, item := range byAward[award.ID] {
			entries = append(entries, billboardEntry{ArtworkWithMeta: item, Award: *award})
		}
	}

	h.SendSuccess(w, entries)
}

// maxBillboardArtworks giới hạn số tác phẩm dựng bảng vinh danh trong một
// lượt. Trần cũ là 100 cho mỗi giải; giữ ở mức tương đương cho tổng số vì
// bảng vinh danh vốn chỉ gồm các tác phẩm đoạt giải.
const maxBillboardArtworks = 100

// maxPublicSearchLength khớp maxlength của ô tìm trên /phong-trien-lam.
const maxPublicSearchLength = 80

func sanitizePublicSearch(raw string) string {
	search := strings.TrimSpace(raw)
	if runes := []rune(search); len(runes) > maxPublicSearchLength {
		return string(runes[:maxPublicSearchLength])
	}
	return search
}
