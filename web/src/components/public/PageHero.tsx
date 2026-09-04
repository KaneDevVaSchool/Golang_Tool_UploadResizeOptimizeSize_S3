import { motion } from "framer-motion";
import type { ReactNode } from "react";

/**
 * Header dùng chung cho mọi trang con /trien-lam/* (Tác phẩm tiêu biểu,
 * Phòng triển lãm, Bảng vàng) - một "mini-hero" nền pastel tiếp nối trực
 * tiếp không khí của Hero ở trang chủ, thay cho page-heading cũ vốn chỉ là
 * chữ đứng trơn trên nền trắng. Cùng 1 component đảm bảo cả 3 trang tuyệt
 * đối đồng nhất (kicker pill, tiêu đề, gạch dưới gradient, mô tả) thay vì
 * mỗi trang tự lặp lại markup riêng.
 *
 * variant "light": nền pastel sáng (mặc định, dùng ở Tác phẩm tiêu biểu +
 * Phòng triển lãm). variant "dark": nền tối đồng bộ .billboard-section
 * (Bảng vàng) - chữ trắng, kicker mờ kính.
 */
export function PageHero({
  kicker,
  title,
  description,
  variant = "light",
  children,
}: {
  kicker: string;
  title: string;
  description?: string;
  variant?: "light" | "dark";
  children?: ReactNode;
}) {
  return (
    <header className={`page-hero page-hero--${variant}`}>
      <div className="page-hero-decor" aria-hidden>
        <span className="page-hero-leaf page-hero-leaf--1">🌿</span>
        <span className="page-hero-leaf page-hero-leaf--2">🍃</span>
      </div>
      <motion.div
        className="page-heading"
        initial={{ opacity: 0, y: 16 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.4 }}
      >
        <span className="section-kicker">{kicker}</span>
        <h1>{title}</h1>
        {description && <p className="page-heading-sub">{description}</p>}
        {children}
      </motion.div>
    </header>
  );
}
