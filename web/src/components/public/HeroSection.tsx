import { motion } from "framer-motion";
import { fadeUp, floatY } from "../../lib/motionPresets";
import type { Region } from "./RegionTabs";

const REGION_BUTTONS: { key: Region; label: string }[] = [
  { key: "saigon", label: "Sài Gòn" },
  { key: "cantho", label: "Cần Thơ" },
  { key: "vungtau", label: "Vũng Tàu" },
];

/**
 * Section 1 - Hero: không gian mở đầu, dùng mascot + wordmark + dragon
 * silhouette có sẵn, animation entrance stagger (fadeUp preset tách từ
 * App.tsx gốc). CTA "Chọn phòng tranh trưng bày" scroll tới Section 3 với
 * region đã chọn.
 */
export function HeroSection({ onSelectRegion }: { onSelectRegion: (region: Region) => void }) {
  return (
    <section className="hero-section">
      <div className="hero-bg-glow" aria-hidden />
      <motion.img
        className="hero-dragon"
        src="/images/vas-dragon-silhouette.png"
        alt=""
        aria-hidden
        initial={{ opacity: 0, scale: 0.9 }}
        animate={{ opacity: 0.35, scale: 1 }}
        transition={{ duration: 1.2, ease: [0.22, 1, 0.36, 1] }}
      />
      <motion.img
        className="hero-mascot"
        src="/images/vas-mascot-wave.png"
        alt=""
        aria-hidden
        variants={floatY}
        animate="animate"
        initial={{ opacity: 0, y: 30 }}
        whileInView={{ opacity: 1 }}
        viewport={{ once: true }}
        transition={{ duration: 0.8 }}
      />

      <div className="hero-content">
        <motion.img
          className="hero-wordmark"
          src="/images/vas-wordmark-stacked.png"
          alt="VA Schools"
          custom={0}
          variants={fadeUp}
          initial="hidden"
          animate="show"
        />
        <motion.p className="hero-kicker" custom={1} variants={fadeUp} initial="hidden" animate="show">
          Kỷ niệm 20 năm thành lập · 2006 – 2026
        </motion.p>
        <motion.h1 className="hero-title" custom={2} variants={fadeUp} initial="hidden" animate="show">
          20 Năm Trường Việt Mỹ Của Em
        </motion.h1>
        <motion.p className="hero-subtitle" custom={3} variants={fadeUp} initial="hidden" animate="show">
          Nơi hội tụ những nét vẽ hồn nhiên và đầy tự hào của học sinh khắp hệ thống — chào mừng hành
          trình 20 năm xây dựng "Trường học của sự lắng nghe".
        </motion.p>

        <motion.div className="hero-region-cta" custom={4} variants={fadeUp} initial="hidden" animate="show">
          <span className="hero-region-label">Chọn phòng tranh trưng bày</span>
          <div className="hero-region-buttons">
            {REGION_BUTTONS.map((r) => (
              <motion.button
                key={r.key}
                type="button"
                className="hero-region-btn"
                onClick={() => onSelectRegion(r.key)}
                whileHover={{ y: -4, scale: 1.03 }}
                whileTap={{ scale: 0.97 }}
              >
                {r.label}
              </motion.button>
            ))}
          </div>
        </motion.div>
      </div>

      <motion.div
        className="hero-scroll-hint"
        animate={{ y: [0, 8, 0] }}
        transition={{ duration: 1.8, repeat: Infinity, ease: "easeInOut" }}
        aria-hidden
      >
        ↓
      </motion.div>
    </section>
  );
}
