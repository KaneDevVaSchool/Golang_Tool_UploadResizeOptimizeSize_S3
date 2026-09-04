import { useEffect, useRef, useState } from "react";

// Hệ số nội suy mỗi khung hình - càng gần 1 càng bám sát scroll thật
// (giật cứng), càng gần 0 càng trễ/mượt (nhưng lag nhiều). 0.12 cho cảm
// giác "đuổi theo mượt" tinh tế, không rõ độ trễ với mắt thường.
const SMOOTHING = 0.12;
// Dưới ngưỡng này (px) coi như đã bắt kịp - dừng vòng lặp rAF để không
// tốn CPU chạy vô ích khi đã đứng yên.
const SETTLE_EPSILON = 0.05;

/**
 * Callback nhận vị trí cuộn đã làm mượt mỗi khung hình - dùng để ghi
 * transform thẳng vào DOM (qua ref) thay vì qua setState, xem
 * useParallaxScrollRef bên dưới.
 */
type Listener = (smoothY: number) => void;

/**
 * Vòng lặp rAF dùng chung cho mọi listener parallax trên trang - chỉ 1
 * listener window "scroll" và 1 vòng lặp rAF duy nhất bất kể có bao nhiêu
 * component gọi useParallaxScrollListener/useParallaxScroll, thay vì mỗi
 * component tự đăng ký scroll/rAF riêng (lãng phí khi Hero + các card khác
 * đều cần smoothY). Tôn trọng prefers-reduced-motion: khi bật, mọi listener
 * luôn nhận 0 và vòng lặp không chạy.
 */
let listeners: Set<Listener> = new Set();
let rafId: number | null = null;
let target = 0;
let current = 0;
let reduceMotion = false;

function tick() {
  const diff = target - current;
  if (Math.abs(diff) < SETTLE_EPSILON) {
    current = target;
    listeners.forEach((fn) => fn(current));
    rafId = null;
    return;
  }
  current += diff * SMOOTHING;
  listeners.forEach((fn) => fn(current));
  rafId = requestAnimationFrame(tick);
}

function onScroll() {
  target = window.scrollY;
  if (rafId === null && !reduceMotion) {
    rafId = requestAnimationFrame(tick);
  }
}

let scrollListenerAttached = false;
function ensureScrollListener() {
  if (scrollListenerAttached || typeof window === "undefined") return;
  scrollListenerAttached = true;
  const media = window.matchMedia("(prefers-reduced-motion: reduce)");
  reduceMotion = media.matches;
  media.addEventListener("change", () => {
    reduceMotion = media.matches;
  });
  target = window.scrollY;
  current = window.scrollY;
  window.addEventListener("scroll", onScroll, { passive: true });
}

/**
 * Đăng ký nhận vị trí cuộn dọc đã làm mượt mỗi khung hình qua callback
 * (không phải React state) - dùng cho các layer parallax ghi transform
 * thẳng vào DOM qua ref (xem HeroSection), tránh việc setState mỗi rAF tick
 * kéo theo re-render + reconciliation toàn bộ cây component ở tần suất tới
 * 60 lần/giây trong lúc cuộn. Trả về true/false đã reduceMotion để component
 * gọi ẩn hẳn phần layer khi cần (không chỉ khựng ở 0).
 */
export function useParallaxScrollListener(onChange: Listener) {
  const onChangeRef = useRef(onChange);
  onChangeRef.current = onChange;

  useEffect(() => {
    ensureScrollListener();
    const fn: Listener = (y) => onChangeRef.current(y);
    listeners.add(fn);
    // Gửi giá trị hiện tại ngay khi mount để layer không "nhảy" từ 0 lên vị
    // trí thật ở lần scroll tiếp theo.
    fn(reduceMotion ? 0 : current);
    return () => {
      listeners.delete(fn);
    };
  }, []);
}

/**
 * Trả về vị trí cuộn dọc đã làm mượt (không phải window.scrollY thô) -
 * mỗi khung hình nội suy dần từ giá trị hiện tại tới scrollY thật
 * (exponential smoothing qua requestAnimationFrame), tạo cảm giác các lớp
 * parallax "đuổi theo" scroll một cách mềm mại thay vì bám cứng theo con
 * lăn chuột, giống hiệu ứng scroll cao cấp (Apple-style smooth parallax).
 *
 * Đây là bản setState - còn dùng ở nơi thực sự cần smoothY dưới dạng giá
 * trị React (ví dụ tính toán khác ngoài transform CSS trực tiếp). Cho các
 * layer chỉ set transform/opacity theo scrollY (trường hợp phổ biến nhất -
 * HeroSection), dùng useParallaxScrollListener + ref để tránh re-render mỗi
 * khung hình.
 *
 * Tôn trọng prefers-reduced-motion: luôn trả về 0 khi user đã bật "giảm
 * chuyển động", vì transform parallax set qua inline style (không phải
 * class) nên CSS "animation: none !important" thông thường không tắt được.
 */
export function useParallaxScroll() {
  const [smoothY, setSmoothY] = useState(0);
  useParallaxScrollListener(setSmoothY);
  return reduceMotion ? 0 : smoothY;
}
