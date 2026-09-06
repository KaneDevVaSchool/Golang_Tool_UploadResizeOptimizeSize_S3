import { adminRequest } from "./adminApi";
import type { ArtworkWithMeta } from "./artworkApi";

export type GradeLevelCount = {
  grade_level_id: number;
  label: string;
  education_level: "primary" | "secondary";
  count: number;
};

export type SchoolCount = {
  school_id: number;
  name: string;
  region: "saigon" | "cantho" | "vungtau";
  count: number;
};

export type TopArtwork = {
  id: number;
  title: string;
  image_url: string;
  thumbnail_url?: string;
  /** Xem ArtworkWithMeta.variants - backend trả cùng bộ cột artworks. */
  variants?: ArtworkWithMeta["variants"];
  /** Backend trả cùng bộ cột artworks (models.Artwork) - dùng để đối chiếu
   *  với school_coverage và suy ra khu vực, không phải trường mới. */
  school_id: number;
  student_name: string;
  view_count: number;
  reaction_count: number;
  comment_count: number;
  engagement_score: number;
};

/** Một ngày trên biểu đồ nhịp hoạt động (backend luôn trả đủ ngày liên tục). */
export type ActivityPoint = {
  /** YYYY-MM-DD */
  date: string;
  uploads: number;
  views: number;
  reactions: number;
  comments: number;
};

/** Mức tham gia của một trường - trường 0 bài vẫn có mặt trong danh sách. */
export type SchoolCoverage = {
  school_id: number;
  name: string;
  region: "saigon" | "cantho" | "vungtau";
  artworks: number;
  grades_covered: number;
  total_grades: number;
  awarded: number;
};

/** Các con số cần hành động, tách khỏi số liệu mô tả quy mô triển lãm. */
export type OperationsSnapshot = {
  pending_artworks: number;
  hidden_comments: number;
  total_comments: number;
  awarded_artworks: number;
  active_awards: number;
  featured_artworks: number;
  silent_artworks: number;
  total_views: number;
  total_reactions: number;
};

export type DashboardStats = {
  total_artworks: number;
  total_by_region: Record<"saigon" | "cantho" | "vungtau", number>;
  total_by_grade: GradeLevelCount[];
  top_schools: SchoolCount[];
  top_artworks: TopArtwork[];
  activity: ActivityPoint[];
  school_coverage: SchoolCoverage[];
  operations: OperationsSnapshot;
};

/** Giá trị 0 cho mọi chỉ số vận hành - dùng khi backend chưa trả khối này. */
const EMPTY_OPERATIONS: OperationsSnapshot = {
  pending_artworks: 0,
  hidden_comments: 0,
  total_comments: 0,
  awarded_artworks: 0,
  active_awards: 0,
  featured_artworks: 0,
  silent_artworks: 0,
  total_views: 0,
  total_reactions: 0,
};

/** Khoảng ngày cho biểu đồ nhịp hoạt động - bỏ trống cả hai thì backend tự
 *  áp mặc định 14 ngày gần nhất. */
export type ActivityRange = {
  from: string; // YYYY-MM-DD
  to: string; // YYYY-MM-DD
};

export async function fetchDashboardStats(
  signal?: AbortSignal,
  range?: ActivityRange,
): Promise<DashboardStats> {
  const query = range ? `?from=${range.from}&to=${range.to}` : "";
  const raw = await adminRequest<DashboardStats>(`/api/v1/admin/dashboard/stats${query}`, { signal });

  // Chuẩn hoá các khối mới trước khi trả ra ngoài. Một backend cũ hơn (chưa
  // deploy kịp, hoặc binary đang chạy còn là bản trước) sẽ không có
  // `operations`/`activity`/`school_coverage`; đọc thẳng vào đó làm cả
  // trang trắng vì lỗi runtime, trong khi phần lớn Dashboard vẫn hiển thị
  // được bình thường. Thà thiếu vài ô còn hơn mất cả trang.
  return {
    ...raw,
    total_by_grade: raw.total_by_grade ?? [],
    top_schools: raw.top_schools ?? [],
    top_artworks: raw.top_artworks ?? [],
    activity: raw.activity ?? [],
    school_coverage: raw.school_coverage ?? [],
    operations: raw.operations ?? EMPTY_OPERATIONS,
  };
}
