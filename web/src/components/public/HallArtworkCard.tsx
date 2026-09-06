import { motion } from "framer-motion";
import { Heart, Sparkles, Trophy } from "lucide-react";
import { useState, type CSSProperties } from "react";
import type { ArtworkWithMeta } from "../../lib/artworkApi";
import { artworkImageURL, artworkPictureSources } from "../../lib/artworkImage";

/** Giải hiển thị trên card - chỉ những trường thực sự dùng để vẽ. */
export type HallCardAward = { name: string; color_hex: string };

/**
 * Tổng lượt thả cảm xúc. reaction_counts là map <loại, số lượng> nên phải
 * cộng dồn; null (chưa ai thả) trả 0 để card vẫn hiện con số thay vì trống.
 */
function totalReactions(item: ArtworkWithMeta): number {
  const counts = item.reaction_counts;
  if (!counts) return 0;
  let sum = 0;
  for (const value of Object.values(counts)) sum += value;
  return sum;
}

/**
 * Card khung gỗ trang trọng nhất — trang Bảng vàng.
 *
 * Dày hơn Tác phẩm tiêu biểu: khuôn walnut kép → gờ vàng → ốc góc →
 * passe-partout → lỗ ảnh. Dùng chung cho bục (variant="podium", chỉ khung
 * ảnh; chữ do bệ bục đảm nhiệm) và dải chuyên đề (variant="grid"). Có giải
 * thì khung mạ và quầng sáng theo màu giải.
 */
export function HallArtworkCard({
  item,
  award,
  index,
  onClick,
  variant = "grid",
  medalLabel,
  accent,
  className,
}: {
  item: ArtworkWithMeta;
  award?: HallCardAward;
  index: number;
  onClick: () => void;
  variant?: "grid" | "podium";
  /** Nhãn huy chương góc trái ảnh: "Vàng" / "Bạc" / "Đồng". */
  medalLabel?: string;
  /** Màu nhấn của card (viền + badge) khi không lấy theo màu giải. */
  accent?: string;
  /** Class thêm (vd. featured-card) để trang khác nhuộm tông riêng. */
  className?: string;
}) {
  const [loaded, setLoaded] = useState(false);
  const likes = totalReactions(item);
  const gradeLine = item.class_name ? `${item.grade_label} · ${item.class_name}` : item.grade_label;
  const badgeColor = accent ?? award?.color_hex;
  const ariaBits = [item.student_name, gradeLine, award?.name, `tác phẩm “${item.title}”`].filter(Boolean);

  return (
    <motion.button
      type="button"
      className={`hall-card hall-card--${variant}${award ? " hall-card--awarded" : ""}${className ? ` ${className}` : ""}`}
      style={badgeColor ? ({ "--hall-accent": badgeColor } as CSSProperties) : undefined}
      onClick={onClick}
      initial={{ opacity: 0, y: 22, rotate: index % 2 === 0 ? -0.8 : 0.8 }}
      whileInView={{ opacity: 1, y: 0, rotate: 0 }}
      viewport={{ once: true, margin: "-40px" }}
      transition={{ duration: 0.55, delay: Math.min(index, 6) * 0.06, ease: [0.22, 1, 0.36, 1] }}
      whileHover={{ y: -10, rotate: 0 }}
      whileTap={{ scale: 0.985 }}
      aria-label={ariaBits.join(", ")}
    >
      <span className="hall-card-fillet">
        <span className="hall-card-key hall-card-key--tl" aria-hidden />
        <span className="hall-card-key hall-card-key--tr" aria-hidden />
        <span className="hall-card-key hall-card-key--bl" aria-hidden />
        <span className="hall-card-key hall-card-key--br" aria-hidden />
        <span className="hall-card-mat">
          <span className="hall-card-media">
            {!loaded && <span className="hall-card-skeleton" aria-hidden />}
            <picture>
              {artworkPictureSources(item, "thumb").map((s) => (
                <source key={s.type} srcSet={s.srcSet} type={s.type} />
              ))}
              <img
                src={artworkImageURL(item, "thumb")}
                alt=""
                loading="lazy"
                decoding="async"
                width={item.width}
                height={item.height}
                onLoad={() => setLoaded(true)}
                onError={() => setLoaded(true)}
                data-loaded={loaded ? "true" : "false"}
              />
            </picture>

            {/* Lớp phủ tối dần về đáy: giữ cho nhãn nổi trên ảnh luôn đọc
                được, kể cả khi tranh có vùng sáng trắng ngay dưới đó. */}
            <span className="hall-card-scrim" aria-hidden />
            {/* Dải sáng quét chéo khi rê chuột. */}
            <span className="hall-card-sheen" aria-hidden />

            {medalLabel && (
              <span className="hall-card-medal">
                <Trophy size={13} strokeWidth={2.4} aria-hidden />
                {medalLabel}
              </span>
            )}

            <span className="hall-card-likes">
              <Heart size={13} strokeWidth={2.2} aria-hidden />
              {likes}
            </span>
          </span>
        </span>
      </span>

      {variant === "grid" && (
        <span className="hall-card-body">
          {award && (
            <span className="hall-card-award-line">
              <Sparkles size={13} strokeWidth={2.2} aria-hidden />
              {award.name}
            </span>
          )}
          <span className="hall-card-label">Tác phẩm</span>
          <span className="hall-card-title">{item.title}</span>
          <span className="hall-card-label">Họa sĩ nhí</span>
          <span className="hall-card-student">{item.student_name}</span>
          <span className="hall-card-label">Lớp</span>
          <span className="hall-card-grade">{gradeLine}</span>
          <span className="hall-card-foot">
            <span className="hall-card-region">{item.school_name}</span>
          </span>
        </span>
      )}
    </motion.button>
  );
}
