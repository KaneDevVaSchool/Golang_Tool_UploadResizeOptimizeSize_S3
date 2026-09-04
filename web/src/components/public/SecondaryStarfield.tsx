import { motion } from "framer-motion";

// Vị trí/độ trễ sao cố định (không random mỗi render) để không "nhảy" chỗ
// giữa các lần re-render - cùng công thức với SECONDARY_STARS trong
// EducationLevelCard.tsx, chỉ rải thưa hơn cho một section rộng hơn.
const STARS = [
  { top: "8%", left: "6%", delay: 0 },
  { top: "14%", left: "92%", delay: 0.5 },
  { top: "22%", left: "34%", delay: 1.1 },
  { top: "6%", left: "60%", delay: 0.8 },
  { top: "30%", left: "78%", delay: 1.6 },
  { top: "18%", left: "16%", delay: 0.3 },
  { top: "26%", left: "50%", delay: 1.9 },
  { top: "10%", left: "44%", delay: 1.3 },
];

/**
 * Rắc sao lấp lánh dùng chung cho các khối nền "đêm hiện đại" của Khối
 * Trung học (đồng bộ .edu-card-star ở EducationLevelCard) - tách thành
 * component riêng để tái dùng ở GalleryPage.tsx (section chọn khối lớp)
 * mà không lặp lại mảng toạ độ.
 */
export function SecondaryStarfield() {
  return (
    <>
      {STARS.map((star, i) => (
        <motion.span
          key={i}
          className="edu-card-star grade-explorer-star"
          style={{ top: star.top, left: star.left }}
          animate={{ opacity: [0.15, 1, 0.15], scale: [0.8, 1.15, 0.8] }}
          transition={{ duration: 2.6, repeat: Infinity, ease: "easeInOut", delay: star.delay }}
        />
      ))}
    </>
  );
}
