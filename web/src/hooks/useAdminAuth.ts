import { useCallback, useEffect, useState } from "react";
import { adminRequest, UnauthorizedError } from "../lib/adminApi";

export type AdminUser = {
  id: number;
  email: string;
  name: string;
  avatar_url: string;
  role: string;
  is_active: boolean;
};

type AuthState = {
  user: AdminUser | null;
  loading: boolean;
  error: string | null;
};

/**
 * useAdminAuth: gọi GET /api/v1/admin/auth/me để biết admin hiện tại có
 * đăng nhập không. AdminLayout dùng hook này để bảo vệ route (redirect
 * /admin/login nếu user === null sau khi loading xong) và để hiển thị
 * header (tên/avatar). refetch() dùng sau khi logout để clear state ngay.
 */
export function useAdminAuth() {
  const [state, setState] = useState<AuthState>({ user: null, loading: true, error: null });

  const refetch = useCallback(async () => {
    setState((s) => ({ ...s, loading: true, error: null }));
    try {
      const user = await adminRequest<AdminUser>("/api/v1/admin/auth/me");
      setState({ user, loading: false, error: null });
    } catch (err) {
      if (err instanceof UnauthorizedError) {
        setState({ user: null, loading: false, error: null });
        return;
      }
      setState({ user: null, loading: false, error: err instanceof Error ? err.message : "Lỗi không xác định" });
    }
  }, []);

  useEffect(() => {
    void refetch();
  }, [refetch]);

  const clearUser = useCallback(() => {
    setState({ user: null, loading: false, error: null });
  }, []);

  return { ...state, refetch, clearUser };
}
