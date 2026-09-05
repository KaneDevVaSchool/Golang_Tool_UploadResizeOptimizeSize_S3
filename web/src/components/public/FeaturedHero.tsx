import { motion } from "framer-motion";
import { useRef } from "react";
import { useParallaxScrollListener } from "../../hooks/useParallaxScroll";
import { fadeUp } from "../../lib/motionPresets";
import { HeroCritters, HeroHillFar, HeroHillMid, HeroHillNear, HeroLeaf, HeroPlant, HeroTree } from "./HeroParallaxHills";

const SPEED = {
  hillFar: 0.08,
  hillMid: 0.16,
  hillNear: 0.26,
  tree: 0.3,
  plant: 0.44,
  leaf: 0.5,
};
const MAX_SCROLL_EFFECT = 280;

/**
 * Banner trang Tác phẩm tiêu biểu - cùng bộ đồi/cây/thú với Hero trang chủ
 * nhưng ôm sát chữ (không ép min-height / dải đất trắng dày). Chữ nằm trong
 * tấm kính mờ kiểu biển triển lãm; đáy là sóng SVG mỏng thay cho ellipse
 * trắng 4.5rem trước đây - đó là khoảng trống người xem đọc ra.
 *
 * `critters` tắt được tầng thú/chim: trang Bảng vàng muốn nền tĩnh để pháo
 * hoa ở khu vinh danh là chuyển động duy nhất kéo mắt.
 */
export function FeaturedHero({
  kicker,
  title,
  description,
  critters = true,
}: {
  kicker: string;
  title: string;
  description?: string;
  critters?: boolean;
}) {
  const hillFarRef = useRef<HTMLDivElement>(null);
  const hillMidRef = useRef<HTMLDivElement>(null);
  const hillNearRef = useRef<HTMLDivElement>(null);
  const crittersRef = useRef<HTMLDivElement>(null);
  const birdsRef = useRef<HTMLDivElement>(null);
  const treeRef = useRef<HTMLDivElement>(null);
  const plantRef = useRef<HTMLDivElement>(null);
  const leafRef = useRef<HTMLDivElement>(null);
  const contentRef = useRef<HTMLDivElement>(null);

  useParallaxScrollListener((scrollY) => {
    const y = Math.min(scrollY, MAX_SCROLL_EFFECT);
    const fade = Math.max(0, 1 - y / (MAX_SCROLL_EFFECT * 0.7));

    if (hillFarRef.current) hillFarRef.current.style.transform = `translate3d(0, ${y * SPEED.hillFar}px, 0)`;
    if (hillMidRef.current) hillMidRef.current.style.transform = `translate3d(0, ${y * SPEED.hillMid}px, 0)`;
    if (hillNearRef.current)
      hillNearRef.current.style.transform = `translate3d(0, ${y * SPEED.hillNear}px, 0) scale(${1 + y / 3600})`;
    if (crittersRef.current) {
      crittersRef.current.style.transform = `translate3d(0, ${y * SPEED.tree}px, 0)`;
      crittersRef.current.style.opacity = String(fade);
    }
    if (birdsRef.current) {
      birdsRef.current.style.transform = `translate3d(0, ${y * SPEED.hillMid}px, 0)`;
      birdsRef.current.style.opacity = String(fade);
    }
    if (treeRef.current) {
      treeRef.current.style.transform = `translate3d(0, ${y * SPEED.tree}px, 0)`;
      treeRef.current.style.opacity = String(fade);
    }
    if (plantRef.current) {
      plantRef.current.style.transform = `translate3d(0, ${y * SPEED.plant}px, 0)`;
      plantRef.current.style.opacity = String(fade);
    }
    if (leafRef.current) {
      leafRef.current.style.transform = `translate3d(0, ${y * SPEED.leaf}px, 0)`;
      leafRef.current.style.opacity = String(fade);
    }
    if (contentRef.current) contentRef.current.style.opacity = String(Math.max(0, 1 - y / MAX_SCROLL_EFFECT));
  });

  return (
    <header className="featured-hero">
      <div className="hero-parallax-hills" aria-hidden>
        <div className="hero-parallax-layer" ref={hillFarRef}>
          <HeroHillFar />
        </div>
        <div className="hero-parallax-layer" ref={hillMidRef}>
          <HeroHillMid />
        </div>
        <div className="hero-parallax-layer" ref={hillNearRef}>
          <HeroHillNear />
        </div>
      </div>

      <div className="hero-decor-layer" ref={treeRef}>
        <HeroTree />
      </div>
      <div className="hero-decor-layer" ref={plantRef}>
        <HeroPlant />
      </div>
      <div className="hero-decor-layer" ref={leafRef}>
        <HeroLeaf />
      </div>
      {critters && (
        <>
          <div className="hero-decor-layer featured-hero-birds" ref={birdsRef} aria-hidden>
            <div className="hero-bird-container hero-bird-container--1">
              <div className="hero-bird" />
            </div>
            <div className="hero-bird-container hero-bird-container--2">
              <div className="hero-bird" />
            </div>
          </div>
          <div className="hero-decor-layer featured-hero-critters" ref={crittersRef}>
            <HeroCritters />
          </div>
        </>
      )}

      <div className="featured-hero-content" ref={contentRef}>
        <motion.div
          className="featured-hero-card"
          custom={0}
          variants={fadeUp}
          initial="hidden"
          animate="show"
        >
          <motion.span className="section-kicker" custom={0} variants={fadeUp} initial="hidden" animate="show">
            {kicker}
          </motion.span>
          <motion.h1 custom={1} variants={fadeUp} initial="hidden" animate="show">
            {title}
          </motion.h1>
          {description && (
            <motion.p className="page-heading-sub" custom={2} variants={fadeUp} initial="hidden" animate="show">
              {description}
            </motion.p>
          )}
        </motion.div>
      </div>

      <div className="featured-hero-wave" aria-hidden>
        <svg viewBox="0 0 1440 64" preserveAspectRatio="none" xmlns="http://www.w3.org/2000/svg">
          <path d="M0,28 C160,56 320,8 480,24 C640,40 800,4 960,22 C1120,40 1280,12 1440,30 L1440,64 L0,64 Z" />
        </svg>
      </div>
    </header>
  );
}
