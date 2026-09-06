package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

// Chống tải trọn site (WinHTTrack, wget -r, HTTPrint...) mà không cản trở
// bot tìm kiếm hợp lệ.
//
// Bối cảnh: tranh của học sinh là nội dung dễ bị lấy nguyên khối rồi đăng lại
// nơi khác. Một lần chạy HTTrack có thể kéo về toàn bộ ảnh trong vài phút, và
// rate limit chung 100 req/phút không chặn được vì công cụ này cho phép hẹn
// nhịp chậm hơn ngưỡng đó.
//
// Ba lớp, xếp theo mức độ chắc chắn giảm dần - lớp nào chắc hơn thì chặn
// thẳng, lớp mơ hồ hơn chỉ siết tốc độ:
//
//  1. User-Agent tự khai là công cụ tải hàng loạt  → chặn thẳng (403)
//  2. Không có User-Agent trên đường HTML          → chặn thẳng (403)
//  3. Nhịp duyệt giống máy quét                    → 429, tự hết sau một lúc
//
// Điều KHÔNG làm: chặn theo "thiếu Referer" hay "thiếu Accept-Language". Trình
// duyệt thật vẫn thiếu các header đó trong nhiều tình huống hợp lệ (mở thẳng
// link, thiết lập riêng tư), chặn theo đó là chặn nhầm người xem thật.

// scraperAgentMarkers là các chuỗi con (đã hạ chữ thường) chỉ xuất hiện ở công
// cụ tải site/thu thập dữ liệu. Cố tình KHÔNG có "bot"/"crawler"/"spider"
// chung chung: Googlebot, Bingbot, và mọi bot mạng xã hội đều chứa các chuỗi
// đó, chặn theo sẽ giết luôn SEO vừa dựng ở P2.15.
var scraperAgentMarkers = []string{
	"httrack",     // WinHTTrack - yêu cầu trực tiếp
	"webcopier",   // WebCopier
	"webzip",      // WebZIP
	"teleport",    // Teleport Pro
	"offline explorer",
	"webreaper",
	"web downloader",
	"sitesucker",
	"webstripper",
	"wget",
	"curl",
	"libwww-perl",
	"python-requests",
	"python-urllib",
	"scrapy",
	"go-http-client",
	"java/",
	"apache-httpclient",
	"okhttp",
	"axios/",
	"node-fetch",
	"httpie",
	"aiohttp",
	"http_request2",
	"zgrab",       // quét lỗ hổng diện rộng
	"masscan",
	"nikto",
	"sqlmap",
	"nmap",
	"dirbuster",
	"gobuster",
	"feroxbuster",
	"wpscan",
}

// legitBotMarkers là bot được miễn hoàn toàn lớp phát hiện theo nhịp: chúng
// crawl nhanh và rộng theo đúng bản chất công việc, không được coi là tấn công.
//
// Không xác minh ngược DNS: chi phí một truy vấn DNS trên đường phục vụ request
// lớn hơn lợi ích ở quy mô một website sự kiện, và hệ quả xấu nhất của việc
// giả mạo ở đây chỉ là được miễn giới hạn nhịp - kẻ giả mạo vẫn dính lớp 1 nếu
// dùng công cụ tải hàng loạt, vẫn dính rate limit chung theo IP.
var legitBotMarkers = []string{
	"googlebot",
	"google-inspectiontool",
	"storebot-google",
	"bingbot",
	"slurp",            // Yahoo
	"duckduckbot",
	"baiduspider",
	"yandexbot",
	"coccocbot",        // Cốc Cốc - trình duyệt phổ biến ở Việt Nam
	"applebot",
	"facebookexternalhit",
	"facebookcatalog",
	"twitterbot",
	"linkedinbot",
	"zalobot",          // Zalo sinh preview khi chia sẻ link
	"telegrambot",
	"whatsapp",
	"discordbot",
	"skypeuripreview",
	"pinterest",
	"redditbot",
	"embedly",
	"ia_archiver",
}

