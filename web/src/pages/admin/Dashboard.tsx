import { Images, MapPin, School, TrendingUp } from "lucide-react";
import { useEffect, useState } from "react";
import {
  Bar,
  BarChart,
  Cell,
  Pie,
  PieChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { AdminPageHeader } from "../../components/admin/AdminPageHeader";
import { StatCard } from "../../components/admin/StatCard";
import { TopArtworksList } from "../../components/admin/TopArtworksList";
import { fetchDashboardStats, type DashboardStats } from "../../lib/dashboardApi";
import { toast } from "../../lib/toastBus";

const REGION_LABEL: Record<string, string> = {
  saigon: "Sài Gòn",
  cantho: "Cần Thơ",
  vungtau: "Vũng Tàu",
};

const REGION_COLOR: Record<string, string> = {
  saigon: "var(--vas-tri-thuc)",
  cantho: "var(--vas-khai-phong)",
  vungtau: "var(--vas-nhan-ai)",
};

const GRADE_COLOR = ["#009082", "#1750b5", "#c49c57", "#9a0036", "#725139"];

export default function Dashboard() {
  const [stats, setStats] = useState<DashboardStats | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const controller = new AbortController();
    setLoading(true);
    fetchDashboardStats(controller.signal)
      .then(setStats)
      .catch((err: unknown) => {
        if (controller.signal.aborted) return;
        toast.error(err instanceof Error ? err.message : "Không tải được số liệu thống kê");
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false);
      });
    return () => controller.abort();
  }, []);

  // Header vẫn hiện ở cả trạng thái đang tải / rỗng, tránh thanh header
  // trống rồi mới "nhảy" ra tiêu đề khi có dữ liệu.
  if (loading && !stats) {
    return (
      <>
        <AdminPageHeader title="Tổng quan" subtitle="Đang gom số liệu…" />
        <div className="admin-page-placeholder">Đang tải số liệu triển lãm, chờ một chút nhé…</div>
      </>
    );
  }
  if (!stats) {
    return (
      <>
        <AdminPageHeader title="Tổng quan" subtitle="Chưa có số liệu" />
        <div className="admin-page-placeholder">
          Chưa có dữ liệu để hiển thị. Hãy tải những tác phẩm đầu tiên lên nhé!
        </div>
      </>
    );
  }

  const regionData = (Object.keys(REGION_LABEL) as Array<keyof typeof REGION_LABEL>).map((key) => ({
    key,
    name: REGION_LABEL[key],
    value: stats.total_by_region[key as "saigon" | "cantho" | "vungtau"] ?? 0,
    color: REGION_COLOR[key],
  }));

  const gradeData = stats.total_by_grade.map((g) => ({
    label: g.label,
    count: g.count,
    level: g.education_level,
  }));

  return (
    <div className="dashboard-page">
      <AdminPageHeader
        title="Tổng quan"
        subtitle={`Triển lãm đang có ${stats.total_artworks} tác phẩm`}
      />

      <section className="dashboard-stat-grid">
        <StatCard label="Tổng số tác phẩm" value={stats.total_artworks} icon={Images} tone="tri-thuc" />
        <StatCard
          label="Tác phẩm Sài Gòn"
          value={stats.total_by_region.saigon ?? 0}
          icon={MapPin}
          tone="khai-phong"
        />
        <StatCard
          label="Tác phẩm Cần Thơ"
          value={stats.total_by_region.cantho ?? 0}
          icon={MapPin}
          tone="nhan-ai"
        />
        <StatCard
          label="Tác phẩm Vũng Tàu"
          value={stats.total_by_region.vungtau ?? 0}
          icon={MapPin}
          tone="trach-nhiem"
        />
      </section>

      <section className="dashboard-grid-2col">
        <div className="dashboard-card">
          <h2>Phân bổ theo khối lớp</h2>
          <div className="dashboard-chart-wrap">
            <ResponsiveContainer width="100%" height={280}>
              <BarChart data={gradeData} margin={{ top: 8, right: 8, bottom: 8, left: 0 }}>
                <XAxis dataKey="label" tick={{ fontSize: 11 }} interval={0} angle={-35} textAnchor="end" height={50} />
                <YAxis allowDecimals={false} tick={{ fontSize: 11 }} />
                <Tooltip formatter={(value) => [`${value ?? 0} tác phẩm`, ""]} labelFormatter={(l) => l} />
                <Bar dataKey="count" radius={[6, 6, 0, 0]}>
                  {gradeData.map((entry, index) => (
                    <Cell key={entry.label} fill={GRADE_COLOR[index % GRADE_COLOR.length]} />
                  ))}
                </Bar>
              </BarChart>
            </ResponsiveContainer>
          </div>
        </div>

        <div className="dashboard-card">
          <h2>Phân bổ theo khu vực</h2>
          <div className="dashboard-chart-wrap">
            <ResponsiveContainer width="100%" height={280}>
              <PieChart>
                <Pie
                  data={regionData}
                  dataKey="value"
                  nameKey="name"
                  innerRadius={55}
                  outerRadius={95}
                  paddingAngle={3}
                >
                  {regionData.map((entry) => (
                    <Cell key={entry.key} fill={entry.color} />
                  ))}
                </Pie>
                <Tooltip formatter={(value) => [`${value ?? 0} tác phẩm`, ""]} />
              </PieChart>
            </ResponsiveContainer>
            <ul className="dashboard-pie-legend">
              {regionData.map((entry) => (
                <li key={entry.key}>
                  <span className="dashboard-pie-legend-dot" style={{ background: entry.color }} />
                  {entry.name} ({entry.value})
                </li>
              ))}
            </ul>
          </div>
        </div>
      </section>

      <section className="dashboard-grid-2col">
        <div className="dashboard-card">
          <h2>
            <School size={18} /> Tác phẩm nhiều nhất
          </h2>
          <ul className="dashboard-school-list">
            {stats.top_schools.map((school) => (
              <li key={school.school_id}>
                <span className="dashboard-school-name">{school.name}</span>
                <span className="dashboard-school-bar-track">
                  <span
                    className="dashboard-school-bar-fill"
                    style={{
                      width: `${stats.top_schools[0]?.count ? (school.count / stats.top_schools[0].count) * 100 : 0}%`,
                    }}
                  />
                </span>
                <span className="dashboard-school-count">{school.count}</span>
              </li>
            ))}
          </ul>
        </div>

        <div className="dashboard-card">
          <h2>
            <TrendingUp size={18} /> Top 10 tác phẩm hàng đầu
          </h2>
          <TopArtworksList items={stats.top_artworks} />
        </div>
      </section>
    </div>
  );
}
