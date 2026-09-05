import { ChevronDown } from "lucide-react";
import { useEffect, useRef, useState, type ComponentType, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { Link } from "react-router-dom";
import { usePageHeaderSlot } from "./pageHeaderPortal";

type IconType = ComponentType<{ size?: number | string; strokeWidth?: number }>;

export type PageHeaderMenuItem = {
  key: string;
  label: string;
  description?: string;
  icon?: IconType;
  onSelect: () => void;
};

export type PageHeaderPrimaryAction = {
  label: string;
  icon?: IconType;
  /** Link nội bộ - nếu có thì render <Link>, bỏ qua onClick. */
  to?: string;
  onClick?: () => void;
  disabled?: boolean;
  /** Có items -> nút mở dropdown thay vì hành động trực tiếp. */
  items?: PageHeaderMenuItem[];
  /** "ghost" cho hành động phụ (VD nút quay lại), mặc định nền brand đặc. */
  variant?: "solid" | "ghost";
};

/**
 * Header của từng trang admin, portal lên thanh AdminHeader - port
 * PageHeader.vue (va-workspace) sang React:
 *   trái = nút hành động chính · giữa = title/subtitle · phải = actions
 *
 * Một hàng duy nhất, cao bằng header nên không tốn thêm chiều dọc - đặc
 * biệt quan trọng trên mobile. Trên màn hình hẹp phần `actions` tự xuống
 * hàng thứ hai (xem .admin-page-header--wrap trong admin.css).
 */
export function AdminPageHeader({
  title,
  subtitle,
  primaryAction,
  actions,
}: {
  title: string;
  subtitle?: string;
  primaryAction?: PageHeaderPrimaryAction;
  actions?: ReactNode;
}) {
  const slot = usePageHeaderSlot();
  const [menuOpen, setMenuOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);

  // Đổi trang -> đóng dropdown đang mở (tránh menu "mồ côi" của trang cũ).
  useEffect(() => {
    setMenuOpen(false);
  }, [title]);

  useEffect(() => {
    if (!menuOpen) return;
    function onPointerDown(e: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) setMenuOpen(false);
    }
    function onKey(e: KeyboardEvent) {
      if (e.key === "Escape") setMenuOpen(false);
    }
    document.addEventListener("mousedown", onPointerDown);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onPointerDown);
      document.removeEventListener("keydown", onKey);
    };
  }, [menuOpen]);

  // Cập nhật <title> tab trình duyệt theo trang đang xem.
  useEffect(() => {
    document.title = `${title} · Quản trị VAS`;
  }, [title]);

  const PrimaryIcon = primaryAction?.icon;
  const hasMenu = (primaryAction?.items?.length ?? 0) > 0;
  const primaryClass = `admin-page-header-primary${
    primaryAction?.variant === "ghost" ? " admin-page-header-primary--ghost" : ""
  }`;

  const primaryContent = PrimaryIcon ? (
    <>
      <PrimaryIcon size={18} strokeWidth={2} />
      <span className="admin-page-header-primary-label">{primaryAction?.label}</span>
    </>
  ) : (
    <span className="admin-page-header-primary-label">{primaryAction?.label}</span>
  );

  const content = (
    <div className="admin-page-header">
      {primaryAction && (
        <div className="admin-page-header-left" ref={menuRef}>
          {primaryAction.to ? (
            <Link
              to={primaryAction.to}
              className={primaryClass}
              title={primaryAction.label}
              aria-label={primaryAction.label}
            >
              {primaryContent}
            </Link>
          ) : (
            <button
              type="button"
              className={`${primaryClass}${menuOpen ? " admin-page-header-primary--open" : ""}`}
              disabled={primaryAction.disabled}
              aria-haspopup={hasMenu ? "menu" : undefined}
              aria-expanded={hasMenu ? menuOpen : undefined}
              title={primaryAction.label}
              aria-label={primaryAction.label}
              onClick={() => {
                if (hasMenu) setMenuOpen((v) => !v);
                else primaryAction.onClick?.();
              }}
            >
              {primaryContent}
              {hasMenu && <ChevronDown size={14} strokeWidth={2} className="admin-page-header-primary-caret" />}
            </button>
          )}

          {hasMenu && menuOpen && (
            <div className="admin-page-header-menu" role="menu">
              {primaryAction.items?.map((item) => {
                const ItemIcon = item.icon;
                return (
                  <button
                    key={item.key}
                    type="button"
                    role="menuitem"
                    className="admin-page-header-menu-item"
                    onClick={() => {
                      setMenuOpen(false);
                      item.onSelect();
                    }}
                  >
                    {ItemIcon && (
                      <span className="admin-page-header-menu-icon">
                        <ItemIcon size={16} strokeWidth={1.75} />
                      </span>
                    )}
                    <span className="admin-page-header-menu-copy">
                      <span className="admin-page-header-menu-label">{item.label}</span>
                      {item.description && (
                        <span className="admin-page-header-menu-desc">{item.description}</span>
                      )}
                    </span>
                  </button>
                );
              })}
            </div>
          )}
        </div>
      )}

      <div className="admin-page-header-title-wrap">
        <h1 className="admin-page-header-title" title={subtitle || title}>
          {title}
        </h1>
        {subtitle && <p className="admin-page-header-subtitle">{subtitle}</p>}
      </div>

      {actions && <div className="admin-page-header-right">{actions}</div>}
    </div>
  );

  return slot ? createPortal(content, slot) : content;
}
