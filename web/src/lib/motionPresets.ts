// Preset animation dùng chung cho UploadTool (cũ) và các trang admin/public
// mới - tách ra từ App.tsx gốc để tránh lặp lại easing curve/timing giữa
// nhiều trang, giữ cảm giác chuyển động nhất quán toàn bộ ứng dụng.
import type { Variants } from "framer-motion";

export const easeSoft = [0.22, 1, 0.36, 1] as const;

// fadeUp: dùng cho danh sách phần tử xuất hiện tuần tự (stagger qua custom
// index) - pattern gốc từ App.tsx (intro section của UploadTool).
export const fadeUp: Variants = {
  hidden: { opacity: 0, y: 18 },
  show: (i: number) => ({
    opacity: 1,
    y: 0,
    transition: { delay: 0.07 * i, duration: 0.5, ease: easeSoft },
  }),
};

// floatY: phần tử trôi nổi nhẹ nhàng lặp vô hạn - dùng cho mascot/decor.
export const floatY: Variants = {
  animate: {
    y: [0, -10, 0],
    transition: { duration: 3.2, repeat: Infinity, ease: "easeInOut" },
  },
};

// popIn: card/modal xuất hiện với hiệu ứng phóng to nhẹ - dùng cho
// EducationLevelCard, StatCard, v.v.
export const popIn: Variants = {
  hidden: { opacity: 0, y: 16, scale: 0.96 },
  show: { opacity: 1, y: 0, scale: 1, transition: { duration: 0.4, ease: easeSoft } },
};
