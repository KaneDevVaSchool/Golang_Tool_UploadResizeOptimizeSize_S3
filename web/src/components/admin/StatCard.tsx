import { animate, motion, useMotionValue, useTransform } from "framer-motion";
import type { LucideIcon } from "lucide-react";
import { useEffect } from "react";
import { formatNumber, REGION_COLOR } from "../../lib/chartTheme";

// Ba tone "saigon"/"cantho"/"vungtau" thêm cho RegionSummaryStrip (dải card
// theo khu vực ở trang Tác phẩm/Giải thưởng/Nhóm chủ đề) - tái dùng nguyên
// StatCard thay vì viết lại component, màu lấy đúng REGION_COLOR để chỉ một
// nơi định nghĩa màu khu vực.
type StatTone =
  | "tri-thuc"
  | "khai-phong"
  | "nhan-ai"
  | "trach-nhiem"
  | "ban-linh"
  | "neutral"
  | "saigon"
  | "cantho"
  | "vungtau";

type StatCardProps = {
  label: string;
  value: number;
  icon: LucideIcon;
  tone: StatTone;
  /** Dòng ngữ cảnh dưới nhãn - trả lời "con số này có ý nghĩa gì". */
  hint?: string;
  /** Chuỗi 12 điểm vẽ sparkline. Bỏ trống thì không vẽ. */
  spark?: number[];
  /** Biến động so với kỳ trước, đơn vị %. */
  delta?: number;
};

/** Sắc độ dùng cho vạch nhấn + icon của từng tone. */
const TONE_INK: Record<StatTone, string> = {
  "tri-thuc": "#b3003f",
  "khai-phong": "#1750b5",
  "nhan-ai": "#009082",
  "trach-nhiem": "#a67f42",
  "ban-linh": "#a8551b",
  neutral: "#6b5a5d",
  saigon: REGION_COLOR.saigon,
  cantho: REGION_COLOR.cantho,
  vungtau: REGION_COLOR.vungtau,
};

function AnimatedCount({ value }: { value: number }) {
  const motionValue = useMotionValue(0);
  const rounded = useTransform(motionValue, (v) => formatNumber(Math.round(v)));

  useEffect(() => {
    const controls = animate(motionValue, value, { duration: 0.9, ease: [0.22, 1, 0.36, 1] });
    return () => controls.stop();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [value]);

  return <motion.span>{rounded}</motion.span>;
}

/**
 * Sparkline 12 điểm vẽ tay bằng SVG - không kéo recharts vào một hình nhỏ
 * cỡ này. viewBox cố định + preserveAspectRatio="none" để hình co giãn theo
 * bề ngang card mà không cần đo DOM.
 */
function Sparkline({ points, color }: { points: number[]; color: string }) {
  if (points.length < 2) return null;
  const max = Math.max(...points, 1);
  const step = 100 / (points.length - 1);
  const path = points
    .map((p, i) => `${i === 0 ? "M" : "L"} ${(i * step).toFixed(2)} ${(28 - (p / max) * 26).toFixed(2)}`)
    .join(" ");
  const area = `${path} L 100 28 L 0 28 Z`;

  return (
    <svg className="stat-card-spark" viewBox="0 0 100 28" preserveAspectRatio="none" aria-hidden>
      <path d={area} fill={color} opacity={0.1} />
      <path d={path} fill="none" stroke={color} strokeWidth={2} strokeLinejoin="round" strokeLinecap="round" vectorEffect="non-scaling-stroke" />
    </svg>
  );
}

/**
 * Ô chỉ số của Dashboard.
 *
 * Cố ý dùng mặt giấy trắng + một vạch màu mảnh thay cho nền gradient đặc như
 * bản trước: bốn khối gradient bão hoà cạnh nhau đọc rất ồn, và chữ trắng
 * trên gradient vàng không bao giờ đủ tương phản. Ở đây màu chỉ làm nhiệm vụ
 * định danh (vạch + icon), còn số liệu mặc mực đen như mọi chữ khác.
 *
 * Số dùng chữ số tỉ lệ (không tabular-nums): ở cỡ lớn, chữ số rộng bằng nhau
 * làm những số như "121" trông rời rạc.
 */
export function StatCard({ label, value, icon: Icon, tone, hint, spark, delta }: StatCardProps) {
  const ink = TONE_INK[tone];
  const hasDelta = typeof delta === "number" && Number.isFinite(delta);

  return (
    <motion.div
      className="stat-card"
      style={{ "--stat-ink": ink } as React.CSSProperties}
      initial={{ opacity: 0, y: 14 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.4, ease: [0.22, 1, 0.36, 1] }}
    >
      <div className="stat-card-top">
        <span className="stat-card-icon">
          <Icon size={18} strokeWidth={2} />
        </span>
        {hasDelta && (
          <span
            className={`stat-card-delta ${delta >= 0 ? "stat-card-delta--up" : "stat-card-delta--down"}`}
          >
            {delta >= 0 ? "▲" : "▼"} {Math.abs(delta).toFixed(0)}%
          </span>
        )}
      </div>

      <div className="stat-card-value">
        <AnimatedCount value={value} />
      </div>
      <div className="stat-card-label">{label}</div>
      {hint && <div className="stat-card-hint">{hint}</div>}
      {spark && spark.length > 1 && <Sparkline points={spark} color={ink} />}
    </motion.div>
  );
}
