import { useState } from "react";
import { Navigate, Outlet } from "react-router-dom";
import { AdminHeader } from "../../components/admin/AdminHeader";
import { AdminSidebar } from "../../components/admin/AdminSidebar";
import { useAdminAuth } from "../../hooks/useAdminAuth";
import { useScrollableBody } from "../../hooks/useScrollableBody";
import "../../styles/admin.css";

/**
 * Layout khung cho toàn bộ /admin/* (trừ /admin/login) - bảo vệ route qua
 * useAdminAuth: chưa đăng nhập -> redirect /admin/login. Render
 * AdminSidebar + AdminHeader + <Outlet/> (nested routes: Dashboard,
 * ArtworksListPage, AwardsPage...).
 */
export default function AdminLayout() {
  useScrollableBody();
  const { user, loading, clearUser } = useAdminAuth();
  const [mobileSidebarOpen, setMobileSidebarOpen] = useState(false);

  if (loading) {
    return (
      <div className="admin-loading-screen">
        <img src="/images/vas-mascot-wave.png" alt="" aria-hidden />
        <p>Đang tải…</p>
      </div>
    );
  }

  if (!user) {
    return <Navigate to="/admin/login" replace />;
  }

  return (
    <div className="admin-shell">
      <AdminSidebar mobileOpen={mobileSidebarOpen} onCloseMobile={() => setMobileSidebarOpen(false)} />
      <div className="admin-main">
        <AdminHeader user={user} onLogout={clearUser} onOpenMobileSidebar={() => setMobileSidebarOpen(true)} />
        <main className="admin-content">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
