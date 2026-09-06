// Diễn giải câu gõ tự do của khách vào ô tìm kiếm của mascot rồng thành một
// trong ba ý định đã biết, rồi map sang đúng API public sẵn có
// (fetchPublicArtworks/fetchBillboard) - KHÔNG gọi AI ngoài, chỉ so khớp từ
// khoá tiếng Việt (có bỏ dấu) và một bước "gõ gần đúng" (Levenshtein khoảng
// cách ngắn) để chịu được lỗi chính tả nhẹ. Tách khỏi component để logic có
// thể test độc lập UI.
import { fetchBillboard, fetchPublicArtworks, type BillboardEntry } from "./publicApi";
import type { ArtworkWithMeta } from "./artworkApi";

export type MascotIntent = "awards" | "search";

export type MascotResult =
  | { intent: "awards"; entries: BillboardEntry[]; matchedBy: "keyword" | "loose" }
  | { intent: "search"; items: ArtworkWithMeta[]; totalCount: number };

/** Bỏ dấu để so khớp từ khoá không phân biệt có dấu/không dấu. */
function fold(value: string): string {
  return value
    .toLowerCase()
    .normalize("NFD")
    .replace(/[̀-ͯ]/g, "")
    .trim();
}

const AWARD_KEYWORDS = [
  "giai nhat",
  "giai nhi",
  "giai ba",
  "giai thuong",
  "giai khuyen khich",
  "ai dat giai",
  "ai doat giai",
  "ai thang",
  "ai duoc giai",
  "bang vang",
  "danh sach giai",
  "ket qua thi",
  "xep hang",
  "hang nhat",
  "hang nhi",
  "hang ba",
  "giai",
];

/** Câu gõ có đang hỏi về giải thưởng/kết quả không, để hiển thị Bảng vàng thay vì tìm tranh. */
export function detectIntent(query: string): MascotIntent {
  const folded = fold(query);
  if (AWARD_KEYWORDS.some((kw) => folded.includes(kw))) return "awards";
  return "search";
}

/**
 * Khoảng cách chỉnh sửa (Levenshtein), dùng để bắt lỗi gõ nhẹ (thiếu/thừa 1-2
 * ký tự, gõ nhầm dấu). Đủ cho câu ngắn (tên tranh/tên học sinh thường dưới 30
 * ký tự) - không cần thư viện ngoài cho việc này.
 */
function editDistance(a: string, b: string): number {
  const dp: number[][] = Array.from({ length: a.length + 1 }, (_, i) => [i, ...Array(b.length).fill(0)]);
  for (let j = 0; j <= b.length; j++) dp[0][j] = j;
  for (let i = 1; i <= a.length; i++) {
    for (let j = 1; j <= b.length; j++) {
      dp[i][j] =
        a[i - 1] === b[j - 1]
          ? dp[i - 1][j - 1]
          : 1 + Math.min(dp[i - 1][j - 1], dp[i - 1][j], dp[i][j - 1]);
    }
  }
  return dp[a.length][b.length];
}

/** Ngưỡng "gần đúng": cho phép sai lệch tỉ lệ với độ dài từ, tối đa 2 ký tự. */
function isCloseMatch(needle: string, haystack: string): boolean {
  if (haystack.includes(needle)) return true;
  const maxDistance = needle.length <= 4 ? 1 : 2;
  const words = haystack.split(/\s+/);
  return words.some((word) => editDistance(needle, word) <= maxDistance);
}

const MAX_RESULTS = 8;

/**
 * Chạy tìm kiếm theo câu gõ. Ý định "awards" lọc ngay trong danh sách bảng
 * vàng đã tải (nếu câu gõ có kèm thêm từ khoá tên/lớp thì lọc tiếp theo tên
 * tác phẩm/học sinh/trường, khớp gần đúng qua isCloseMatch để chịu lỗi gõ);
 * ý định "search" gọi thẳng /api/v1/public/artworks?search= - đúng hợp đồng
 * LIKE đã dùng ở GallerySearch, và nếu backend không trả gì thì thử lại một
 * lần với "gõ gần đúng" trên tập billboard + trang đầu artworks để gợi ý,
 * thay vì im lặng trả rỗng.
 */
export async function runMascotSearch(query: string, signal?: AbortSignal): Promise<MascotResult> {
  const trimmed = query.trim();
  const intent = detectIntent(trimmed);

  if (intent === "awards") {
    // Backend trả "data": null (không phải []) khi bảng vàng rỗng - cùng quy
    // ước đã thấy ở fetchPublicArtworks bên dưới. Luôn chốt về mảng rỗng
    // ngay tại nguồn để mọi .length/.filter phía sau không cần tự phòng thủ.
    const all = (await fetchBillboard(signal)) ?? [];
    const extra = fold(trimmed).replace(new RegExp(AWARD_KEYWORDS.join("|"), "g"), "").trim();
    if (!extra) return { intent: "awards", entries: all, matchedBy: "keyword" };

    const exact = all.filter((entry) =>
      [entry.title, entry.student_name, entry.school_name, entry.award.name].some((field) =>
        fold(field).includes(extra),
      ),
    );
    if (exact.length > 0) return { intent: "awards", entries: exact, matchedBy: "keyword" };

    const loose = all.filter((entry) =>
      [entry.title, entry.student_name, entry.school_name].some((field) => isCloseMatch(extra, fold(field))),
    );
    return { intent: "awards", entries: loose, matchedBy: "loose" };
  }

  const result = await fetchPublicArtworks({ search: trimmed, page: 1, page_size: MAX_RESULTS }, signal);
  // Cùng quy ước null-khi-rỗng ở trên: "items" vắng mặt hoàn toàn (không phải
  // mảng rỗng) khi không có tác phẩm nào khớp - đã gây crash panel trước khi
  // sửa (Cannot read properties of null (reading 'length')).
  return { intent: "search", items: result.items ?? [], totalCount: result.total_count };
}

/** Câu gõ có đủ dài để tìm không, hay chỉ là 1 ký tự gõ nhầm/chưa kịp gõ xong. */
export function isQueryTooShort(query: string): boolean {
  return query.trim().length > 0 && query.trim().length < 2;
}
