import { ChevronLeft, ChevronRight } from "lucide-react";
import { HallArtworkCard } from "./HallArtworkCard";
import { useRailScroll } from "../../hooks/useRailScroll";
import type { BillboardEntry } from "../../lib/publicApi";

/**
 * Dãy card cuộn ngang cho khu "Giải Thưởng Đặc Biệt & Chuyên Đề".
 *
 * Cuộn snap từng card, có nút mũi tên hai bên và mép mờ báo còn tranh phía
 * sau. Dùng chung cơ chế cuộn với ArtworkRail qua useRailScroll, nhưng
 * markup/CSS riêng vì card ở đây là card bo tròn chứ không phải khung
 * tranh bảo tàng.
 */
export function HallRail({
  entries,
  onOpen,
}: {
  entries: BillboardEntry[];
  onOpen: (entry: BillboardEntry) => void;
}) {
  const { scrollerRef, canPrev, canNext, scrollByDir } = useRailScroll(".hall-rail-cell", [entries.length]);

  return (
    <div className={`hall-rail-wrap${canPrev ? " hall-rail-wrap--prev" : ""}${canNext ? " hall-rail-wrap--next" : ""}`}>
      <button
        type="button"
        className="hall-rail-nav hall-rail-nav--prev"
        onClick={() => scrollByDir(-1)}
        disabled={!canPrev}
        aria-label="Xem các giải trước đó"
      >
        <ChevronLeft size={20} strokeWidth={2.2} />
      </button>

      <div ref={scrollerRef} className="hall-rail" tabIndex={0} aria-label="Dãy tác phẩm đạt giải chuyên đề">
        {entries.map((entry, index) => (
          <div key={`${entry.id}-${entry.award.id}`} className="hall-rail-cell">
            <HallArtworkCard
              item={entry}
              award={entry.award}
              index={index}
              onClick={() => onOpen(entry)}
            />
          </div>
        ))}
      </div>

      <button
        type="button"
        className="hall-rail-nav hall-rail-nav--next"
        onClick={() => scrollByDir(1)}
        disabled={!canNext}
        aria-label="Xem các giải tiếp theo"
      >
        <ChevronRight size={20} strokeWidth={2.2} />
      </button>
    </div>
  );
}
