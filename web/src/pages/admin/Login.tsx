import { motion } from "framer-motion";
import { useEffect } from "react";
import { useSearchParams } from "react-router-dom";
import { useScrollableBody } from "../../hooks/useScrollableBody";
import { toast } from "../../lib/toastBus";
import "../../styles/admin.css";

const ERROR_MESSAGES: Record<string, string> = {
  invalid_state: "Phiên đăng nhập đã hết hạn, vui lòng thử lại.",
  missing_code: "Đăng nhập Google không thành công, vui lòng thử lại.",
  exchange_failed: "Không xác thực được với Google, vui lòng thử lại.",
  userinfo_failed: "Không lấy được thông tin tài khoản Google.",
  incomplete_profile: "Tài khoản Google thiếu thông tin cần thiết.",
  domain_not_allowed: "Email này không thuộc hệ thống Trường Việt Mỹ.",
  account_disabled: "Tài khoản của bạn đã bị vô hiệu hoá.",
  server_error: "Có lỗi xảy ra, vui lòng thử lại sau.",
};

/**
 * Trang đăng nhập admin - phong cách tham khảo va-workspace: card trắng nổi
 * trên nền --vas-tri-thuc, watermark logo full-background invert trắng, nút
 * Google tròn chỉ icon (không text). Click nút = redirect thật tới
 * /auth/google/login (browser-redirect flow, không phải popup/AJAX).
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
      <img className="admin-login-watermark" src="/images/background-logo.png" alt="" aria-hidden />
      <div className="admin-login-scrim" aria-hidden />

      <motion.div
        className="admin-login-container"
        initial={{ opacity: 0, y: 24 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.5, ease: [0.22, 1, 0.36, 1] }}
      >
        <img className="admin-login-logo" src="/images/vas-white.png" alt="VA Schools" />

        <div className="admin-login-card">
          <h1>Đăng nhập</h1>
          <p>Đăng nhập bằng tài khoản Google do nhà trường cung cấp</p>

          <button type="button" className="admin-login-google-btn" onClick={loginWithGoogle}>
            <svg width="22" height="22" viewBox="0 0 48 48" aria-hidden>
              <path
                fill="#FFC107"
                d="M43.611 20.083H42V20H24v8h11.303c-1.649 4.657-6.08 8-11.303 8-6.627 0-12-5.373-12-12s5.373-12 12-12c3.059 0 5.842 1.154 7.961 3.039l5.657-5.657C34.046 6.053 29.268 4 24 4 12.955 4 4 12.955 4 24s8.955 20 20 20 20-8.955 20-20c0-1.341-.138-2.65-.389-3.917z"
              />
              <path
                fill="#FF3D00"
                d="m6.306 14.691 6.571 4.819C14.655 15.108 18.961 12 24 12c3.059 0 5.842 1.154 7.961 3.039l5.657-5.657C34.046 6.053 29.268 4 24 4 16.318 4 9.656 8.337 6.306 14.691z"
              />
              <path
                fill="#4CAF50"
                d="M24 44c5.166 0 9.86-1.977 13.409-5.192l-6.19-5.238A11.91 11.91 0 0 1 24 36c-5.202 0-9.619-3.317-11.283-7.946l-6.522 5.025C9.505 39.556 16.227 44 24 44z"
              />
              <path
                fill="#1976D2"
                d="M43.611 20.083H42V20H24v8h11.303a12.04 12.04 0 0 1-4.087 5.571l.003-.002 6.19 5.238C36.971 39.205 44 34 44 24c0-1.341-.138-2.65-.389-3.917z"
              />
            </svg>
          </button>
        </div>
      </motion.div>
    </div>
  );
}
