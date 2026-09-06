package handlers

import (
	"encoding/json"
	"html"
	"html/template"
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
  <meta name="twitter:card" content="summary_large_image">
  <meta name="twitter:title" content="{{.Title}}">
  <meta name="twitter:description" content="{{.Description}}">
  <meta name="twitter:image" content="{{.ImageURL}}">
  <meta http-equiv="refresh" content="0;url={{.AppURL}}">
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
}

// PublicHandler expose API không cần đăng nhập cho trang public (danh sách
// tác phẩm, billboard vinh danh, reaction/comment/view ẩn danh). Ghi dữ
// liệu (reaction/comment) được bảo vệ bằng rate-limit riêng ở container.go
// (nghiêm hơn API chung), không dùng session/API key vì đây là tương tác
// công khai theo đúng yêu cầu "ẩn danh tự do".
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

// HandleListArtworks GET /api/v1/public/artworks?search=&region=&grade_level_id=&education_level=&page=&page_size=
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
	// region lọc theo school_id gián tiếp không khả dụng ở filter hiện có
	// (ArtworkFilter chỉ có SchoolID) - FE nên tự truyền school_id cụ thể
	// nếu cần lọc theo 1 trường; lọc theo cả region cần mở rộng filter sau
	// nếu có nhu cầu thực tế (hiện tại billboard/section 3 dùng school_id
	// đơn lẻ do metaHandler cung cấp danh sách trường theo region).
	if v, err := strconv.ParseInt(r.URL.Query().Get("school_id"), 10, 64); err == nil && v > 0 {
		filter.SchoolID = &v
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

	result, err := h.artworkService.ListArtworks(r.Context(), filter)
	if err != nil {
		h.SendError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Không tải được tác phẩm tiêu biểu")
		return
	}

	region := strings.TrimSpace(r.URL.Query().Get("region"))
	items := result.Items
	if region != "" {
		filtered := make([]*models.ArtworkWithMeta, 0, len(items))
		for _, item := range items {
			if item.Region == region {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}

	h.SendSuccess(w, map[string]any{"items": items, "total_count": len(items)})
}

// HandleGetArtwork GET /api/v1/public/artworks/{id} - kèm ghi nhận 1 lượt
// xem (chống đếm trùng trong 24h qua visitor_token gửi ở query param).
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
	if visitorToken != "" {
		if counted, err := h.viewRepo.RecordView(r.Context(), id, visitorToken); err == nil && counted {
			artwork.ViewCount++
		}
	}

	h.SendSuccess(w, artwork)
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

	scheme := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto"))
	if scheme != "http" && scheme != "https" {
		scheme = "http"
		if r.TLS != nil {
			scheme = "https"
		}
	}
	origin := &url.URL{Scheme: scheme, Host: r.Host}
	appURL := origin.ResolveReference(&url.URL{
		Path:     "/tac-pham-tieu-bieu",
		RawQuery: url.Values{"tranh": {strconv.FormatInt(id, 10)}}.Encode(),
	})
	shareURL := origin.ResolveReference(r.URL)
	imageURL, parseErr := url.Parse(artwork.S3URL)
	if parseErr != nil || !imageURL.IsAbs() {
		imageURL = origin.ResolveReference(&url.URL{Path: artwork.S3URL})
	}

	data := artworkSharePageData{
		Title:       "“" + artwork.Title + "” — " + artwork.StudentName,
		Description: "Ngắm tác phẩm “" + artwork.Title + "” của " + artwork.StudentName + " tại " + artwork.SchoolName + " trong Khu vườn nghệ thuật VA Schools.",
		ShareURL:    shareURL.String(),
		AppURL:      appURL.String(),
		ImageURL:    imageURL.String(),
		ImageAlt:    artwork.Title,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	if err := artworkShareTemplate.Execute(w, data); err != nil {
		http.Error(w, "Không thể mở trang chia sẻ", http.StatusInternalServerError)
	}
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
