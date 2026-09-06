package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
)

// Vì sao file này tồn tại: trước đây GetClientIP tin tuyệt đối vào header
// X-Forwarded-For do client gửi. Bất kỳ ai cũng chỉ cần thêm
// "X-Forwarded-For: <ngẫu nhiên>" vào mỗi request là có một "IP" mới, nên
// toàn bộ rate limit theo IP (kể cả bộ 20 req/phút cho bình luận) trở thành
// vô hiệu - đúng thứ đáng lo nhất khi đối tượng người dùng là học sinh.
//
// Sửa bằng nguyên tắc chuẩn của reverse proxy: chỉ đọc header chuyển tiếp khi
// chặng kết nối trực tiếp (RemoteAddr) nằm trong danh sách proxy tin cậy.
// Kiến trúc dự án là Nginx trên cùng máy proxy sang 127.0.0.1:8080, nên mặc
// định tin loopback là đủ và an toàn.

// trustedProxies giữ danh sách CIDR được phép đặt X-Forwarded-For/X-Real-IP.
// Mặc định loopback vì Nginx chạy cùng máy (xem deploy/nginx/*.conf).
var (
	trustedProxyMu   sync.RWMutex
	trustedProxyNets = defaultTrustedProxyNets()
)

func defaultTrustedProxyNets() []*net.IPNet {
	nets := make([]*net.IPNet, 0, 2)
	for _, cidr := range []string{"127.0.0.0/8", "::1/128"} {
		if _, n, err := net.ParseCIDR(cidr); err == nil {
			nets = append(nets, n)
		}
	}
	return nets
}

// SetTrustedProxies cấu hình lại dải proxy tin cậy từ danh sách CIDR (hoặc IP
// đơn lẻ, tự hiểu là /32 và /128). Giá trị không phân giải được bị bỏ qua và
// trả về trong danh sách lỗi để nơi gọi ghi log - không làm sập khởi động, vì
// một dòng cấu hình sai không đáng đánh đổi bằng việc server không lên.
//
// Danh sách rỗng nghĩa là KHÔNG tin proxy nào: luôn dùng RemoteAddr. Đó là
// trạng thái an toàn nhất, dùng khi app phơi trực tiếp ra Internet.
func SetTrustedProxies(cidrs []string) []string {
	var invalid []string
	nets := make([]*net.IPNet, 0, len(cidrs))

	for _, raw := range cidrs {
		entry := strings.TrimSpace(raw)
		if entry == "" {
			continue
		}

		if _, n, err := net.ParseCIDR(entry); err == nil {
			nets = append(nets, n)
			continue
		}

		// Cho phép ghi IP trần trong cấu hình - người vận hành thường viết
		// "10.0.0.5" chứ không viết "10.0.0.5/32".
		if ip := net.ParseIP(entry); ip != nil {
			bits := 32
			if ip.To4() == nil {
				bits = 128
			}
			nets = append(nets, &net.IPNet{
				IP:   ip,
				Mask: net.CIDRMask(bits, bits),
			})
			continue
		}

		invalid = append(invalid, entry)
	}

	trustedProxyMu.Lock()
	trustedProxyNets = nets
	trustedProxyMu.Unlock()

	return invalid
}

// isTrustedProxy kiểm tra IP của chặng kết nối trực tiếp.
func isTrustedProxy(ip net.IP) bool {
	if ip == nil {
		return false
	}

	trustedProxyMu.RLock()
	defer trustedProxyMu.RUnlock()

	for _, n := range trustedProxyNets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// remoteIP tách IP khỏi RemoteAddr. Không dùng LastIndex(":") như bản cũ vì
// địa chỉ IPv6 ("[::1]:54321") có nhiều dấu hai chấm.
func remoteIP(r *http.Request) net.IP {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	return net.ParseIP(strings.TrimSpace(host))
}

// GetClientIP trả IP thật của client dưới dạng chuỗi đã chuẩn hoá.
//
// Chỉ đọc X-Forwarded-For/X-Real-IP khi request đến từ proxy tin cậy; ngược
// lại dùng thẳng RemoteAddr. Trong X-Forwarded-For, duyệt từ PHẢI sang TRÁI và
// lấy IP đầu tiên không thuộc dải tin cậy: phần bên trái do client tự bịa,
// phần bên phải do các proxy tin cậy nối thêm nên mới đáng tin.
func GetClientIP(r *http.Request) string {
	direct := remoteIP(r)

	if !isTrustedProxy(direct) {
		if direct == nil {
			return strings.TrimSpace(r.RemoteAddr)
		}
		return direct.String()
	}

	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		for i := len(parts) - 1; i >= 0; i-- {
			ip := net.ParseIP(strings.TrimSpace(parts[i]))
			if ip == nil {
				continue
			}
			if !isTrustedProxy(ip) {
				return ip.String()
			}
		}
	}

	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		if ip := net.ParseIP(strings.TrimSpace(xri)); ip != nil {
			return ip.String()
		}
	}

	if direct == nil {
		return strings.TrimSpace(r.RemoteAddr)
	}
	return direct.String()
}

// ClientIPKey gom IP về khoá dùng cho bộ đếm rate limit.
//
// IPv6 cấp cho người dùng cả một khối /64 trở lên, nên đếm theo địa chỉ đầy đủ
// là vô nghĩa: đổi sang địa chỉ khác trong cùng khối là có bộ đếm mới. Gom về
// /64 khiến một thuê bao IPv6 tính chung một bộ đếm, tương đương cách IPv4
// tính theo một địa chỉ.
func ClientIPKey(r *http.Request) string {
	raw := GetClientIP(r)
	ip := net.ParseIP(raw)
	if ip == nil {
		return raw
	}

	if v4 := ip.To4(); v4 != nil {
		return v4.String()
	}

	masked := ip.Mask(net.CIDRMask(64, 128))
	return masked.String() + "/64"
}
