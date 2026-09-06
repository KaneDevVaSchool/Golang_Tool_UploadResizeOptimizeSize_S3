import { adminRequest } from "./adminApi";
import type { Award } from "./artworkApi";

export type { Award };

export function fetchAwards(includeInactive: boolean, signal?: AbortSignal): Promise<Award[]> {
  const path = includeInactive ? "/api/v1/admin/awards" : "/api/v1/awards";
  return adminRequest<Award[]>(path, { signal });
}

export type AwardPayload = {
  name: string;
  slug?: string;
  /** Giải gắn riêng cho 1 khối lớp - null/không gửi = giải dùng chung toàn hệ thống. */
  grade_level_id?: number | null;
  rank_order: number;
  color_hex: string;
  icon_key?: string | null;
  is_active: boolean;
};

export function createAward(payload: AwardPayload): Promise<Award> {
  return adminRequest<Award>("/api/v1/admin/awards", { method: "POST", body: payload });
}

export function updateAward(id: number, payload: AwardPayload): Promise<Award> {
  return adminRequest<Award>(`/api/v1/admin/awards/${id}`, { method: "PUT", body: payload });
}

export function deleteAward(id: number): Promise<{ deleted: boolean }> {
  return adminRequest<{ deleted: boolean }>(`/api/v1/admin/awards/${id}`, { method: "DELETE" });
}
