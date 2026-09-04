import { motion, useReducedMotion, type PanInfo } from "framer-motion";
import { ChevronLeft, ChevronRight } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { EducationLevelCard } from "./EducationLevelCard";
import { HillDivider } from "./HillDivider";

const LEVELS = ["primary", "secondary"] as const;
type Level = (typeof LEVELS)[number];

const SWIPE_DISTANCE_THRESHOLD = 60;
const SWIPE_VELOCITY_THRESHOLD = 300;
const MOBILE_QUERY = "(max-width: 768px)";
// Ngưỡng di chuyển (px) để coi 1 cử chỉ là "kéo" thay vì "chạm chọn" - card
// và slot chồng lệch (drag) là 2 motion-node cha/con tách biệt nên Framer
// Motion không tự phân biệt tap/drag xuyên qua chúng như trên 1 node duy
// nhất; cờ didDragRef bù việc đó bằng tay (xem onDrag/handleDragEnd).
const DRAG_CLICK_SUPPRESS_PX = 8;

/**
 * true dưới breakpoint mobile - dưới ngưỡng này bố cục chồng lệch đổi tỉ lệ
 * (xem STACK_LAYOUT bên dưới): card back thu hẹp thành 1 dải "peek" hẹp lộ
 * ra ở mép phải thay vì gần bằng front, đủ để gợi ý vuốt/kéo mà không cần
 * chấm dot hay mũi tên (vốn đã ẩn ở mobile) để chỉ dẫn.
 */
function useIsMobile() {
  const [isMobile, setIsMobile] = useState(
    () => typeof window !== "undefined" && window.matchMedia(MOBILE_QUERY).matches,
  );
  useEffect(() => {
    const mql = window.matchMedia(MOBILE_QUERY);
    const onChange = () => setIsMobile(mql.matches);
    onChange();
    mql.addEventListener("change", onChange);
    return () => mql.removeEventListener("change", onChange);
  }, []);
  return isMobile;
}

/** Vị trí/kích thước front-back theo breakpoint - % tính theo .edu-gate-slot
 * (width CSS cố định, xem public.css). Desktop: back nhô ra đủ rộng để đọc
 * nội dung (peeking offset trong EducationLevelCard). Mobile: back chỉ hé
 * một dải mỏng bên phải - đủ thấy có "thứ gì đó" tiếp theo để vuốt qua, thể
 * hiện tinh thần "vẫn thấy tí nội dung bên kia" mà không cần dot/mũi tên. */
const STACK_LAYOUT = {
  desktop: { frontLeft: "-6%", backLeft: "40%", backScale: 0.94, backOpacity: 0.85 },
  mobile: { frontLeft: "0%", backLeft: "84%", backScale: 0.92, backOpacity: 0.9 },
};

/**
 * Section 2 - Cổng thông tin chọn khối: 2 card chồng lệch kiểu "card deck"
 * thay vì side-by-side bằng nhau - card đang chọn (front) nằm trái, gần trọn
 * sân khấu; card còn lại (back) nhô ra bên phải, đè lên khoảng 1/3 phải của
 * front, thu nhỏ + mờ nhẹ để tạo chiều sâu. Bấm/kéo card back (hoặc mũi
 * tên/dot) đẩy nó lên làm front - swap vị trí bằng spring animation. Dải đồi
 * SVG (HillDivider) mở đầu section, tiếp nối chân trời Hero.
 */
