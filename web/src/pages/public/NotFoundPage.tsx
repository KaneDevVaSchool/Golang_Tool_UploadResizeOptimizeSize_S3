import { motion } from "framer-motion";
import { Home, Images, Sparkles } from "lucide-react";
import { Link } from "react-router-dom";
import { FeaturedGardenScene } from "../../components/public/FeaturedGardenScene";
import { usePageMeta } from "../../hooks/usePageMeta";
import { fadeUp } from "../../lib/motionPresets";

const PAGE_TITLE = "Không tìm thấy trang — VA Schools";
const PAGE_DESCRIPTION =
  "Trang bạn tìm không tồn tại. Quay lại Khu vườn nghệ thuật VA Schools để khám phá các tác phẩm dự thi.";

const SHORTCUTS = [
  { to: "/", label: "Trang chủ", icon: Home },
  { to: "/tac-pham-tieu-bieu", label: "Tác phẩm tiêu biểu", icon: Sparkles },
  { to: "/phong-trien-lam", label: "Phòng triển lãm", icon: Images },
];

/**
 * Trang 404 khu vực public - dùng lại nền "khu vườn tổ tiên" của
 * FeaturedGardenScene (đồi núi + bướm, không có tầng thú chạy) thay vì dựng
 * một nền riêng, để không lệch phong cách với Tác phẩm tiêu biểu/Bảng vàng.
 *
 * Mascot rồng (vas-mascot-wave.png) đứng giữa, "lạc lối" - chỉ khác trạng
 * thái vẫy tay bình thường ở chỗ nghiêng đầu + có biển gỗ ghi 404, gợi ý
 * "trang này không có ở đây" thay vì lỗi khô khan.
 */
export default function NotFoundPage() {
  usePageMeta({ title: PAGE_TITLE, description: PAGE_DESCRIPTION });

  return (
    <div className="not-found-page">
      <FeaturedGardenScene critters={false} />

      <div className="not-found-content">
        <motion.div className="not-found-mascot-wrap" custom={0} variants={fadeUp} initial="hidden" animate="show">
          <motion.img
            className="not-found-mascot"
            src="/images/vas-mascot-wave.png"
            alt="Rồng nhỏ VASchools ngơ ngác vì lạc đường"
            animate={{ rotate: [-3, 3, -3], y: [0, -6, 0] }}
            transition={{ duration: 3.2, repeat: Infinity, ease: "easeInOut" }}
          />
          <span className="not-found-mascot-shadow" aria-hidden />
          <span className="not-found-signpost" aria-hidden>
            404
          </span>
        </motion.div>

        <motion.p className="not-found-kicker" custom={1} variants={fadeUp} initial="hidden" animate="show">
          Ơ kìa, lạc đường rồi
        </motion.p>
        <motion.h1 className="not-found-title" custom={2} variants={fadeUp} initial="hidden" animate="show">
          Không tìm thấy trang này
        </motion.h1>
        <motion.p className="not-found-subtitle" custom={3} variants={fadeUp} initial="hidden" animate="show">
          Đường dẫn bạn vào có thể đã đổi chỗ, hoặc chưa từng tồn tại. Chú rồng nhỏ của trường mời bạn quay lại
          khu vườn triển lãm nhé.
        </motion.p>

        <motion.div className="not-found-actions" custom={4} variants={fadeUp} initial="hidden" animate="show">
          {SHORTCUTS.map(({ to, label, icon: Icon }) => (
            <Link key={to} to={to} className={`not-found-btn${to === "/" ? " not-found-btn--primary" : ""}`}>
              <Icon size={18} aria-hidden />
              {label}
            </Link>
          ))}
        </motion.div>
      </div>
    </div>
  );
}
