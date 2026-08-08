export type UploadMode = "s3" | "wp";

export type HealthResponse = {
  success: boolean;
  data?: {
    status?: string;
    max_size?: number;
    max_size_formatted?: string;
    absolute_max_size?: number;
    absolute_max_size_formatted?: string;
    chunk_upload?: boolean;
    checks?: Record<string, unknown>;
  };
  error?: { code?: string; message?: string };
};

export type S3UploadData = {
  url: string;
  key: string;
  size: number;
  name: string;
};

export type WPSizeInfo = {
  name: string;
  file: string;
  width: number;
  height: number;
  url: string;
};

export type WPUploadData = {
  file: {
    name: string;
    type: string;
    url: string;
    size: number;
  };
  sizes: WPSizeInfo[];
};

export type UploadResult = S3UploadData | WPUploadData;

export type ApiErrorBody = {
  success: false;
  error?: { code?: string; message?: string };
};

const API_BASE = (import.meta.env.VITE_API_BASE_URL as string | undefined)?.replace(/\/$/, "") ?? "";
const API_KEY = (import.meta.env.VITE_API_KEY as string | undefined)?.trim() ?? "";

function apiUrl(path: string): string {
  if (path.startsWith("http://") || path.startsWith("https://")) return path;
  return `${API_BASE}${path}`;
}

function getCookie(name: string): string {
  const match = document.cookie.match(
    new RegExp(`(?:^|; )${name.replace(/[$()*+.?[\\\]^{|}]/g, "\\$&")}=([^;]*)`),
  );
  return match ? decodeURIComponent(match[1]) : "";
}

export function getCsrfToken(): string {
  return getCookie("csrf_token");
}

function authHeaders(json = false): HeadersInit {
  const headers: Record<string, string> = {};
  const token = getCsrfToken();
  if (token) headers["X-CSRF-Token"] = token;
  if (API_KEY) headers["X-API-Key"] = API_KEY;
  if (json) headers["Content-Type"] = "application/json";
  return headers;
}

function applyXhrAuth(xhr: XMLHttpRequest) {
  const token = getCsrfToken();
  if (token) xhr.setRequestHeader("X-CSRF-Token", token);
  if (API_KEY) xhr.setRequestHeader("X-API-Key", API_KEY);
}

function messageFromBody(body: unknown, status: number, fallbackText = ""): string {
  if (body && typeof body === "object") {
    const err = body as ApiErrorBody & { error?: { message?: string }; message?: string };
    if (err.error?.message) return err.error.message;
    if (typeof err.message === "string" && err.message) return err.message;
  }
  if (fallbackText.trim()) return fallbackText.slice(0, 200);
  if (status === 403) return "Bị từ chối (CSRF hoặc API key). Tải lại trang rồi thử lại.";
  if (status === 401) return "Thiếu API key. Cấu hình VITE_API_KEY nếu máy chủ yêu cầu.";
  return `Request failed (${status})`;
}

async function parseApiJson<T>(res: Response): Promise<T> {
  const text = await res.text();
  let body: unknown = {};
  if (text) {
    try {
      body = JSON.parse(text);
    } catch {
      throw new Error(text.slice(0, 200) || `Request failed (${res.status})`);
    }
  }
  if (!res.ok) {
    throw new Error(messageFromBody(body, res.status, text));
  }
  const ok = body as { success?: boolean; data?: T; error?: { message?: string } };
  if (!ok.success || ok.data === undefined) {
    throw new Error(ok.error?.message || "Unexpected API response");
  }
  return ok.data;
}

export async function fetchHealth(signal?: AbortSignal): Promise<HealthResponse> {
  const res = await fetch(apiUrl("/api/v1/health"), {
    credentials: "same-origin",
    headers: API_KEY ? { "X-API-Key": API_KEY } : undefined,
    signal,
  });
  const text = await res.text();
  let body: HealthResponse = { success: false };
  if (text) {
    try {
      body = JSON.parse(text) as HealthResponse;
    } catch {
      throw new Error(`Health check failed (${res.status})`);
    }
  }
  if (!res.ok && res.status !== 503) {
    throw new Error(body.error?.message || `Health check failed (${res.status})`);
  }
  return body;
}

export type UploadProgressEvent = {
  loaded: number;
  total: number;
  percent: number;
};

export class UploadAbortedError extends Error {
  constructor(message = "Đã hủy gửi ảnh") {
    super(message);
    this.name = "UploadAbortedError";
  }
}

type ActiveXhr = { xhr: XMLHttpRequest; abort: () => void };

let activeUpload: ActiveXhr | null = null;
let activeChunkSession: string | null = null;

export function abortActiveUpload() {
  const session = activeChunkSession;
  activeChunkSession = null;
  if (activeUpload) {
    activeUpload.abort();
    activeUpload = null;
  }
  if (session) {
    void abortChunkUpload(session);
  }
}

