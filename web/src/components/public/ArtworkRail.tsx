import { ChevronLeft, ChevronRight } from "lucide-react";
import type { ArtworkWithMeta } from "../../lib/artworkApi";
import { useRailScroll } from "../../hooks/useRailScroll";
import { FeaturedArtworkFrame } from "./FeaturedArtworkFrame";

/**
 * Hàng tranh cuộn ngang (snap từng khung). Nút mũi tên + kéo/vuốt; mép
 * mờ để người xem biết còn ảnh phía sau.
 */
export function ArtworkRail({
  items,
  onOpen,
  loading = false,
  emptyText,
  hasMore = false,
  onLoadMore,
}: {
  items: ArtworkWithMeta[];
  onOpen: (index: number) => void;
  loading?: boolean;
  emptyText?: string;
  hasMore?: boolean;
  onLoadMore?: () => void;
}) {
  const { scrollerRef, canPrev, canNext, scrollByDir } = useRailScroll(".gallery-rail-cell", [items.length, loading]);

  if (!loading && items.length === 0) {
    return <p className="admin-empty-note">{emptyText ?? "Chưa có tác phẩm nào."}</p>;
  }

  return (
    <div className={`gallery-rail-wrap${canPrev ? " gallery-rail-wrap--prev" : ""}${canNext ? " gallery-rail-wrap--next" : ""}`}>
      <button
        type="button"
        className="gallery-rail-nav gallery-rail-nav--prev"
        onClick={() => scrollByDir(-1)}
        disabled={!canPrev}
        aria-label="Cuộn tranh sang trái"
      >
        <ChevronLeft size={22} strokeWidth={1.75} />
      </button>

      <div ref={scrollerRef} className="gallery-rail" tabIndex={0} aria-label="Dãy tranh cuộn ngang">
        {loading && items.length === 0
          ? Array.from({ length: 4 }, (_, i) => <div key={`sk-${i}`} className="gallery-rail-cell gallery-rail-cell--skeleton" aria-hidden />)
          : items.map((item, index) => (
              <div key={item.id} className="gallery-rail-cell">
                <FeaturedArtworkFrame item={item} index={index} onClick={() => onOpen(index)} />
              </div>
            ))}
        {hasMore && onLoadMore && (
          <button type="button" className="gallery-rail-more" onClick={onLoadMore}>
            Xem thêm
          </button>
        )}
      </div>

      <button
        type="button"
        className="gallery-rail-nav gallery-rail-nav--next"
        onClick={() => scrollByDir(1)}
        disabled={!canNext}
        aria-label="Cuộn tranh sang phải"
      >
        <ChevronRight size={22} strokeWidth={1.75} />
      </button>
    </div>
  );
}
