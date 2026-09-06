import { Outlet, useLocation } from "react-router-dom";
import { MascotAssistant } from "../../components/public/MascotAssistant";
import { PublicFooter } from "../../components/public/PublicFooter";
import { PublicNavbar } from "../../components/public/PublicNavbar";
import { useDeviceTier } from "../../hooks/useDeviceTier";
import { useScrollableBody } from "../../hooks/useScrollableBody";
import "../../styles/public.css";

/** Lá và hoa của lớp trang trí nền, xếp theo thứ tự xuất hiện trên màn hình. */
const FOREST_LEAVES = ["🍃", "🌿", "🍂", "🌸", "🍁", "🌼", "🍃", "🌺"];

/**
 * Số lá theo sức máy. Mỗi lá là một phần tử position:fixed mang
 * `will-change: transform` + `drop-shadow` và animation vô hạn - tức là một
 * lớp compositor thường trú phải vẽ lại filter mỗi khung hình, nhân với số
 * lá, trên MỌI trang public. Desktop kham được; điện thoại thì đây là chi
 * phí nền cộng dồn vào mọi thứ khác đang chạy.
 */
const LEAF_BUDGET: Record<string, number> = {
  full: FOREST_LEAVES.length,
  light: 4,
  minimal: 3,
};

/**
 * Layout gốc cho toàn bộ khu vực /trien-lam/*: navbar cố định dùng chung +
 * <Outlet /> render trang con theo route. Thay cho PublicGallery cũ vốn gộp
 * tất cả section vào 1 trang cuộn dài kiểu landing page.
 */
export default function PublicLayout() {
  useScrollableBody();
  const tier = useDeviceTier();
  const location = useLocation();
  const leafCount = LEAF_BUDGET[tier] ?? FOREST_LEAVES.length;

  return (
    <div className="public-gallery-page">
      {/* Lớp trang trí "khu vườn" cố định toàn màn hình: lá + hoa bay nhẹ,
          thuần CSS (không ảnh, không JS), đứng sau mọi nội dung.
          Số lá giảm dần trên máy yếu - xem LEAF_BUDGET. Lấy cách quãng để
          lá còn rải đều ngang màn hình thay vì dồn về một phía. */}
      <div className="forest-ambient" aria-hidden="true">
        {FOREST_LEAVES.filter((_, i) => i % Math.ceil(FOREST_LEAVES.length / leafCount) === 0).map((leaf, i) => (
          <span key={i} className={`forest-leaf forest-leaf--${i + 1}`}>
            {leaf}
          </span>
        ))}
      </div>
      <PublicNavbar />
      {/* key theo pathname để mỗi trang con vào bằng một hoạt cảnh fade+trượt
          ngắn: sau quãng chờ chunk, nội dung "hiện ra" thay vì bụp một cái.
          Đặt key ở <main> chứ không ở layout - navbar, footer và lớp lá nền
          giữ nguyên, không nhấp nháy theo. */}
      <main className="public-page-main route-enter" key={location.pathname}>
        <Outlet />
      </main>
      <PublicFooter />
      <MascotAssistant />
    </div>
  );
}
