import { useEffect, useState } from "react";

/**
 * Mức "sức" của thiết bị, dùng để quyết định cắt bớt bao nhiêu hiệu ứng.
 *
 * - "full": desktop — giữ nguyên toàn bộ hiệu ứng như thiết kế ban đầu.
 * - "light": tablet và màn hình vừa — giảm số hạt, bỏ blur nền.
 * - "minimal": điện thoại, hoặc người đã bật "giảm chuyển động" — chỉ giữ
 *   những gì thực sự cần để trang không bị trống trải.
 */
export type DeviceTier = "full" | "light" | "minimal";

// Ngưỡng khớp thang breakpoint trong tokens.css. 1024px trở lên coi là
// desktop; dưới 768px là điện thoại.
const DESKTOP_QUERY = "(min-width: 1024px)";
const PHONE_QUERY = "(max-width: 767px)";
const REDUCED_MOTION_QUERY = "(prefers-reduced-motion: reduce)";

function detect(): DeviceTier {
  if (typeof window === "undefined" || !window.matchMedia) return "full";

  // Người đã tắt chuyển động thì mọi thứ trang trí đều thừa, bất kể máy mạnh
  // hay yếu - đây là lựa chọn của họ, không phải giới hạn phần cứng.
  if (window.matchMedia(REDUCED_MOTION_QUERY).matches) return "minimal";
  if (window.matchMedia(PHONE_QUERY).matches) return "minimal";
  if (!window.matchMedia(DESKTOP_QUERY).matches) return "light";

  // Máy để bàn nhưng ít nhân CPU (máy văn phòng cũ trong trường) cũng nên
  // nhẹ đi. deviceMemory chỉ Chromium có, thiếu thì bỏ qua điều kiện này.
  const cores = navigator.hardwareConcurrency ?? 8;
  const memory = (navigator as Navigator & { deviceMemory?: number }).deviceMemory;
  if (cores <= 4 || (memory !== undefined && memory <= 4)) return "light";

  return "full";
}

/**
 * Theo dõi mức thiết bị và cập nhật khi người dùng xoay máy, đổi cỡ cửa sổ,
 * hoặc bật/tắt "giảm chuyển động" ngay trong lúc đang xem.
 *
 * Sinh ra vì trang public có rất nhiều hiệu ứng chạy liên tục (pháo hoa ở
 * Bảng vàng, lá bay nền, parallax, blur nền) - đẹp trên desktop nhưng là
 * nguyên nhân chính gây giật khi cuộn trên điện thoại tầm trung.
 */
export function useDeviceTier(): DeviceTier {
  const [tier, setTier] = useState<DeviceTier>(detect);

  useEffect(() => {
    const queries = [DESKTOP_QUERY, PHONE_QUERY, REDUCED_MOTION_QUERY].map((q) => window.matchMedia(q));
    const sync = () => setTier(detect());

    queries.forEach((mq) => mq.addEventListener("change", sync));
    // Chạy một lần sau khi mount: giá trị khởi tạo có thể lệch nếu HTML được
    // render sẵn ở kích thước khác (SSR/prerender).
    sync();

    return () => queries.forEach((mq) => mq.removeEventListener("change", sync));
  }, []);

  return tier;
}
