import { Suspense, lazy } from "react";
import { Navigate, Route, Routes } from "react-router-dom";
import { RouteFallback } from "./components/RouteFallback";
import { RouteProgress } from "./components/RouteProgress";
import { ToastHost } from "./components/ToastHost";
import PublicLayout from "./pages/public/PublicLayout";
import HomePage from "./pages/public/HomePage";

/**
 * Router gốc của toàn bộ app:
 * - "/" render thẳng trang triển lãm (PublicLayout + HomePage) — đây là
 *   trang chủ khi truy cập tên miền gốc, URL giữ nguyên là "/".
 * - "/trien-lam" là alias trỏ Navigate về "/" (giữ tương thích link cũ).
 * - "/admin/login" trang đăng nhập Google OAuth.
 * - "/admin/*" khu vực quản trị, bảo vệ bởi AdminLayout (redirect login nếu
 *   chưa đăng nhập).
 * - "/tac-pham-tieu-bieu", "/phong-trien-lam", "/bang-vang", "/thu-ngo" các
 *   trang con của khu vực public, dùng chung PublicLayout (navbar cố định).
 *
 * Tách bundle theo route (React.lazy):
 * PublicLayout + HomePage nạp tĩnh vì đó là điểm vào của gần như mọi khách
 * truy cập — lazy chúng chỉ thêm một vòng chờ mạng trước khi thấy nội dung.
 * Mọi thứ còn lại nạp theo nhu cầu. Quan trọng nhất là khu /admin: trước
 * đây phụ huynh vào xem tranh phải tải kèm cả dashboard quản trị và thư
 * viện biểu đồ recharts (~400KB) — không truy cập được từ trang public.
 */

const FeaturedArtworksPage = lazy(() => import("./pages/public/FeaturedArtworksPage"));
const GalleryPage = lazy(() => import("./pages/public/GalleryPage"));
const HallOfFamePage = lazy(() => import("./pages/public/HallOfFamePage"));
const OpenLetterPage = lazy(() => import("./pages/public/OpenLetterPage"));

const AdminLayout = lazy(() => import("./pages/admin/AdminLayout"));
const AdminLogin = lazy(() => import("./pages/admin/Login"));
const Dashboard = lazy(() => import("./pages/admin/Dashboard"));
const ArtworksListPage = lazy(() => import("./pages/admin/ArtworksListPage"));
const ArtworksUploadPage = lazy(() => import("./pages/admin/ArtworksUploadPage"));
const AwardsPage = lazy(() => import("./pages/admin/AwardsPage"));
const TopicCategoriesPage = lazy(() => import("./pages/admin/TopicCategoriesPage"));
const AdminNotFoundPage = lazy(() => import("./pages/admin/NotFoundPage"));

const NotFoundPage = lazy(() => import("./pages/public/NotFoundPage"));

export default function App() {
  return (
    <>
      <ToastHost />
      {/* Thanh tiến trình đỉnh màn hình. Đặt NGOÀI Suspense: điều hướng bằng
          <Link> chạy trong startTransition nên trang cũ được giữ lại và
          fallback không hiện - lúc đó đây là chỉ báo duy nhất cho người dùng
          biết cú bấm đã được nhận. */}
      <RouteProgress />
      {/* Một Suspense bọc ngoài toàn bộ Routes là đủ: mỗi lần chỉ có một
          route đang khớp, nên không có hai chunk cùng treo fallback. */}
      <Suspense fallback={<RouteFallback />}>
        <Routes>
          <Route path="/" element={<PublicLayout />}>
            <Route index element={<HomePage />} />
            <Route path="tac-pham-tieu-bieu" element={<FeaturedArtworksPage />} />
            <Route path="phong-trien-lam" element={<GalleryPage />} />
            <Route path="bang-vang" element={<HallOfFamePage />} />
            <Route path="thu-ngo" element={<OpenLetterPage />} />
            {/* Mọi đường dẫn lạ dưới layout public rơi vào đây, thay vì hiện
                nhầm HomePage - xem P2.4 trong docs/plan/02-roadmap.md. */}
            <Route path="*" element={<NotFoundPage />} />
          </Route>
          <Route path="/trien-lam" element={<Navigate to="/" replace />} />
          <Route path="/admin/login" element={<AdminLogin />} />
          <Route path="/admin" element={<AdminLayout />}>
            <Route index element={<Dashboard />} />
            <Route path="artworks" element={<ArtworksListPage />} />
            <Route path="artworks/upload" element={<ArtworksUploadPage />} />
            <Route path="awards" element={<AwardsPage />} />
            <Route path="topic-categories" element={<TopicCategoriesPage />} />
            {/* 404 riêng cho khu quản trị: giữ nguyên sidebar/header thay vì
                rơi ra ngoài layout public. */}
            <Route path="*" element={<AdminNotFoundPage />} />
          </Route>
        </Routes>
      </Suspense>
    </>
  );
}
