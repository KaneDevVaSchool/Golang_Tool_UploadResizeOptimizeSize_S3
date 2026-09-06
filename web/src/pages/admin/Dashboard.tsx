import { Eye, Heart, ImageOff, Images, MapPin, Star } from "lucide-react";
import { useMemo, useEffect, useState } from "react";
import { AdminPageHeader } from "../../components/admin/AdminPageHeader";
import { StatCard } from "../../components/admin/StatCard";
import { artworkImageURL, artworkPictureSources } from "../../lib/artworkImage";
import { REGION_COLOR, REGION_LABEL, formatNumber } from "../../lib/chartTheme";
import {
  fetchDashboardStats,
  type ActivityPoint,
  type ActivityRange,
  type DashboardStats,
  type GradeLevelCount,
  type SchoolCoverage,
  type TopArtwork,
} from "../../lib/dashboardApi";
import { toast } from "../../lib/toastBus";

// Định dạng "lượt xem trung bình / tác phẩm". Làm tròn số nguyên bình
// thường sẽ ra "0" khi tỉ lệ chưa tới 0.5 (vd 28 lượt / 60 tác phẩm) - dễ
// đọc nhầm thành lỗi ngay giai đoạn đầu hội thi, khi bài đăng nhiều nhưng
// lượt xem tích luỹ còn ít. Dưới 1 thì giữ 1 chữ số thập phân để thấy đúng
// là "gần 0" chứ không phải "bằng 0".
function formatAverage(value: number): string {
  if (value >= 1 || value === 0) return formatNumber(Math.round(value));
  return value.toFixed(1);
}

// Bộ lọc thời gian cho biểu đồ nhịp hoạt động: "mặc định" (14 ngày gần nhất,
// không gửi from/to lên backend - khớp hành vi cũ), "tháng" (1 dropdown chọn
// tháng/năm) hoặc "khoảng ngày" (2 input date tuỳ ý). Tách khỏi ActivityRange
// (kiểu chỉ có from/to gửi lên API) vì "mặc định" không mang ngày cụ thể nào
// ở phía FE - để backend tự quyết như trước khi có bộ lọc này.
type ActivityRangeValue =
  | { mode: "default" }
  | { mode: "month"; month: string } // YYYY-MM
  | { mode: "custom"; from: string; to: string }; // YYYY-MM-DD

function toDateInputValue(d: Date): string {
  return d.toISOString().slice(0, 10);
}

function defaultMonthRange(): ActivityRangeValue {
  return { mode: "default" };
}

/** Ngày cuối cùng của tháng YYYY-MM (28-31 tuỳ tháng/năm nhuận). */
function lastDayOfMonth(month: string): string {
  const [y, m] = month.split("-").map(Number);
  // Ngày 0 của tháng sau = ngày cuối tháng này, theo đúng lịch dương lịch
  // kể cả tháng 2 năm nhuận - không cần bảng tra cứu số ngày/tháng thủ công.
  return toDateInputValue(new Date(y, m, 0));
}

function rangeToQuery(range: ActivityRangeValue): ActivityRange | undefined {
  if (range.mode === "default") return undefined;
  if (range.mode === "month") return { from: `${range.month}-01`, to: lastDayOfMonth(range.month) };
  return { from: range.from, to: range.to };
}

