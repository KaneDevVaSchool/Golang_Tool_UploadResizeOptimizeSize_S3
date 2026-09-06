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
  /**
   * Ẩn phần chữ, chỉ còn icon (nút tròn/vuông gọn). `label` vẫn bắt buộc -
   * dùng làm title/aria-label để không mất khả năng tiếp cận khi bỏ chữ.
   */
  iconOnly?: boolean;
};

/**
 * Tách tiêu đề thành từng ký tự để chạy hiệu ứng dồn chữ (staggered) khi
 * đổi trang. Khoảng trắng giữ nguyên bằng   vì span inline-block sẽ
 * nuốt mất khoảng trắng thường.
 *
 * Chỉ tách ký tự cho tiêu đề ngắn (<= LIMIT). Tiêu đề dài mà tách ra thì
 * vừa tốn DOM vừa làm hiệu ứng lê thê; lúc đó cả cụm mờ vào một lần.
 */
const TITLE_STAGGER_LIMIT = 28;

function AnimatedTitle({ title }: { title: string }) {
  if (title.length > TITLE_STAGGER_LIMIT) {
    return <span className="admin-page-header-title-chars">{title}</span>;
  }

  return (
    <span className="admin-page-header-title-chars" aria-hidden>
      {Array.from(title).map((ch, i) => (
        <span
          key={`${ch}-${i}`}
          className="admin-page-header-char"
          style={{ animationDelay: `${i * 26}ms` }}
        >
          {ch === " " ? " " : ch}
        </span>
      ))}
    </span>
  );
}

/**
 * Header của từng trang admin, portal lên thanh AdminHeader:
 *   trái = nút hành động chính · giữa = breadcrumb + title/subtitle · phải = actions
 *
 * Một hàng duy nhất, cao bằng header nên không tốn thêm chiều dọc - đặc
 * biệt quan trọng trên mobile.
 *
 * Tiêu đề chạy hiệu ứng dồn chữ mỗi lần đổi trang: chuyển trang trong SPA
 * không có phản hồi "đã tải xong trang mới" như trình duyệt tải lại, nên
 * hiệu ứng này đóng vai trò báo hiệu. `key={title}` buộc React dựng lại
 * nhánh -> animation chạy lại; không có nó thì CSS animation chỉ chạy đúng
 * lần gắn đầu tiên.
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
    document.title = `${title} · Quản trị VASchools`;
  }, [title]);

  const PrimaryIcon = primaryAction?.icon;
  const hasMenu = (primaryAction?.items?.length ?? 0) > 0;
  const primaryClass = `admin-page-header-primary${
    primaryAction?.variant === "ghost" ? " admin-page-header-primary--ghost" : ""
  }${primaryAction?.iconOnly ? " admin-page-header-primary--icon-only" : ""}`;

  const primaryContent = PrimaryIcon ? (
    <>
      <PrimaryIcon size={18} strokeWidth={2} />
      {!primaryAction?.iconOnly && (
        <span className="admin-page-header-primary-label">{primaryAction?.label}</span>
      )}
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

      <div className="admin-page-header-title-wrap" key={title}>
        <h1 className="admin-page-header-title" title={subtitle || title}>
          {/* Chuỗi thật cho screen reader; bản tách ký tự bên trong đã
              aria-hidden nên không bị đọc thành từng chữ cái rời rạc. */}
          <span className="admin-page-header-title-sr">{title}</span>
          <AnimatedTitle title={title} />
        </h1>
        {subtitle && <p className="admin-page-header-subtitle">{subtitle}</p>}
      </div>

      {actions && <div className="admin-page-header-right">{actions}</div>}
    </div>
  );

  return slot ? createPortal(content, slot) : content;
}