// BotGuardConfig gom các ngưỡng của lớp phát hiện theo nhịp.
type BotGuardConfig struct {
	// Window là khoảng thời gian quan sát hành vi.
	Window time.Duration
	// MaxRequests là số request tối đa trong Window trước khi bị coi là quét.
	// Đặt rộng hơn nhiều so với người xem thật: một trang gallery tải khoảng
	// 30-60 ảnh, nên người dùng bình thường vẫn ở xa ngưỡng này.
	MaxRequests int
	// MaxDistinctPaths là số đường dẫn KHÁC NHAU tối đa trong Window. Đây mới
	// là dấu hiệu đặc trưng của trình tải site: người thật xem đi xem lại vài
	// trang, còn máy quét đi qua mỗi URL đúng một lần rồi sang URL mới.
	MaxDistinctPaths int
	// BlockDuration là thời gian giữ hình phạt sau khi vượt ngưỡng.
	BlockDuration time.Duration
	// CleanupInterval là nhịp dọn bộ nhớ theo dõi.
	CleanupInterval time.Duration
}

// DefaultBotGuardConfig chọn ngưỡng theo lưu lượng thực tế của trang: một phiên
// xem tranh sôi nổi vào khoảng 60-120 request/phút (ảnh biến thể, API, asset),
// nên 240 request hoặc 150 URL khác nhau trong một phút là mức người dùng bằng
// trình duyệt gần như không chạm tới.
func DefaultBotGuardConfig() BotGuardConfig {
	return BotGuardConfig{
		Window:           time.Minute,
		MaxRequests:      240,
		MaxDistinctPaths: 150,
		BlockDuration:    10 * time.Minute,
		CleanupInterval:  5 * time.Minute,
	}
}

type botTracker struct {
	mu           sync.Mutex
	windowStart  time.Time
	requests     int
	paths        map[string]struct{}
	blockedUntil time.Time
	lastSeen     time.Time
}

// BotGuard theo dõi nhịp truy cập theo khoá IP.
type BotGuard struct {
	cfg      BotGuardConfig
	mu       sync.RWMutex
	trackers map[string]*botTracker
	stop     chan struct{}
	stopOnce sync.Once
}

// NewBotGuard tạo bộ chống quét và chạy goroutine dọn bộ nhớ, theo đúng mẫu
// vòng đời của RateLimiter (Stop() gọi trong Container.Shutdown).
func NewBotGuard(cfg BotGuardConfig) *BotGuard {
	if cfg.Window <= 0 {
		cfg.Window = time.Minute
	}
	if cfg.MaxRequests <= 0 {
		cfg.MaxRequests = 240
	}
	if cfg.MaxDistinctPaths <= 0 {
		cfg.MaxDistinctPaths = 150
	}
	if cfg.BlockDuration <= 0 {
		cfg.BlockDuration = 10 * time.Minute
	}
	if cfg.CleanupInterval <= 0 {
		cfg.CleanupInterval = 5 * time.Minute
	}

	g := &BotGuard{
		cfg:      cfg,
		trackers: make(map[string]*botTracker),
		stop:     make(chan struct{}),
	}
	go g.cleanup()
	return g
}

// Stop dừng goroutine dọn bộ nhớ. An toàn khi gọi nhiều lần.
func (g *BotGuard) Stop() {
	g.stopOnce.Do(func() { close(g.stop) })
}

func (g *BotGuard) cleanup() {
	ticker := time.NewTicker(g.cfg.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-g.stop:
			return
		case <-ticker.C:
			now := time.Now()
			cutoff := g.cfg.Window*2 + g.cfg.BlockDuration

			g.mu.Lock()
			for key, t := range g.trackers {
				t.mu.Lock()
				idle := now.Sub(t.lastSeen) > cutoff
				stillBlocked := now.Before(t.blockedUntil)
				t.mu.Unlock()

				if idle && !stillBlocked {
					delete(g.trackers, key)
				}
			}
			g.mu.Unlock()
		}
	}
}

func (g *BotGuard) tracker(key string) *botTracker {
	g.mu.RLock()
	t, ok := g.trackers[key]
	g.mu.RUnlock()
	if ok {
		return t
	}

	g.mu.Lock()
	defer g.mu.Unlock()
	if t, ok := g.trackers[key]; ok {
		return t
	}
	t = &botTracker{
		windowStart: time.Now(),
		paths:       make(map[string]struct{}),
		lastSeen:    time.Now(),
	}
	g.trackers[key] = t
	return t
}

