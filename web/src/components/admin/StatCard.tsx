import { motion, useMotionValue, useTransform, animate } from "framer-motion";
import type { LucideIcon } from "lucide-react";
import { useEffect } from "react";

type StatCardProps = {
  label: string;
  value: number;
  icon: LucideIcon;
  tone: "nhan-ai" | "ban-linh" | "tri-thuc" | "trach-nhiem" | "khai-phong";
};

const TONE_GRADIENT: Record<StatCardProps["tone"], string> = {
  "nhan-ai": "linear-gradient(135deg, #009082 0%, #00b3a1 100%)",
  "ban-linh": "linear-gradient(135deg, #725139 0%, #96714f 100%)",
  "tri-thuc": "linear-gradient(135deg, #7a002b 0%, #b3003f 100%)",
  "trach-nhiem": "linear-gradient(135deg, #a87e3d 0%, #c49c57 100%)",
  "khai-phong": "linear-gradient(135deg, #123f8c 0%, #1750b5 100%)",
};

function AnimatedCount({ value }: { value: number }) {
  const motionValue = useMotionValue(0);
  const rounded = useTransform(motionValue, (v) => Math.round(v).toLocaleString("vi-VN"));

  useEffect(() => {
    const controls = animate(motionValue, value, { duration: 0.9, ease: [0.22, 1, 0.36, 1] });
    return () => controls.stop();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [value]);

  return <motion.span>{rounded}</motion.span>;
}

/**
 * Card thống kê Dashboard - gradient theo 1 trong 5 màu cốt lõi VAS, số đếm
 * animate tăng dần khi có dữ liệu mới (framer-motion animate()), blur
 * decoration trang trí góc card.
 */
export function StatCard({ label, value, icon: Icon, tone }: StatCardProps) {
  return (
    <motion.div
      className="stat-card"
      style={{ background: TONE_GRADIENT[tone] }}
      initial={{ opacity: 0, y: 16 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.4, ease: [0.22, 1, 0.36, 1] }}
    >
      <div className="stat-card-decor" aria-hidden />
      <div className="stat-card-icon">
        <Icon size={22} strokeWidth={2} />
      </div>
      <div className="stat-card-value">
        <AnimatedCount value={value} />
      </div>
      <div className="stat-card-label">{label}</div>
    </motion.div>
  );
}
