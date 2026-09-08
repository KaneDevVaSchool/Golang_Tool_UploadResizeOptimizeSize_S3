import { motion, useMotionValue, useReducedMotion, useSpring, useTransform } from "framer-motion";
import { ArrowRight, Rocket } from "lucide-react";
import type { PointerEvent } from "react";
import { fadeUp } from "../../lib/motionPresets";
import { webpOf } from "../../lib/staticImage";

type EducationLevelCardProps = {
  level: "primary" | "secondary";
  artworkCount: number;
  onSelect: () => void;
  /** true khi card đang ở vị trí "back" của stack chồng lệch (bị card kia đè
   * lên từ bên trái) - đẩy nội dung sang phải để tiêu đề/mô tả không bị che
   * khuất bởi mép card phía trước. */
  peeking?: boolean;
};

const COPY = {
  primary: {
    kicker: "VƯỜN ƯƠM SẮC MÀU",
    title: "Khối Tiểu Học",
    subtitle: "Lớp 1 đến Lớp 5",
    body: "Ở tuổi này, mặt trời có thể màu tím và cả nhà mình đều biết bay. Các em vẽ đúng những gì mình thấy trong đầu — không rào đón, không sợ sai.",
    gradeRange: "5 khối lớp (Lớp 1 – 5)",
  },
  secondary: {
    kicker: "XƯỞNG VẼ CỦA TUỔI TRẺ",
    title: "Khối Trung Học",
    subtitle: "THCS & THPT · Lớp 6 đến Lớp 12",
    body: "Nét cọ đã vững hơn, và điều muốn nói cũng nhiều hơn. Mỗi bức tranh là một lần các em thử trả lời: mình là ai, và mình nhìn thế giới thế nào.",
    gradeRange: "7 khối lớp (Lớp 6 – 12)",
  },
} as const;

const SECONDARY_STARS = [
  { top: "12%", left: "18%", delay: 0 },
  { top: "20%", left: "72%", delay: 0.4 },
  { top: "38%", left: "88%", delay: 0.9 },
  { top: "28%", left: "8%", delay: 1.3 },
  { top: "16%", left: "48%", delay: 1.8 },
  { top: "42%", left: "62%", delay: 0.6 },
];

const PARALLAX_SPRING = { stiffness: 70, damping: 18, mass: 0.45 };

/**
 * Theo dõi con trỏ trong card Trung học - mỗi lớp cảnh (sao / sóng / đảo)
 * lấy cùng một cặp toạ độ đã spring, nhân hệ số sâu khác nhau. Lớp càng
 * gần "máy quay" càng dịch nhiều. Tôn trọng prefers-reduced-motion và bỏ
 * qua pointer thô (touch) vì parallax theo ngón tay dễ giật trên mobile.
 */
function useSecondaryParallax(enabled: boolean) {
  const reduceMotion = useReducedMotion();
  const mx = useMotionValue(0);
  const my = useMotionValue(0);
  const sx = useSpring(mx, PARALLAX_SPRING);
  const sy = useSpring(my, PARALLAX_SPRING);
  const live = enabled && !reduceMotion;

  const starsX = useTransform(sx, (v) => v * 16);
  const starsY = useTransform(sy, (v) => v * 10);
  const waveFarX = useTransform(sx, (v) => v * 28);
  const waveFarY = useTransform(sy, (v) => v * 8);
  const waveNearX = useTransform(sx, (v) => v * 48);
  const waveNearY = useTransform(sy, (v) => v * 14);
  const islandX = useTransform(sx, (v) => v * 70);
  const islandY = useTransform(sy, (v) => v * 30);
  const rotateX = useTransform(sy, (v) => (live ? v * -6 : 0));
  const rotateY = useTransform(sx, (v) => (live ? v * 8 : 0));

  function onPointerMove(e: PointerEvent<HTMLButtonElement>) {
    if (!live || e.pointerType !== "mouse") return;
    const rect = e.currentTarget.getBoundingClientRect();
    mx.set((e.clientX - rect.left) / rect.width - 0.5);
    my.set((e.clientY - rect.top) / rect.height - 0.5);
  }

  function onPointerLeave() {
    mx.set(0);
    my.set(0);
  }

  return {
    reduceMotion: Boolean(reduceMotion),
    live,
    layers: { starsX, starsY, waveFarX, waveFarY, waveNearX, waveNearY, islandX, islandY, rotateX, rotateY },
    onPointerMove,
    onPointerLeave,
  };
}

