import { useMemo } from "react";
import type { CSSProperties } from "react";

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
  // Toạ độ tia tính một lần: mỗi tia là một góc trên đường tròn, dịch ra
  // bằng transform nên trình duyệt chỉ phải xử lý transform/opacity.
  const sparks = useMemo(
    () =>
      Array.from({ length: SPARK_COUNT }, (_, i) => {
        const angle = (360 / SPARK_COUNT) * i;
        return { angle, key: i };
      }),
    [],
  );

  return (
    <div className="hall-fireworks" aria-hidden>
      {BURSTS.map((burst, i) => (
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