function xhrUpload(
  endpoint: string,
  form: FormData,
  onProgress?: (event: UploadProgressEvent) => void,
  signal?: AbortSignal,
): Promise<UploadResult> {
  return new Promise((resolve, reject) => {
    if (signal?.aborted) {
      reject(new UploadAbortedError());
      return;
    }

    const xhr = new XMLHttpRequest();
    xhr.open("POST", apiUrl(endpoint));
    xhr.withCredentials = true;
    applyXhrAuth(xhr);

    const cleanup = () => {
      if (activeUpload?.xhr === xhr) activeUpload = null;
      signal?.removeEventListener("abort", onAbort);
    };

    const onAbort = () => {
      xhr.abort();
      cleanup();
      reject(new UploadAbortedError());
    };

    activeUpload = {
      xhr,
      abort: () => {
        xhr.abort();
      },
    };
    signal?.addEventListener("abort", onAbort);

    xhr.upload.onprogress = (e) => {
      if (!e.lengthComputable || !onProgress) return;
      onProgress({
        loaded: e.loaded,
        total: e.total,
        percent: Math.round((e.loaded / e.total) * 100),
      });
    };

    xhr.onerror = () => {
      cleanup();
      reject(new Error("Network error during upload"));
    };

    xhr.onabort = () => {
      cleanup();
      reject(new UploadAbortedError());
    };

    xhr.onload = () => {
      cleanup();
      let body: unknown;
      const raw = xhr.responseText || "";
      try {
        body = JSON.parse(raw || "{}");
      } catch {
        reject(new Error(messageFromBody(null, xhr.status, raw)));
        return;
      }

      if (xhr.status >= 200 && xhr.status < 300) {
        const ok = body as { success?: boolean; data?: UploadResult };
        if (ok.success && ok.data) {
          resolve(ok.data);
          return;
        }
      }

      reject(new Error(messageFromBody(body, xhr.status, raw)));
    };

    xhr.send(form);
  });
}

type ChunkInitData = {
  upload_id: string;
  chunk_size: number;
  total_chunks: number;
  total_size: number;
  filename: string;
};

async function initChunkUpload(file: File, signal?: AbortSignal): Promise<ChunkInitData> {
  const res = await fetch(apiUrl("/api/v1/upload/init"), {
    method: "POST",
    credentials: "same-origin",
    headers: authHeaders(true),
    body: JSON.stringify({ filename: file.name, total_size: file.size }),
    signal,
  });
  return parseApiJson<ChunkInitData>(res);
}

async function uploadChunkPart(
  uploadId: string,
  index: number,
  blob: Blob,
  onProgress?: (loaded: number, total: number) => void,
  signal?: AbortSignal,
): Promise<void> {
  await new Promise<void>((resolve, reject) => {
    if (signal?.aborted) {
      reject(new UploadAbortedError());
      return;
    }

    const xhr = new XMLHttpRequest();
    xhr.open("POST", apiUrl("/api/v1/upload/chunk"));
    xhr.withCredentials = true;
    applyXhrAuth(xhr);

    const cleanup = () => {
      if (activeUpload?.xhr === xhr) activeUpload = null;
      signal?.removeEventListener("abort", onAbort);
    };

    const onAbort = () => {
      xhr.abort();
      cleanup();
      reject(new UploadAbortedError());
    };

    activeUpload = { xhr, abort: () => xhr.abort() };
    signal?.addEventListener("abort", onAbort);

    xhr.upload.onprogress = (e) => {
      if (!e.lengthComputable || !onProgress) return;
      onProgress(e.loaded, e.total);
    };
    xhr.onerror = () => {
      cleanup();
      reject(new Error(`Network error on chunk ${index}`));
    };
    xhr.onabort = () => {
      cleanup();
      reject(new UploadAbortedError());
    };
    xhr.onload = () => {
      cleanup();
      const raw = xhr.responseText || "";
      try {
        const body = JSON.parse(raw || "{}") as ApiErrorBody & { success?: boolean };
        if (xhr.status >= 200 && xhr.status < 300 && body.success) {
          resolve();
          return;
        }
        reject(new Error(messageFromBody(body, xhr.status, raw)));
      } catch {
        reject(new Error(messageFromBody(null, xhr.status, raw)));
      }
    };

    const form = new FormData();
    form.append("upload_id", uploadId);
    form.append("index", String(index));
    form.append("chunk", blob, `part-${index}`);
    xhr.send(form);
  });
}

async function completeChunkUpload(uploadId: string, signal?: AbortSignal): Promise<S3UploadData> {
  const res = await fetch(apiUrl("/api/v1/upload/complete"), {
    method: "POST",
    credentials: "same-origin",
    headers: authHeaders(true),
    body: JSON.stringify({ upload_id: uploadId }),
    signal,
  });
  return parseApiJson<S3UploadData>(res);
}

async function abortChunkUpload(uploadId: string): Promise<void> {
  try {
    await fetch(apiUrl("/api/v1/upload/abort"), {
      method: "POST",
      credentials: "same-origin",
      headers: authHeaders(true),
      body: JSON.stringify({ upload_id: uploadId }),
    });
  } catch {
    // best-effort cleanup
  }
}

