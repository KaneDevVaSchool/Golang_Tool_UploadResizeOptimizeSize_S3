import { Outlet } from "react-router-dom";
import { PublicNavbar } from "../../components/public/PublicNavbar";
import { useScrollableBody } from "../../hooks/useScrollableBody";
import "../../styles/public.css";

/**
 * Layout gốc cho toàn bộ khu vực /trien-lam/*: navbar cố định dùng chung +
 * <Outlet /> render trang con theo route. Thay cho PublicGallery cũ vốn gộp
 * tất cả section vào 1 trang cuộn dài kiểu landing page.
 */
export default function PublicLayout() {
  useScrollableBody();

  return (
    <div className="public-gallery-page">
      {/* Lớp trang trí "khu vườn" cố định toàn màn hình: lá + hoa bay nhẹ,
          thuần CSS (không ảnh, không JS), đứng sau mọi nội dung. */}
      <div className="forest-ambient" aria-hidden="true">
        <span className="forest-leaf forest-leaf--1">🍃</span>
        <span className="forest-leaf forest-leaf--2">🌿</span>
        <span className="forest-leaf forest-leaf--3">🍂</span>
        <span className="forest-leaf forest-leaf--4">🌸</span>
        <span className="forest-leaf forest-leaf--5">🍁</span>
        <span className="forest-leaf forest-leaf--6">🌼</span>
        <span className="forest-leaf forest-leaf--7">🍃</span>
        <span className="forest-leaf forest-leaf--8">🌺</span>
      </div>
      <PublicNavbar />
      <main className="public-page-main">
        <Outlet />
      </main>
      <footer className="public-footer">
        <span className="public-footer-leaf" aria-hidden>
          🌿
        </span>
        <p>Kỷ niệm 20 năm thành lập Trường Việt Mỹ · 2006 – 2026</p>
        <span className="public-footer-leaf public-footer-leaf--right" aria-hidden>
          🌿
        </span>
      </footer>
    </div>
  );
}
