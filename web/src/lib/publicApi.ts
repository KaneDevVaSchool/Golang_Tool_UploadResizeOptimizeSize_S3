// Wrapper fetch cho /api/v1/public/* - không cần session/API key (đúng yêu
// cầu "ẩn danh tự do"), vẫn cần CSRF token cho POST/DELETE (double-submit
// cookie pattern áp dụng toàn cục, xem internal/middleware/csrf.go) và
// visitor_token cho dedupe/rate-limit ở backend.
import { getCsrfToken } from "./api";
import { getOrCreateVisitorToken } from "./visitorToken";
import type { ArtworkWithMeta } from "./artworkApi";

const API_BASE = (import.meta.env.VITE_API_BASE_URL as string | undefined)?.replace(/\/$/, "") ?? "";

function apiUrl(path: string): string {
  if (path.startsWith("http://") || path.startsWith("https://")) return path;
  return `${API_BASE}${path}`;
}

type ApiEnvelope<T> = { success: boolean; data?: T; error?: { code?: string; message?: string } };

async function parseEnvelope<T>(res: Response): Promise<T> {
  const text = await res.text();
  let body: ApiEnvelope<T> = { success: false };
  if (text) {
    try {
      body = JSON.parse(text) as ApiEnvelope<T>;
    } catch {
      throw new Error(text.slice(0, 200) || `Request failed (${res.status})`);
    }
  }
  if (res.status === 429) {
    throw new Error("Bạn thao tác hơi nhanh, vui lòng thử lại sau ít phút.");
  }
  if (!res.ok || !body.success) {
    throw new Error(body.error?.message || `Request failed (${res.status})`);
  }
  return body.data as T;
}

async function publicGet<T>(path: string, signal?: AbortSignal): Promise<T> {
  const res = await fetch(apiUrl(path), { signal });
  return parseEnvelope<T>(res);
}

async function publicWrite<T>(path: string, method: "POST" | "DELETE", body?: unknown): Promise<T> {
  const headers: Record<string, string> = {};
  const token = getCsrfToken();
  if (token) headers["X-CSRF-Token"] = token;
  if (body !== undefined) headers["Content-Type"] = "application/json";

  const res = await fetch(apiUrl(path), {
    method,
    credentials: "same-origin",
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });
  return parseEnvelope<T>(res);
}

export type PublicArtworkFilter = {
  region?: "saigon" | "cantho" | "vungtau";
  school_id?: number;
  grade_level_id?: number;
  education_level?: "primary" | "secondary";
  search?: string;
  page?: number;
  page_size?: number;
};

export type PublicArtworkListResult = { items: ArtworkWithMeta[]; total_count: number; page: number; page_size: number };

export function fetchPublicArtworks(filter: PublicArtworkFilter, signal?: AbortSignal): Promise<PublicArtworkListResult> {
  const params = new URLSearchParams();
  if (filter.school_id) params.set("school_id", String(filter.school_id));
  if (filter.grade_level_id) params.set("grade_level_id", String(filter.grade_level_id));
  if (filter.education_level) params.set("education_level", filter.education_level);
  if (filter.search) params.set("search", filter.search);
  params.set("page", String(filter.page ?? 1));
  params.set("page_size", String(filter.page_size ?? 24));
  return publicGet(`/api/v1/public/artworks?${params.toString()}`, signal);
}

export function fetchFeaturedArtworks(
  region?: "saigon" | "cantho" | "vungtau",
  signal?: AbortSignal,
): Promise<{ items: ArtworkWithMeta[]; total_count: number }> {
  const params = region ? `?region=${region}` : "";
  return publicGet(`/api/v1/public/artworks/featured${params}`, signal);
}

export function fetchPublicArtwork(id: number, signal?: AbortSignal): Promise<ArtworkWithMeta> {
  const token = getOrCreateVisitorToken();
  return publicGet(`/api/v1/public/artworks/${id}?visitor_token=${encodeURIComponent(token)}`, signal);
}

export type ReactionCounts = Record<string, number>;

export function addReaction(artworkId: number, reactionType: string): Promise<{ reaction_counts: ReactionCounts }> {
  const visitor_token = getOrCreateVisitorToken();
  return publicWrite(`/api/v1/public/artworks/${artworkId}/reactions`, "POST", { reaction_type: reactionType, visitor_token });
}

export function removeReaction(artworkId: number, reactionType: string): Promise<{ reaction_counts: ReactionCounts }> {
  const visitor_token = getOrCreateVisitorToken();
  return publicWrite(
    `/api/v1/public/artworks/${artworkId}/reactions/${reactionType}?visitor_token=${encodeURIComponent(visitor_token)}`,
    "DELETE",
  );
}

export type PublicComment = {
  id: number;
  artwork_id: number;
  display_name: string;
  content: string;
  created_at: string;
  can_delete: boolean;
};

export function fetchComments(artworkId: number, signal?: AbortSignal): Promise<PublicComment[]> {
  const visitorToken = getOrCreateVisitorToken();
  return publicGet(
    `/api/v1/public/artworks/${artworkId}/comments?visitor_token=${encodeURIComponent(visitorToken)}`,
    signal,
  );
}

export function createComment(artworkId: number, displayName: string, content: string): Promise<PublicComment> {
  const visitor_token = getOrCreateVisitorToken();
  return publicWrite(`/api/v1/public/artworks/${artworkId}/comments`, "POST", {
    display_name: displayName,
    content,
    visitor_token,
  });
}

export function deleteComment(artworkId: number, commentId: number): Promise<{ deleted: boolean }> {
  const visitorToken = getOrCreateVisitorToken();
  return publicWrite(
    `/api/v1/public/artworks/${artworkId}/comments/${commentId}?visitor_token=${encodeURIComponent(visitorToken)}`,
    "DELETE",
  );
}

export type BillboardEntry = ArtworkWithMeta & {
  award: { id: number; name: string; slug: string; rank_order: number; color_hex: string; icon_key?: string };
};

export function fetchBillboard(signal?: AbortSignal): Promise<BillboardEntry[]> {
  return publicGet("/api/v1/public/billboard", signal);
}
