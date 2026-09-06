package middleware

import (
	"net/http"
	"testing"
)

// Các test dưới đây khoá lại một quyết định dễ bị vô tình đảo ngược khi ai đó
// "sửa cho tiện": đọc thẳng X-Forwarded-For mà không xét nguồn. Làm vậy là mở
// lại đúng lỗ hổng khiến mọi rate limit theo IP trở thành vô hiệu.

func newRequest(remoteAddr string, headers map[string]string) *http.Request {
	r, _ := http.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = remoteAddr
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	return r
}

func withTrustedProxies(t *testing.T, cidrs []string) {
	t.Helper()
	original := append([]string(nil), "127.0.0.0/8", "::1/128")
	SetTrustedProxies(cidrs)
	t.Cleanup(func() { SetTrustedProxies(original) })
}

func TestGetClientIP_KhongTinHeaderTuNguonLa(t *testing.T) {
	withTrustedProxies(t, []string{"127.0.0.0/8"})

	// Client kết nối trực tiếp (không qua proxy tin cậy) tự khai một IP giả.
	r := newRequest("203.0.113.9:41234", map[string]string{
		"X-Forwarded-For": "1.2.3.4",
		"X-Real-IP":       "5.6.7.8",
	})

	if got := GetClientIP(r); got != "203.0.113.9" {
		t.Fatalf("phải bỏ qua header giả và dùng RemoteAddr, nhận được %q", got)
	}
}

func TestGetClientIP_TinHeaderKhiQuaProxyTinCay(t *testing.T) {
	withTrustedProxies(t, []string{"127.0.0.0/8"})

	r := newRequest("127.0.0.1:8080", map[string]string{
		"X-Forwarded-For": "198.51.100.7",
	})

	if got := GetClientIP(r); got != "198.51.100.7" {
		t.Fatalf("qua proxy tin cậy phải lấy IP từ X-Forwarded-For, nhận được %q", got)
	}
}

func TestGetClientIP_ChonIPPhaiNhatNgoaiDaiTinCay(t *testing.T) {
	withTrustedProxies(t, []string{"127.0.0.0/8", "10.0.0.0/8"})

	// Client bịa thêm phần bên trái; chỉ hai chặng bên phải là do proxy thật nối.
	r := newRequest("127.0.0.1:8080", map[string]string{
		"X-Forwarded-For": "1.1.1.1, 198.51.100.7, 10.0.0.5",
	})

	if got := GetClientIP(r); got != "198.51.100.7" {
		t.Fatalf("phải lấy IP ngoài dải tin cậy ở gần proxy nhất, nhận được %q", got)
	}
}

func TestGetClientIP_DanhSachProxyRongThiKhongTinGiCa(t *testing.T) {
	withTrustedProxies(t, nil)

	r := newRequest("127.0.0.1:8080", map[string]string{
		"X-Forwarded-For": "1.2.3.4",
	})

	if got := GetClientIP(r); got != "127.0.0.1" {
		t.Fatalf("không cấu hình proxy nào thì phải dùng RemoteAddr, nhận được %q", got)
	}
}

func TestGetClientIP_TachDungIPv6(t *testing.T) {
	withTrustedProxies(t, nil)

	// Bản cũ dùng LastIndex(":") nên cắt nhầm địa chỉ IPv6 thành "[2001:db8::1]".
	r := newRequest("[2001:db8::1]:54321", nil)

	if got := GetClientIP(r); got != "2001:db8::1" {
		t.Fatalf("phải tách đúng host IPv6, nhận được %q", got)
	}
}

func TestClientIPKey_GomIPv6VeKhoi64(t *testing.T) {
	withTrustedProxies(t, nil)

	// Hai địa chỉ khác nhau trong cùng một khối /64 phải chung một bộ đếm,
	// nếu không người dùng IPv6 chỉ cần đổi địa chỉ là thoát rate limit.
	a := ClientIPKey(newRequest("[2001:db8:1:2::abcd]:1", nil))
	b := ClientIPKey(newRequest("[2001:db8:1:2::9999]:1", nil))

	if a != b {
		t.Fatalf("hai địa chỉ cùng khối /64 phải cho cùng khoá, nhận %q và %q", a, b)
	}
}

func TestClientIPKey_IPv4GiuNguyen(t *testing.T) {
	withTrustedProxies(t, nil)

	if got := ClientIPKey(newRequest("203.0.113.9:1", nil)); got != "203.0.113.9" {
		t.Fatalf("IPv4 phải giữ nguyên địa chỉ đầy đủ, nhận được %q", got)
	}
}
