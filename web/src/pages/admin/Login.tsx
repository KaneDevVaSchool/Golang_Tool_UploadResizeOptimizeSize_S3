import { motion } from "framer-motion";
import { useEffect } from "react";
import { useSearchParams } from "react-router-dom";
import { useScrollableBody } from "../../hooks/useScrollableBody";
import { toast } from "../../lib/toastBus";
import { webpOf } from "../../lib/staticImage";
import "../../styles/admin.css";

const ERROR_MESSAGES: Record<string, string> = {
  invalid_state: "Phiên đăng nhập đã hết hạn, vui lòng thử lại.",
  missing_code: "Đăng nhập Google không thành công, vui lòng thử lại.",
  exchange_failed: "Không xác thực được với Google, vui lòng thử lại.",
  userinfo_failed: "Không lấy được thông tin tài khoản Google.",
  incomplete_profile: "Tài khoản Google thiếu thông tin cần thiết.",
  domain_not_allowed: "Email này không thuộc hệ thống Trường Việt Mỹ.",
  email_not_allowed: "Tài khoản này không có quyền truy cập hệ thống. Vui lòng liên hệ quản trị viên.",
  email_not_verified: "Email Google của bạn chưa được xác thực.",
  account_disabled: "Tài khoản của bạn đã bị vô hiệu hoá.",
  server_error: "Có lỗi xảy ra, vui lòng thử lại sau.",
};

/**
 * Trang đăng nhập admin - bố cục port 1-1 từ nguyên mẫu gốc va-hrm
 * (resources/js/Pages/Auth/Login.tsx, React/Tailwind) sang CSS thuần theo
 * token --vas-* của dự án này, đối chiếu thêm bản CSS thuần va-workspace
 * (Modules/Identity/resources/js/pages/Login.vue).
 *
 * Giữ nguyên bản gốc: nền --vas-tri-thuc (#9a0036), watermark
 * background-logo.png invert trắng KHÔNG giảm opacity (độ mờ đến từ chính
 * ảnh nguồn - hạ opacity nữa là họa tiết biến mất), logo-2.png cỡ lớn ở
 * header, card trắng bo góc, nút Google tròn chỉ icon google.png.
 *
 * Click nút = redirect thật tới /auth/google/login (browser-redirect flow,
 * không phải popup/AJAX).
 */
export default function Login() {
  useScrollableBody();
  const [params, setParams] = useSearchParams();

  useEffect(() => {
    const errorCode = params.get("error");
    if (!errorCode) return;
    toast.error(ERROR_MESSAGES[errorCode] ?? "Đăng nhập không thành công, vui lòng thử lại.");
    const next = new URLSearchParams(params);
    next.delete("error");
    setParams(next, { replace: true });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  function loginWithGoogle() {
    // Dùng VITE_API_BASE_URL nếu FE deploy khác origin với backend (mặc
    // định rỗng = same-origin, khớp .env.example / Vite proxy dev mode).
    const apiBase = (import.meta.env.VITE_API_BASE_URL as string | undefined)?.replace(/\/$/, "") ?? "";
    window.location.href = `${apiBase}/auth/google/login`;
  }

  return (
    <div className="admin-login">
      <picture>
        <source srcSet={webpOf("/images/background-logo.png")} type="image/webp" />
        <img className="admin-login-watermark" src="/images/background-logo.png" alt="" aria-hidden />
      </picture>
      <div className="admin-login-scrim" aria-hidden />

      <motion.div
        className="admin-login-container"
        initial={{ opacity: 0, y: 24 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.5, ease: [0.22, 1, 0.36, 1] }}
      >
        <header className="admin-login-header">
          <picture>
            <source srcSet={webpOf("/images/logo-2.png")} type="image/webp" />
            <img
              className="admin-login-logo"
              src="/images/logo-2.png"
              alt="Vietnam America Schools — Trường học của sự lắng nghe"
              width={320}
              height={92}
            />
          </picture>
        </header>

        <div className="admin-login-card">
          <h1>Đăng nhập</h1>
          <p>Đăng nhập thông qua tài khoản mail do nhà trường cung cấp</p>

          <div className="admin-login-actions">
            <button
              type="button"
              className="admin-login-google-btn"
              aria-label="Đăng nhập bằng Google"
              onClick={loginWithGoogle}
            >
              <img
                className="admin-login-google-icon"
                src="/images/google.png"
                alt=""
                width={40}
                height={40}
                loading="eager"
                decoding="async"
              />
            </button>
          </div>
        </div>
      </motion.div>
    </div>
  );
}
