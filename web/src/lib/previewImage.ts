/** Downscale a blob/file for thumbnails — avoids decoding multi‑MB originals in the strip. */
export async function createThumbObjectURL(
  source: Blob,
  maxEdge = 160,
  quality = 0.82,
): Promise<string | null> {
  if (!source.type.startsWith("image/")) return null;

  try {
    const bitmap = await createImageBitmap(source);
    try {
      const scale = Math.min(1, maxEdge / Math.max(bitmap.width, bitmap.height));
      const w = Math.max(1, Math.round(bitmap.width * scale));
      const h = Math.max(1, Math.round(bitmap.height * scale));
      const canvas = document.createElement("canvas");
      canvas.width = w;
      canvas.height = h;
      const ctx = canvas.getContext("2d");
      if (!ctx) return null;
      ctx.imageSmoothingEnabled = true;
      ctx.imageSmoothingQuality = "high";
      ctx.drawImage(bitmap, 0, 0, w, h);

      const blob = await new Promise<Blob | null>((resolve) => {
        canvas.toBlob(resolve, "image/jpeg", quality);
      });
      if (!blob) return null;
      return URL.createObjectURL(blob);
    } finally {
      bitmap.close();
    }
  } catch {
    return null;
  }
}

export function revokeObjectUrl(url: string | null | undefined) {
  if (url) URL.revokeObjectURL(url);
}
