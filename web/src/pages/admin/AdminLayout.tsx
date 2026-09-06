import { useCallback, useEffect, useState } from "react";
import { Navigate, Outlet, useLocation } from "react-router-dom";
import { AdminHeader } from "../../components/admin/AdminHeader";
import { AdminSidebar } from "../../components/admin/AdminSidebar";
import { PageHeaderSlotProvider } from "../../components/admin/pageHeaderPortal";
import { useAdminAuth } from "../../hooks/useAdminAuth";
import "../../styles/admin.css";

const SIDEBAR_COLLAPSE_KEY = "vas_admin_sidebar_collapsed";
const DESKTOP_BREAKPOINT = "(min-width: 1280px)";

function readCollapsed(): boolean {
  try {
    return localStorage.getItem(SIDEBAR_COLLAPSE_KEY) === "true";
  } catch {
    return false;
  }
}

/**
 * Layout khung cho toàn bộ /admin/* (trừ /admin/login) - bảo vệ route qua
 * useAdminAuth: chưa đăng nhập -> redirect /admin/login.
 *
 * Bố cục port từ AppLayout.vue (va-workspace): sidebar cố định bên trái +
 * header một hàng + vùng nội dung tự cuộn nội bộ (shell cao 100dvh, KHÔNG
 * để cả trang cuộn) - nhờ vậy header/sidebar luôn nằm yên, trên mobile
 * không bị nhảy khi bàn phím ảo bật lên.
 *
 * Trạng thái thu gọn sidebar do layout giữ (nút toggle nằm trên header như
 * bản gốc) và lưu localStorage.
 */
export default function AdminLayout() {
  const { user, loading, clearUser } = useAdminAuth();
  const location = useLocation();
  const [mobileSidebarOpen, setMobileSidebarOpen] = useState(false);
  const [collapsed, setCollapsed] = useState(readCollapsed);
  const [isDesktop, setIsDesktop] = useState(
    () => typeof window !== "undefined" && window.matchMedia(DESKTOP_BREAKPOINT).matches,
  );

  useEffect(() => {
    const mq = window.matchMedia(DESKTOP_BREAKPOINT);
    function sync() {
      setIsDesktop(mq.matches);
      // Về desktop thì drawer mobile không còn ý nghĩa - đóng lại để
      // overlay không kẹt lại che nội dung.
      if (mq.matches) setMobileSidebarOpen(false);
    }
    sync();
    mq.addEventListener("change", sync);
    return () => mq.removeEventListener("change", sync);
  }, []);

  useEffect(() => {
    try {
      localStorage.setItem(SIDEBAR_COLLAPSE_KEY, String(collapsed));
    } catch {
      // localStorage có thể bị chặn (private mode) - bỏ qua, không ảnh hưởng chức năng chính
    }
  }, [collapsed]);

  // Đổi route -> đóng drawer (bấm link trong drawer đã tự đóng, nhưng còn
  // các đường vào khác: nút quay lại của trình duyệt, redirect trong trang).
  useEffect(() => {
    setMobileSidebarOpen(false);
  }, [location.pathname]);

  const toggleSidebar = useCallback(() => {
    if (window.matchMedia(DESKTOP_BREAKPOINT).matches) {
      setCollapsed((v) => !v);
      return;
    }
    setMobileSidebarOpen((v) => !v);
  }, []);

  const closeMobile = useCallback(() => setMobileSidebarOpen(false), []);

  if (loading) {
    return (
      <div className="admin-loading-screen" role="status" aria-live="polite">
        <img src="/images/vas-mascot-wave.png" alt="" aria-hidden />
        <div className="admin-loading-shadow" aria-hidden />
        {/* Nói rõ đang chờ CÁI GÌ: màn hình này đợi API xác thực phiên, có
            thể lâu hơn hẳn một lần tải trang, và "Đang tải…" trơ ra vài giây
            khiến người dùng tưởng hỏng. */}
        <p>Đang kiểm tra phiên đăng nhập…</p>
        <div className="admin-loading-track" aria-hidden />
      </div>
    );
  }

  if (!user) {
    return <Navigate to="/admin/login" replace />;
  }

  return (
    <PageHeaderSlotProvider>
      <div className="admin-shell">
        <AdminSidebar
          mobileOpen={mobileSidebarOpen}
          collapsed={collapsed}
          isDesktop={isDesktop}
          onCloseMobile={closeMobile}
        />

        <div className="admin-main">
          <AdminHeader
            user={user}
            collapsed={collapsed}
            isDesktop={isDesktop}
            onToggleSidebar={toggleSidebar}
            onLogout={clearUser}
          />

          <main className="admin-content">
            {/* background-logo.png có alpha gốc rất thấp (~5%) - CSS filter
                (invert/sepia/...) chỉ đổi màu RGB, không khuếch đại alpha,
                nên watermark từng tàng hình thực sự dù đã set opacity. Cần
                feColorMatrix để nhân alpha lên trước khi tô màu, đúng kỹ
                thuật bản gốc (ProjectCreate.vue bên va-workspace). */}
            <svg className="admin-content-watermark-defs" aria-hidden focusable="false">
              <filter id="admin-watermark-boost" colorInterpolationFilters="sRGB">
                <feColorMatrix type="matrix" values="0 0 0 0 0.6  0 0 0 0 0  0 0 0 0 0.21  0 0 0 18 0" />
              </filter>
            </svg>
            {/* Watermark logo: cố định trong khung nội dung, không cuộn
                theo, nằm dưới mọi card (z-index 0 + card position:relative).
                Wrapper canh giữa logo + quầng sáng (::before trong CSS). */}
            <div className="admin-content-watermark" aria-hidden>
              <div className="admin-content-watermark-logo" />
            </div>
            {/* Viền trang trí góc - thuần thẩm mỹ, gợi khung "tấm ảnh" quanh
                vùng nội dung mà không đóng khung cứng. */}
            <div className="admin-content-watermark-deco" aria-hidden>
              <svg className="corner-tl" width="96" height="96" viewBox="0 0 120 120">
                <path d="M0 40 L0 0 L40 0" fill="none" stroke="rgba(154, 0, 54, 0.18)" strokeWidth="2" />
                <circle cx="0" cy="0" r="3" fill="rgba(154, 0, 54, 0.25)" />
              </svg>
              <svg className="corner-br" width="96" height="96" viewBox="0 0 120 120">
                <path d="M0 40 L0 0 L40 0" fill="none" stroke="rgba(23, 80, 181, 0.16)" strokeWidth="2" />
                <circle cx="0" cy="0" r="3" fill="rgba(23, 80, 181, 0.22)" />
              </svg>
            </div>
            {/* key theo pathname: mỗi trang admin vào bằng hoạt cảnh ngắn
                thay vì bụp một cái sau quãng chờ chunk. Đặt ở lớp trong
                cùng - sidebar, header và watermark không nhấp nháy theo. */}
            <div className="admin-content-inner route-enter" key={location.pathname}>
              <Outlet />
            </div>
          </main>
        </div>
      </div>
    </PageHeaderSlotProvider>
  );
}
