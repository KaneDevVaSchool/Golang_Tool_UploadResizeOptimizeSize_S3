import { motion } from "framer-motion";
import { ArrowRight, Rocket } from "lucide-react";

type EducationLevelCardProps = {
  level: "primary" | "secondary";
  artworkCount: number;
  onSelect: () => void;
};

const COPY = {
  primary: {
    kicker: "VƯỜN ƯƠM SẮC MÀU TUỔI THƠ",
    title: "Khối Tiểu Học",
    subtitle: "Dành cho học sinh Lớp 1 đến Lớp 5",
    body: "Bước vào khu vườn thần tiên nơi trí tưởng tượng bay bổng cùng những nét cọ ngây thơ, rực rỡ và tràn đầy niềm vui hồn nhiên.",
    gradeRange: "5 Khối lớp (Lớp 1 - 5)",
  },
  secondary: {
    kicker: "XƯỞNG NGHỆ THUẬT HIỆN ĐẠI",
    title: "Khối Trung Học",
    subtitle: "THCS & THPT (Lớp 6 đến Lớp 12)",
    body: "Khám phá không gian nghệ thuật đương đại đậm chất cá tính, tư duy trừu tượng và khát vọng bản lĩnh của tuổi trẻ.",
    gradeRange: "7 Khối lớp (Lớp 6 - 12)",
  },
} as const;

/**
 * 1 trong 2 card lớn ở Section 2 (cổng chọn khối) - chuyển đổi tinh thần từ
 * mẫu Tailwind user cung cấp sang CSS thuần: badge kicker, tiêu đề lớn
 * (font Fraunces), mô tả, stat pill, CTA, floating decoration, hover
 * lift+scale, theme riêng từng cấp học (tiểu học pastel sáng / trung học
 * dark navy).
 */
export function EducationLevelCard({ level, artworkCount, onSelect }: EducationLevelCardProps) {
  const copy = COPY[level];

  return (
    <motion.button
      type="button"
      className={`edu-card edu-card--${level}`}
      onClick={onSelect}
      whileHover={{ y: -8, scale: 1.015 }}
      whileTap={{ scale: 0.99 }}
      transition={{ duration: 0.35, ease: [0.22, 1, 0.36, 1] }}
    >
      <div className="edu-card-decor" aria-hidden>
        {level === "primary" ? (
          <>
            <motion.span
              className="edu-card-float edu-card-float--1"
              animate={{ y: [0, -10, 0], rotate: [0, 6, 0] }}
              transition={{ duration: 4, repeat: Infinity, ease: "easeInOut" }}
            >
              🦋
            </motion.span>
            <motion.span
              className="edu-card-float edu-card-float--2"
              animate={{ y: [0, -8, 0], rotate: [0, -6, 0] }}
              transition={{ duration: 3.4, repeat: Infinity, ease: "easeInOut", delay: 0.6 }}
            >
              🌿
            </motion.span>
          </>
        ) : (
          <>
            <motion.span
              className="edu-card-float edu-card-float--1"
              animate={{ opacity: [0.4, 1, 0.4] }}
              transition={{ duration: 2.2, repeat: Infinity, ease: "easeInOut" }}
            >
              ✨
            </motion.span>
            <motion.span
              className="edu-card-float edu-card-float--2"
              animate={{ opacity: [0.4, 1, 0.4] }}
              transition={{ duration: 2.6, repeat: Infinity, ease: "easeInOut", delay: 0.8 }}
            >
              ✨
            </motion.span>
          </>
        )}
      </div>

      <div className="edu-card-content">
        <span className="edu-card-kicker">
          {level === "secondary" && <Rocket size={13} />}
          {copy.kicker}
        </span>
        <h3 className="edu-card-title">{copy.title}</h3>
        <p className="edu-card-subtitle">{copy.subtitle}</p>
        <p className="edu-card-body">{copy.body}</p>
        <div className="edu-card-stats">
          <span className="edu-card-stat-pill">{artworkCount} tác phẩm</span>
          <span className="edu-card-stat-sep">• {copy.gradeRange}</span>
        </div>
      </div>

      <div className="edu-card-cta">
        <span>Khám phá ngay</span>
        <ArrowRight size={18} className="edu-card-cta-arrow" />
      </div>
    </motion.button>
  );
}