export function EducationLevelGate({
  primaryCount,
  secondaryCount,
  onSelectLevel,
}: {
  primaryCount: number;
  secondaryCount: number;
  onSelectLevel: (level: "primary" | "secondary") => void;
}) {
  const [frontIndex, setFrontIndex] = useState(0);
  const reduceMotion = useReducedMotion();
  const isMobile = useIsMobile();
  const layout = isMobile ? STACK_LAYOUT.mobile : STACK_LAYOUT.desktop;
  const counts: Record<Level, number> = { primary: primaryCount, secondary: secondaryCount };
  // true trong lúc 1 cử chỉ kéo đang/vừa vượt ngưỡng "là drag chứ không phải
  // tap" - đọc bởi wrapper onSelect bên dưới để nuốt click giả sinh ra khi
  // nhả chuột/ngón tay đúng lúc con trỏ đang nằm trên nút CTA/card.
  const didDragRef = useRef(false);

  function goTo(next: number) {
    setFrontIndex((next + LEVELS.length) % LEVELS.length);
  }

  function handleDrag(_e: unknown, info: PanInfo) {
    if (Math.abs(info.offset.x) > DRAG_CLICK_SUPPRESS_PX) {
      didDragRef.current = true;
    }
  }

  function handleDragEnd(_e: unknown, info: PanInfo) {
    const { offset, velocity } = info;
    if (offset.x < -SWIPE_DISTANCE_THRESHOLD || velocity.x < -SWIPE_VELOCITY_THRESHOLD) {
      goTo(frontIndex + 1);
    } else if (offset.x > SWIPE_DISTANCE_THRESHOLD || velocity.x > SWIPE_VELOCITY_THRESHOLD) {
      goTo(frontIndex - 1);
    }
    // để tap event (bubble ngay sau dragend) vẫn thấy didDragRef=true trước
    // khi reset lại cho lần chạm tiếp theo.
    setTimeout(() => {
      didDragRef.current = false;
    }, 0);
  }

  function withDragGuard(action: () => void) {
    return () => {
      if (didDragRef.current) return;
      action();
    };
  }

  return (
    <div className="edu-gate-wrap">
      <HillDivider />
      <section className="edu-gate-section">
        <motion.div
          className="section-heading"
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true, margin: "-80px" }}
          transition={{ duration: 0.5 }}
        >
          <span className="section-kicker">Cổng thông tin</span>
          <h2>Chọn khối để khám phá</h2>
        </motion.div>

        <div className="edu-gate-carousel">
          <button
            type="button"
            className="edu-gate-arrow edu-gate-arrow--prev"
            aria-label="Khối trước"
            onClick={() => goTo(frontIndex - 1)}
          >
            <ChevronLeft size={22} />
          </button>
          <button
            type="button"
            className="edu-gate-arrow edu-gate-arrow--next"
            aria-label="Khối tiếp theo"
            onClick={() => goTo(frontIndex + 1)}
          >
            <ChevronRight size={22} />
          </button>

          <div className="edu-gate-stage">
            {LEVELS.map((level, i) => {
              // vị trí tương đối so với front hiện tại: 0 = đang trước, 1 = đang sau
              const rank = (i - frontIndex + LEVELS.length) % LEVELS.length;
              const isFront = rank === 0;

              return (
                <motion.div
                  key={level}
                  className={`edu-gate-slot${isFront ? " edu-gate-slot--front" : " edu-gate-slot--back"}`}
                  style={{ zIndex: LEVELS.length - rank }}
                  // Desktop: chỉ card "back" (đang hé ra) kéo được - front
                  // chủ yếu để click chọn/CTA, tránh nhầm khi rê chuột. Mobile:
                  // cả front cũng kéo được vì đó là cách chính để vuốt sang
                  // card kia (không còn mũi tên/dot chỉ dẫn ở mobile).
                  drag={reduceMotion ? false : !isFront || isMobile ? "x" : false}
                  dragConstraints={{ left: 0, right: 0 }}
                  dragElastic={0.5}
                  onDrag={handleDrag}
                  onDragEnd={handleDragEnd}
                  initial={false}
                  animate={{
                    left: isFront ? layout.frontLeft : layout.backLeft,
                    scale: isFront ? 1 : layout.backScale,
                    opacity: isFront ? 1 : layout.backOpacity,
                  }}
                  transition={{ type: "spring", stiffness: 240, damping: 28 }}
                >
                  <EducationLevelCard
                    level={level}
                    artworkCount={counts[level]}
                    peeking={!isFront && !isMobile}
                    onSelect={withDragGuard(() => (isFront ? onSelectLevel(level) : goTo(i)))}
                  />
                </motion.div>
              );
            })}
          </div>
        </div>

        <div className="edu-gate-dots" role="tablist" aria-label="Chọn khối">
          {LEVELS.map((level, i) => (
            <button
              key={level}
              type="button"
              role="tab"
              aria-selected={i === frontIndex}
              className={`edu-gate-dot edu-gate-dot--${level}${i === frontIndex ? " edu-gate-dot--active" : ""}`}
              onClick={() => goTo(i)}
            >
              <span className="edu-gate-dot-mark" aria-hidden />
              <span className="edu-gate-dot-label">{level === "primary" ? "Tiểu học" : "Trung học"}</span>
            </button>
          ))}
        </div>
      </section>
    </div>
  );
}
