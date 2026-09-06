import { useEffect, useMemo, useRef, useState } from "react";
import type { CSSProperties } from "react";
import { useDeviceTier } from "../../hooks/useDeviceTier";

/** Một quả pháo: vị trí nổ, thời điểm, màu, cỡ. */
type Burst = {
  left: number;
  top: number;
  delay: number;
  hue: string;
  scale: number;
  /** Quãng đường viên pháo bay lên trước khi nổ, tính bằng % chiều cao khu. */
  rise: number;
};

/**
 * Vị trí cố định, không random: pháo phải nổ hai bên và phía trên bục,
 * chừa khoảng giữa cho khung quán quân. Random mỗi lần render sẽ có lúc
 * dồn hết một góc, lại làm ảnh nhấp nháy lệch nhau giữa các lần tải.
 *
 * Delay rải đều trong chu kỳ 6s và không trùng nhau, nên gần như lúc nào
 * cũng có một quả đang nở - đây là chuyển động duy nhất của trang nên nó
 * phải liên tục, không để khoảng lặng.
 */
const BURSTS: Burst[] = [
  { left: 11, top: 20, delay: 0, hue: "#f0b429", scale: 1.15, rise: 26 },
  { left: 88, top: 25, delay: 0.7, hue: "#e8734a", scale: 1, rise: 22 },
  { left: 24, top: 9, delay: 1.5, hue: "#f5d76e", scale: 0.85, rise: 18 },
  { left: 74, top: 8, delay: 2.2, hue: "#ffd166", scale: 1.05, rise: 20 },
  { left: 4, top: 45, delay: 2.9, hue: "#f0b429", scale: 0.75, rise: 16 },
  { left: 95, top: 48, delay: 3.5, hue: "#e8c56a", scale: 0.8, rise: 16 },
  { left: 38, top: 6, delay: 4.1, hue: "#ff8fa3", scale: 0.7, rise: 14 },
  { left: 62, top: 5, delay: 4.7, hue: "#7fd4c1", scale: 0.72, rise: 14 },
  { left: 17, top: 58, delay: 5.3, hue: "#ffd166", scale: 0.62, rise: 12 },
  { left: 83, top: 62, delay: 5.8, hue: "#e8734a", scale: 0.65, rise: 12 },
];

/** Số tia mỗi quả - 16 tia đủ dày để ra hình cầu mà chưa thành đốm đặc. */
const SPARK_COUNT = 16;

/**
 * Lượng pháo theo sức máy. Bản đầy đủ là 10 quả × (1 đạn + 1 chớp + 16 tia)
 * = 180 phần tử, tất cả đều animation vô hạn và cố ý không có khoảng lặng.
 * Trên desktop thì không sao, nhưng đó chính là thứ làm điện thoại tầm trung
 * giật khi cuộn trang Bảng vàng.
 *
 * Cắt theo hai chiều - ít quả hơn VÀ ít tia hơn mỗi quả - vì chi phí là tích
 * của hai số này. Vẫn giữ vài quả lệch pha nhau để cảm giác "luôn có pháo"
 * không mất đi.
 */
const BURST_BUDGET: Record<string, { bursts: number; sparks: number }> = {
  full: { bursts: BURSTS.length, sparks: SPARK_COUNT },
  light: { bursts: 6, sparks: 10 },
  minimal: { bursts: 3, sparks: 6 },
};

/**
 * Pháo hoa nền cho bục vinh danh. Thuần CSS (không canvas, không thư
 * viện): mỗi quả gồm một viên đạn bay vọt lên, một chớp sáng ở tâm rồi
 * SPARK_COUNT tia bung ra, lặp vô hạn với độ trễ lệch nhau nên không bao
 * giờ nổ đồng loạt.
 *
 * Toàn bộ nằm dưới lớp nội dung và pointer-events:none, nên không cản
 * thao tác bấm vào khung tranh. Người bật "giảm chuyển động" trong hệ
 * điều hành sẽ không thấy gì (ẩn hẳn trong CSS).
 */
export function HallFireworks() {
  const tier = useDeviceTier();
  const budget = BURST_BUDGET[tier] ?? BURST_BUDGET.full;
  const rootRef = useRef<HTMLDivElement>(null);
  // Chỉ chạy khi khu vực pháo hoa thực sự nằm trong tầm nhìn. Trang Bảng
  // vàng cuộn dài, mà pháo hoa chỉ ở phần bục vinh danh trên cùng - trước
  // đây nó vẫn vẽ lại mỗi khung hình kể cả khi người xem đã cuộn xuống tận
  // danh sách bên dưới và không còn nhìn thấy gì.
  const [visible, setVisible] = useState(true);

  useEffect(() => {
    const el = rootRef.current;
    if (!el || typeof IntersectionObserver === "undefined") return;

    const observer = new IntersectionObserver(
      ([entry]) => setVisible(entry.isIntersecting),
      // rootMargin dương: bật lại trước khi khu vực kịp lọt vào màn hình, để
      // người cuộn ngược lên không thấy pháo "khởi động" giữa chừng.
      { rootMargin: "200px" },
    );
    observer.observe(el);
    return () => observer.disconnect();
  }, []);

  // Lấy các quả rải đều trong danh sách thay vì cắt lấy phần đầu: BURSTS xếp
  // theo delay tăng dần, cắt đầu sẽ dồn hết pháo vào 3 giây đầu chu kỳ rồi
  // im lặng phần còn lại.
  const bursts = useMemo(() => {
    if (budget.bursts >= BURSTS.length) return BURSTS;
    const step = BURSTS.length / budget.bursts;
    return Array.from({ length: budget.bursts }, (_, i) => BURSTS[Math.floor(i * step)]);
  }, [budget.bursts]);

  // Toạ độ tia tính một lần: mỗi tia là một góc trên đường tròn, dịch ra
  // bằng transform nên trình duyệt chỉ phải xử lý transform/opacity.
  const sparks = useMemo(
    () =>
      Array.from({ length: budget.sparks }, (_, i) => {
        const angle = (360 / budget.sparks) * i;
        return { angle, key: i };
      }),
    [budget.sparks],
  );

  return (
    <div ref={rootRef} className="hall-fireworks" aria-hidden>
      {visible && bursts.map((burst, i) => (
        <span
          key={i}
          className="hall-firework"
          style={
            {
              left: `${burst.left}%`,
              top: `${burst.top}%`,
              "--fw-delay": `${burst.delay}s`,
              "--fw-color": burst.hue,
              "--fw-scale": burst.scale,
              "--fw-rise": `${burst.rise}vh`,
            } as CSSProperties
          }
        >
          {/* Viên đạn bay lên - phần dẫn mắt tới chỗ sắp nổ. */}
          <span className="hall-firework-trail" />
          {/* Chớp sáng ở tâm ngay khoảnh khắc nổ. */}
          <span className="hall-firework-flash" />
          {sparks.map((spark) => (
            <span
              key={spark.key}
              className="hall-firework-spark"
              style={{ "--fw-angle": `${spark.angle}deg` } as CSSProperties}
            />
          ))}
        </span>
      ))}
    </div>
  );
}
