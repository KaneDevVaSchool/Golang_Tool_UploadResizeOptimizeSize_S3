import { motion } from "framer-motion";
import { Heart, Sparkles, Trophy } from "lucide-react";
import { useState, type CSSProperties } from "react";
import type { ArtworkWithMeta } from "../../lib/artworkApi";

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
 * Card tác phẩm bo tròn cho trang Bảng vàng: ảnh bo góc, badge giải nổi ở
 * góc trên ảnh, số lượt thích ở góc dưới, chân card là tên tác phẩm + học
 * sinh + lớp.
 *
 * Dùng chung cho cả 3 ô podium (variant="podium", chỉ ảnh + huy chương +
 * lượt thích, phần chữ do bệ bục đảm nhiệm) lẫn lưới bên dưới
 * (variant="grid", đầy đủ chân card). Một component cho cả hai chỗ để card
 * ở mọi khu vực của trang co giãn và hover giống hệt nhau.
 */
export function HallArtworkCard({
  item,
  award,
  index,
  onClick,
  variant = "grid",
  medalLabel,
  accent,
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
}) {
  const [loaded, setLoaded] = useState(false);
  const likes = totalReactions(item);
  const gradeLine = item.class_name ? `${item.grade_label} · ${item.class_name}` : item.grade_label;
  const badgeColor = accent ?? award?.color_hex;
  const ariaBits = [item.student_name, gradeLine, award?.name, `tác phẩm “${item.title}”`].filter(Boolean);

  return (
    <motion.button
      type="button"
      className={`hall-card hall-card--${variant}`}
      style={badgeColor ? ({ "--hall-accent": badgeColor } as CSSProperties) : undefined}
      onClick={onClick}
      initial={{ opacity: 0, y: 22 }}
      whileInView={{ opacity: 1, y: 0 }}
      viewport={{ once: true, margin: "-40px" }}
      transition={{ duration: 0.5, delay: Math.min(index, 6) * 0.06, ease: [0.22, 1, 0.36, 1] }}
      whileHover={{ y: -8 }}
      whileTap={{ scale: 0.985 }}
      aria-label={ariaBits.join(", ")}
    >
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

        {/* Ở lưới, tên giải nằm trên ảnh (podium đã có badge riêng nổi
            trên đỉnh bục nên không lặp lại ở đây). */}
        {variant === "grid" && award && (
          <span className="hall-card-award">
            <Sparkles size={13} strokeWidth={2.4} aria-hidden />
            {award.name}
          </span>
        )}

        <span className="hall-card-likes">
          <Heart size={13} strokeWidth={2.2} aria-hidden />
          {likes}
        </span>
      </span>

      {variant === "grid" && (
        <span className="hall-card-body">
          <strong className="hall-card-title">{item.title}</strong>
          <span className="hall-card-student">
            {item.student_name} · {gradeLine}
          </span>
          <span className="hall-card-foot">
            <span className="hall-card-region">{item.school_name}</span>
            <span className="hall-card-more">Xem chi tiết →</span>
          </span>
        </span>
      )}
    </motion.button>
  );
}
