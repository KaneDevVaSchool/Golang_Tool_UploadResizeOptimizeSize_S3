import { motion } from "framer-motion";
import { Heart, Sparkles } from "lucide-react";
import { useState, type CSSProperties } from "react";
import type { ArtworkWithMeta } from "../../lib/artworkApi";
import { artworkImageURL, artworkPictureSources } from "../../lib/artworkImage";

/** Giải trên card tiêu biểu — chỉ những trường dùng để vẽ badge. */
export type FeaturedCardAward = { name: string; color_hex: string };

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
 * Card tranh trang Tác phẩm tiêu biểu.
 *
 * Khung gỗ gụ chữ nhật + móc treo: khuôn mahogany → gờ vàng → passe-partout
 * kem → lỗ ảnh 4:3. Khác walnut của phòng triển lãm / bảng vàng.
 * Tên học sinh luôn mang danh xưng "Họa sĩ nhí". Có giải thì khung mạ thêm.
 */
export function FeaturedArtworkCard({
  item,
  award,
  index,
  onClick,
}: {
  item: ArtworkWithMeta;
  award?: FeaturedCardAward;
  index: number;
  onClick: () => void;
}) {
  const [loaded, setLoaded] = useState(false);
  const likes = totalReactions(item);
  const gradeLine = item.class_name ? `${item.grade_label} · ${item.class_name}` : item.grade_label;
  const ariaBits = [item.student_name, gradeLine, award?.name, `tác phẩm “${item.title}”`].filter(Boolean);

  return (
    <motion.button
      type="button"
      className={`featured-card${award ? " featured-card--awarded" : ""}`}
      style={award ? ({ "--featured-accent": award.color_hex } as CSSProperties) : undefined}
      onClick={onClick}
      initial={{ opacity: 0, y: 22, rotate: index % 2 === 0 ? -0.6 : 0.6 }}
      whileInView={{ opacity: 1, y: 0, rotate: 0 }}
      viewport={{ once: true, margin: "-40px" }}
      transition={{ duration: 0.55, delay: Math.min(index, 6) * 0.06, ease: [0.22, 1, 0.36, 1] }}
      whileHover={{ y: -10, rotate: 0 }}
      whileTap={{ scale: 0.985 }}
      aria-label={ariaBits.join(", ")}
    >
      <span className="featured-card-hanger" aria-hidden>
        <span className="featured-card-nail" />
        <span className="featured-card-wire" />
      </span>
      <span className="featured-card-fillet">
        <span className="featured-card-mat">
          <span className="featured-card-media">
            {!loaded && <span className="featured-card-skeleton" aria-hidden />}
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
                draggable={false}
                onContextMenu={(event) => event.preventDefault()}
                onLoad={() => setLoaded(true)}
                onError={() => setLoaded(true)}
                data-loaded={loaded ? "true" : "false"}
              />
            </picture>
            <span className="featured-card-scrim" aria-hidden />
            <span className="featured-card-sheen" aria-hidden />
            {award && (
              <span className="featured-card-award">
                <Sparkles size={13} strokeWidth={2.4} aria-hidden />
                {award.name}
              </span>
            )}
            <span className="featured-card-likes">
              <Heart size={13} strokeWidth={2.2} aria-hidden />
              {likes}
            </span>
          </span>
        </span>
      </span>

      <span className="featured-card-body">
        <span className="featured-card-label">Tác phẩm</span>
        <strong className="featured-card-title">{item.title}</strong>
        <span className="featured-card-label">Họa sĩ nhí</span>
        <span className="featured-card-student">{item.student_name}</span>
        <span className="featured-card-grade">{gradeLine}</span>
        <span className="featured-card-region">{item.school_name}</span>
      </span>
    </motion.button>
  );
}
