// Wrapper fetch riêng cho /api/v1/admin/* và /auth/* - dùng session cookie
// (credentials: 'include'), KHÔNG dùng API key như lib/api.ts (upload tool
// cũ). Giữ nguyên lib/api.ts, không sửa - 2 file độc lập hoàn toàn.
import { getCsrfToken } from "./api";

const API_BASE = (import.meta.env.VITE_API_BASE_URL as string | undefined)?.replace(/\/$/, "") ?? "";

function apiUrl(path: string): string {
  if (path.startsWith("http://") || path.startsWith("https://")) return path;
  return `${API_BASE}${path}`;
}

export class UnauthorizedError extends Error {
  constructor() {
    super("Phiên đăng nhập đã hết hạn");
    this.name = "UnauthorizedError";
  }
}

type ApiEnvelope<T> = {
  success: boolean;
  data?: T;
  error?: { code?: string; message?: string };
};

function authHeaders(hasBody: boolean): HeadersInit {
  const headers: Record<string, string> = {};
  const token = getCsrfToken();
  if (token) headers["X-CSRF-Token"] = token;
  if (hasBody) headers["Content-Type"] = "application/json";
  return headers;
}

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

  if (res.status === 401) {
    throw new UnauthorizedError();
  }
  if (!res.ok || !body.success) {
    throw new Error(body.error?.message || `Request failed (${res.status})`);
  }
  return body.data as T;
}

type RequestOptions = {
  method?: string;
  body?: unknown;
  signal?: AbortSignal;
};

/**
 * adminRequest: gọi 1 endpoint /api/v1/admin/* (hoặc bất kỳ path nào), tự
 * đính X-CSRF-Token + credentials cookie, parse response envelope
 * {success,data,error} chuẩn của backend. Ném UnauthorizedError riêng để
 * caller (useAdminAuth) tự điều hướng về /admin/login qua router thay vì
 * window.location cứng (giữ SPA navigation mượt).
 */
export async function adminRequest<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { method = "GET", body, signal } = options;
  const hasBody = body !== undefined;

  const res = await fetch(apiUrl(path), {
    method,
    credentials: "include",
    headers: authHeaders(hasBody),
    body: hasBody ? JSON.stringify(body) : undefined,
    signal,
  });

  return parseEnvelope<T>(res);
}

/**
 * adminUpload: multipart form upload tới /api/v1/admin/* - dùng cho
 * bulk-upload tác phẩm. Không set Content-Type thủ công (browser tự set
 * boundary đúng cho FormData).
 */
export async function adminUpload<T>(path: string, form: FormData, signal?: AbortSignal): Promise<T> {
  const token = getCsrfToken();
  const res = await fetch(apiUrl(path), {
    method: "POST",
    credentials: "include",
    headers: token ? { "X-CSRF-Token": token } : undefined,
    body: form,
    signal,
  });
  return parseEnvelope<T>(res);
}