type SecondaryLayers = ReturnType<typeof useSecondaryParallax>["layers"];

/**
 * Cảnh đêm nhiều lớp của card Trung học: stars.jpg (xa nhất) → sóng xa →
 * đảo → sóng gần. Mỗi lớp bọc motion.div riêng để pointer-parallax không
 * đè lên animation idle (ken-burns / sóng trôi / đảo nhấp) của thẻ img.
 */
function SecondaryNightScene({ layers, reduceMotion }: { layers: SecondaryLayers; reduceMotion: boolean }) {
  return (
    <>
      <motion.div className="edu-card-layer edu-card-layer--stars" style={{ x: layers.starsX, y: layers.starsY }}>
        <img className="edu-card-stars" src="/images/stars.jpg" alt="" />
        <img className="edu-card-stars edu-card-stars--twinkle" src="/images/stars.jpg" alt="" />
      </motion.div>

      {SECONDARY_STARS.map((star, i) => (
        <motion.span
          key={i}
          className="edu-card-star"
          style={{ top: star.top, left: star.left }}
          animate={reduceMotion ? undefined : { opacity: [0.15, 1, 0.15], scale: [0.8, 1.15, 0.8] }}
          transition={{ duration: 2.6, repeat: Infinity, ease: "easeInOut", delay: star.delay }}
        />
      ))}

      <span className="edu-card-shooting-star" />

      <motion.div className="edu-card-layer edu-card-layer--wave-far" style={{ x: layers.waveFarX, y: layers.waveFarY }}>
        <img className="edu-card-wave edu-card-wave--far" src="/images/wave.png" alt="" />
      </motion.div>

      <motion.div className="edu-card-layer edu-card-layer--island" style={{ x: layers.islandX, y: layers.islandY }}>
        <picture>
          <source srcSet={webpOf("/images/island.png")} type="image/webp" />
          <motion.img
            className="edu-card-island"
            src="/images/island.png"
            alt=""
            animate={reduceMotion ? undefined : { y: [0, -10, 0], rotate: [0, 1.1, 0] }}
            transition={{ duration: 5.8, repeat: Infinity, ease: "easeInOut" }}
          />
        </picture>
      </motion.div>

      <motion.div className="edu-card-layer edu-card-layer--wave-near" style={{ x: layers.waveNearX, y: layers.waveNearY }}>
        <img className="edu-card-wave edu-card-wave--near" src="/images/wave.png" alt="" />
      </motion.div>

      <div className="edu-card-vignette" />
    </>
  );
}

/**
 * 1 trong 2 card lớn ở Section 2 (cổng chọn khối) - badge kicker, tiêu đề
 * lớn (font Fraunces), mô tả, stat pill, CTA, hover lift+scale, theme riêng
 * từng cấp học. Tiểu học: đồi lá xanh + cây cọ + lá dương xỉ (cùng hệ ảnh
 * Hero). Trung học: diorama đêm nhiều lớp (stars.jpg / wave.png / island.png)
 * với parallax theo con trỏ và animation idle, tương phản khu vườn ban ngày.
 */
