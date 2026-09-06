// Preset animation dùng chung cho các trang admin/public - tách riêng để
// easing curve/timing không bị lặp lại giữa nhiều trang, giữ cảm giác
// chuyển động nhất quán toàn bộ ứng dụng.
import type { Variants } from "framer-motion";

const easeSoft = [0.22, 1, 0.36, 1] as const;

// fadeUp: dùng cho danh sách phần tử xuất hiện tuần tự (stagger qua custom
// index).
export const fadeUp: Variants = {
  hidden: { opacity: 0, y: 18 },
  show: (i: number) => ({
    opacity: 1,
    y: 0,
    transition: { delay: 0.07 * i, duration: 0.5, ease: easeSoft },
  }),
};
