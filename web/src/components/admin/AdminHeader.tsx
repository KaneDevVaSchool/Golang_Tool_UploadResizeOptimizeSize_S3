import { ChevronDown, LogOut, Menu } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { ConfirmDialog } from "../ConfirmDialog";
import type { AdminUser } from "../../hooks/useAdminAuth";
import { adminRequest } from "../../lib/adminApi";
import { toast } from "../../lib/toastBus";

/**
 * Header admin: avatar tròn (ảnh Google hoặc chữ cái đầu), dropdown
 * tên/email/role, nút đăng xuất mở ConfirmDialog (tái dùng component có
 * sẵn) - chỉ thực sự logout khi người dùng xác nhận trong dialog.
 */
export function AdminHeader({
  user,
  onLogout,
  onOpenMobileSidebar,
}: {
  user: AdminUser;
  onLogout: () => void;
  onOpenMobileSidebar: () => void;
}) {
  const [menuOpen, setMenuOpen] = useState(false);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [loggingOut, setLoggingOut] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);
  const navigate = useNavigate();

  useEffect(() => {
    if (!menuOpen) return;
    function onClickOutside(e: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) setMenuOpen(false);
    }
    document.addEventListener("mousedown", onClickOutside);
    return () => document.removeEventListener("mousedown", onClickOutside);
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

  return (
    <header className="admin-header">
      <button
        type="button"
        className="admin-header-menu-btn"
        aria-label="Mở menu điều hướng"
        onClick={onOpenMobileSidebar}
      >
        <Menu size={22} />
      </button>

      <div className="admin-header-spacer" />

      <div className="admin-header-account" ref={menuRef}>
        <button type="button" className="admin-header-account-btn" onClick={() => setMenuOpen((v) => !v)}>
          {user.avatar_url ? (
            <img src={user.avatar_url} alt="" className="admin-header-avatar" referrerPolicy="no-referrer" />
          ) : (
            <span className="admin-header-avatar admin-header-avatar--fallback">{initial}</span>
          )}
          <ChevronDown size={16} className={`admin-header-chevron${menuOpen ? " admin-header-chevron--open" : ""}`} />
        </button>

        {menuOpen && (
          <div className="admin-header-dropdown" role="menu">
            <div className="admin-header-dropdown-info">
              <strong>{user.name || "Quản trị viên"}</strong>
              <span>{user.email}</span>
              <span className="admin-header-role-badge">{user.role === "super_admin" ? "Quản trị cấp cao" : "Quản trị viên"}</span>
            </div>
            <button
              type="button"
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
