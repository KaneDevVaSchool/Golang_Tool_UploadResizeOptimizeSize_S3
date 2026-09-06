import { adminRequest } from "./adminApi";

export type TopicCategory = {
  id: number;
  name: string;
  slug: string;
  /** Nhóm chủ đề áp dụng cho cấp học nào - null/undefined = dùng chung mọi cấp. */
  education_level?: "primary" | "secondary" | null;
  display_order: number;
  is_active: boolean;
};

export function fetchTopicCategories(includeInactive: boolean, signal?: AbortSignal): Promise<TopicCategory[]> {
  const path = includeInactive ? "/api/v1/admin/topic-categories" : "/api/v1/topic-categories";
  return adminRequest<TopicCategory[]>(path, { signal });
}

export type TopicCategoryPayload = {
  name: string;
  slug?: string;
  education_level?: "primary" | "secondary" | null;
  display_order: number;
  is_active: boolean;
};

export function createTopicCategory(payload: TopicCategoryPayload): Promise<TopicCategory> {
  return adminRequest<TopicCategory>("/api/v1/admin/topic-categories", { method: "POST", body: payload });
}

export function updateTopicCategory(id: number, payload: TopicCategoryPayload): Promise<TopicCategory> {
  return adminRequest<TopicCategory>(`/api/v1/admin/topic-categories/${id}`, { method: "PUT", body: payload });
}

export function deleteTopicCategory(id: number): Promise<{ deleted: boolean }> {
  return adminRequest<{ deleted: boolean }>(`/api/v1/admin/topic-categories/${id}`, { method: "DELETE" });
}
