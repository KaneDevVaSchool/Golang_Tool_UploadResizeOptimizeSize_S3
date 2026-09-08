import { motion } from "framer-motion";
import { useRef } from "react";
import { HeroCritters, HeroHillFar, HeroHillMid, HeroHillNear, HeroLeaf, HeroPlant, HeroTree } from "./HeroParallaxHills";
import { useParallaxScrollListener } from "../../hooks/useParallaxScroll";
import { fadeUp } from "../../lib/motionPresets";
import { webpOf } from "../../lib/staticImage";
import type { Region } from "./RegionTabs";

const REGION_BUTTONS: { key: Region; label: string; icon: string }[] = [
  { key: "saigon", label: "Sài Gòn", icon: "🏙️" },
  { key: "cantho", label: "Cần Thơ", icon: "🌾" },
  { key: "vungtau", label: "Vũng Tàu", icon: "🌊" },
];

// Hệ số tốc độ mỗi lớp - lớp "xa" di chuyển chậm hơn scroll thật (< 1),
// lớp "gần" di chuyển nhanh hơn (gợi cảm giác trôi qua trước mắt), giống
// cơ chế multi-layer parallax cổ điển (mỗi ảnh nền một background-position
// riêng). Ở đây dùng transform: translateY áp trực tiếp cho từng layer.
const SPEED = {
  hillFar: 0.08,
  hillMid: 0.16,
  hillNear: 0.26,
  flora: 0.34,
  tree: 0.3,
  plant: 0.44,
  leaf: 0.5,
  content: 0.4,
};

// Quá ngưỡng này (px cuộn), nội dung Hero coi như đã cuộn qua - dừng dịch
// chuyển/fade thêm để không "âm" khi người dùng cuộn rất sâu xuống các
// section khác (lúc đó Hero đã ra khỏi viewport từ lâu).
const MAX_SCROLL_EFFECT = 640;

/**
 * Section 1 - Hero, dạng parallax scroll nhiều lớp (lấy cảm hứng từ mẫu
 * layered-hill parallax cổ điển): các lớp trang trí (đồi, cây/cỏ/lá, hoa
 * lá) và khối nội dung text đều dịch chuyển theo scrollY với tốc độ khác
 * nhau - lớp càng "xa" (đồi mờ ở nền) di chuyển càng chậm, lớp càng "gần"
 * (cây, lá) di chuyển càng nhanh và mờ dần, tạo cảm giác chiều sâu khi
 * cuộn qua Hero. Toàn bộ hiệu ứng tôn trọng prefers-reduced-motion (tắt
 * qua CSS, xem public.css).
 */