export function EducationLevelCard({ level, artworkCount, onSelect, peeking = false }: EducationLevelCardProps) {
  const copy = COPY[level];
  const parallax = useSecondaryParallax(level === "secondary");

  return (
    <motion.button
      type="button"
      className={`edu-card edu-card--${level}${peeking ? " edu-card--peeking" : ""}`}
      onClick={onSelect}
      onPointerMove={parallax.onPointerMove}
      onPointerLeave={parallax.onPointerLeave}
      initial={{ opacity: 0, y: 32 }}
      whileInView={{ opacity: 1, y: 0 }}
      viewport={{ once: true, margin: "-80px" }}
      whileHover={{ y: -8, scale: 1.015 }}
      whileTap={{ scale: 0.99 }}
      style={
        level === "secondary"
          ? {
              rotateX: parallax.layers.rotateX,
              rotateY: parallax.layers.rotateY,
              transformPerspective: 900,
            }
          : undefined
      }
      transition={{
        duration: 0.55,
        ease: [0.22, 1, 0.36, 1],
        rotateX: { duration: 0 },
        rotateY: { duration: 0 },
      }}
    >
      <div className="edu-card-decor" aria-hidden>
        {level === "primary" ? (
          <>
            <picture>
              <source srcSet={webpOf("/images/parallax/hill5.png")} type="image/webp" />
              <img className="edu-card-hill" src="/images/parallax/hill5.png" alt="" />
            </picture>
            <picture>
              <source srcSet={webpOf("/images/parallax/tree.png")} type="image/webp" />
              <motion.img
                className="edu-card-art edu-card-art--tree"
                src="/images/parallax/tree.png"
                alt=""
                animate={{ y: [0, -6, 0], rotate: [0, 1.5, 0] }}
                transition={{ duration: 5, repeat: Infinity, ease: "easeInOut" }}
              />
            </picture>
            <picture>
              <source srcSet={webpOf("/images/parallax/leaf.png")} type="image/webp" />
              <motion.img
                className="edu-card-art edu-card-art--leaf"
                src="/images/parallax/leaf.png"
                alt=""
                animate={{ rotate: [0, -3, 0] }}
                transition={{ duration: 4.2, repeat: Infinity, ease: "easeInOut", delay: 0.4 }}
              />
            </picture>
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
          <SecondaryNightScene layers={parallax.layers} reduceMotion={parallax.reduceMotion} />
        )}
      </div>

      {/* Nội dung chữ tự stagger riêng (kicker -> title -> subtitle -> body ->
          stats -> CTA) qua fadeUp/custom={i}, tách khỏi whileInView của
          .edu-card cha (vốn chỉ lo lift+fade cả khối) - viewport "once" đặt
          lại ở đây vì .edu-card-content không phải cùng node với button cha,
          Framer Motion không tự propagate variants qua 2 lần whileInView
          khác trigger nên mỗi container animate độc lập theo đúng lúc chính
          nó vào khung nhìn (2 node cùng nằm trong .edu-card nên thực tế vào
          khung nhìn gần như đồng thời). */}
      <motion.div
        className="edu-card-content"
        initial="hidden"
        whileInView="show"
        viewport={{ once: true, margin: "-80px" }}
      >
        <motion.span className="edu-card-kicker" custom={0} variants={fadeUp}>
          {level === "secondary" && <Rocket size={13} />}
          {copy.kicker}
        </motion.span>
        <motion.h3 className="edu-card-title" custom={1} variants={fadeUp}>
          {copy.title}
        </motion.h3>
        <motion.p className="edu-card-subtitle" custom={2} variants={fadeUp}>
          {copy.subtitle}
        </motion.p>
        <motion.p className="edu-card-body" custom={3} variants={fadeUp}>
          {copy.body}
        </motion.p>
        <motion.div className="edu-card-stats" custom={4} variants={fadeUp}>
          <span className="edu-card-stat-pill">{artworkCount} tác phẩm</span>
          <span className="edu-card-stat-sep">{copy.gradeRange}</span>
        </motion.div>
      </motion.div>

      <motion.div
        className="edu-card-cta"
        initial="hidden"
        whileInView="show"
        viewport={{ once: true, margin: "-80px" }}
        custom={5}
        variants={fadeUp}
      >
        <span>Xem tranh khối này</span>
        <ArrowRight size={18} className="edu-card-cta-arrow" />
      </motion.div>
    </motion.button>
  );
}
