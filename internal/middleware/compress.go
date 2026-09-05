package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

// minCompressSize - dưới ngưỡng này gzip thường làm response TO HƠN (header
// gzip ~18 byte + block overhead) và tốn CPU vô ích. 1KB là ngưỡng quen thuộc,
// cùng mức nginx dùng mặc định.
const minCompressSize = 1024

// compressibleTypes - chỉ nén thứ thực sự co lại được. Ảnh/video/font woff2
// đã nén sẵn trong định dạng, gzip thêm chỉ đốt CPU mà gần như không giảm byte.
var compressibleTypes = []string{
	"text/html",
	"text/css",
	"text/plain",
	"text/xml",
	"application/javascript",
	"application/json",
	"application/xml",
	"image/svg+xml",
}

// gzipWriterPool tái dùng gzip.Writer giữa các request - mỗi writer giữ sẵn
// window buffer 32KB, cấp phát mới cho từng request sẽ tạo áp lực GC không cần thiết.
var gzipWriterPool = sync.Pool{
	New: func() any {
		w, _ := gzip.NewWriterLevel(io.Discard, gzip.BestSpeed)
		return w
	},
}

func isCompressible(contentType string) bool {
	// Content-Type thường kèm charset ("text/css; charset=utf-8") nên so khớp tiền tố.
	if i := strings.IndexByte(contentType, ';'); i >= 0 {
		contentType = contentType[:i]
	}
	contentType = strings.TrimSpace(strings.ToLower(contentType))
	for _, t := range compressibleTypes {
		if contentType == t {
			return true
		}
	}
	return false
}

// gzipResponseWriter hoãn quyết định nén tới lúc biết Content-Type và kích
// thước thật của body.
//
// Không thể quyết ngay ở đầu request vì lúc đó handler chưa đặt Content-Type,
// mà cũng không thể đệm toàn bộ body (file 950KB sẽ nằm hết trong RAM). Cách
// làm: đệm tối đa minCompressSize byte đầu tiên, tới đó là đã đủ dữ liệu để
// quyết định - body ngắn hơn ngưỡng thì ghi thẳng không nén, dài hơn thì bật
// gzip và stream phần còn lại.
type gzipResponseWriter struct {
	http.ResponseWriter

	gz       *gzip.Writer
	buf      []byte
	decided  bool // đã chọn xong nén hay không?
	compress bool
	status   int
	wroteHdr bool
}

func (w *gzipResponseWriter) WriteHeader(status int) {
	if w.wroteHdr {
		return
	}
	w.status = status
	// Chưa gọi ResponseWriter.WriteHeader ở đây: Content-Encoding phải được
	// đặt TRƯỚC khi header đi ra, mà lúc này còn chưa biết có nén hay không.
	// Việc gửi header thật dời sang decide().
	w.wroteHdr = true
}

// decide chốt nén/không nén rồi gửi header đi. Gọi đúng một lần.
func (w *gzipResponseWriter) decide(bodyLargerThanBuf bool) {
	if w.decided {
		return
	}
	w.decided = true

	header := w.ResponseWriter.Header()
	// 204/304 không có body; Content-Encoding trên chúng là sai và gây lỗi ở
	// một số proxy. Response đã tự nén (ví dụ file .gz) cũng để nguyên.
	noBody := w.status == http.StatusNoContent || w.status == http.StatusNotModified
	alreadyEncoded := header.Get("Content-Encoding") != ""

	w.compress = bodyLargerThanBuf &&
		!noBody &&
		!alreadyEncoded &&
		isCompressible(header.Get("Content-Type"))

	if w.compress {
		header.Set("Content-Encoding", "gzip")
		// Content-Length của body gốc không còn đúng sau khi nén; để trống
		// thì Go tự dùng chunked encoding.
		header.Del("Content-Length")
		w.gz = gzipWriterPool.Get().(*gzip.Writer)
		w.gz.Reset(w.ResponseWriter)
	} else if !bodyLargerThanBuf {
		// Body ngắn và đã nằm trọn trong buffer -> biết chính xác độ dài,
		// đặt lại Content-Length để client không phải dùng chunked.
		if header.Get("Content-Length") == "" {
			header.Set("Content-Length", strconv.Itoa(len(w.buf)))
		}
	}

	if w.status == 0 {
		w.status = http.StatusOK
	}
	w.ResponseWriter.WriteHeader(w.status)
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	if !w.wroteHdr {
		w.WriteHeader(http.StatusOK)
	}

	if !w.decided {
		// Còn đang gom đủ dữ liệu để quyết định.
		if len(w.buf)+len(b) < minCompressSize {
			w.buf = append(w.buf, b...)
			return len(b), nil
		}
		// Đã vượt ngưỡng -> chốt nén, xả buffer rồi ghi tiếp bình thường.
		w.buf = append(w.buf, b...)
		w.decide(true)
		pending := w.buf
		w.buf = nil
		if _, err := w.writeOut(pending); err != nil {
			return 0, err
		}
		return len(b), nil
	}

	return w.writeOut(b)
}

func (w *gzipResponseWriter) writeOut(b []byte) (int, error) {
	if w.compress {
		return w.gz.Write(b)
	}
	return w.ResponseWriter.Write(b)
}

// finish xả nốt phần còn treo. Bắt buộc gọi sau khi handler trả về, kể cả khi
// handler không ghi byte nào (ví dụ 204) - lúc đó header vẫn cần được gửi.
func (w *gzipResponseWriter) finish() {
	if !w.decided {
		w.decide(false)
		if len(w.buf) > 0 {
			_, _ = w.ResponseWriter.Write(w.buf)
		}
		w.buf = nil
		return
	}
	if w.compress && w.gz != nil {
		_ = w.gz.Close()
		gzipWriterPool.Put(w.gz)
		w.gz = nil
	}
}

// Flush - giữ cho handler nào cần đẩy dữ liệu sớm vẫn hoạt động khi bị bọc.
func (w *gzipResponseWriter) Flush() {
	if !w.decided {
		w.decide(len(w.buf) >= minCompressSize)
		if len(w.buf) > 0 {
			_, _ = w.writeOut(w.buf)
			w.buf = nil
		}
	}
	if w.compress && w.gz != nil {
		_ = w.gz.Flush()
	}
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// GzipMiddleware nén response bằng gzip khi client hỗ trợ.
//
// Bundle của trang này là ~950KB JS + ~180KB CSS gửi thô; gzip đưa về khoảng
// một phần tư, và đây là thay đổi rẻ nhất trong toàn bộ đợt tối ưu tốc độ tải.
//
// Đặt Vary: Accept-Encoding trên MỌI response (kể cả không nén) để cache dùng
// chung không phục vụ nhầm bản đã nén cho client không hiểu gzip.
func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Accept-Encoding")

		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		gw := &gzipResponseWriter{ResponseWriter: w, buf: make([]byte, 0, minCompressSize)}
		defer gw.finish()
		next.ServeHTTP(gw, r)
	})
}
