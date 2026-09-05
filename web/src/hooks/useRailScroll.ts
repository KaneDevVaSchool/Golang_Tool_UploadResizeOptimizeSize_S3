import { useCallback, useEffect, useRef, useState } from "react";

/**
 * Logic chung cho một dãy cuộn ngang: theo dõi còn cuộn được về trái/phải
 * (để hiện mép mờ + bật/tắt nút mũi tên) và cuộn từng "bước" đúng bằng bề
 * rộng một thẻ.
 *
 * Tách khỏi component để ArtworkRail (khung tranh bảo tàng) và HallRail
 * (card bo tròn) dùng chung một cơ chế - hai nơi có markup và CSS khác hẳn
 * nhau nhưng hành vi cuộn thì giống hệt.
 *
 * @param cellSelector selector của một ô trong dãy, dùng để đo bước cuộn.
 * @param deps đổi thì đo lại (số lượng item, trạng thái loading...).
 */
export function useRailScroll(cellSelector: string, deps: unknown[] = []) {
  const scrollerRef = useRef<HTMLDivElement>(null);
  const [canPrev, setCanPrev] = useState(false);
  const [canNext, setCanNext] = useState(false);

  const updateEdges = useCallback(() => {
    const el = scrollerRef.current;
    if (!el) return;
    const max = el.scrollWidth - el.clientWidth;
    // Ngưỡng 8px: bỏ qua sai số làm tròn của subpixel, nếu không mũi tên
    // sẽ nhấp nháy bật/tắt khi cuộn tới sát mép.
    setCanPrev(el.scrollLeft > 8);
    setCanNext(max - el.scrollLeft > 8);
  }, []);

  useEffect(() => {
    const el = scrollerRef.current;
    if (!el) return;
    updateEdges();
    el.addEventListener("scroll", updateEdges, { passive: true });
    const ro = new ResizeObserver(updateEdges);
    ro.observe(el);
    return () => {
      el.removeEventListener("scroll", updateEdges);
      ro.disconnect();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [updateEdges, ...deps]);

  const scrollByDir = useCallback(
    (dir: -1 | 1) => {
      const el = scrollerRef.current;
      if (!el) return;
      const cell = el.querySelector<HTMLElement>(cellSelector);
      // Bước = bề rộng thẻ + khoảng cách thật giữa hai thẻ (đọc từ CSS để
      // không phải đồng bộ tay mỗi lần đổi gap trong stylesheet).
      const gap = cell ? parseFloat(getComputedStyle(el).columnGap || "0") || 0 : 0;
      const step = cell ? cell.offsetWidth + gap : Math.round(el.clientWidth * 0.72);
      // Cuộn mượt là hiệu ứng chuyển động - người đã tắt animation trong
      // hệ điều hành thì nhảy thẳng tới nơi (CSS scroll-behavior không
      // chặn được lệnh gọi có behavior:"smooth" tường minh này).
      const reduced = window.matchMedia?.("(prefers-reduced-motion: reduce)").matches;
      el.scrollBy({ left: dir * step, behavior: reduced ? "auto" : "smooth" });
    },
    [cellSelector],
  );

  return { scrollerRef, canPrev, canNext, scrollByDir };
}
