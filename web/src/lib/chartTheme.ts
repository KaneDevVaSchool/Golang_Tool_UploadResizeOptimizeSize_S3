/**
 * Bảng màu và hằng số dùng chung cho khu vực quản trị.
 *
 * Từng phục vụ cả biểu đồ recharts ở Dashboard (nhịp hoạt động, phân bổ khối
 * lớp, top tác phẩm) - phần đó đã bỏ để nhường chỗ cho card thống kê theo
 * khu vực, nên các export chỉ-dùng-cho-biểu-đồ (CHART_*, LEVEL_*,
 * formatDayLabel, formatCompact) đã gỡ theo. Còn lại REGION_* + formatNumber
 * vì card khu vực vẫn cần màu định danh Sài Gòn/Cần Thơ/Vũng Tàu và cách
 * đọc số kiểu Việt Nam thống nhất.
 *
 * Đừng đổi các mã màu này bằng cảm tính - khoảng cách màu giữa ba khu vực là
 * thứ không nhìn bằng mắt mà biết được (đã qua kiểm tra tương phản/CVD).
 */

/** Màu cố định theo khu vực - không bao giờ đổi theo thứ hạng hiện tại, để
 *  người đọc quen "Sài Gòn màu đỏ" không bị đánh lừa khi lọc bớt một khu
 *  vực hoặc khu vực đổi thứ hạng số liệu. */
export const REGION_COLOR: Record<string, string> = {
  saigon: "#b3003f",
  cantho: "#1750b5",
  vungtau: "#009082",
};

export const REGION_LABEL: Record<string, string> = {
  saigon: "Sài Gòn",
  cantho: "Cần Thơ",
  vungtau: "Vũng Tàu",
};

/** Định dạng số kiểu Việt Nam (1.284) - dùng thống nhất mọi nơi. */
export function formatNumber(value: number): string {
  return value.toLocaleString("vi-VN");
}
