import { LayoutDashboard } from "lucide-react";
import { Link } from "react-router-dom";
import { AdminPageHeader } from "../../components/admin/AdminPageHeader";
import { webpOf } from "../../lib/staticImage";

/**
 * 404 trong khu quản trị - render bên trong AdminLayout (sidebar + header
 * vẫn còn nguyên, người dùng đã đăng nhập) nên KHÔNG dùng theme khu vườn của
 * bản public: chỉ mascot đứng yên + card gọn, giữ đúng phong cách nghiêm túc
 * đã có ở admin-loading-screen/ConfirmDialog (xem 06-frontend.md mục 7 -
 * không lặp ngôn ngữ thị giác giữa 2 khu vực).
 */
export default function AdminNotFoundPage() {
  return (
    <>
      <AdminPageHeader title="Không tìm thấy trang" subtitle="Đường dẫn này không tồn tại trong khu quản trị" />

      <div className="admin-not-found">
        <picture>
          <source srcSet={webpOf("/images/vas-mascot-wave.png")} type="image/webp" />
          <img className="admin-not-found-mascot" src="/images/vas-mascot-wave.png" alt="" aria-hidden />
        </picture>
        <p className="admin-not-found-code">404</p>
        <h2 className="admin-not-found-title">Trang bạn tìm không có ở đây</h2>
        <p className="admin-not-found-text">
          Có thể đường dẫn đã đổi, hoặc mục này chưa từng tồn tại. Kiểm tra lại menu bên trái, hoặc quay về
          trang tổng quan.
        </p>
        <Link to="/admin" className="admin-not-found-btn">
          <LayoutDashboard size={18} strokeWidth={2} />
          Về trang tổng quan
        </Link>
      </div>
    </>
  );
}