async function sleep(ms: number, signal?: AbortSignal) {
  if (signal?.aborted) throw new UploadAbortedError();
  await new Promise<void>((resolve, reject) => {
    const t = window.setTimeout(() => {
      signal?.removeEventListener("abort", onAbort);
      resolve();
    }, ms);
    const onAbort = () => {
      window.clearTimeout(t);
      reject(new UploadAbortedError());
    };
    signal?.addEventListener("abort", onAbort, { once: true });
  });
}

async function uploadChunkPartWithRetry(
  uploadId: string,
  index: number,
  blob: Blob,
  onProgress?: (loaded: number, total: number) => void,
  signal?: AbortSignal,
): Promise<void> {
  let lastErr: unknown;
  for (let attempt = 0; attempt < 3; attempt++) {
    try {
      await uploadChunkPart(uploadId, index, blob, onProgress, signal);
      return;
    } catch (err) {
      if (err instanceof UploadAbortedError) throw err;
      lastErr = err;
      if (attempt < 2) await sleep(400 * (attempt + 1), signal);
    }
  }
  throw lastErr instanceof Error ? lastErr : new Error(`Chunk ${index} failed`);
}

async function uploadChunked(
  file: File,
  onProgress?: (event: UploadProgressEvent) => void,
  signal?: AbortSignal,
): Promise<S3UploadData> {
  const session = await initChunkUpload(file, signal);
  activeChunkSession = session.upload_id;

  if (session.total_size !== file.size) {
    await abortChunkUpload(session.upload_id);
    activeChunkSession = null;
    throw new Error("Server rejected file size for chunked upload");
  }

  let uploadedBytes = 0;

  try {
    for (let i = 0; i < session.total_chunks; i++) {
      if (signal?.aborted) throw new UploadAbortedError();
      const start = i * session.chunk_size;
      const end = Math.min(start + session.chunk_size, file.size);
      const blob = file.slice(start, end);

      await uploadChunkPartWithRetry(session.upload_id, i, blob, (loaded) => {
        const overall = uploadedBytes + loaded;
        onProgress?.({
          loaded: overall,
          total: file.size,
          percent: Math.min(99, Math.round((overall / file.size) * 100)),
        });
      }, signal);
      uploadedBytes += blob.size;
      onProgress?.({
        loaded: uploadedBytes,
        total: file.size,
        percent: Math.min(99, Math.round((uploadedBytes / file.size) * 100)),
      });
    }

    const result = await completeChunkUpload(session.upload_id, signal);
    activeChunkSession = null;
    onProgress?.({ loaded: file.size, total: file.size, percent: 100 });
    return result;
  } catch (err) {
    const id = session.upload_id;
    activeChunkSession = null;
    await abortChunkUpload(id);
    throw err;
  }
}

export type UploadLimits = {
  maxSize: number;
  absoluteMaxSize: number;
  chunkUpload?: boolean;
};

export function validateClientFile(file: File, mode: UploadMode, limits: UploadLimits): string | null {
  if (mode === "wp") {
    if (!file.type.startsWith("image/")) {
      return `“${file.name}” không phải ảnh. Chế độ thu nhỏ chỉ nhận ảnh.`;
    }
    if (file.size > limits.maxSize) {
      return `“${file.name}” vượt ${formatBytes(limits.maxSize)}. Dùng “Lưu ảnh” cho file lớn hơn.`;
    }
    return null;
  }
  if (file.size > limits.absoluteMaxSize) {
    return `“${file.name}” vượt giới hạn ${formatBytes(limits.absoluteMaxSize)}.`;
  }
  if (file.size > limits.maxSize && limits.chunkUpload === false) {
    return `“${file.name}” cần gửi theo phần nhưng máy chủ chưa bật chunk upload.`;
  }
  return null;
}

export async function uploadFile(
  file: File,
  mode: UploadMode,
  limits: UploadLimits,
  onProgress?: (event: UploadProgressEvent) => void,
  signal?: AbortSignal,
): Promise<UploadResult> {
  const clientErr = validateClientFile(file, mode, limits);
  if (clientErr) throw new Error(clientErr);

  if (mode === "wp") {
    const form = new FormData();
    form.append("file", file);
    return xhrUpload("/api/v1/wp-upload", form, onProgress, signal);
  }

  if (file.size > limits.maxSize) {
    return uploadChunked(file, onProgress, signal);
  }

  const form = new FormData();
  form.append("file", file);
  return xhrUpload("/api/v1/upload", form, onProgress, signal);
}

export function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(2)} MB`;
}

export function isWPResult(data: UploadResult): data is WPUploadData {
  return "file" in data && "sizes" in data;
}

export function resultUrl(data: UploadResult): string {
  return isWPResult(data) ? data.file.url : data.url;
}

export function resultName(data: UploadResult): string {
  return isWPResult(data) ? data.file.name : data.name;
}

export function resultSize(data: UploadResult): number {
  return isWPResult(data) ? data.file.size : data.size;
}
