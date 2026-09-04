import { Route, Routes } from "react-router-dom";
import { ToastHost } from "./components/ToastHost";
import AdminLayout from "./pages/admin/AdminLayout";
import ArtworksListPage from "./pages/admin/ArtworksListPage";
import ArtworksUploadPage from "./pages/admin/ArtworksUploadPage";
import AwardsPage from "./pages/admin/AwardsPage";
import Dashboard from "./pages/admin/Dashboard";
import Login from "./pages/admin/Login";
import PublicGallery from "./pages/public/PublicGallery";
import UploadTool from "./pages/UploadTool";

/**
 * Router gốc của toàn bộ app:
 * - "/" giữ nguyên UploadTool (công cụ upload S3 nội bộ hiện có) để không
 *   gián đoạn người dùng đang dùng công cụ này.
 * - "/admin/login" trang đăng nhập Google OAuth.
 * - "/admin/*" khu vực quản trị, bảo vệ bởi AdminLayout (redirect login nếu
 *   chưa đăng nhập).
 * - "/trien-lam" trang public kỷ niệm 20 năm VAS, không cần đăng nhập.
 */
export default function App() {
  return (
    <>
      <ToastHost />
      <Routes>
        <Route path="/" element={<UploadTool />} />
        <Route path="/admin/login" element={<Login />} />
        <Route path="/admin" element={<AdminLayout />}>
          <Route index element={<Dashboard />} />
          <Route path="artworks" element={<ArtworksListPage />} />
          <Route path="artworks/upload" element={<ArtworksUploadPage />} />
          <Route path="awards" element={<AwardsPage />} />
        </Route>
        <Route path="/trien-lam" element={<PublicGallery />} />
      </Routes>
    </>
  );
}
