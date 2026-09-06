import { adminRequest, adminUpload } from "./adminApi";

export type BulkUploadItem = {
  temp_key: string;
  file_name: string;
  s3_key?: string;
  s3_url?: string;
  file_size?: number;
  width?: number;
  height?: number;
  /**
   * Các cỡ ảnh backend đã sinh sẵn lúc upload. Vắng mặt nếu ảnh gốc nhỏ hơn
   * mọi cỡ đích hoặc khâu sinh biến thể lỗi - lúc đó tác phẩm vẫn tạo được,
   * chỉ là trang hiển thị bằng ảnh gốc.
   */
  variants?: Partial<Record<ArtworkVariantKey, string>>;
  error?: string;
};

export type BulkUploadResponse = {
  items: BulkUploadItem[];
  open_errors: string[];
};

/**
 * bulkUploadArtworks: gửi nhiều file trong 1 multipart request tới
 * /api/v1/admin/artworks/bulk-upload - backend tự giới hạn concurrency,
 * CHỈ đẩy lên S3, chưa ghi bảng artworks (xem CreateArtworkRequest bước 2).
 */
export function bulkUploadArtworks(files: File[], signal?: AbortSignal): Promise<BulkUploadResponse> {
  const form = new FormData();
  files.forEach((file) => form.append("files", file));
  return adminUpload<BulkUploadResponse>("/api/v1/admin/artworks/bulk-upload", form, signal);
}

export type CreateArtworkPayload = {
  title: string;
  student_name: string;
  school_id: number;
  grade_level_id: number;
  class_name?: string;
  s3_key: string;
  s3_url: string;
  file_size: number;
  width?: number;
  height?: number;
  /** Gửi lại nguyên vẹn từ kết quả bulk-upload để backend lưu vào DB. */
  variants?: Partial<Record<ArtworkVariantKey, string>>;
  award_id?: number | null;
};

export type Award = {
  id: number;
  name: string;
  slug: string;
  rank_order: number;
  color_hex: string;
  icon_key?: string;
  is_active: boolean;
};

/** Khoá của map variants: "<cỡ>_<định dạng>", khớp GeneratedVariant bên Go. */
export type ArtworkVariantKey =
  | "thumb_webp"
  | "thumb_jpg"
  | "medium_webp"
  | "medium_jpg"
  | "large_webp"
  | "large_jpg";

export type ArtworkWithMeta = {
  id: number;
  title: string;
  student_id: number;
  school_id: number;
  grade_level_id: number;
  image_url: string;
  thumbnail_url?: string;
  /**
   * Các cỡ ảnh dẫn xuất sinh lúc upload, khoá dạng "<cỡ>_<định dạng>"
   * ("thumb_webp", "medium_jpg"...). Có thể vắng mặt: tác phẩm upload trước
   * khi có tính năng này, hoặc ảnh gốc vốn đã nhỏ hơn mọi cỡ đích.
   * Dùng qua helper trong lib/artworkImage.ts thay vì đọc trực tiếp.
   */
  variants?: Partial<Record<ArtworkVariantKey, string>>;
  file_size: number;
  width?: number;
  height?: number;
  is_featured: boolean;
  is_published: boolean;
  view_count: number;
  created_at: string;
  updated_at: string;
  student_name: string;
  school_name: string;
  region: "saigon" | "cantho" | "vungtau";
  grade_label: string;
  education_level: "primary" | "secondary";
  class_name?: string;
  comment_count: number;
  reaction_counts: Record<string, number> | null;
  awards?: Award[];
};

export function createArtwork(payload: CreateArtworkPayload): Promise<ArtworkWithMeta> {
  return adminRequest<ArtworkWithMeta>("/api/v1/admin/artworks", { method: "POST", body: payload });
}

export type UpdateArtworkPayload = {
  title: string;
  student_id: number;
  school_id: number;
  grade_level_id: number;
  is_featured: boolean;
  is_published: boolean;
  award_id?: number | null;
};

export function updateArtwork(id: number, payload: UpdateArtworkPayload): Promise<ArtworkWithMeta> {
  return adminRequest<ArtworkWithMeta>(`/api/v1/admin/artworks/${id}`, { method: "PUT", body: payload });
}

export function deleteArtwork(id: number): Promise<{ deleted: boolean }> {
  return adminRequest<{ deleted: boolean }>(`/api/v1/admin/artworks/${id}`, { method: "DELETE" });
}

export function toggleFeatured(id: number, featured: boolean): Promise<{ is_featured: boolean }> {
  return adminRequest<{ is_featured: boolean }>(`/api/v1/admin/artworks/${id}/featured`, {
    method: "PATCH",
    body: { featured },
  });
}

export type ArtworkFilter = {
  search?: string;
  school_id?: number;
  grade_level_id?: number;
  education_level?: "primary" | "secondary";
  award_id?: number;
  featured?: boolean;
  page?: number;
  page_size?: number;
};

export type ArtworkListResult = {
  items: ArtworkWithMeta[];
  total_count: number;
  page: number;
  page_size: number;
};

export function fetchArtworks(filter: ArtworkFilter, signal?: AbortSignal): Promise<ArtworkListResult> {
  const params = new URLSearchParams();
  if (filter.search) params.set("search", filter.search);
  if (filter.school_id) params.set("school_id", String(filter.school_id));
  if (filter.grade_level_id) params.set("grade_level_id", String(filter.grade_level_id));
  if (filter.education_level) params.set("education_level", filter.education_level);
  if (filter.award_id) params.set("award_id", String(filter.award_id));
  if (filter.featured !== undefined) params.set("featured", String(filter.featured));
  params.set("page", String(filter.page ?? 1));
  params.set("page_size", String(filter.page_size ?? 20));

  return adminRequest<ArtworkListResult>(`/api/v1/admin/artworks?${params.toString()}`, { signal });
}

export type School = {
  id: number;
  name: string;
  region: "saigon" | "cantho" | "vungtau";
  display_order: number;
  is_active: boolean;
};

export type GradeLevel = {
  id: number;
  education_level: "primary" | "secondary";
  grade_number: number;
  label: string;
  display_order: number;
};

export function fetchSchools(signal?: AbortSignal): Promise<School[]> {
  return adminRequest<School[]>("/api/v1/schools", { signal });
}

export function fetchGradeLevels(signal?: AbortSignal): Promise<GradeLevel[]> {
  return adminRequest<GradeLevel[]>("/api/v1/grade-levels", { signal });
}
