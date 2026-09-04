import { Navigate, Route, Routes } from "react-router-dom";
import { ToastHost } from "./components/ToastHost";
import AdminLayout from "./pages/admin/AdminLayout";
import ArtworksListPage from "./pages/admin/ArtworksListPage";
import ArtworksUploadPage from "./pages/admin/ArtworksUploadPage";
import AwardsPage from "./pages/admin/AwardsPage";
import Dashboard from "./pages/admin/Dashboard";
import Login from "./pages/admin/Login";
import FeaturedArtworksPage from "./pages/public/FeaturedArtworksPage";
import GalleryPage from "./pages/public/GalleryPage";
import HallOfFamePage from "./pages/public/HallOfFamePage";
import HomePage from "./pages/public/HomePage";
import PublicLayout from "./pages/public/PublicLayout";
import UploadTool from "./pages/UploadTool";

/**
 * Router gốc của toàn bộ app:
 * - "/" render thẳng trang triển lãm (PublicLayout + HomePage) — đây là
 *   trang chủ khi truy cập tên miền gốc, URL giữ nguyên là "/".
 * - "/trien-lam" là alias trỏ Navigate về "/" (giữ tương thích link cũ).
 * - "/upload" công cụ upload S3 nội bộ hiện có (trước đây ở "/"), giữ
 *   nguyên chức năng, chỉ đổi đường dẫn để nhường "/" cho trang triển lãm.
 * - "/admin/login" trang đăng nhập Google OAuth.
 * - "/admin/*" khu vực quản trị, bảo vệ bởi AdminLayout (redirect login nếu
 *   chưa đăng nhập).
 * - "/tac-pham-tieu-bieu", "/phong-trien-lam", "/bang-vang" các trang con
 *   của khu vực public, dùng chung PublicLayout (navbar cố định).
 */
export default function App() {
  return (
    <>
      <ToastHost />
      <Routes>
        <Route path="/" element={<PublicLayout />}>
          <Route index element={<HomePage />} />
          <Route path="tac-pham-tieu-bieu" element={<FeaturedArtworksPage />} />
          <Route path="phong-trien-lam" element={<GalleryPage />} />
          <Route path="bang-vang" element={<HallOfFamePage />} />
        </Route>
        <Route path="/trien-lam" element={<Navigate to="/" replace />} />
        <Route path="/upload" element={<UploadTool />} />
        <Route path="/admin/login" element={<Login />} />
        <Route path="/admin" element={<AdminLayout />}>
          <Route index element={<Dashboard />} />
          <Route path="artworks" element={<ArtworksListPage />} />
          <Route path="artworks/upload" element={<ArtworksUploadPage />} />
          <Route path="awards" element={<AwardsPage />} />
        </Route>
      </Routes>
    </>
  );
}
