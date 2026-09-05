import { ChevronLeft, ChevronRight } from "lucide-react";

/**
 * Thanh phân trang cho lưới tác phẩm. Rút gọn kiểu "1 … 4 5 6 … 12": luôn
 * hiện trang đầu/cuối, trang hiện tại và 1 trang liền kề mỗi bên, phần bị
 * lược thay bằng dấu "…" - để số nút không phình ra khi có hàng chục trang.
 */
function buildPageList(current: number, total: number): (number | "gap")[] {
  if (total <= 7) return Array.from({ length: total }, (_, i) => i + 1);

  const pages: (number | "gap")[] = [1];
  const from = Math.max(2, current - 1);
  const to = Math.min(total - 1, current + 1);

  if (from > 2) pages.push("gap");
  for (let p = from; p <= to; p++) pages.push(p);
  if (to < total - 1) pages.push("gap");
  pages.push(total);

  return pages;
}

export function GalleryPagination({
  page,
  totalPages,
  onChange,
}: {
  page: number;
  totalPages: number;
  onChange: (page: number) => void;
}) {
  if (totalPages <= 1) return null;

  return (
    <nav className="gallery-pagination" aria-label="Phân trang tác phẩm">
      <button
        type="button"
        className="gallery-page-btn gallery-page-btn--nav"
        onClick={() => onChange(page - 1)}
        disabled={page <= 1}
        aria-label="Trang trước"
      >
        <ChevronLeft size={18} />
      </button>

      {buildPageList(page, totalPages).map((entry, i) =>
        entry === "gap" ? (
          <span key={`gap-${i}`} className="gallery-page-gap" aria-hidden>
            …
          </span>
        ) : (
          <button
            key={entry}
            type="button"
            className={`gallery-page-btn${entry === page ? " gallery-page-btn--active" : ""}`}
            onClick={() => onChange(entry)}
            aria-current={entry === page ? "page" : undefined}
            aria-label={`Trang ${entry}`}
          >
            {entry}
          </button>
        ),
      )}

      <button
        type="button"
        className="gallery-page-btn gallery-page-btn--nav"
        onClick={() => onChange(page + 1)}
        disabled={page >= totalPages}
        aria-label="Trang sau"
      >
        <ChevronRight size={18} />
      </button>
    </nav>
  );
}
