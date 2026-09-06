import { useEffect, useState } from "react";
import { fetchRegionSummary, type RegionSummaryItem } from "../lib/dashboardApi";

type State = {
  data: RegionSummaryItem[];
  loading: boolean;
};

/**
 * useRegionSummary: gọi GET /api/v1/admin/dashboard/region-summary. Dùng
 * chung cho 3 trang quản trị (Tác phẩm/Giải thưởng/Nhóm chủ đề) hiển thị
 * dải card "số tác phẩm + số học sinh theo khu vực" ở đầu trang - tránh lặp
 * useEffect/useState/AbortController 3 lần (tiền lệ: useAdminAuth.ts dùng
 * chung cho AdminLayout/AdminHeader).
 */
export function useRegionSummary() {
  const [state, setState] = useState<State>({ data: [], loading: true });

  useEffect(() => {
    const controller = new AbortController();
    fetchRegionSummary(controller.signal)
      .then((data) => setState({ data, loading: false }))
      .catch((err: unknown) => {
        if (controller.signal.aborted) return;
        // Dải card phụ - lỗi ở đây không đáng toast làm phiền thao tác
        // chính (khác Dashboard, nơi số liệu là nội dung chính của trang).
        void err;
        setState({ data: [], loading: false });
      });
    return () => controller.abort();
  }, []);

  return state;
}
