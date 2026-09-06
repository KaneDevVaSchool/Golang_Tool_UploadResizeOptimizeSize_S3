// Bảng màu dùng chung cho nhóm chủ đề sáng tạo - tách riêng khỏi AwardsPage
// vì 2 nơi tạo/sửa nhóm chủ đề (TopicCategoryQuickCreateModal và
// TopicCategoriesPage) đều cần chung 1 danh sách, tránh lệch màu preset giữa
// 2 chỗ. Dùng lại đúng bộ màu của AwardsPage.COLOR_PRESETS cho nhất quán
// thị giác giữa giải thưởng và nhóm chủ đề trong toàn bộ trang quản trị.
export const TOPIC_CATEGORY_COLOR_PRESETS = [
  "#725139",
  "#c49c57",
  "#9a0036",
  "#1750b5",
  "#009082",
  "#a67f42",
  "#2e7d32",
  "#d81b60",
  "#5e35b1",
  "#ef6c00",
  "#00838f",
  "#455a64",
];

/** Mặc định trùng màu icon Layers cũ trước khi có color picker - giữ diện mạo cũ cho nhóm tạo mới nếu admin không đổi màu. */
export const TOPIC_CATEGORY_DEFAULT_COLOR = "#725139";
