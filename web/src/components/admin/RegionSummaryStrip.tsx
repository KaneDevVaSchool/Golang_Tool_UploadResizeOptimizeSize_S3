import type { LucideIcon } from "lucide-react";
import { StatCard } from "./StatCard";
import { REGION_LABEL } from "../../lib/chartTheme";
import type { RegionSummaryItem } from "../../lib/dashboardApi";

type RegionSummaryStripProps = {
  data: RegionSummaryItem[];
  loading: boolean;
  /** Icon dùng chung cho mọi ô "tác phẩm". */
  artworkIcon: LucideIcon;
  /** Icon dùng chung cho mọi ô "học sinh". */
  studentIcon: LucideIcon;
};

/**
 * Dải 6 ô nhỏ (3 khu vực x tác phẩm/học sinh) ở đầu trang Tác phẩm/Giải
 * thưởng/Nhóm chủ đề - tái dùng nguyên StatCard với tone theo khu vực
 * (saigon/cantho/vungtau, xem StatCard.tsx) thay vì viết lại component
 * riêng, và nguyên CSS .dashboard-stat-grid của Dashboard.
 */
export function RegionSummaryStrip({ data, loading, artworkIcon, studentIcon }: RegionSummaryStripProps) {
  if (loading && data.length === 0) {
    return (
      <div className="dashboard-stat-grid">
        {[0, 1, 2, 3, 4, 5].map((i) => (
          <div key={i} className="dashboard-skeleton-card" />
        ))}
      </div>
    );
  }

  // Lỗi tải hoặc chưa có dữ liệu - ẩn hẳn dải, không chiếm chỗ trống vô
  // nghĩa trên trang mà chức năng chính không phụ thuộc vào nó.
  if (data.length === 0) return null;

  return (
    <div className="dashboard-stat-grid">
      {data.map((region) => (
        <StatCard
          key={`${region.region}-artworks`}
          label={`Tác phẩm · ${REGION_LABEL[region.region]}`}
          value={region.artworks}
          icon={artworkIcon}
          tone={region.region}
        />
      ))}
      {data.map((region) => (
        <StatCard
          key={`${region.region}-students`}
          label={`Học sinh · ${REGION_LABEL[region.region]}`}
          value={region.students}
          icon={studentIcon}
          tone={region.region}
        />
      ))}
    </div>
  );
}
