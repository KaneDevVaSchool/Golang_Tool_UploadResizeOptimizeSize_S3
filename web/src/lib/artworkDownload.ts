import { zipSync, type Zippable } from "fflate";
import { downloadArtworkOriginal } from "./artworkApi";

/**
 * Tải nhiều ảnh gốc rồi nén thành 1 file .zip để tải xuống - dùng cho thao
 * tác bulk ở trang danh sách tác phẩm. Ảnh nằm trên S3 (cross-origin), nên
 * không thể lặp thẻ <a download> trỏ thẳng tới S3: trình duyệt sẽ MỞ ảnh
 * (điều hướng) thay vì tải, vì thuộc tính download chỉ có hiệu lực same-origin.
 * Phải tải qua backend (same-origin blob) rồi tự đóng gói ở client.
 *
 * zipSync (đồng bộ) đủ dùng ở quy mô vài chục ảnh vài MB/ảnh của hội thi này -
 * nếu sau này cần hàng trăm ảnh cùng lúc, cân nhắc zip() bất đồng bộ của
 * fflate để không chặn main thread khi nén.
 */
export type BulkDownloadProgress = {
  done: number;
  total: number;
};

export type BulkDownloadResult = {
  blob: Blob;
  succeeded: number;
  failed: Array<{ id: number; error: string }>;
};

export async function buildArtworkZip(
  ids: number[],
  onProgress?: (progress: BulkDownloadProgress) => void,
): Promise<BulkDownloadResult> {
  const files: Zippable = {};
  const failed: BulkDownloadResult["failed"] = [];
  // Tên trùng nhau (2 tác phẩm cùng tên) phải được làm khác biệt trong zip -
  // đếm số lần đã dùng mỗi tên để tự thêm hậu tố " (2)", " (3)"...
  const usedNames = new Map<string, number>();

  for (let i = 0; i < ids.length; i++) {
    try {
      const { blob, fileName } = await downloadArtworkOriginal(ids[i]);
      const uniqueName = dedupeName(fileName, usedNames);
      files[uniqueName] = new Uint8Array(await blob.arrayBuffer());
    } catch (err) {
      failed.push({ id: ids[i], error: err instanceof Error ? err.message : "Không tải được ảnh" });
    }
    onProgress?.({ done: i + 1, total: ids.length });
  }

  const zipped = zipSync(files, { level: 6 });
  // zipSync trả Uint8Array<ArrayBufferLike> (có thể là SharedArrayBuffer khi
  // chạy trong worker) - ép về ArrayBuffer thường để khớp kiểu BlobPart, an
  // toàn vì file chạy trên main thread nên buffer luôn là ArrayBuffer thật.
  return { blob: new Blob([zipped.buffer as ArrayBuffer], { type: "application/zip" }), succeeded: Object.keys(files).length, failed };
}

function dedupeName(name: string, used: Map<string, number>): string {
  const count = used.get(name) ?? 0;
  used.set(name, count + 1);
  if (count === 0) return name;

  const dotIndex = name.lastIndexOf(".");
  const base = dotIndex > 0 ? name.slice(0, dotIndex) : name;
  const ext = dotIndex > 0 ? name.slice(dotIndex) : "";
  return `${base} (${count + 1})${ext}`;
}

/**
 * triggerBlobDownload: kích hoạt tải 1 Blob xuống máy người dùng qua thẻ <a>
 * tạm same-origin (blob: URL) - khác thẻ <a download> trỏ S3 trực tiếp (bị
 * chặn bởi lý do same-origin nêu trên), blob: URL luôn tải được vì nó không
 * phải cross-origin.
 */
export function triggerBlobDownload(blob: Blob, fileName: string) {
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = fileName;
  document.body.appendChild(a);
  a.click();
  a.remove();
  // Trì hoãn revoke để Firefox/Safari kịp bắt đầu tải trước khi URL bị huỷ.
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}
