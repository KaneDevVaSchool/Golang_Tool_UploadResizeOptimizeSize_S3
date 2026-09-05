import { motion } from "framer-motion";

/**
 * Dải đồi SVG ngăn giữa Hero và section "Cổng thông tin" (EducationLevelGate)
 * - lấy cảm hứng từ mẫu layered-hill parallax cổ điển (CodePen
 * aarongriffis/rYPERJ: nhiều lớp <path> núi vẽ tay + cây mọc lên khi vào
 * khung hình), nhưng viết lại bằng SVG path thuần + Framer Motion
 * (whileInView) thay cho GSAP/TimelineMax/ScrollMagic/DrawSVGPlugin để khớp
 * stack React đang dùng trong dự án, không cần thêm thư viện mới.
 *
 * Thay cho dải ::before/::after cũ của .edu-gate-wrap (radial-gradient +
 * chuỗi ký tự cây lặp) - đồi thật ở đây tiếp nối trực quan chân trời của
 * HeroSection phía trên, đồng thời tông màu tự chuyển từ xanh lá (khớp
 * .edu-card--primary/Tiểu học, bên trái) sang navy (khớp
 * .edu-card--secondary/Trung học, bên phải) để dẫn mắt người xem xuống đúng
 * 2 card bên dưới trước khi họ chọn khối.
 *
 * 2 lớp: đồi xa (mờ, thấp, tĩnh) và đồi gần (đậm, cao hơn, mang cụm cây) -
 * chỉ 2 lớp (không 6 như bản gốc) vì đây là dải chuyển tiếp ngắn giữa 2
 * section, không phải toàn bộ khung cảnh Hero.
 */
export function HillDivider() {
  return (
    <div className="hill-divider" aria-hidden>
      {/* defs dùng chung cho cả 2 lớp - gradient ngang xanh lá (trái, khớp
          Tiểu học) → navy (phải, khớp Trung học); đặt trong 1 svg 0x0 riêng
          để không nhân đôi id khi HillDivider render nhiều lần trên trang. */}
      <svg width="0" height="0">
        <defs>
          <linearGradient id="hill-divider-gradient-far" x1="0%" y1="0%" x2="100%" y2="0%">
            <stop offset="0%" stopColor="#7dd3c0" />
            <stop offset="55%" stopColor="#a7c4e8" />
            <stop offset="100%" stopColor="#3b3f6b" />
          </linearGradient>
          <linearGradient id="hill-divider-gradient-near" x1="0%" y1="0%" x2="100%" y2="0%">
            <stop offset="0%" stopColor="#3fae86" />
            <stop offset="55%" stopColor="#4c6a9e" />
            <stop offset="100%" stopColor="#1e1b4b" />
          </linearGradient>
        </defs>
      </svg>
      {/* Mỗi lớp đồi trượt lên từ dưới + fade khi vào khung nhìn, so le delay
          (xa trước, gần sau) - cùng tinh thần "lớp xa chậm/tĩnh hơn, lớp gần
          nổi bật hơn" như HeroSection phía trên, chỉ chạy 1 lần (viewport
          once) vì đây là điểm chuyển tiếp đi qua 1 lần khi cuộn xuống, không
          phải yếu tố lặp lại. */}
      <motion.svg
        className="hill-divider-far"
        viewBox="0 0 1200 120"
        preserveAspectRatio="none"
        xmlns="http://www.w3.org/2000/svg"
        initial={{ opacity: 0, y: 24 }}
        whileInView={{ opacity: 1, y: 0 }}
        viewport={{ once: true, margin: "-40px" }}
        transition={{ duration: 0.7, ease: [0.22, 1, 0.36, 1] }}
      >
        <path d="M0,90 C150,40 300,100 450,70 C600,45 750,95 900,60 C1030,32 1120,75 1200,55 L1200,120 L0,120 Z" />
      </motion.svg>
      <motion.svg
        className="hill-divider-near"
        viewBox="0 0 1200 120"
        preserveAspectRatio="none"
        xmlns="http://www.w3.org/2000/svg"
        initial={{ opacity: 0, y: 32 }}
        whileInView={{ opacity: 1, y: 0 }}
        viewport={{ once: true, margin: "-40px" }}
        transition={{ duration: 0.7, delay: 0.12, ease: [0.22, 1, 0.36, 1] }}
      >
        <path d="M0,70 C120,100 220,55 360,80 C520,108 620,60 760,85 C900,110 1000,68 1200,92 L1200,120 L0,120 Z" />
      </motion.svg>

      {/* Cụm cây mọc lên khi cuộn tới - dùng lại HillTree bên dưới, so le
          delay để không "mọc" đồng loạt cứng nhắc. */}
      <div className="hill-divider-trees">
        <HillTree className="hill-divider-tree hill-divider-tree--1" delay={0} />
        <HillTree className="hill-divider-tree hill-divider-tree--2" delay={0.15} />
        <HillTree className="hill-divider-tree hill-divider-tree--3" delay={0.3} />
        <HillTree className="hill-divider-tree hill-divider-tree--4" delay={0.1} />
      </div>
    </div>
  );
}

/**
 * 1 cây thông cách điệu (path đơn giản, không phải bản branch-by-branch
 * đầy đủ như CodePen gốc - đủ để gợi "rừng thưa" ở quy mô nhỏ của dải chia
 * này). scale từ gốc (bottom) lên, giống cảm giác "mọc" của DrawSVG gốc,
 * chỉ chạy 1 lần khi vào viewport (viewport once) để không lặp lại mỗi lần
 * cuộn qua lại.
 */
function HillTree({ className, delay = 0 }: { className?: string; delay?: number }) {
  return (
    <motion.svg
      className={className}
      viewBox="0 0 20 40"
      initial={{ scaleY: 0, opacity: 0 }}
      whileInView={{ scaleY: 1, opacity: 1 }}
      viewport={{ once: true, margin: "-40px" }}
      transition={{ duration: 0.6, delay, ease: [0.22, 1, 0.36, 1] }}
      style={{ transformOrigin: "bottom" }}
    >
      <path className="hill-divider-tree-trunk" d="M9,26 L9,38 a1,1 0 0,0 2,0 L11,26 Z" />
      <path className="hill-divider-tree-leaf" d="M10,2 L18,20 L12,20 L12,17 L2,17 L10,2 Z" />
      <path className="hill-divider-tree-leaf" d="M10,10 L17,26 L3,26 L10,10 Z" />
    </motion.svg>
  );
}
