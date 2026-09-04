import { motion } from "framer-motion";
import { forwardRef, useEffect, useState } from "react";
import { fetchFeaturedArtworks } from "../../lib/publicApi";
import type { ArtworkWithMeta } from "../../lib/artworkApi";
import { RegionTabs, type Region } from "./RegionTabs";
import { PublicLightbox } from "./PublicLightbox";

/**
 * Section 3 - Tác phẩm tiêu biểu: tab theo khu vực, gallery hover-overlay,
 * click mở PublicLightbox.
 */
export const FeaturedGallery = forwardRef<HTMLElement, { initialRegion?: Region }>(function FeaturedGallery(
  { initialRegion = "all" },
  ref,
) {
  const [region, setRegion] = useState<Region>(initialRegion);
  const [items, setItems] = useState<ArtworkWithMeta[]>([]);
  const [loading, setLoading] = useState(true);
  const [activeIndex, setActiveIndex] = useState<number | null>(null);

  useEffect(() => {
    setRegion(initialRegion);
  }, [initialRegion]);

  useEffect(() => {
    const controller = new AbortController();
    setLoading(true);
    fetchFeaturedArtworks(region === "all" ? undefined : region, controller.signal)
      .then((res) => setItems(res.items ?? []))
      .catch(() => {
        if (!controller.signal.aborted) setItems([]);
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false);
      });
    return () => controller.abort();
  }, [region]);

  return (
    <section className="featured-gallery-section" ref={ref}>
      <motion.div
        className="section-heading"
        initial={{ opacity: 0, y: 20 }}
        whileInView={{ opacity: 1, y: 0 }}
        viewport={{ once: true, margin: "-80px" }}
        transition={{ duration: 0.5 }}
      >
        <span className="section-kicker">Vinh danh</span>
        <h2>Tác phẩm tiêu biểu</h2>
      </motion.div>

      <RegionTabs value={region} onChange={setRegion} />

      {loading ? (
        <p className="admin-empty-note">Đang tải…</p>
      ) : items.length === 0 ? (
        <p className="admin-empty-note">Chưa có tác phẩm tiêu biểu nào ở khu vực này.</p>
      ) : (
        <div className="featured-gallery-grid">
          {items.map((item, index) => (
            <motion.button
              key={item.id}
              type="button"
              className="featured-gallery-item"
              onClick={() => setActiveIndex(index)}
              initial={{ opacity: 0, y: 16 }}
              whileInView={{ opacity: 1, y: 0 }}
              viewport={{ once: true, margin: "-40px" }}
              transition={{ duration: 0.35, delay: Math.min(index, 8) * 0.04 }}
              whileHover={{ scale: 1.03 }}
            >
              <img src={item.thumbnail_url || item.image_url} alt={item.title} loading="lazy" />
              <span className="featured-gallery-overlay">
                <strong>{item.title}</strong>
                <span>{item.student_name}</span>
              </span>
            </motion.button>
          ))}
        </div>
      )}

      {activeIndex !== null && (
        <PublicLightbox items={items} activeIndex={activeIndex} onNavigate={setActiveIndex} onClose={() => setActiveIndex(null)} />
      )}
    </section>
  );
});
