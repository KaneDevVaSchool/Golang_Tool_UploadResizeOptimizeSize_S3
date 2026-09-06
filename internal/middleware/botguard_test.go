package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func doRequest(h http.Handler, path, userAgent, remoteAddr string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodGet, path, nil)
	r.RemoteAddr = remoteAddr
	if userAgent != "" {
		r.Header.Set("User-Agent", userAgent)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestBotGuard_ChanCongCuTaiHangLoat(t *testing.T) {
	guard := NewBotGuard(DefaultBotGuardConfig())
	defer guard.Stop()
	h := BotGuardMiddleware(guard)(okHandler())

	// Đây là lý do tính năng tồn tại - yêu cầu trực tiếp là chặn WinHTTrack.
	agents := []string{
		"Mozilla/4.0 (compatible; MSIE 8.0; Windows NT 6.0) HTTrack 3.0",
		"Wget/1.21.3",
		"curl/8.4.0",
		"Python-urllib/3.11",
		"Scrapy/2.11 (+https://scrapy.org)",
		"Teleport Pro/1.29",
	}

	for _, ua := range agents {
		w := doRequest(h, "/tac-pham-tieu-bieu", ua, "203.0.113.5:1000")
		if w.Code != http.StatusForbidden {
			t.Errorf("User-Agent %q phải bị chặn 403, nhận được %d", ua, w.Code)
		}
	}
}

func TestBotGuard_KhongChanBotTimKiemHopLe(t *testing.T) {
	guard := NewBotGuard(DefaultBotGuardConfig())
	defer guard.Stop()
	h := BotGuardMiddleware(guard)(okHandler())

	// Khoá lại quyết định của P2.15: SEO phải tiếp tục hoạt động. Nếu ai đó
	// thêm "bot"/"crawler" chung chung vào scraperAgentMarkers, test này đổ.
	agents := []string{
		"Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)",
		"Mozilla/5.0 (compatible; bingbot/2.0; +http://www.bing.com/bingbot.htm)",
		"facebookexternalhit/1.1 (+http://www.facebook.com/externalhit_uatext.php)",
		"Mozilla/5.0 (compatible; coccocbot-web/1.0; +http://help.coccoc.com/searchengine)",
		"Twitterbot/1.0",
	}

	for _, ua := range agents {
		w := doRequest(h, "/tac-pham-tieu-bieu", ua, "203.0.113.6:1000")
		if w.Code != http.StatusOK {
			t.Errorf("bot hợp lệ %q phải đi lọt, nhận được %d", ua, w.Code)
		}
	}
}

func TestBotGuard_TrinhDuyetThatDiLot(t *testing.T) {
	guard := NewBotGuard(DefaultBotGuardConfig())
	defer guard.Stop()
	h := BotGuardMiddleware(guard)(okHandler())

	chrome := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0 Safari/537.36"
	w := doRequest(h, "/phong-trien-lam", chrome, "203.0.113.7:1000")

	if w.Code != http.StatusOK {
		t.Fatalf("trình duyệt thật phải đi lọt, nhận được %d", w.Code)
	}
}

func TestBotGuard_ChanKhiQuetQuaNhieuDuongDan(t *testing.T) {
	cfg := DefaultBotGuardConfig()
	cfg.MaxDistinctPaths = 10
	cfg.MaxRequests = 10000 // cô lập: chỉ xét tiêu chí số đường dẫn
	guard := NewBotGuard(cfg)
	defer guard.Stop()
	h := BotGuardMiddleware(guard)(okHandler())

	chrome := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/120.0 Safari/537.36"
	blocked := false
	for i := 0; i < 20; i++ {
		w := doRequest(h, fmt.Sprintf("/tranh/%d", i), chrome, "203.0.113.8:1000")
		if w.Code == http.StatusTooManyRequests {
			blocked = true
			break
		}
	}

	if !blocked {
		t.Fatal("quét qua nhiều đường dẫn khác nhau phải bị chặn 429")
	}
}

func TestBotGuard_KhongChanKhiXemLaiCungTrang(t *testing.T) {
	cfg := DefaultBotGuardConfig()
	cfg.MaxDistinctPaths = 10
	guard := NewBotGuard(cfg)
	defer guard.Stop()
	h := BotGuardMiddleware(guard)(okHandler())

	// Người thật tải lại cùng một trang nhiều lần: số đường dẫn KHÁC NHAU vẫn
	// là 1, nên không được coi là máy quét dù số request cao.
	chrome := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/120.0 Safari/537.36"
	for i := 0; i < 30; i++ {
		w := doRequest(h, "/phong-trien-lam", chrome, "203.0.113.11:1000")
		if w.Code != http.StatusOK {
			t.Fatalf("tải lại cùng một trang không được bị chặn (lần %d, mã %d)", i, w.Code)
		}
	}
}

func TestBotGuard_MienTruSitemapVaRobots(t *testing.T) {
	guard := NewBotGuard(DefaultBotGuardConfig())
	defer guard.Stop()
	h := BotGuardMiddleware(guard)(okHandler())

	// Hai đường này tồn tại để phục vụ crawler; chặn chúng là tự phá SEO.
	// Kiểm bằng chính User-Agent nằm trong danh sách chặn.
	for _, path := range []string{"/robots.txt", "/sitemap.xml", "/api/v1/health"} {
		w := doRequest(h, path, "Wget/1.21", "203.0.113.9:1000")
		if w.Code != http.StatusOK {
			t.Errorf("%s phải luôn phục vụ được, nhận được %d", path, w.Code)
		}
	}
}

func TestBotGuard_ChanRequestKhongCoUserAgentTrenDuongHTML(t *testing.T) {
	guard := NewBotGuard(DefaultBotGuardConfig())
	defer guard.Stop()
	h := BotGuardMiddleware(guard)(okHandler())

	w := doRequest(h, "/phong-trien-lam", "", "203.0.113.10:1000")
	if w.Code != http.StatusForbidden {
		t.Fatalf("thiếu User-Agent trên đường HTML phải bị chặn, nhận được %d", w.Code)
	}

	// Nhưng đường API thì không: client tích hợp gọi bằng script là hợp lệ
	// theo API.md, miễn có API key.
	w = doRequest(h, "/api/v1/public/artworks", "", "203.0.113.10:1000")
	if w.Code != http.StatusOK {
		t.Fatalf("đường API không được chặn chỉ vì thiếu User-Agent, nhận được %d", w.Code)
	}
}

func TestBotGuard_GiuHinhPhatTrongThoiGianChan(t *testing.T) {
	cfg := DefaultBotGuardConfig()
	cfg.MaxRequests = 3
	cfg.Window = time.Minute
	cfg.BlockDuration = time.Minute
	guard := NewBotGuard(cfg)
	defer guard.Stop()

	key := "203.0.113.12"
	for i := 0; i < 4; i++ {
		guard.observe(key, "/a")
	}

	// Sau khi vượt ngưỡng, request tiếp theo vẫn bị chặn dù đổi đường dẫn:
	// hình phạt tính theo khoá IP, không phải theo từng URL.
	if !guard.observe(key, "/hoan-toan-khac") {
		t.Fatal("hình phạt phải còn hiệu lực trong BlockDuration")
	}
}
