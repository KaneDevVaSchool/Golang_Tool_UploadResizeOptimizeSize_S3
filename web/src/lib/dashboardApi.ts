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
  student_name: string;
  view_count: number;
  reaction_count: number;
  comment_count: number;
  engagement_score: number;
};

export type DashboardStats = {
  total_artworks: number;
  total_by_region: Record<"saigon" | "cantho" | "vungtau", number>;
  total_by_grade: GradeLevelCount[];
  top_schools: SchoolCount[];
  top_artworks: TopArtwork[];
};

export function fetchDashboardStats(signal?: AbortSignal): Promise<DashboardStats> {
  return adminRequest<DashboardStats>("/api/v1/admin/dashboard/stats", { signal });
}
