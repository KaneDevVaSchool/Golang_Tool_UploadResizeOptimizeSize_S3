// Chọn nguồn ảnh cho một tác phẩm.
//
// Backend sinh sẵn các cỡ nhỏ lúc upload (xem internal/service/image_variants.go)
// nhưng KHÔNG phải tác phẩm nào cũng có: ảnh upload trước khi có tính năng này,
// ảnh gốc vốn đã nhỏ hơn cỡ đích, hoặc khâu sinh biến thể lỗi. Vì vậy mọi chỗ
// hiển thị đều phải có đường lui, và gom vào đây để không mỗi nơi lui một kiểu.
//
// Thứ tự lui: biến thể đúng cỡ → thumbnail_url → image_url (ảnh gốc).
import type { ArtworkVariantKey, ArtworkWithMeta } from "./artworkApi";

/** Cỡ hiển thị, tính theo chỗ ảnh thực sự xuất hiện trên trang. */
export type ArtworkImageSize = "thumb" | "medium" | "large";

/** Phần tối thiểu của một tác phẩm mà helper cần - nhận cả DTO rút gọn
 *  (dashboard, billboard) chứ không riêng ArtworkWithMeta đầy đủ. */
type ImageSource = Pick<ArtworkWithMeta, "image_url" | "thumbnail_url" | "variants">;

function variantURL(item: ImageSource, key: ArtworkVariantKey): string | undefined {
  return item.variants?.[key] || undefined;
}

/**
 * URL dùng cho thuộc tính src - luôn là JPEG (hoặc ảnh gốc) để trình duyệt
 * nào cũng đọc được. WebP đi qua <source> trong pictureSources().
 */
export function artworkImageURL(item: ImageSource, size: ArtworkImageSize = "thumb"): string {
  return (
    variantURL(item, `${size}_jpg` as ArtworkVariantKey) ??
    item.thumbnail_url ??
    item.image_url
  );
}

/**
 * Danh sách <source> cho <picture>, xếp theo thứ tự ưu tiên: trình duyệt lấy
 * cái đầu tiên nó hiểu. Rỗng nghĩa là không có biến thể nào - lúc đó dùng thẻ
 * <img> trần với artworkImageURL() là đủ.
 */
export function artworkPictureSources(
  item: ImageSource,
  size: ArtworkImageSize = "thumb",
): Array<{ srcSet: string; type: string }> {
  const webp = variantURL(item, `${size}_webp` as ArtworkVariantKey);
  return webp ? [{ srcSet: webp, type: "image/webp" }] : [];
}