export function HeroSection({ onSelectRegion }: { onSelectRegion: (region: Region) => void }) {
  // Mỗi layer parallax giữ 1 ref DOM - vị trí cuộn được ghi thẳng vào
  // transform/opacity của các node này trong callback rAF (xem
  // useParallaxScrollListener bên dưới) thay vì đi qua React state, để
  // tránh setState + re-render toàn bộ HeroSection ở tần suất tới 60
  // lần/giây trong lúc cuộn - chỉ 1 layout/paint trực tiếp trên DOM node
  // liên quan, giống cách các thư viện scroll-parallax hiệu năng cao vẫn làm.
  const hillFarRef = useRef<HTMLDivElement>(null);
  const hillMidRef = useRef<HTMLDivElement>(null);
  const hillNearRef = useRef<HTMLDivElement>(null);
  const crittersRef = useRef<HTMLDivElement>(null);
  const treeRef = useRef<HTMLDivElement>(null);
  const plantRef = useRef<HTMLDivElement>(null);
  const leafRef = useRef<HTMLDivElement>(null);
  const floraRef = useRef<HTMLDivElement>(null);
  const butterflyRef = useRef<HTMLDivElement>(null);
  const birdRef = useRef<HTMLDivElement>(null);
  const petalRef = useRef<HTMLDivElement>(null);
  const contentRef = useRef<HTMLDivElement>(null);
  const scrollHintRef = useRef<HTMLDivElement>(null);

  useParallaxScrollListener((scrollY) => {
    const y = Math.min(scrollY, MAX_SCROLL_EFFECT);

    // Độ mờ dần khi cuộn: nội dung/nhân vật tan biến trước khi Hero cuộn
    // hết hẳn, giống hiệu ứng "text biến mất khi cuộn" của mẫu tham chiếu.
    const contentFade = Math.max(0, 1 - y / (MAX_SCROLL_EFFECT * 0.7));
    const heroFade = Math.max(0, 1 - y / MAX_SCROLL_EFFECT);
    // Tiền cảnh (cây/cỏ/lá - SPEED.tree/plant/leaf) trôi nhanh hơn hẳn các
    // lớp khác (0.3-0.5 so với 0.08-0.26 của đồi) nên tại cùng mốc scroll,
    // chúng đã dịch chuyển ra xa khỏi vị trí neo hơn nhiều - nếu dùng chung
    // heroFade (chỉ về 0 khi cuộn hết MAX_SCROLL_EFFECT), có một khoảng
    // scroll giữa chừng mà chúng vẫn còn hiện rõ nhưng đã trôi vượt khỏi
    // .hero-section (overflow:hidden cắt theo khung Hero, không theo
    // layer), lộ ra thành một dải ảnh "kẹt" ngay trước section kế tiếp. Cho
    // tiền cảnh mờ hẳn ở nửa đầu quãng cuộn (45%) để luôn biến mất trước
    // khi kịp trôi ra ngoài khung nhìn của chính Hero.
    const foregroundFade = Math.max(0, 1 - y / (MAX_SCROLL_EFFECT * 0.45));

    if (hillFarRef.current) hillFarRef.current.style.transform = `translate3d(0, ${y * SPEED.hillFar}px, 0)`;
    if (hillMidRef.current) hillMidRef.current.style.transform = `translate3d(0, ${y * SPEED.hillMid}px, 0)`;
    if (hillNearRef.current)
      hillNearRef.current.style.transform = `translate3d(0, ${y * SPEED.hillNear}px, 0) scale(${1 + y / 4000})`;

    if (crittersRef.current) {
      crittersRef.current.style.transform = `translate3d(0, ${y * SPEED.tree}px, 0)`;
      crittersRef.current.style.opacity = String(foregroundFade);
    }
    if (treeRef.current) {
      treeRef.current.style.transform = `translate3d(0, ${y * SPEED.tree}px, 0)`;
      treeRef.current.style.opacity = String(foregroundFade);
    }
    if (plantRef.current) {
      plantRef.current.style.transform = `translate3d(0, ${y * SPEED.plant}px, 0)`;
      plantRef.current.style.opacity = String(foregroundFade);
    }
    if (leafRef.current) {
      leafRef.current.style.transform = `translate3d(0, ${y * SPEED.leaf}px, 0)`;
      leafRef.current.style.opacity = String(foregroundFade);
    }
    if (floraRef.current) {
      floraRef.current.style.transform = `translate3d(0, ${y * SPEED.flora}px, 0)`;
      Array.from(floraRef.current.children).forEach((child) => {
        (child as HTMLElement).style.opacity = String(0.55 * heroFade);
      });
    }

    if (butterflyRef.current) butterflyRef.current.style.opacity = String(heroFade);
    if (birdRef.current) birdRef.current.style.opacity = String(heroFade);
    if (petalRef.current) petalRef.current.style.opacity = String(heroFade);
    // hero-scroll-hint dùng motion.div với animate={{ y: [...] }} riêng -
    // Framer Motion ghi transform qua style của chính nó mỗi frame của
    // animation lặp đó; set opacity trực tiếp qua CSS custom property (đọc
    // bởi .hero-scroll-hint trong public.css) thay vì .style.opacity để
    // không phụ thuộc thứ tự ghi giữa 2 nguồn cùng set style trên 1 node.
    if (scrollHintRef.current) scrollHintRef.current.style.setProperty("--scroll-hint-fade", String(heroFade));

    if (contentRef.current) {
      contentRef.current.style.transform = `translate3d(0, ${y * SPEED.content}px, 0)`;
      contentRef.current.style.opacity = String(contentFade);
    }
  });

  return (
    <section className="hero-section">
      {/* Lớp đồi vector - xa đến gần, mỗi lớp một tốc độ trôi riêng */}
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

      {/* Bộ thú rừng (sóc/thỏ/voi) chạy dọc gờ đồi tiền cảnh - đường chạy
          riêng của từng con (translateX lặp vô hạn, xem .hero-critter--*
          trong public.css) cắm trực tiếp vào transform của chính nó nên
          cũng cần div "giá đỡ" riêng chỉ lo dịch chuyển theo scroll, giống
          cơ chế đã dùng cho .hero-decor-layer--flora bên dưới. */}
      <div className="hero-decor-layer hero-decor-layer--critters" ref={crittersRef}>
        <HeroCritters />
      </div>

      {/* Cây/cỏ/lá tiền cảnh (ảnh thật từ mẫu tham chiếu) - trôi nhanh nhất
          trong các lớp trang trí vì đứng gần "máy quay" nhất. */}
      <div className="hero-decor-layer hero-decor-layer--tree" ref={treeRef}>
        <HeroTree />
      </div>
      <div className="hero-decor-layer hero-decor-layer--plant" ref={plantRef}>
        <HeroPlant />
      </div>
      <div className="hero-decor-layer hero-decor-layer--leaf" ref={leafRef}>
        <HeroLeaf />
      </div>

      {/* Mỗi phần tử trang trí có animation CSS/framer riêng (sway, scale,
          fade) cắm trực tiếp vào transform của chính nó - nên translate
          parallax không thể ghi đè lên cùng thuộc tính đó mà phải bọc
          trong 1 div "giá đỡ" riêng chỉ lo việc dịch chuyển theo scroll,
          để phần tử con tự do animate transform/opacity của nó. */}
      <div className="hero-decor-layer hero-decor-layer--flora" ref={floraRef}>
        <span className="hero-flora hero-flora--1" aria-hidden>
          🌸
        </span>
        <span className="hero-flora hero-flora--2" aria-hidden>
          🍃
        </span>
        <span className="hero-flora hero-flora--3" aria-hidden>
          🌼
        </span>
      </div>

      {/* Bướm bay quỹ đạo hình số 8 - đường bay riêng (hero-butterfly-fly),
          tách khỏi .hero-flora vì cần animation transform phức tạp hơn
          (translate theo path + xoay cánh) không hợp với sway đơn giản. */}
      <div className="hero-decor-layer" ref={butterflyRef}>
        <span className="hero-butterfly hero-butterfly--1" aria-hidden>
          🦋
        </span>
        <span className="hero-butterfly hero-butterfly--2" aria-hidden>
          🦋
        </span>
      </div>

      {/* Đàn chim bay ngang qua trời - sprite sheet vỗ cánh thật (không
          phải emoji): mỗi .hero-bird-container lo đường bay (translate
          theo %viewport + scale phối cảnh), .hero-bird bên trong lo vỗ
          cánh (background-position chạy qua steps() trên sprite 10 khung),
          tách 2 animation vì khác thuộc tính/chu kỳ. So le --fly-delay để
          trông như một đàn tự nhiên thay vì 3 bản sao đồng bộ. */}
      <div className="hero-decor-layer" ref={birdRef} aria-hidden>
        <div className="hero-bird-container hero-bird-container--1">
          <div className="hero-bird" />
        </div>
        <div className="hero-bird-container hero-bird-container--2">
          <div className="hero-bird" />
        </div>
        <div className="hero-bird-container hero-bird-container--3">
          <div className="hero-bird" />
        </div>
      </div>

      {/* Hoa rơi nhẹ nhàng từ trên xuống - khác với .hero-flora (trôi nổi
          tại chỗ): đường rơi thẳng có lắc ngang + xoay, lặp vô hạn, so le
          delay để không rơi thành hàng đồng loạt. */}
      <div className="hero-decor-layer" ref={petalRef}>
        <span className="hero-petal hero-petal--1" aria-hidden>
          🌸
        </span>
        <span className="hero-petal hero-petal--2" aria-hidden>
          🌺
        </span>
        <span className="hero-petal hero-petal--3" aria-hidden>
          🌸
        </span>
      </div>

      <div className="hero-content" ref={contentRef}>
        <picture>
          <source srcSet={webpOf("/images/vas-wordmark-stacked.png")} type="image/webp" />
          <motion.img
            className="hero-wordmark"
            src="/images/vas-wordmark-stacked.png"
            alt="VA Schools"
            custom={0}
            variants={fadeUp}
            initial="hidden"
            animate="show"
          />
        </picture>
        {/* Huy hiệu kỷ niệm - thay dòng kicker chữ hoa phẳng bằng một khối
            có viền/nền riêng: mốc "20" được tách ra làm con số lớn để mắt
            có một điểm dừng trước khi vào tiêu đề, thay vì 6 khối chữ cùng
            cỡ xếp dọc đều nhau. */}
        <motion.p className="hero-kicker" custom={1} variants={fadeUp} initial="hidden" animate="show">
          <span className="hero-kicker-num">20</span>
          <span className="hero-kicker-text">
            Năm thành lập
            <span className="hero-kicker-years">2006 – 2026</span>
          </span>
        </motion.p>
        {/* Tiêu đề tách 2 dòng: dòng dẫn nhỏ hơn, dòng nhấn lớn - tạo bậc
            thang cỡ chữ trong cùng một tiêu đề thay vì một khối đồng cỡ. */}
        <motion.h1 className="hero-title" custom={2} variants={fadeUp} initial="hidden" animate="show">
          <span className="hero-title-lead">20 Năm Trường Việt Mỹ</span>
          <span className="hero-title-main">Của Em</span>
        </motion.h1>
        <motion.p className="hero-subtitle" custom={3} variants={fadeUp} initial="hidden" animate="show">
          Hai mươi mùa tựu trường, hai mươi mùa phượng nở. Bao thế hệ học trò đã lớn lên dưới mái trường này,
          mang theo những kỷ niệm chẳng lời nào tả hết. Năm nay, các em kể lại câu chuyện ấy theo cách của
          riêng mình — bằng nét cọ và sắc màu tuổi thơ.
        </motion.p>

        <motion.div className="hero-region-cta" custom={4} variants={fadeUp} initial="hidden" animate="show">
          {/* Nhãn nằm giữa 2 đường kẻ mảnh (::before/::after) - vạch một
              đường ngang cắt qua cột nội dung, tách phần "mời xem tranh"
              khỏi phần giới thiệu phía trên. */}
          <span className="hero-region-label">Mời bạn ghé thăm phòng tranh của từng cơ sở</span>
          <div className="hero-region-buttons">
            {REGION_BUTTONS.map((r) => (
              <motion.button
                key={r.key}
                type="button"
                className={`hero-region-btn hero-region-btn--${r.key}`}
                onClick={() => onSelectRegion(r.key)}
                whileHover={{ y: -4, scale: 1.03 }}
                whileTap={{ scale: 0.97 }}
              >
                <span className="hero-region-btn-icon" aria-hidden>
                  {r.icon}
                </span>
                <span className="hero-region-btn-label">{r.label}</span>
              </motion.button>
            ))}
          </div>
        </motion.div>
      </div>

      <motion.div
        className="hero-scroll-hint"
        ref={scrollHintRef}
        animate={{ y: [0, 8, 0] }}
        transition={{ duration: 1.8, repeat: Infinity, ease: "easeInOut" }}
        aria-hidden
      >
        ↓
      </motion.div>
    </section>
  );
}
