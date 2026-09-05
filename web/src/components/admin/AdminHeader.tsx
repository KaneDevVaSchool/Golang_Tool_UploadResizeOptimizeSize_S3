import { ChevronDown, ExternalLink, LogOut, Menu, PanelLeftClose, PanelLeftOpen } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { ConfirmDialog } from "../ConfirmDialog";
import type { AdminUser } from "../../hooks/useAdminAuth";
import { adminRequest } from "../../lib/adminApi";
import { toast } from "../../lib/toastBus";
import { useRegisterPageHeaderSlot } from "./pageHeaderPortal";

/**
 * Header admin - port AppHeader.vue (va-workspace): MỘT hàng thấp
 * (--admin-header-h) gồm
 *   nút menu · vùng portal cho AdminPageHeader · lối tắt + tài khoản.
 *
 * Nút menu đa dụng như bản gốc: desktop thu gọn/mở rộng sidebar, dưới
 * desktop mở drawer off-canvas.
 */
export function AdminHeader({
  user,
  collapsed,
  isDesktop,
  onToggleSidebar,
  onLogout,
}: {
  user: AdminUser;
  collapsed: boolean;
  isDesktop: boolean;
  onToggleSidebar: () => void;
  onLogout: () => void;
}) {
  const [menuOpen, setMenuOpen] = useState(false);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [loggingOut, setLoggingOut] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);
  const navigate = useNavigate();
  const registerSlot = useRegisterPageHeaderSlot();

  useEffect(() => {
    if (!menuOpen) return;
    function onClickOutside(e: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) setMenuOpen(false);
    }
    function onKey(e: KeyboardEvent) {
      if (e.key === "Escape") setMenuOpen(false);
    }
    document.addEventListener("mousedown", onClickOutside);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onClickOutside);
      document.removeEventListener("keydown", onKey);
    };
  }, [menuOpen]);

  async function confirmLogout() {
    setLoggingOut(true);
    try {
      await adminRequest("/api/v1/admin/auth/logout", { method: "POST", body: {} });
      onLogout();
      toast.success("Đã đăng xuất.");
      navigate("/admin/login", { replace: true });
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Đăng xuất thất bại, vui lòng thử lại.");
    } finally {
      setLoggingOut(false);
      setConfirmOpen(false);
    }
  }

  const initial = (user.name || user.email || "?").trim().charAt(0).toUpperCase();

  // Desktop: icon phản ánh trạng thái thu gọn; mobile: icon hamburger.
  const ToggleIcon = !isDesktop ? Menu : collapsed ? PanelLeftOpen : PanelLeftClose;
  const toggleLabel = !isDesktop
    ? "Mở menu điều hướng"
    : collapsed
      ? "Mở rộng menu"
      : "Thu gọn menu";

  return (
    <header className="admin-header">
      <button
        type="button"
        className="admin-header-icon-btn"
        aria-label={toggleLabel}
        title={toggleLabel}
        aria-expanded={isDesktop ? !collapsed : undefined}
        onClick={onToggleSidebar}
      >
        <ToggleIcon size={20} strokeWidth={2} />
      </button>

      {/* Đích portal của AdminPageHeader (xem pageHeaderPortal.tsx) */}
      <div id="admin-content-header" className="admin-header-page" ref={registerSlot} />

      <div className="admin-header-actions">
        <a
          href="/"
          target="_blank"
          rel="noreferrer"
          className="admin-header-icon-btn admin-header-view-site"
          aria-label="Xem trang triển lãm"
          title="Xem trang triển lãm"
        >
          <ExternalLink size={18} strokeWidth={2} />
        </a>

        <div className="admin-header-account" ref={menuRef}>
          <button
            type="button"
            className="admin-header-account-btn"
            aria-haspopup="menu"
            aria-expanded={menuOpen}
            aria-label="Menu tài khoản"
            onClick={() => setMenuOpen((v) => !v)}
          >
            {user.avatar_url ? (
              <img src={user.avatar_url} alt="" className="admin-header-avatar" referrerPolicy="no-referrer" />
            ) : (
              <span className="admin-header-avatar admin-header-avatar--fallback">{initial}</span>
            )}
            <span className="admin-header-account-name">{user.name || "Quản trị viên"}</span>
            <ChevronDown
              size={16}
              className={`admin-header-chevron${menuOpen ? " admin-header-chevron--open" : ""}`}
            />
          </button>

          {menuOpen && (
            <div className="admin-header-dropdown" role="menu">
              <div className="admin-header-dropdown-info">
                <strong>{user.name || "Quản trị viên"}</strong>
                <span>{user.email}</span>
                <span className="admin-header-role-badge">
                  {user.role === "super_admin" ? "Quản trị cấp cao" : "Quản trị viên"}
                </span>
              </div>
              <button
                type="button"
                role="menuitem"
                className="admin-header-dropdown-item admin-header-dropdown-item--danger"
                onClick={() => {
                  setMenuOpen(false);
                  setConfirmOpen(true);
                }}
              >
                <LogOut size={16} />
                Đăng xuất
              </button>
            </div>
          )}
        </div>
      </div>

      <ConfirmDialog
        open={confirmOpen}
        title="Xác nhận đăng xuất"
        message="Bạn có chắc muốn đăng xuất khỏi trang quản trị? Phiên làm việc hiện tại sẽ kết thúc."
        confirmLabel="Đăng xuất"
        cancelLabel="Ở lại"
        busyLabel="Đang đăng xuất…"
        busy={loggingOut}
        onConfirm={confirmLogout}
        onCancel={() => setConfirmOpen(false)}
      />
    </header>
  );
}
