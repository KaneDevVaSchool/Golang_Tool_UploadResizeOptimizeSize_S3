import { AnimatePresence, motion } from "framer-motion";
import { useEffect, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { GradeNode } from "../../components/public/GradeNode";
import { PageHero } from "../../components/public/PageHero";
import { PublicLightbox } from "../../components/public/PublicLightbox";
import { SecondaryStarfield } from "../../components/public/SecondaryStarfield";
import type { ArtworkWithMeta } from "../../lib/artworkApi";
import { fetchGradeLevels, type GradeLevel } from "../../lib/artworkApi";
import { fetchPublicArtworks } from "../../lib/publicApi";

type EduLevel = "primary" | "secondary";

const LEVEL_TABS: { key: EduLevel; label: string }[] = [
  { key: "primary", label: "Khối Tiểu học" },
  { key: "secondary", label: "Khối Trung học" },
];

/**
 * Trang /trien-lam/phong-trien-lam - chọn cấp học (tab) rồi chọn khối lớp
 * (node) để xem tác phẩm. Cấp học khởi tạo đọc từ query "?khoi=" (điều
 * hướng từ Gate ở trang chủ), mặc định "primary".
 */
export default function GalleryPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const initialLevel = searchParams.get("khoi") === "secondary" ? "secondary" : "primary";
  const [level, setLevel] = useState<EduLevel>(initialLevel);
  const [grades, setGrades] = useState<GradeLevel[]>([]);
  const [selectedGradeId, setSelectedGradeId] = useState<number | null>(null);
  const [items, setItems] = useState<ArtworkWithMeta[]>([]);
  const [loading, setLoading] = useState(false);
  const [activeIndex, setActiveIndex] = useState<number | null>(null);

  useEffect(() => {
    fetchGradeLevels().then(setGrades).catch(() => setGrades([]));
  }, []);

  function handleChangeLevel(next: EduLevel) {
    setLevel(next);
    setSearchParams({ khoi: next });
    setSelectedGradeId(null);
    setItems([]);
  }

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

  const isSecondary = level === "secondary";

  return (
    <>
      <PageHero
        kicker="Phòng triển lãm"
        title="Tác phẩm theo khối lớp"
        description="Chọn cấp học rồi chọn khối lớp để xem tác phẩm học sinh."
        variant={isSecondary ? "dark" : "light"}
      />
      <section
        className={`grade-explorer-section public-page-section${
          isSecondary ? " grade-explorer-section--secondary" : ""
        }`}
      >
        {isSecondary && (
          <div className="grade-explorer-decor" aria-hidden>
            <img className="grade-explorer-hill" src="/images/parallax/hill2.png" alt="" />
            <SecondaryStarfield />
          </div>
        )}
        <div className="level-tabs" role="tablist" aria-label="Chọn cấp học">
          {LEVEL_TABS.map((tab) => (
            <button
              key={tab.key}
              type="button"
              role="tab"
              aria-selected={level === tab.key}
              className={`level-tab${level === tab.key ? " level-tab--active" : ""}`}
              onClick={() => handleChangeLevel(tab.key)}
            >
              {tab.label}
            </button>
          ))}
        </div>

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
