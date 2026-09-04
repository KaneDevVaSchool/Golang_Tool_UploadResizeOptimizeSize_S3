import { AnimatePresence, motion } from "framer-motion";
import { forwardRef, useEffect, useState } from "react";
import { fetchGradeLevels, type GradeLevel } from "../../lib/artworkApi";
import { fetchPublicArtworks } from "../../lib/publicApi";
import type { ArtworkWithMeta } from "../../lib/artworkApi";
import { GradeNode } from "./GradeNode";
import { PublicLightbox } from "./PublicLightbox";

/**
 * Section 4 - Phòng triển lãm theo khối: lưới node cho từng khối lớp (1-5
 * hoặc 6-12 tuỳ level đang chọn), click 1 node -> load tác phẩm khối đó ->
 * hiện lưới -> click ảnh mở PublicLightbox.
 */
export const GradeLevelExplorer = forwardRef<HTMLElement, { level: "primary" | "secondary" | null }>(
  function GradeLevelExplorer({ level }, ref) {
    const [grades, setGrades] = useState<GradeLevel[]>([]);
    const [selectedGradeId, setSelectedGradeId] = useState<number | null>(null);
    const [items, setItems] = useState<ArtworkWithMeta[]>([]);
    const [loading, setLoading] = useState(false);
    const [activeIndex, setActiveIndex] = useState<number | null>(null);

    useEffect(() => {
      fetchGradeLevels().then(setGrades).catch(() => setGrades([]));
    }, []);

    useEffect(() => {
      setSelectedGradeId(null);
      setItems([]);
    }, [level]);

    const filteredGrades = grades.filter((g) => g.education_level === level);

    useEffect(() => {
      if (!selectedGradeId) return;
      const controller = new AbortController();
      setLoading(true);
      fetchPublicArtworks({ grade_level_id: selectedGradeId, page_size: 60 }, controller.signal)
        .then((res) => setItems(res.items ?? []))
        .catch(() => {
          if (!controller.signal.aborted) setItems([]);
        })
        .finally(() => {
          if (!controller.signal.aborted) setLoading(false);
        });
      return () => controller.abort();
    }, [selectedGradeId]);

    if (!level) return null;

    return (
      <section className="grade-explorer-section" ref={ref}>
        <motion.div
          className="section-heading"
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true, margin: "-80px" }}
          transition={{ duration: 0.5 }}
        >
          <span className="section-kicker">Phòng triển lãm</span>
          <h2>{level === "primary" ? "Tác phẩm khối Tiểu học" : "Tác phẩm khối Trung học"}</h2>
        </motion.div>

        <div className="grade-node-grid">
          {filteredGrades.map((g) => (
            <GradeNode
              key={g.id}
              label={g.label}
              active={g.id === selectedGradeId}
              onClick={() => setSelectedGradeId(g.id === selectedGradeId ? null : g.id)}
            />
          ))}
        </div>

        <AnimatePresence mode="wait">
          {selectedGradeId && (
            <motion.div
              key={selectedGradeId}
              initial={{ opacity: 0, y: 12 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -8 }}
              transition={{ duration: 0.3 }}
              className="grade-artworks-wrap"
            >
              {loading ? (
                <p className="admin-empty-note">Đang tải…</p>
              ) : items.length === 0 ? (
                <p className="admin-empty-note">Chưa có tác phẩm nào cho khối này.</p>
              ) : (
                <div className="featured-gallery-grid">
                  {items.map((item, index) => (
                    <button
                      key={item.id}
                      type="button"
                      className="featured-gallery-item"
                      onClick={() => setActiveIndex(index)}
                    >
                      <img src={item.thumbnail_url || item.image_url} alt={item.title} loading="lazy" />
                      <span className="featured-gallery-overlay">
                        <strong>{item.title}</strong>
                        <span>{item.student_name}</span>
                      </span>
                    </button>
                  ))}
                </div>
              )}
            </motion.div>
          )}
        </AnimatePresence>

        {activeIndex !== null && (
          <PublicLightbox items={items} activeIndex={activeIndex} onNavigate={setActiveIndex} onClose={() => setActiveIndex(null)} />
        )}
      </section>
    );
  },
);
