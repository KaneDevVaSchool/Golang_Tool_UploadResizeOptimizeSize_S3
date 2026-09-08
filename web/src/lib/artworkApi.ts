import { adminRequest, adminUpload, adminDownload } from "./adminApi";

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
  topic_category_id?: number | null;
  class_name?: string;
  s3_key: string;
  s3_url: string;
  file_size: number;
  width?: number;
  height?: number;
  /** Gửi lại nguyên vẹn từ kết quả bulk-upload để backend lưu vào DB. */
  variants?: Partial<Record<ArtworkVariantKey, string>>;
  /** Một tác phẩm có thể nhận nhiều giải cùng lúc (giải chính + Đặc biệt phụ). */
  award_ids?: number[];
};

export type Award = {
  id: number;
  name: string;
  slug: string;
  /** Giải gắn riêng cho 1 khối lớp (vd Tiểu học chia giải theo từng khối 1-5).
   * null/undefined = giải dùng chung toàn hệ thống, không tách khối. */
  grade_level_id?: number | null;
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
  topic_category_id?: number;
  topic_category_name?: string;
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
  topic_category_id?: number | null;
  is_featured: boolean;
  is_published: boolean;
  /**
   * undefined = giữ nguyên giải hiện tại (không gửi trường trong body);
   * [] = gỡ hết giải; danh sách = thay toàn bộ giải hiện tại bằng danh sách
   * này. Không còn giới hạn 1 giải/tác phẩm.
   */
  award_ids?: number[];
};

export function updateArtwork(id: number, payload: UpdateArtworkPayload): Promise<ArtworkWithMeta> {
  return adminRequest<ArtworkWithMeta>(`/api/v1/admin/artworks/${id}`, { method: "PUT", body: payload });
}

export function deleteArtwork(id: number): Promise<{ deleted: boolean }> {
  return adminRequest<{ deleted: boolean }>(`/api/v1/admin/artworks/${id}`, { method: "DELETE" });
}

/**
 * deleteArtworkBatch: xoá nhiều tác phẩm cùng lúc (thao tác bulk ở trang
 * danh sách) - 1 request thay vì lặp deleteArtwork cho từng id. Backend xoá
 * từng tác phẩm một ở tầng service (ảnh trên S3 không gộp xoá được), nên
 * response trả `deleted` có thể nhỏ hơn số id gửi lên nếu 1 vài tác phẩm lỗi
 * xoá S3 - tác phẩm đó vẫn còn nguyên, không mất dữ liệu.
 */
export function deleteArtworkBatch(ids: number[]): Promise<{ deleted: number; requested: number }> {
  return adminRequest<{ deleted: number; requested: number }>("/api/v1/admin/artworks/bulk-delete", {
    method: "DELETE",
    body: { ids },
  });
}

/**
 * downloadArtworkOriginal: tải ảnh gốc (không watermark) của 1 tác phẩm qua
 * proxy backend /admin/artworks/{id}/download - cùng cơ chế endpoint public
 * dùng cho khách tải ảnh, nhưng nhánh admin không watermark và không đòi
 * is_published (admin phải tải được cả ảnh đang ẩn). Trả về Blob + tên file
 * gợi ý từ header Content-Disposition, dùng cho cả tải đơn lẫn gộp zip.
 */
export function downloadArtworkOriginal(id: number, signal?: AbortSignal): Promise<{ blob: Blob; fileName: string }> {
  return adminDownload(`/api/v1/admin/artworks/${id}/download`, signal);
}

export function toggleFeatured(id: number, featured: boolean): Promise<{ is_featured: boolean }> {
  return adminRequest<{ is_featured: boolean }>(`/api/v1/admin/artworks/${id}/featured`, {
    method: "PATCH",
    body: { featured },
  });
}

/**
 * setFeaturedBatch: bật/tắt tiêu biểu cho nhiều tác phẩm cùng lúc (thao tác
 * bulk trên trang danh sách) - 1 request PATCH thay vì gọi lặp toggleFeatured
 * cho từng id, tránh trạng thái nửa vời nếu một request giữa chừng lỗi.
 */
export function setFeaturedBatch(
  ids: number[],
  featured: boolean,
): Promise<{ updated: number; is_featured: boolean }> {
  return adminRequest<{ updated: number; is_featured: boolean }>("/api/v1/admin/artworks/bulk-featured", {
    method: "PATCH",
    body: { ids, featured },
  });
}

export type ArtworkFilter = {
  search?: string;
  region?: "saigon" | "cantho" | "vungtau";
  school_id?: number;
  grade_level_id?: number;
  education_level?: "primary" | "secondary";
  topic_category_id?: number;
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
  if (filter.region) params.set("region", filter.region);
  if (filter.school_id) params.set("school_id", String(filter.school_id));
  if (filter.grade_level_id) params.set("grade_level_id", String(filter.grade_level_id));
  if (filter.education_level) params.set("education_level", filter.education_level);
  if (filter.topic_category_id) params.set("topic_category_id", String(filter.topic_category_id));
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

export type ArtworkComment = {
  id: number;
  artwork_id: number;
  display_name: string;
  content: string;
  is_hidden: boolean;
  created_at: string;
};

/**
 * fetchArtworkComments: lấy TOÀN BỘ bình luận (kể cả đã ẩn) của 1 tác phẩm
 * cho màn hình kiểm duyệt - khác endpoint public luôn lọc bỏ comment ẩn.
 */
export function fetchArtworkComments(artworkId: number, signal?: AbortSignal): Promise<ArtworkComment[]> {
  return adminRequest<ArtworkComment[]>(`/api/v1/admin/artworks/${artworkId}/comments`, { signal });
}

/**
 * setCommentHidden: ẩn/hiện 1 bình luận. Backend áp ngay cho API public (mọi
 * lượt gọi tiếp theo tới /public/artworks/{id}/comments đều lọc theo is_hidden).
 */
export function setCommentHidden(
  artworkId: number,
  commentId: number,
  isHidden: boolean,
): Promise<{ is_hidden: boolean }> {
  return adminRequest<{ is_hidden: boolean }>(`/api/v1/admin/artworks/${artworkId}/comments/${commentId}`, {
    method: "PATCH",
    body: { is_hidden: isHidden },
  });
}

export function fetchSchools(signal?: AbortSignal): Promise<School[]> {
  return adminRequest<School[]>("/api/v1/schools", { signal });
}

export function fetchGradeLevels(signal?: AbortSignal): Promise<GradeLevel[]> {
  return adminRequest<GradeLevel[]>("/api/v1/grade-levels", { signal });
}
