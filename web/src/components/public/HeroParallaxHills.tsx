import { webpOf } from "../../lib/staticImage";

/**
 * Lớp "đồi" parallax cho Hero - dùng nguyên bộ ảnh PNG từ mẫu tham chiếu
 * (C:\Users\ASUS\Desktop\theme: hill1-5, tree, plant, leaf), giữ đúng tông
 * xanh lá/ngọc bích gốc của mẫu thay vì vẽ lại theo màu thương hiệu VASchools -
 * theo yêu cầu dùng thẳng asset gốc. File nằm ở
 * public/images/parallax/*.png. Mỗi <img> là một dải đồi/cụm cây kéo dài
 * toàn chiều rộng (hoặc neo 1 góc); layer cha (HeroSection) áp transform
 * theo scroll (xa chậm, gần nhanh) - các component ở đây chỉ render ảnh,
 * không tự animate.
 *
 * Mỗi ảnh bọc <picture> với nguồn WebP trước (nhẹ hơn PNG gốc 66-80%),
 * PNG làm dự phòng cho trình duyệt cũ - cùng mẫu artworkImage.ts dùng cho
 * ảnh tác phẩm, nhưng ở đây không cần helper JS vì chỉ có 1 cỡ duy nhất.
 */

/** Lớp xa nhất - dãy núi nhiều tầng nhạt/đậm lồng sẵn trong 1 ảnh, phủ
 * toàn chiều rộng, làm hậu cảnh mờ ở chân trời. */
export function HeroHillFar() {
  const src = "/images/parallax/hill1.png";
  return (
    <picture>
      <source srcSet={webpOf(src)} type="image/webp" />
      <img className="hero-hill hero-hill--far" src={src} alt="" aria-hidden />
    </picture>
  );
}

/** Lớp giữa - đồi đơn với cụm cây nhỏ trên đỉnh, đứng lệch phải. */
export function HeroHillMid() {
  const src = "/images/parallax/hill3.png";
  return (
    <picture>
      <source srcSet={webpOf(src)} type="image/webp" />
      <img className="hero-hill hero-hill--mid" src={src} alt="" aria-hidden />
    </picture>
  );
}

/** Lớp gần nhất - đồi đậm với cây cọ + bụi cây trên đỉnh, đứng lệch trái,
 * chân đồi chạm đáy Hero, làm nền cho mascot/nội dung đứng trước. */
export function HeroHillNear() {
  const src = "/images/parallax/hill4.png";
  return (
    <picture>
      <source srcSet={webpOf(src)} type="image/webp" />
      <img className="hero-hill hero-hill--near" src={src} alt="" aria-hidden />
    </picture>
  );
}

/** Cây cọ đơn lẻ - phần tử tiền cảnh rời rạc, neo góc trái đáy Hero. */
export function HeroTree() {
  const src = "/images/parallax/tree.png";
  return (
    <picture>
      <source srcSet={webpOf(src)} type="image/webp" />
      <img className="hero-parallax-tree" src={src} alt="" aria-hidden />
    </picture>
  );
}

/** Dải cỏ/dương xỉ - viền đáy Hero, phần tử tiền cảnh gần nhất. */
export function HeroPlant() {
  const src = "/images/parallax/plant.png";
  return (
    <picture>
      <source srcSet={webpOf(src)} type="image/webp" />
      <img className="hero-parallax-plant" src={src} alt="" aria-hidden />
    </picture>
  );
}

/** Lá dương xỉ lớn - góc trên, phần tử tiền cảnh rơi vào khung hình. */
export function HeroLeaf() {
  const src = "/images/parallax/leaf.png";
  return (
    <picture>
      <source srcSet={webpOf(src)} type="image/webp" />
      <img className="hero-parallax-leaf" src={src} alt="" aria-hidden />
    </picture>
  );
}

/** Bộ 3 con thú rừng (sóc, voi, thỏ - SVG đơn sắc) chạy dọc theo gờ đồi
 * tiền cảnh, mỗi con một đường chạy/tốc độ riêng (CSS animation, xem
 * public.css) để trông như thú tự nhiên băng qua khung hình chứ không di
 * chuyển đồng bộ. Đứng trong .hero-decor-layer riêng (không lồng theo lớp
 * flora) vì animation "chạy" (translateX lặp vô hạn) khác hẳn sway tại chỗ. */
export function HeroCritters() {
  return (
    <>
      <img className="hero-critter hero-critter--squirrel" src="/images/parallax/squirrel.svg" alt="" aria-hidden />
      <img className="hero-critter hero-critter--rabbit" src="/images/parallax/rabbit.svg" alt="" aria-hidden />
      <img className="hero-critter hero-critter--elephant" src="/images/parallax/ele.svg" alt="" aria-hidden />
    </>
  );
}