/** Nhãn hiển thị ở tiêu đề biểu đồ, thay cho "Nhịp 14 ngày gần nhất" cố định. */
function rangeLabel(range: ActivityRangeValue): string {
  if (range.mode === "default") return "Nhịp 14 ngày gần nhất";
  if (range.mode === "month") {
    const [y, m] = range.month.split("-");
    return `Nhịp tháng ${Number(m)}/${y}`;
  }
  return `Nhịp ${formatChartDay(range.from)} – ${formatChartDay(range.to)}`;
}

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
  const [range, setRange] = useState<ActivityRangeValue>(() => defaultMonthRange());

  useEffect(() => {
    const controller = new AbortController();
    setLoading(true);
    fetchDashboardStats(controller.signal, rangeToQuery(range))
      .then(setStats)
      .catch((err: unknown) => {
        if (controller.signal.aborted) return;
        toast.error(err instanceof Error ? err.message : "Không tải được số liệu thống kê");
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false);
      });
    return () => controller.abort();
  }, [range]);

  const activity = useMemo(() => stats?.activity ?? [], [stats]);

  // So sánh nửa đầu với nửa sau của khoảng đang xem để ra biến động % cho
  // các ô chỉ số đầu trang - tổng quát cho mọi độ dài khoảng (14 ngày mặc
  // định, cả tháng, hay khoảng ngày tuỳ chọn), không neo cứng theo 1 con số
  // ngày cụ thể như trước. Khoảng quá ngắn (< 4 điểm) thì so sánh nửa/nửa dễ
  // nhiễu (vd 3 ngày) nên bỏ qua, không hiển thị biến động.
  const deltas = useMemo(() => {
    if (activity.length < 4) return null;
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
          Không tải được số liệu thống kê. Thử tải lại trang; nếu vẫn lỗi, kiểm tra kết nối máy chủ.
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
              ? `Trung bình ${formatAverage(ops.total_views / stats.total_artworks)} lượt/tác phẩm`
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
          label="Tác phẩm nổi bật"
          value={ops.featured_artworks}
          icon={Star}
          tone="trach-nhiem"
          hint={
            stats.total_artworks > 0
              ? `${((ops.featured_artworks / stats.total_artworks) * 100).toFixed(1)}% tổng số tác phẩm`
              : undefined
          }
        />
      </section>

      <section className="dashboard-chart-grid" aria-label="Biểu đồ">
        <ActivityChart points={activity} range={range} onRangeChange={setRange} loading={loading} />
        <GradeChart grades={stats.total_by_grade} />
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

const ACTIVITY_SERIES = [
  { key: "uploads" as const, label: "Tải lên", color: "#b3003f" },
  { key: "views" as const, label: "Lượt xem", color: "#1750b5" },
  { key: "reactions" as const, label: "Cảm xúc", color: "#009082" },
  { key: "comments" as const, label: "Bình luận", color: "#a67f42" },
];

function formatChartDay(iso: string): string {
  const parts = iso.split("-");
  if (parts.length < 3) return iso;
  return `${Number(parts[2])}/${Number(parts[1])}`;
}

/** Ước lượng số nhãn trục hoành vừa mắt cho một biểu đồ rộng 640 điểm ảo -
 *  đích khoảng 60px/nhãn. Khoảng dài (cả tháng, hay khoảng ngày tuỳ ý nhiều
 *  tháng) sẽ tự giãn bước nhảy thay vì luôn nhảy 2 như trước (chỉ đúng cho
 *  14 ngày, 30+ điểm sẽ chồng chữ lên nhau). */
function labelStepFor(pointCount: number): number {
  const targetLabels = Math.max(Math.floor(640 / 60), 2);
  return Math.max(1, Math.ceil(pointCount / targetLabels));
}

type ActivityChartProps = {
  points: ActivityPoint[];
  range: ActivityRangeValue;
  onRangeChange: (range: ActivityRangeValue) => void;
  loading: boolean;
};

function ActivityChart({ points, range, onRangeChange, loading }: ActivityChartProps) {
  const width = 640;
  const height = 220;
  const pad = { l: 36, r: 12, t: 16, b: 28 };
  const innerW = width - pad.l - pad.r;
  const innerH = height - pad.t - pad.b;
  const max = Math.max(1, ...points.flatMap((p) => [p.uploads, p.views, p.reactions, p.comments]));
  const last = Math.max(points.length - 1, 1);
  const xAt = (i: number) => pad.l + (i / last) * innerW;
  const yAt = (value: number) => pad.t + innerH - (value / max) * innerH;
  const ticks = [0, Math.round(max / 2), max];
  const labelStep = labelStepFor(points.length);

  function seriesPath(key: (typeof ACTIVITY_SERIES)[number]["key"]) {
    return points
      .map((p, i) => `${i === 0 ? "M" : "L"} ${xAt(i).toFixed(1)} ${yAt(p[key]).toFixed(1)}`)
      .join(" ");
  }

  return (
    <article className="dashboard-chart-card">
      <header className="dashboard-chart-header">
        <h2>{rangeLabel(range)}</h2>
        <ul className="dashboard-chart-legend">
          {ACTIVITY_SERIES.map((s) => (
            <li key={s.key}>
              <span style={{ background: s.color }} />
              {s.label}
            </li>
          ))}
        </ul>
      </header>

      <ActivityRangePicker value={range} onChange={onRangeChange} disabled={loading} />

      {points.length < 2 ? (
        <p className="admin-empty-note">Chưa đủ dữ liệu để vẽ biểu đồ</p>
      ) : (
        <svg
          className="dashboard-chart-svg"
          viewBox={`0 0 ${width} ${height}`}
          role="img"
          aria-label={`${rangeLabel(range)}: tải lên, lượt xem, cảm xúc, bình luận`}
        >
          {ticks.map((tick) => (
            <g key={tick}>
              <line
                x1={pad.l}
                x2={width - pad.r}
                y1={yAt(tick)}
                y2={yAt(tick)}
                className="dashboard-chart-grid"
              />
              <text x={pad.l - 6} y={yAt(tick) + 4} className="dashboard-chart-tick">
                {formatNumber(tick)}
              </text>
            </g>
          ))}
          {ACTIVITY_SERIES.map((s) => (
            <path
              key={s.key}
              d={seriesPath(s.key)}
              fill="none"
              stroke={s.color}
              strokeWidth={2}
              strokeLinejoin="round"
              strokeLinecap="round"
            />
          ))}
          {points.map((p, i) =>
            i % labelStep === 0 || i === points.length - 1 ? (
              <text key={p.date} x={xAt(i)} y={height - 8} className="dashboard-chart-xlabel">
                {formatChartDay(p.date)}
              </text>
            ) : null,
          )}
        </svg>
      )}
    </article>
  );
}

/** Tháng hiện tại dạng YYYY-MM - giá trị mặc định khi người dùng bấm sang
 *  chế độ "Theo tháng" lần đầu, trước khi tự chọn tháng khác. */
function currentMonthValue(): string {
  return toDateInputValue(new Date()).slice(0, 7);
}

/** Hôm nay dạng YYYY-MM-DD - mốc "to" mặc định khi chuyển sang "Theo khoảng
 *  ngày" lần đầu. */
function todayValue(): string {
  return toDateInputValue(new Date());
}

type ActivityRangePickerProps = {
  value: ActivityRangeValue;
  onChange: (range: ActivityRangeValue) => void;
  disabled: boolean;
};

/**
 * Bộ chọn khoảng thời gian cho biểu đồ nhịp hoạt động - 3 chế độ dàn ngang
 * trên một dòng, co lại thành 2 dòng ở mobile (`.activity-range-picker` xử lý
 * responsive bằng flex-wrap, không cần breakpoint riêng).
 *
 * Segmented control 3 nút quyết định chế độ; input tương ứng chỉ hiện đúng 1
 * loại tại một thời điểm - tránh vừa có dropdown tháng vừa có 2 ô ngày cùng
 * lúc gây rối mắt cho một biểu đồ vốn đã nhỏ.
 */
function ActivityRangePicker({ value, onChange, disabled }: ActivityRangePickerProps) {
  return (
    <div className="activity-range-picker">
      <div className="activity-range-picker-modes" role="tablist" aria-label="Chọn kiểu khoảng thời gian">
        <button
          type="button"
          role="tab"
          aria-selected={value.mode === "default"}
          className={`activity-range-picker-mode ${value.mode === "default" ? "activity-range-picker-mode--active" : ""}`}
          onClick={() => onChange({ mode: "default" })}
          disabled={disabled}
        >
          14 ngày gần nhất
        </button>
        <button
          type="button"
          role="tab"
          aria-selected={value.mode === "month"}
          className={`activity-range-picker-mode ${value.mode === "month" ? "activity-range-picker-mode--active" : ""}`}
          onClick={() => onChange({ mode: "month", month: currentMonthValue() })}
          disabled={disabled}
        >
          Theo tháng
        </button>
        <button
          type="button"
          role="tab"
          aria-selected={value.mode === "custom"}
          className={`activity-range-picker-mode ${value.mode === "custom" ? "activity-range-picker-mode--active" : ""}`}
          onClick={() => onChange({ mode: "custom", from: currentMonthValue() + "-01", to: todayValue() })}
          disabled={disabled}
        >
          Khoảng ngày
        </button>
      </div>

      {value.mode === "month" && (
        <input
          type="month"
          className="activity-range-picker-input"
          value={value.month}
          max={currentMonthValue()}
          disabled={disabled}
          aria-label="Chọn tháng"
          onChange={(e) => e.target.value && onChange({ mode: "month", month: e.target.value })}
        />
      )}

      {value.mode === "custom" && (
        <div className="activity-range-picker-dates">
          <input
            type="date"
            className="activity-range-picker-input"
            value={value.from}
            max={value.to}
            disabled={disabled}
            aria-label="Từ ngày"
            onChange={(e) => e.target.value && onChange({ mode: "custom", from: e.target.value, to: value.to })}
          />
          <span aria-hidden>–</span>
          <input
            type="date"
            className="activity-range-picker-input"
            value={value.to}
            min={value.from}
            max={todayValue()}
            disabled={disabled}
            aria-label="Đến ngày"
            onChange={(e) => e.target.value && onChange({ mode: "custom", from: value.from, to: e.target.value })}
          />
        </div>
      )}
    </div>
  );
}

function GradeChart({ grades }: { grades: GradeLevelCount[] }) {
  const max = Math.max(1, ...grades.map((g) => g.count));

  return (
    <article className="dashboard-chart-card">
      <header className="dashboard-chart-header">
        <h2>Phân bổ theo khối lớp</h2>
      </header>
      {grades.length === 0 ? (
        <p className="admin-empty-note">Chưa có dữ liệu khối lớp</p>
      ) : (
        <ul className="dashboard-grade-bars">
          {grades.map((g) => (
            <li key={g.grade_level_id} className="dashboard-grade-bar">
              <span className="dashboard-grade-bar-label">{g.label}</span>
              <div className="dashboard-grade-bar-track">
                <div
                  className="dashboard-grade-bar-fill"
                  style={{ width: `${(g.count / max) * 100}%` }}
                />
              </div>
              <span className="dashboard-grade-bar-count">{formatNumber(g.count)}</span>
            </li>
          ))}
        </ul>
      )}
    </article>
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
