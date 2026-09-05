package handlers

import "testing"

// TestIsEmailAllowed_Whitelist kiểm tra whitelist email chính xác: chỉ đúng
// các tài khoản được liệt kê mới đăng nhập được, email khác cùng domain vẫn
// bị chặn.
func TestIsEmailAllowed_Whitelist(t *testing.T) {
	h := NewAdminAuthHandler(nil, nil, nil,
		[]string{"vaschools.edu.vn", "hcm.vaschools.edu.vn"},
		[]string{"khoana@hcm.vaschools.edu.vn", "toanbq@vaschools.edu.vn"},
		false,
	)

	tests := []struct {
		name  string
		email string
		want  bool
	}{
		{"email trong whitelist", "khoana@hcm.vaschools.edu.vn", true},
		{"email trong whitelist - domain khác", "toanbq@vaschools.edu.vn", true},
		{"viết hoa vẫn khớp", "KhoaNA@HCM.VASchools.edu.vn", true},
		{"có khoảng trắng thừa", "  toanbq@vaschools.edu.vn  ", true},
		{"cùng domain nhưng ngoài whitelist", "nguoila@vaschools.edu.vn", false},
		{"ngoài whitelist và ngoài domain", "attacker@gmail.com", false},
		{"email rỗng", "", false},
		{"prefix trùng nhưng domain lạ", "khoana@hcm.vaschools.edu.vn.evil.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := h.isEmailAllowed(tt.email); got != tt.want {
				t.Errorf("isEmailAllowed(%q) = %v, muốn %v", tt.email, got, tt.want)
			}
		})
	}
}

// TestIsEmailAllowed_FallbackDomain kiểm tra khi whitelist rỗng thì rơi về
// kiểm tra domain như hành vi cũ.
func TestIsEmailAllowed_FallbackDomain(t *testing.T) {
	h := NewAdminAuthHandler(nil, nil, nil,
		[]string{"vaschools.edu.vn"},
		nil,
		false,
	)

	if !h.isEmailAllowed("batky@vaschools.edu.vn") {
		t.Error("email đúng domain phải được phép khi whitelist rỗng")
	}
	if h.isEmailAllowed("nguoila@gmail.com") {
		t.Error("email sai domain phải bị chặn")
	}
}

// TestIsEmailAllowed_NoRestriction kiểm tra whitelist và domain đều rỗng thì
// không giới hạn (chế độ dev).
func TestIsEmailAllowed_NoRestriction(t *testing.T) {
	h := NewAdminAuthHandler(nil, nil, nil, nil, nil, false)

	if !h.isEmailAllowed("batky@gmail.com") {
		t.Error("không cấu hình giới hạn thì mọi email phải được phép")
	}
}
