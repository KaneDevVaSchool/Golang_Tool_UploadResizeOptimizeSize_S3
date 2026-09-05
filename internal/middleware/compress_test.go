package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// serve chạy body qua GzipMiddleware và trả về response đã ghi.
func serve(t *testing.T, acceptEncoding string, h http.HandlerFunc) *http.Response {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if acceptEncoding != "" {
		req.Header.Set("Accept-Encoding", acceptEncoding)
	}
	rec := httptest.NewRecorder()
	GzipMiddleware(h).ServeHTTP(rec, req)
	return rec.Result()
}

func TestGzipCompressesLargeCompressibleBody(t *testing.T) {
	body := strings.Repeat("nội dung tiếng Việt để nén ", 200) // vượt xa 1KB

	res := serve(t, "gzip", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		_, _ = io.WriteString(w, body)
	})

	if got := res.Header.Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("Content-Encoding = %q, muốn gzip", got)
	}
	// Content-Length của body gốc phải bị xoá, nếu không client sẽ đọc thiếu/thừa.
	if got := res.Header.Get("Content-Length"); got != "" {
		t.Errorf("Content-Length = %q, muốn rỗng khi đã nén", got)
	}

	zr, err := gzip.NewReader(res.Body)
	if err != nil {
		t.Fatalf("không đọc được gzip: %v", err)
	}
	got, err := io.ReadAll(zr)
	if err != nil {
		t.Fatalf("giải nén lỗi: %v", err)
	}
	if string(got) != body {
		t.Error("body sau khi giải nén khác body gốc")
	}
}

func TestGzipSkipsWhenClientDoesNotAccept(t *testing.T) {
	body := strings.Repeat("x", 5000)

	res := serve(t, "", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css")
		_, _ = io.WriteString(w, body)
	})

	if got := res.Header.Get("Content-Encoding"); got != "" {
		t.Errorf("Content-Encoding = %q, muốn rỗng khi client không nhận gzip", got)
	}
	got, _ := io.ReadAll(res.Body)
	if string(got) != body {
		t.Error("body bị đổi dù không nén")
	}
	// Vary vẫn phải có, nếu không cache chung sẽ phục vụ nhầm bản đã nén.
	if !strings.Contains(res.Header.Get("Vary"), "Accept-Encoding") {
		t.Error("thiếu Vary: Accept-Encoding")
	}
}

func TestGzipSkipsSmallBody(t *testing.T) {
	body := "quá ngắn để đáng nén"

	res := serve(t, "gzip", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, body)
	})

	if got := res.Header.Get("Content-Encoding"); got != "" {
		t.Errorf("Content-Encoding = %q, muốn rỗng với body dưới ngưỡng", got)
	}
	got, _ := io.ReadAll(res.Body)
	if string(got) != body {
		t.Errorf("body = %q, muốn %q", got, body)
	}
}

func TestGzipSkipsIncompressibleType(t *testing.T) {
	body := bytes.Repeat([]byte{0xFF, 0xD8, 0xFF}, 2000) // giả lập JPEG

	res := serve(t, "gzip", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write(body)
	})

	if got := res.Header.Get("Content-Encoding"); got != "" {
		t.Errorf("Content-Encoding = %q, muốn rỗng với ảnh đã nén sẵn", got)
	}
	got, _ := io.ReadAll(res.Body)
	if !bytes.Equal(got, body) {
		t.Error("body ảnh bị thay đổi")
	}
}

func TestGzipPreservesStatusCode(t *testing.T) {
	res := serve(t, "gzip", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, strings.Repeat("lỗi không tìm thấy ", 200))
	})

	if res.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, muốn 404", res.StatusCode)
	}
	if got := res.Header.Get("Content-Encoding"); got != "gzip" {
		t.Errorf("Content-Encoding = %q, muốn gzip (body lỗi vẫn nén được)", got)
	}
}

// 204 không có body - gắn Content-Encoding vào là sai và một số proxy sẽ lỗi.
func TestGzipSkipsNoContent(t *testing.T) {
	res := serve(t, "gzip", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	if res.StatusCode != http.StatusNoContent {
		t.Errorf("status = %d, muốn 204", res.StatusCode)
	}
	if got := res.Header.Get("Content-Encoding"); got != "" {
		t.Errorf("Content-Encoding = %q, muốn rỗng với 204", got)
	}
}

// Handler ghi nhiều lần quanh ngưỡng: phần đệm trước khi chốt quyết định phải
// được xả ra đủ và đúng thứ tự.
func TestGzipWriteAcrossThreshold(t *testing.T) {
	chunk := strings.Repeat("a", 400)
	want := strings.Repeat(chunk, 10) // 4000 byte, vượt ngưỡng ở lần ghi thứ 3

	res := serve(t, "gzip", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		for i := 0; i < 10; i++ {
			_, _ = io.WriteString(w, chunk)
		}
	})

	if got := res.Header.Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("Content-Encoding = %q, muốn gzip", got)
	}
	zr, err := gzip.NewReader(res.Body)
	if err != nil {
		t.Fatalf("không đọc được gzip: %v", err)
	}
	got, _ := io.ReadAll(zr)
	if string(got) != want {
		t.Errorf("độ dài body = %d, muốn %d", len(got), len(want))
	}
}

// Response đã tự nén sẵn không được nén chồng lần nữa.
func TestGzipSkipsAlreadyEncoded(t *testing.T) {
	res := serve(t, "gzip", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		w.Header().Set("Content-Encoding", "br")
		_, _ = io.WriteString(w, strings.Repeat("x", 5000))
	})

	if got := res.Header.Get("Content-Encoding"); got != "br" {
		t.Errorf("Content-Encoding = %q, muốn giữ nguyên br", got)
	}
}
