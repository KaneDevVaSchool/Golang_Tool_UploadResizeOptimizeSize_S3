import { Image, LayoutDashboard, Menu, Trophy, X } from "lucide-react";
import { useEffect, useState } from "react";
import { NavLink } from "react-router-dom";

const SIDEBAR_COLLAPSE_KEY = "vas_admin_sidebar_collapsed";
const DESKTOP_BREAKPOINT = "(min-width: 1280px)";

type MenuItem = {
  id: string;
  label: string;
  path: string;
  icon: typeof LayoutDashboard;
  end?: boolean;
};

const MENU_ITEMS: MenuItem[] = [
  { id: "dashboard", label: "Dashboard", path: "/admin", icon: LayoutDashboard, end: true },
  { id: "artworks", label: "Quản lý tác phẩm", path: "/admin/artworks", icon: Image },
  { id: "awards", label: "Quản lý giải thưởng", path: "/admin/awards", icon: Trophy },
];

function readCollapsed(): boolean {
  try {
    return localStorage.getItem(SIDEBAR_COLLAPSE_KEY) === "true";
  } catch {
    return false;
  }
}

/**
 * Sidebar admin - port UX từ AppSidebar.vue (va-workspace, chỉ tham khảo
 * pattern, code React thuần):
 * - Desktop (>=1280px): collapse/expand toggle, trạng thái lưu localStorage.
 * - Mobile/tablet (<1280px): off-canvas drawer, Escape để đóng, khoá scroll
 *   body khi mở.
 * - Active link theo NavLink của react-router (tự so path).
 */
export function AdminSidebar({
  mobileOpen,
  onCloseMobile,
}: {
  mobileOpen: boolean;
  onCloseMobile: () => void;
}) {
  const [collapsed, setCollapsed] = useState(readCollapsed);
  const [isDesktop, setIsDesktop] = useState(
    () => typeof window !== "undefined" && window.matchMedia(DESKTOP_BREAKPOINT).matches,
  );

  useEffect(() => {
    const mq = window.matchMedia(DESKTOP_BREAKPOINT);
    const sync = () => setIsDesktop(mq.matches);
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

  useEffect(() => {
    if (!mobileOpen) return;
    function onKey(e: KeyboardEvent) {
      if (e.key === "Escape") onCloseMobile();
    }
    document.body.style.overflow = "hidden";
    window.addEventListener("keydown", onKey);
    return () => {
      document.body.style.overflow = "";
      window.removeEventListener("keydown", onKey);
    };
  }, [mobileOpen, onCloseMobile]);

  const showCollapsed = isDesktop && collapsed;

  return (
    <>
      {!isDesktop && mobileOpen && (
        <button type="button" className="admin-sidebar-overlay" aria-label="Đóng menu" onClick={onCloseMobile} />
      )}

      <aside
        className={`admin-sidebar${showCollapsed ? " admin-sidebar--collapsed" : ""}${
          !isDesktop ? (mobileOpen ? " admin-sidebar--drawer-open" : " admin-sidebar--drawer-closed") : ""
        }`}
      >
        <div className="admin-sidebar-brand">
          {showCollapsed ? (
            <img src="/images/vas-white-mark.png" alt="VA Schools" className="admin-sidebar-mark" />
          ) : (
            <img src="/images/vas-white.png" alt="VA Schools" className="admin-sidebar-logo" />
          )}
          {!isDesktop && (
            <button type="button" className="admin-sidebar-close" aria-label="Đóng menu" onClick={onCloseMobile}>
              <X size={20} />
            </button>
          )}
        </div>

        <nav className="admin-sidebar-nav" aria-label="Điều hướng quản trị">
          {MENU_ITEMS.map((item) => {
            const Icon = item.icon;
            return (
              <NavLink
                key={item.id}
                to={item.path}
                end={item.end}
                className={({ isActive }) => `admin-sidebar-link${isActive ? " admin-sidebar-link--active" : ""}`}
                onClick={() => {
                  if (!isDesktop) onCloseMobile();
                }}
                title={showCollapsed ? item.label : undefined}
              >
                <span className="admin-sidebar-icon">
                  <Icon size={20} strokeWidth={2} />
                </span>
                {!showCollapsed && <span className="admin-sidebar-label">{item.label}</span>}
                {showCollapsed && <span className="admin-sidebar-flyout">{item.label}</span>}
              </NavLink>
            );
          })}
        </nav>

        {isDesktop && (
          <button
            type="button"
            className="admin-sidebar-collapse-toggle"
            onClick={() => setCollapsed((v) => !v)}
            aria-label={collapsed ? "Mở rộng menu" : "Thu gọn menu"}
          >
            <Menu size={18} />
          </button>
        )}
      </aside>
    </>
  );
}
