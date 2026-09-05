import { motion } from "framer-motion";
import { Crown, Trophy } from "lucide-react";
import { useState, type CSSProperties } from "react";
import type { ArtworkWithMeta, Award } from "../../lib/artworkApi";

/**
 * Khung tranh phòng trưng bày cho lưới Tác phẩm tiêu biểu.
 *
 * Từ ngoài vào trong, mô phỏng khung bảo tàng thật:
 * móc + dây treo → khuôn gỗ vân (moulding) → gờ vàng (fillet) →
 * passe-partout kem → lỗ ảnh lõm + kính → tấm đồng khắc tên gắn đáy khung.
 * Tỉ lệ khung theo A3 (1 : √2), đổi hướng theo metadata width/height.
 */

export type FrameOrientation = "portrait" | "landscape";

/**
 * Chỉ những trường tấm đồng thực sự dùng để hiển thị giải. Khai báo hẹp
 * như vậy để nhận được cả Award đầy đủ (admin) lẫn payload giải rút gọn
 * của API public (/billboard không trả is_active).
 */
export type FrameAward = Pick<Award, "name" | "color_hex">;

/**
 * Hướng tranh suy ra từ kích thước thật của file (width/height do backend
 * ghi lúc upload). Ảnh vuông hoặc thiếu metadata -> mặc định dọc.
 *
 * Hướng KHÔNG làm khung to nhỏ khác nhau - mọi khung cùng một cỡ - nó chỉ
 * quyết định hình dạng lỗ cắt trong passe-partout.
 */
function orientationOf(item: ArtworkWithMeta): FrameOrientation {
  if (item.width && item.height && item.width > item.height) return "landscape";
  return "portrait";
}

export function FeaturedArtworkFrame({
  item,
  index,
  onClick,
  podium = 0,
  award,
}: {
  item: ArtworkWithMeta;
  index: number;
  onClick: () => void;
  podium?: number;
  /**
   * Giải cần hiện trên tấm đồng. Mặc định lấy giải đầu trong item.awards,
   * nhưng một tác phẩm có thể mang nhiều giải - trang Bảng vàng xếp theo
   * từng hạng nên phải chỉ đích danh giải ứng với hạng đang trưng bày.
   */
  award?: FrameAward;
}) {
  const topAward = award ?? item.awards?.[0];
  const orientation = orientationOf(item);
  const [loaded, setLoaded] = useState(false);
  const awarded = Boolean(topAward);
  const podiumClass = podium >= 1 && podium <= 3 ? ` artwork-frame--podium-${podium}` : "";
  const awardClass = podium >= 1 && podium <= 3 ? ` artwork-frame-award--podium-${podium}` : "";
  const gradeLine = item.class_name ? `${item.grade_label} · ${item.class_name}` : item.grade_label;
  const ariaBits = [`tác phẩm “${item.title}”`, item.student_name, gradeLine, topAward?.name].filter(Boolean);

  return (
    <motion.button
      type="button"
      className={`artwork-frame artwork-frame--${orientation}${awarded ? " artwork-frame--awarded" : ""}${podiumClass}`}
      style={topAward ? ({ "--award-color": topAward.color_hex } as CSSProperties) : undefined}
      onClick={onClick}
      initial={{ opacity: 0, y: 28, rotate: index % 2 === 0 ? -1.6 : 1.6 }}
      whileInView={{ opacity: 1, y: 0, rotate: 0 }}
      viewport={{ once: true, margin: "-40px" }}
      transition={{ duration: 0.55, delay: Math.min(index % 6, 5) * 0.06, ease: [0.22, 1, 0.36, 1] }}
      whileHover={{ y: -10 }}
      whileTap={{ scale: 0.985 }}
      aria-label={ariaBits.join(", ")}
    >
      <span className="artwork-frame-hanger" aria-hidden>
        <span className="artwork-frame-nail" />
        <span className="artwork-frame-wire" />
      </span>

      <span className="artwork-frame-body">
        <span className="artwork-frame-fillet">
          <span className="artwork-frame-key artwork-frame-key--tl" aria-hidden />
          <span className="artwork-frame-key artwork-frame-key--tr" aria-hidden />
          <span className="artwork-frame-key artwork-frame-key--bl" aria-hidden />
          <span className="artwork-frame-key artwork-frame-key--br" aria-hidden />
          <span className="artwork-frame-mat">
            {/* Tỉ lệ lỗ cắt do class .artwork-frame--landscape trên nút cha
                quyết định (xem .artwork-frame-window trong public.css), không
                đặt inline nữa - style inline sẽ ghi đè mất quy tắc đảo
                height/width mà khổ ngang cần để nằm gọn trong vùng mat vuông. */}
            <span className="artwork-frame-window-slot">
              <span className="artwork-frame-window">
                {!loaded && <span className="artwork-frame-skeleton" aria-hidden />}
                <img
                  src={item.thumbnail_url || item.image_url}
                  alt=""
                  loading="lazy"
                  decoding="async"
                  width={item.width}
                  height={item.height}
                  onLoad={() => setLoaded(true)}
                  onError={() => setLoaded(true)}
                  data-loaded={loaded ? "true" : "false"}
                />
                <span className="artwork-frame-glass" aria-hidden />
              </span>
            </span>

            <span className="artwork-frame-plaque">
              {podium === 1 && (
                <span className="artwork-frame-crown" aria-hidden>
                  <Crown strokeWidth={1.75} />
                </span>
              )}
              <span className="artwork-frame-screw" aria-hidden />
              <span className="artwork-frame-plaque-inner">
                {topAward && (
                  <span className={`artwork-frame-award${awardClass}`} style={{ backgroundColor: topAward.color_hex }}>
                    <Trophy strokeWidth={2.4} aria-hidden />
                    {topAward.name}
                  </span>
                )}
                <span className="artwork-frame-work">
                  <span className="artwork-frame-title-label">Tác phẩm</span>
                  <em className="artwork-frame-title-text">{item.title}</em>
                </span>
                <span className="artwork-frame-plaque-rule" aria-hidden />
                <strong className="artwork-frame-student">{item.student_name}</strong>
                <span className="artwork-frame-byline">
                  <span className="artwork-frame-grade">{gradeLine}</span>
                </span>
              </span>
              <span className="artwork-frame-screw" aria-hidden />
            </span>
          </span>
        </span>
      </span>
    </motion.button>
  );
}
