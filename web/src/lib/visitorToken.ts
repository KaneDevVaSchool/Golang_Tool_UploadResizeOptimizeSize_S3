// Định danh trình duyệt ẩn danh cho trang public - KHÔNG phải xác thực
// danh tính thật; visitor_token gắn vào mỗi lượt xem / reaction / comment và
// rate-limit chống spam. Lưu localStorage, không mất khi
// tải lại trang, mất khi xoá dữ liệu trình duyệt (chấp nhận được).
const VISITOR_TOKEN_KEY = "vas_visitor_token";
const DISPLAY_NAME_KEY = "vas_visitor_display_name";

function generateToken(): string {
  if (typeof crypto !== "undefined" && crypto.randomUUID) {
    return crypto.randomUUID();
  }
  return `${Date.now()}-${Math.random().toString(36).slice(2)}-${Math.random().toString(36).slice(2)}`;
}

export function getOrCreateVisitorToken(): string {
  try {
    const existing = localStorage.getItem(VISITOR_TOKEN_KEY);
    if (existing) return existing;
    const token = generateToken();
    localStorage.setItem(VISITOR_TOKEN_KEY, token);
    return token;
  } catch {
    // localStorage có thể bị chặn (private mode) - dùng token tạm thời cho
    // phiên hiện tại, không lưu được qua lần tải lại (chấp nhận được vì
    // đây chỉ ảnh hưởng dedupe, không phải lỗi chức năng).
    return generateToken();
  }
}

export function getSavedDisplayName(): string {
  try {
    return localStorage.getItem(DISPLAY_NAME_KEY) ?? "";
  } catch {
    return "";
  }
}

export function saveDisplayName(name: string) {
  try {
    localStorage.setItem(DISPLAY_NAME_KEY, name);
  } catch {
    // bỏ qua nếu không lưu được - không ảnh hưởng luồng chính
  }
}
