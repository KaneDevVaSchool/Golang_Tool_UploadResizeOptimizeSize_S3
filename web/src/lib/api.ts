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
  };
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

export type ApiErrorBody = {
  success: false;
  error?: { code?: string; message?: string };
};

function getCookie(name: string): string {
  const match = document.cookie.match(
    new RegExp(`(?:^|; )${name.replace(/[$()*+.?[\\\]^{|}]/g, "\\$&")}=([^;]*)`),
  );
  return match ? decodeURIComponent(match[1]) : "";
}

export function getCsrfToken(): string {
  return getCookie("csrf_token");
}

function csrfHeaders(json = false): HeadersInit {
  const headers: Record<string, string> = {};
  const token = getCsrfToken();
  if (token) headers["X-CSRF-Token"] = token;
  if (json) headers["Content-Type"] = "application/json";
  return headers;
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
    const err = body as ApiErrorBody;
    throw new Error(err.error?.message || `Request failed (${res.status})`);
  }
  const ok = body as { success?: boolean; data?: T; error?: { message?: string } };
  if (!ok.success || ok.data === undefined) {
    throw new Error(ok.error?.message || "Unexpected API response");
  }
  return ok.data;
}

export async function fetchHealth(): Promise<HealthResponse> {
  const res = await fetch("/api/v1/health", { credentials: "same-origin" });
  if (!res.ok) {
    throw new Error(`Health check failed (${res.status})`);
  }
  return res.json();
}

export type UploadProgressEvent = {
  loaded: number;
  total: number;
  percent: number;
};

function xhrUpload(
  endpoint: string,
  form: FormData,
  onProgress?: (event: UploadProgressEvent) => void,
): Promise<S3UploadData | WPUploadData> {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    xhr.open("POST", endpoint);
    xhr.withCredentials = true;

    const token = getCsrfToken();
    if (token) xhr.setRequestHeader("X-CSRF-Token", token);

    xhr.upload.onprogress = (e) => {
      if (!e.lengthComputable || !onProgress) return;
      onProgress({
        loaded: e.loaded,
        total: e.total,
        percent: Math.round((e.loaded / e.total) * 100),
      });
    };

    xhr.onerror = () => reject(new Error("Network error during upload"));

    xhr.onload = () => {
      let body: unknown;
      try {
        body = JSON.parse(xhr.responseText || "{}");
      } catch {
        reject(new Error(xhr.responseText || "Invalid server response"));
        return;
      }

      if (xhr.status >= 200 && xhr.status < 300) {
        const ok = body as { success?: boolean; data?: S3UploadData | WPUploadData };
        if (ok.success && ok.data) {
          resolve(ok.data);
          return;
        }
      }

      const err = body as ApiErrorBody;
      reject(new Error(err.error?.message || `Upload failed (${xhr.status})`));
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

async function initChunkUpload(file: File): Promise<ChunkInitData> {
  const res = await fetch("/api/v1/upload/init", {
    method: "POST",
    credentials: "same-origin",
    headers: csrfHeaders(true),
    body: JSON.stringify({ filename: file.name, total_size: file.size }),
  });
  return parseApiJson<ChunkInitData>(res);
}

async function uploadChunkPart(
  uploadId: string,
  index: number,
  blob: Blob,
  onProgress?: (loaded: number, total: number) => void,
): Promise<void> {
  await new Promise<void>((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    xhr.open("POST", "/api/v1/upload/chunk");
    xhr.withCredentials = true;
    const token = getCsrfToken();
    if (token) xhr.setRequestHeader("X-CSRF-Token", token);

    xhr.upload.onprogress = (e) => {
      if (!e.lengthComputable || !onProgress) return;
      onProgress(e.loaded, e.total);
    };
    xhr.onerror = () => reject(new Error(`Network error on chunk ${index}`));
    xhr.onload = () => {
      try {
        const body = JSON.parse(xhr.responseText || "{}") as ApiErrorBody & {
          success?: boolean;
        };
        if (xhr.status >= 200 && xhr.status < 300 && body.success) {
          resolve();
          return;
        }
        reject(new Error(body.error?.message || `Chunk ${index} failed (${xhr.status})`));
      } catch {
        reject(new Error(`Chunk ${index} failed (${xhr.status})`));
      }
    };

    const form = new FormData();
    form.append("upload_id", uploadId);
    form.append("index", String(index));
    form.append("chunk", blob, `part-${index}`);
    xhr.send(form);
  });
}

async function completeChunkUpload(uploadId: string): Promise<S3UploadData> {
  const res = await fetch("/api/v1/upload/complete", {
    method: "POST",
    credentials: "same-origin",
    headers: csrfHeaders(true),
    body: JSON.stringify({ upload_id: uploadId }),
  });
  return parseApiJson<S3UploadData>(res);
}

async function abortChunkUpload(uploadId: string): Promise<void> {
  try {
    await fetch("/api/v1/upload/abort", {
      method: "POST",
      credentials: "same-origin",
      headers: csrfHeaders(true),
      body: JSON.stringify({ upload_id: uploadId }),
    });
  } catch {
    // best-effort cleanup
  }
}

async function uploadChunkPartWithRetry(
  uploadId: string,
  index: number,
  blob: Blob,
  onProgress?: (loaded: number, total: number) => void,
): Promise<void> {
  try {
    await uploadChunkPart(uploadId, index, blob, onProgress);
  } catch (firstErr) {
    // One retry for transient network blips
    await new Promise((r) => setTimeout(r, 400));
    try {
      await uploadChunkPart(uploadId, index, blob, onProgress);
    } catch {
      throw firstErr;
    }
  }
}

async function uploadChunked(
  file: File,
  onProgress?: (event: UploadProgressEvent) => void,
): Promise<S3UploadData> {
  const session = await initChunkUpload(file);
  if (session.total_size !== file.size) {
    await abortChunkUpload(session.upload_id);
    throw new Error("Server rejected file size for chunked upload");
  }

  let uploadedBytes = 0;

  try {
    for (let i = 0; i < session.total_chunks; i++) {
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
      });
      uploadedBytes += blob.size;
      onProgress?.({
        loaded: uploadedBytes,
        total: file.size,
        percent: Math.min(99, Math.round((uploadedBytes / file.size) * 100)),
      });
    }

    const result = await completeChunkUpload(session.upload_id);
    onProgress?.({ loaded: file.size, total: file.size, percent: 100 });
    return result;
  } catch (err) {
    await abortChunkUpload(session.upload_id);
    throw err;
  }
}

export type UploadLimits = {
  maxSize: number;
  absoluteMaxSize: number;
};

export async function uploadFile(
  file: File,
  mode: UploadMode,
  limits: UploadLimits,
  onProgress?: (event: UploadProgressEvent) => void,
): Promise<S3UploadData | WPUploadData> {
  if (mode === "wp") {
    if (file.size > limits.maxSize) {
      throw new Error(
        `WP resize supports files up to ${formatBytes(limits.maxSize)}. Use S3 upload for larger files (chunked up to ${formatBytes(limits.absoluteMaxSize)}).`,
      );
    }
    const form = new FormData();
    form.append("file", file);
    return xhrUpload("/api/v1/wp-upload", form, onProgress);
  }

  if (file.size > limits.absoluteMaxSize) {
    throw new Error(`File exceeds absolute limit of ${formatBytes(limits.absoluteMaxSize)}`);
  }

  if (file.size > limits.maxSize) {
    return uploadChunked(file, onProgress);
  }

  const form = new FormData();
  form.append("file", file);
  return xhrUpload("/api/v1/upload", form, onProgress);
}

export function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(2)} MB`;
}
