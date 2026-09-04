import { motion } from "framer-motion";
import { useEffect, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { PageHero } from "../../components/public/PageHero";
import { PublicLightbox } from "../../components/public/PublicLightbox";
import { RegionTabs, type Region } from "../../components/public/RegionTabs";
import type { ArtworkWithMeta } from "../../lib/artworkApi";
import { fetchFeaturedArtworks } from "../../lib/publicApi";

const VALID_REGIONS: Region[] = ["all", "saigon", "cantho", "vungtau"];

/**
 * Trang /trien-lam/tac-pham-tieu-bieu - tab theo khu vực + lưới gallery.
 * Khu vực khởi tạo đọc từ query "?khu-vuc=" (điều hướng từ Hero ở trang
 * chủ); đổi tab cập nhật lại query để có thể chia sẻ/back-forward được.
 */
export default function FeaturedArtworksPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const initialRegion = (searchParams.get("khu-vuc") as Region) ?? "all";
  const [region, setRegion] = useState<Region>(VALID_REGIONS.includes(initialRegion) ? initialRegion : "all");
  const [items, setItems] = useState<ArtworkWithMeta[]>([]);
  const [loading, setLoading] = useState(true);
  const [activeIndex, setActiveIndex] = useState<number | null>(null);

  function handleChangeRegion(next: Region) {
    setRegion(next);
    setSearchParams(next === "all" ? {} : { "khu-vuc": next });
  }

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
    <>
      <PageHero
        kicker="Vinh danh"
        title="Tác phẩm tiêu biểu"
        description="Những tác phẩm nổi bật được tuyển chọn từ khắp các cơ sở VAS."
      />
      <section className="featured-gallery-section public-page-section">
        <RegionTabs value={region} onChange={handleChangeRegion} />

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
                animate={{ opacity: 1, y: 0 }}
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
          <PublicLightbox
            items={items}
            activeIndex={activeIndex}
            onNavigate={setActiveIndex}
            onClose={() => setActiveIndex(null)}
          />
        )}
      </section>
    </>
  );
}
