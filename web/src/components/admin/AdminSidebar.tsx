import { Image, LayoutDashboard, Trophy, Upload, X } from "lucide-react";
import { useEffect, type ComponentType } from "react";
import { NavLink } from "react-router-dom";

type IconType = ComponentType<{ size?: number | string; strokeWidth?: number }>;

type MenuItem = {
  id: string;
  label: string;
  path: string;
  icon: IconType;
  end?: boolean;
};

type MenuSection = {
  id: string;
  label: string;
  items: MenuItem[];
};

/** Menu chia nhóm giống MENU_SECTIONS của AppSidebar.vue (va-workspace). */
const MENU_SECTIONS: MenuSection[] = [
  {
    id: "general",
    label: "Điều hướng",
    items: [{ id: "dashboard", label: "Tổng quan", path: "/admin", icon: LayoutDashboard, end: true }],
  },
  {
    id: "content",
    label: "Nội dung",
    items: [
      { id: "artworks", label: "Quản lý tác phẩm", path: "/admin/artworks", icon: Image, end: true },
      { id: "upload", label: "Tải tác phẩm mới", path: "/admin/artworks/upload", icon: Upload },
    ],
  },
  {
    id: "config",
    label: "Cấu hình",
    items: [{ id: "awards", label: "Quản lý giải thưởng", path: "/admin/awards", icon: Trophy }],
  },
];

/**
 * Sidebar admin - port nguyên bố cục + hành vi AppSidebar.vue của
 * va-workspace:
 * - Brand logo trên cùng (mark khi thu gọn, wordmark khi mở rộng).
 * - Menu chia nhóm có nhãn section chữ hoa nhỏ; icon nằm trong "well" bo góc.
 * - Desktop (>=1280px): rail 4rem khi thu gọn + flyout nhãn khi hover;
 *   trạng thái collapsed do AdminLayout giữ (nút toggle nằm trên header,
 *   đúng như va-workspace) và lưu localStorage.
 * - Tablet/mobile (<1280px): off-canvas drawer + overlay, Escape để đóng,
 *   khoá scroll body khi mở.
 */
export function AdminSidebar({
  mobileOpen,
  collapsed,
  isDesktop,
  onCloseMobile,
}: {
  mobileOpen: boolean;
  collapsed: boolean;
  isDesktop: boolean;
  onCloseMobile: () => void;
}) {
  useEffect(() => {
    if (isDesktop || !mobileOpen) return;
    function onKey(e: KeyboardEvent) {
      if (e.key === "Escape") onCloseMobile();
    }
    document.body.style.overflow = "hidden";
    window.addEventListener("keydown", onKey);
    return () => {
      document.body.style.overflow = "";
      window.removeEventListener("keydown", onKey);
    };
  }, [mobileOpen, isDesktop, onCloseMobile]);

  const showCollapsed = isDesktop && collapsed;

  return (
    <div className={`admin-sidebar-wrap${!isDesktop && mobileOpen ? " admin-sidebar-wrap--open" : ""}`}>
      <button
        type="button"
        className="admin-sidebar-overlay"
        aria-label="Đóng menu"
        tabIndex={!isDesktop && mobileOpen ? 0 : -1}
        onClick={onCloseMobile}
      />

      <aside
        className={`admin-sidebar${showCollapsed ? " admin-sidebar--collapsed" : ""}`}
        aria-label={showCollapsed ? "Menu thu gọn" : "Menu chính"}
      >
        <div className="admin-sidebar-brand">
          <NavLink to="/admin" end className="admin-sidebar-brand-link" onClick={onCloseMobile}>
            {showCollapsed ? (
              <img src="/images/vas-white-mark.png" alt="VA Schools" className="admin-sidebar-mark" />
            ) : (
              <img src="/images/vas-white.png" alt="VA Schools" className="admin-sidebar-logo" />
            )}
          </NavLink>

          <button type="button" className="admin-sidebar-close" aria-label="Đóng menu" onClick={onCloseMobile}>
            <X size={18} strokeWidth={2} />
          </button>
        </div>

        <nav className="admin-sidebar-nav" aria-label="Điều hướng quản trị">
          {MENU_SECTIONS.map((section) => (
            <section key={section.id} className="admin-sidebar-section">
              {!showCollapsed && <p className="admin-sidebar-section-label">{section.label}</p>}

              {section.items.map((item) => {
                const Icon = item.icon;
                return (
                  <NavLink
                    key={item.id}
                    to={item.path}
                    end={item.end}
                    className={({ isActive }) =>
                      `admin-sidebar-link${isActive ? " admin-sidebar-link--active" : ""}`
                    }
                    aria-label={showCollapsed ? item.label : undefined}
                    onClick={() => {
                      if (!isDesktop) onCloseMobile();
                    }}
                  >
                    <span className="admin-sidebar-icon">
                      <Icon size={18} strokeWidth={2} />
                    </span>
                    {showCollapsed ? (
                      <span className="admin-sidebar-flyout">{item.label}</span>
                    ) : (
                      <span className="admin-sidebar-label">{item.label}</span>
                    )}
                  </NavLink>
                );
              })}
            </section>
          ))}
        </nav>
      </aside>
    </div>
  );
}
