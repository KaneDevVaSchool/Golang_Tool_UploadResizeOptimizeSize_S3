import { useRef } from "react";
import { useParallaxScrollListener } from "../../hooks/useParallaxScroll";

// Tốc độ trôi khi cuộn - cùng cơ chế multi-layer parallax mà HeroSection
// dùng (xem SPEED trong HeroSection.tsx), nhưng biên độ nhỏ hơn nhiều vì
// đây là nền trang trí sau một trang nội dung dài (gallery), không phải
// hero full-screen: chuyển động phải đủ nhẹ để không "đua" với thao tác
// lướt xem ảnh của người dùng.
const SPEED = {
  ranges: 0.018,
  critterFooter: 0.03,
};

/**
 * Một dãy núi: <rect> bị cắt bởi <clipPath> path SVG. Nhiều lớp chồng
 * nhau (xa → gần) tạo chiều sâu thay vì một ảnh PNG đồi phẳng.
 */
function ClippedRange({
  id,
  d,
  fill,
}: {
  id: string;
  d: string;
  fill: string;
}) {
  return (
    <g>
      <defs>
        <clipPath id={id}>
          <path d={d} />
        </clipPath>
      </defs>
      <rect width="1440" height="560" fill={fill} clipPath={`url(#${id})`} />
    </g>
  );
}

/**
 * Lớp nền "khu vườn tổ tiên" cho trang Tác phẩm tiêu biểu.
 *
 * Tách 2 tầng: mặt đất tĩnh (nhiều dãy núi SVG clip-path + sóng / dương xỉ)
 * đứng dưới lưới tranh, và tầng sinh vật (bướm + sóc/thỏ). Tầng đất luôn
 * hiện; tầng sống tắt khi prefers-reduced-motion.
 *
 * `critters={false}` bỏ hẳn tầng sinh vật - trang Bảng vàng chỉ muốn nền
 * đồi núi tĩnh để pháo hoa trên bục là chuyển động duy nhất.
 */
export function FeaturedGardenScene({ critters = true }: { critters?: boolean }) {
  const rangesRef = useRef<HTMLDivElement>(null);
  const footerRef = useRef<HTMLDivElement>(null);

  useParallaxScrollListener((scrollY) => {
    if (rangesRef.current) rangesRef.current.style.transform = `translate3d(0, ${scrollY * SPEED.ranges}px, 0)`;
    if (footerRef.current)
      footerRef.current.style.transform = `translate3d(0, ${scrollY * SPEED.critterFooter}px, 0)`;
  });

  return (
    <div className="featured-garden-scene" aria-hidden>
      <div className="featured-garden-ground">
        <div className="featured-garden-wave" />

        <div className="featured-garden-peaks">
          <span className="featured-mtn-peak featured-mtn-peak--1" />
          <span className="featured-mtn-peak featured-mtn-peak--2" />
          <span className="featured-mtn-peak featured-mtn-peak--3" />
          <span className="featured-mtn-peak featured-mtn-peak--4" />
          <span className="featured-mtn-peak featured-mtn-peak--5" />
          <span className="featured-mtn-peak featured-mtn-peak--6" />
        </div>

        <div className="featured-garden-ranges" ref={rangesRef}>
          <svg viewBox="0 0 1440 560" preserveAspectRatio="xMidYMax slice" xmlns="http://www.w3.org/2000/svg">
            <defs>
              <linearGradient id="fg-mtn-far" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor="#c8e8dc" />
                <stop offset="100%" stopColor="#a8d4c4" />
              </linearGradient>
              <linearGradient id="fg-mtn-mid" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor="#9dceba" />
                <stop offset="100%" stopColor="#6fb89a" />
              </linearGradient>
              <linearGradient id="fg-mtn-near" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor="#5aa888" />
                <stop offset="100%" stopColor="#3fae86" />
              </linearGradient>
              <linearGradient id="fg-mtn-gold" x1="0" y1="0" x2="1" y2="1">
                <stop offset="0%" stopColor="#c4b07a" />
                <stop offset="55%" stopColor="#8fbe9a" />
                <stop offset="100%" stopColor="#3fae86" />
              </linearGradient>
              <linearGradient id="fg-mtn-deep" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor="#2f8f72" />
                <stop offset="100%" stopColor="#1f6f58" />
              </linearGradient>
            </defs>

            <ClippedRange
              id="fg-clip-skyline"
              fill="url(#fg-mtn-far)"
              d="M0,560 L0,210 C90,150 180,248 300,118 C420,12 540,188 720,78 C900,-8 1020,168 1180,68 C1300,16 1380,148 1440,88 L1440,560 Z"
            />
            <ClippedRange
              id="fg-clip-far"
              fill="url(#fg-mtn-mid)"
              d="M0,560 L0,278 C110,198 230,328 390,218 C560,108 700,288 860,176 C1020,82 1180,258 1440,158 L1440,560 Z"
            />
            <ClippedRange
              id="fg-clip-left"
              fill="url(#fg-mtn-gold)"
              d="M0,560 L0,300 C70,236 150,368 250,248 C360,148 490,348 640,278 C760,228 840,368 960,420 L960,560 Z"
            />
            <ClippedRange
              id="fg-clip-right"
              fill="url(#fg-mtn-near)"
              d="M480,560 L480,410 C600,328 760,398 900,288 C1040,186 1180,358 1320,248 C1380,208 1420,298 1440,268 L1440,560 Z"
            />
            <ClippedRange
              id="fg-clip-mid"
              fill="url(#fg-mtn-near)"
              d="M0,560 L0,348 C150,286 270,408 430,308 C590,214 760,388 940,298 C1100,218 1280,368 1440,286 L1440,560 Z"
            />
            <ClippedRange
              id="fg-clip-near"
              fill="url(#fg-mtn-deep)"
              d="M0,560 L0,428 C100,396 200,478 340,416 C480,354 620,468 780,406 C940,344 1100,458 1260,396 C1360,364 1400,438 1440,416 L1440,560 Z"
            />
          </svg>
        </div>

        <span className="featured-garden-fern featured-garden-fern--tl" />
        <span className="featured-garden-fern featured-garden-fern--tr" />
        <span className="featured-garden-fern featured-garden-fern--bl" />
        <span className="featured-garden-fern featured-garden-fern--br" />
      </div>

      {critters && (
        <>
          <img className="featured-butterfly featured-butterfly--1" src="/images/parallax/garden-butterfly.svg" alt="" />
          <img className="featured-butterfly featured-butterfly--2" src="/images/parallax/garden-butterfly.svg" alt="" />

          <div className="featured-critter-footer" ref={footerRef}>
            <img className="hero-critter hero-critter--squirrel" src="/images/parallax/squirrel.svg" alt="" />
            <img className="hero-critter hero-critter--rabbit" src="/images/parallax/rabbit.svg" alt="" />
          </div>
        </>
      )}
    </div>
  );
}
