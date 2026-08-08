export type ImageTransform = {
  /** Clockwise degrees: 0 | 90 | 180 | 270 */
  rotation: number;
  flipH: boolean;
  flipV: boolean;
};

export const DEFAULT_TRANSFORM: ImageTransform = {
  rotation: 0,
  flipH: false,
  flipV: false,
};

export function normalizeRotation(deg: number): number {
  const n = ((deg % 360) + 360) % 360;
  return n === 0 || n === 90 || n === 180 || n === 270 ? n : 0;
}

export function hasTransform(t: ImageTransform): boolean {
  return normalizeRotation(t.rotation) !== 0 || t.flipH || t.flipV;
}

export function cssImageTransform(t: ImageTransform): string {
  const parts = [`rotate(${normalizeRotation(t.rotation)}deg)`];
  if (t.flipH) parts.push("scaleX(-1)");
  if (t.flipV) parts.push("scaleY(-1)");
  return parts.join(" ");
}

function loadImage(src: string): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const img = new Image();
    img.onload = () => resolve(img);
    img.onerror = () => reject(new Error("Không đọc được ảnh để xoay"));
    img.src = src;
  });
}

function pickMime(file: File): { type: string; quality?: number } {
  if (file.type === "image/png" || file.type === "image/webp" || file.type === "image/gif") {
    return { type: file.type };
  }
  return { type: "image/jpeg", quality: 0.92 };
}

function canvasToFile(canvas: HTMLCanvasElement, file: File): Promise<File> {
  const { type, quality } = pickMime(file);
  return new Promise((resolve, reject) => {
    canvas.toBlob(
      (blob) => {
        if (!blob) {
          reject(new Error("Không xuất được ảnh đã chỉnh"));
          return;
        }
        const base = file.name.replace(/\.[^.]+$/, "") || "image";
        const ext =
          type === "image/png" ? ".png" : type === "image/webp" ? ".webp" : ".jpg";
        const name = file.name.includes(".") ? file.name.replace(/\.[^.]+$/, ext) : `${base}${ext}`;
        resolve(new File([blob], name, { type, lastModified: Date.now() }));
      },
      type,
      quality,
    );
  });
}

/** Bake rotation + flips into a new File. Returns original when no-op or non-image. */
export async function bakeImageTransform(file: File, transform: ImageTransform): Promise<File> {
  if (!file.type.startsWith("image/") || !hasTransform(transform)) {
    return file;
  }

  const rotation = normalizeRotation(transform.rotation);
  const url = URL.createObjectURL(file);
  try {
    const img = await loadImage(url);
    const swap = rotation === 90 || rotation === 270;
    const canvas = document.createElement("canvas");
    canvas.width = swap ? img.naturalHeight : img.naturalWidth;
    canvas.height = swap ? img.naturalWidth : img.naturalHeight;

    const ctx = canvas.getContext("2d");
    if (!ctx) throw new Error("Canvas không khả dụng");

    ctx.translate(canvas.width / 2, canvas.height / 2);
    ctx.rotate((rotation * Math.PI) / 180);
    ctx.scale(transform.flipH ? -1 : 1, transform.flipV ? -1 : 1);
    ctx.drawImage(img, -img.naturalWidth / 2, -img.naturalHeight / 2);

    return canvasToFile(canvas, file);
  } finally {
    URL.revokeObjectURL(url);
  }
}