// observe ghi nhận một request và trả về true nếu khoá này đang bị coi là máy quét.
func (g *BotGuard) observe(key, path string) bool {
	t := g.tracker(key)

	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	t.lastSeen = now

	if now.Before(t.blockedUntil) {
		return true
	}

	if now.Sub(t.windowStart) > g.cfg.Window {
		t.windowStart = now
		t.requests = 0
		// Cấp map mới thay vì xoá từng khoá: map của một máy quét có thể lên
		// tới hàng nghìn phần tử, cấp lại rẻ hơn và trả bộ nhớ cũ về GC ngay.
		t.paths = make(map[string]struct{})
	}

	t.requests++
	// Chặn trần số đường dẫn ghi nhớ để bản thân cơ chế phòng thủ không trở
	// thành đường làm cạn RAM: qua ngưỡng là đã đủ kết luận, không cần đếm tiếp.
	if len(t.paths) <= g.cfg.MaxDistinctPaths*2 {
		t.paths[path] = struct{}{}
	}

	if t.requests > g.cfg.MaxRequests || len(t.paths) > g.cfg.MaxDistinctPaths {
		t.blockedUntil = now.Add(g.cfg.BlockDuration)
		return true
	}

	return false
}

// IsLegitBot cho biết User-Agent thuộc nhóm bot tìm kiếm/mạng xã hội hợp lệ.
func IsLegitBot(userAgent string) bool {
	ua := strings.ToLower(userAgent)
	for _, marker := range legitBotMarkers {
		if strings.Contains(ua, marker) {
			return true
		}
	}
	return false
}

// IsScraperAgent cho biết User-Agent tự khai là công cụ tải hàng loạt.
func IsScraperAgent(userAgent string) bool {
	ua := strings.ToLower(userAgent)
	for _, marker := range scraperAgentMarkers {
		if strings.Contains(ua, marker) {
			return true
		}
	}
	return false
}

// BotGuardMiddleware áp ba lớp bảo vệ đã mô tả ở đầu file.
//
// Chỉ áp cho đường public. Các đường sau được bỏ qua hoàn toàn:
//   - /api/v1/health   : giám sát và load balancer phải luôn thăm dò được
//   - /robots.txt, /sitemap.xml : chính là thứ để phục vụ crawler
//   - /api/v1/admin/*  : đã có session, và admin dùng công cụ dòng lệnh là
//     việc bình thường (vd script đối soát dữ liệu)
func BotGuardMiddleware(guard *BotGuard) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if guard == nil || isBotGuardExempt(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			ua := r.Header.Get("User-Agent")

			// Bot hợp lệ đi thẳng: chúng crawl nhanh theo đúng bản chất công việc.
			if IsLegitBot(ua) {
				next.ServeHTTP(w, r)
				return
			}

			// Lớp 1 - tự khai là công cụ tải hàng loạt.
			if IsScraperAgent(ua) {
				writeScraperBlocked(w)
				return
			}

			// Lớp 2 - không có User-Agent trên đường duyệt HTML. Mọi trình duyệt
			// đều gửi header này; thiếu nó gần như chắc chắn là script tự viết.
			// Chỉ áp cho đường HTML, không áp cho /api/: client tích hợp hợp lệ
			// gọi API bằng script là việc được phép (xem API.md).
			if strings.TrimSpace(ua) == "" && !strings.HasPrefix(r.URL.Path, "/api/") {
				writeScraperBlocked(w)
				return
			}

			// Lớp 3 - nhịp truy cập giống máy quét.
			if guard.observe(ClientIPKey(r), r.URL.Path) {
				WriteRateLimited(w, guard.cfg.BlockDuration)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func isBotGuardExempt(path string) bool {
	switch path {
	case "/robots.txt", "/sitemap.xml", "/api/v1/health":
		return true
	}
	return strings.HasPrefix(path, "/api/v1/admin/") ||
		strings.HasPrefix(path, "/auth/")
}

func writeScraperBlocked(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusForbidden)
	_, _ = w.Write([]byte(`{"success":false,"error":{"code":"AUTOMATED_ACCESS_BLOCKED","message":"Yêu cầu bị từ chối. Nội dung tác phẩm của học sinh không được phép tải hàng loạt bằng công cụ tự động."}}`))
}
