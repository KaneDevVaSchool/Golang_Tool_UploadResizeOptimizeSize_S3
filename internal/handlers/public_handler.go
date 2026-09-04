package handlers

import (
	"encoding/json"
	"html"
	"net/http"
	"strconv"
	"strings"

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

// HandleListArtworks GET /api/v1/public/artworks?region=&grade_level_id=&education_level=&page=&page_size=
// Luôn ép is_published=true - không public/trả về draft.
func (h *PublicHandler) HandleListArtworks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.SendError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Chỉ hỗ trợ GET")
		return
	}

	published := true
	filter := models.ArtworkFilter{
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
	h.SendSuccess(w, comments)
}

type commentRequestBody struct {
	DisplayName  string `json:"display_name"`
	Content      string `json:"content"`
	VisitorToken string `json:"visitor_token"`
}

// HandleCreateComment POST /api/v1/public/artworks/{id}/comments - ẩn danh,
// chỉ yêu cầu display_name tự nhập (không xác thực danh tính thật). Nội
// dung được escape qua html.EscapeString để chống XSS khi FE render lại
// (double bảo vệ - React tự escape khi render text, nhưng escape ở backend
// đảm bảo dữ liệu lưu DB cũng an toàn nếu có nơi khác đọc trực tiếp).
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
		ArtworkID:    artworkID,
		DisplayName:  html.EscapeString(displayName),
		Content:      html.EscapeString(content),
		VisitorToken: visitorToken,
		IPAddress:    middleware.GetClientIP(r),
	}

	created, err := h.commentRepo.Create(r.Context(), comment)
	if err != nil {
		h.SendError(w, http.StatusInternalServerError, "COMMENT_FAILED", "Không gửi được bình luận")
		return
	}
	h.SendSuccess(w, created)
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

	published := true
	var entries []billboardEntry
	for _, award := range awards {
		filter := models.ArtworkFilter{IsPublished: &published, AwardID: &award.ID, Page: 1, PageSize: 100}
		result, err := h.artworkService.ListArtworks(r.Context(), filter)
		if err != nil {
			continue
		}
		for _, item := range result.Items {
			entries = append(entries, billboardEntry{ArtworkWithMeta: item, Award: *award})
		}
	}

	h.SendSuccess(w, entries)
}
