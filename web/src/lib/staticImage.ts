/**
 * Đường dẫn WebP cho ảnh tĩnh trong public/images (asset trang trí, không
 * phải ảnh tác phẩm học sinh - loại đó dùng lib/artworkImage.ts). Mỗi PNG ở
 * đây có sẵn bản .webp cùng thư mục (sinh một lần bằng sharp, xem
 * scripts/optimize-static-images ghi trong git log) - hàm này chỉ đổi đuôi,
 * không cần biết thư mục con.
 */
export function webpOf(pngPath: string): string {
  return pngPath.replace(/\.png$/i, ".webp");
}
