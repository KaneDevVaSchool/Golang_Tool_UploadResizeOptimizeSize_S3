import { Eye, Heart, ImageOff, Images, MapPin, MessageSquare } from "lucide-react";
import { useMemo, useEffect, useState } from "react";
import { AdminPageHeader } from "../../components/admin/AdminPageHeader";
import { StatCard } from "../../components/admin/StatCard";
import { artworkImageURL, artworkPictureSources } from "../../lib/artworkImage";
import { REGION_COLOR, REGION_LABEL, formatNumber } from "../../lib/chartTheme";
import {
  fetchDashboardStats,
  type DashboardStats,
  type SchoolCoverage,
  type TopArtwork,
} from "../../lib/dashboardApi";
import { toast } from "../../lib/toastBus";

type RegionKey = "saigon" | "cantho" | "vungtau";

type RegionSummary = {
  key: RegionKey;
  name: string;
  color: string;
  artworks: number;
  schools: SchoolCoverage[];
  /** Tác phẩm nổi bật nhất của khu vực trong top tương tác toàn triển lãm -
   *  có thể rỗng nếu khu vực đó không có bài nào lọt top. */
  heroArtwork?: TopArtwork;
};

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

  const activity = useMemo(() => stats?.activity ?? [], [stats]);

  // So sánh 7 ngày gần nhất với 7 ngày liền trước để ra biến động % cho các
  // ô chỉ số đầu trang. Toàn bộ chuỗi 14 ngày do backend trả (đã điền đủ
  // ngày trống) nên chỉ cần cắt đôi, không phải lo ngày thiếu làm lệch mốc.
  const deltas = useMemo(() => {
    if (activity.length < 14) return null;
    const half = Math.floor(activity.length / 2);
    const sum = (from: number, to: number, key: "uploads" | "views" | "reactions" | "comments") =>
      activity.slice(from, to).reduce((acc, p) => acc + p[key], 0);

    function pct(key: "uploads" | "views" | "reactions" | "comments"): number | undefined {
      const prev = sum(0, half, key);
      const curr = sum(half, activity.length, key);
      if (prev === 0) return undefined;
      return ((curr - prev) / prev) * 100;
    }

    return {
      uploads: pct("uploads"),
      views: pct("views"),
      reactions: pct("reactions"),
      comments: pct("comments"),
    };
  }, [activity]);

  // Gộp school_coverage + top_artworks theo khu vực - cả hai danh sách này
  // backend đã trả sẵn trong cùng 1 lần gọi, nên đây thuần là suy diễn ở FE,
  // không cần endpoint mới. school_id là cầu nối duy nhất giữa "tác phẩm nổi
  // bật" (không mang theo region) và "trường" (có region).
  const regions = useMemo<RegionSummary[]>(() => {
    if (!stats) return [];
    const coverage = stats.school_coverage ?? [];
    const schoolRegion = new Map(coverage.map((s) => [s.school_id, s.region]));

    return (Object.keys(REGION_LABEL) as RegionKey[]).map((key) => {
      const schools = coverage.filter((s) => s.region === key).sort((a, b) => b.artworks - a.artworks);
      const heroArtwork = stats.top_artworks.find((a) => schoolRegion.get(a.school_id) === key);
      return {
        key,
        name: REGION_LABEL[key],
        color: REGION_COLOR[key],
        artworks: stats.total_by_region[key] ?? 0,
        schools,
        heroArtwork,
      };
    });
  }, [stats]);

  if (loading && !stats) {
    return (
      <>
        <AdminPageHeader title="Tổng quan" subtitle="Đang gom số liệu…" />
        <div className="dashboard-skeleton">
          <div className="dashboard-skeleton-row">
            {[0, 1, 2, 3].map((i) => (
              <div key={i} className="dashboard-skeleton-card" />
            ))}
          </div>
          <div className="dashboard-skeleton-chart" />
        </div>
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

  const ops = stats.operations;

  return (
    <div className="dashboard-page">
      <AdminPageHeader
        title="Tổng quan"
        subtitle={`Triển lãm đang có ${formatNumber(stats.total_artworks)} tác phẩm`}
      />

      {/* ---- Hàng chỉ số dẫn dắt ---- */}
      <section className="dashboard-stat-grid" aria-label="Chỉ số chính">
        <StatCard
          label="Tác phẩm đã xuất bản"
          value={stats.total_artworks}
          icon={Images}
          tone="tri-thuc"
          hint={ops.pending_artworks > 0 ? `${formatNumber(ops.pending_artworks)} bài còn chờ duyệt` : "Không còn bài chờ duyệt"}
          spark={activity.map((p) => p.uploads)}
          delta={deltas?.uploads}
        />
        <StatCard
          label="Lượt xem"
          value={ops.total_views}
          icon={Eye}
          tone="khai-phong"
          hint={
            stats.total_artworks > 0
              ? `Trung bình ${formatNumber(Math.round(ops.total_views / stats.total_artworks))} lượt/tác phẩm`
              : undefined
          }
          spark={activity.map((p) => p.views)}
          delta={deltas?.views}
        />
        <StatCard
          label="Lượt cảm xúc"
          value={ops.total_reactions}
          icon={Heart}
          tone="nhan-ai"
          hint={
            ops.total_views > 0
              ? `${((ops.total_reactions / ops.total_views) * 100).toFixed(1)}% lượt xem để lại cảm xúc`
              : undefined
          }
          spark={activity.map((p) => p.reactions)}
          delta={deltas?.reactions}
        />
        <StatCard
          label="Bình luận"
          value={ops.total_comments}
          icon={MessageSquare}
          tone="trach-nhiem"
          hint={
            ops.hidden_comments > 0
              ? `${formatNumber(ops.hidden_comments)} bình luận đã bị ẩn`
              : "Chưa phải ẩn bình luận nào"
          }
          spark={activity.map((p) => p.comments)}
          delta={deltas?.comments}
        />
      </section>

      {/* ---- Ba khu vực trưng bày ---- */}
      <section className="dashboard-region-grid" aria-label="Thống kê theo khu vực">
        {regions.map((region) => (
          <RegionCard key={region.key} region={region} />
        ))}
      </section>
    </div>
  );
}

function RegionCard({ region }: { region: RegionSummary }) {
  const totalAwarded = region.schools.reduce((acc, s) => acc + s.awarded, 0);
  const totalGradesCovered = region.schools.reduce((acc, s) => acc + s.grades_covered, 0);
  const totalGrades = region.schools.reduce((acc, s) => acc + s.total_grades, 0);
  const coverageRatio = totalGrades > 0 ? totalGradesCovered / totalGrades : 0;

  return (
    <article className="region-card" style={{ "--region-ink": region.color } as React.CSSProperties}>
      <div className="region-card-hero">
        {region.heroArtwork ? (
          <picture>
            {artworkPictureSources(region.heroArtwork, "medium").map((s) => (
              <source key={s.type} srcSet={s.srcSet} type={s.type} />
            ))}
            <img
              src={artworkImageURL(region.heroArtwork, "medium")}
              alt=""
              loading="lazy"
              className="region-card-hero-img"
            />
          </picture>
        ) : (
          <div className="region-card-hero-empty" aria-hidden>
            <ImageOff size={28} strokeWidth={1.5} />
          </div>
        )}
        <div className="region-card-hero-scrim" aria-hidden />
        <div className="region-card-hero-tag">
          <MapPin size={13} strokeWidth={2.2} />
          {region.name}
        </div>
        {region.heroArtwork && (
          <div className="region-card-hero-caption">
            <strong>{region.heroArtwork.title}</strong>
            <span>{region.heroArtwork.student_name}</span>
          </div>
        )}
      </div>

      <div className="region-card-body">
        <div className="region-card-metrics">
          <div className="region-card-metric">
            <span className="region-card-metric-value">{formatNumber(region.artworks)}</span>
            <span className="region-card-metric-label">Tác phẩm</span>
          </div>
          <div className="region-card-metric">
            <span className="region-card-metric-value">{formatNumber(totalAwarded)}</span>
            <span className="region-card-metric-label">Đã trao giải</span>
          </div>
          <div className="region-card-metric">
            <span className="region-card-metric-value">{Math.round(coverageRatio * 100)}%</span>
            <span className="region-card-metric-label">Phủ khối lớp</span>
          </div>
        </div>

        <ul className="region-card-schools">
          {region.schools.map((s) => (
            <li key={s.school_id} className="region-card-school">
              <span className="region-card-school-name">{s.name}</span>
              <span className="region-card-school-count">{formatNumber(s.artworks)} bài</span>
            </li>
          ))}
          {region.schools.length === 0 && (
            <li className="region-card-school region-card-school--empty">Chưa có cơ sở nào ở khu vực này</li>
          )}
        </ul>
      </div>
    </article>
  );
}
