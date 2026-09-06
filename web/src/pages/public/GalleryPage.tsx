import { useCallback, useEffect, useRef, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { FeaturedGardenScene } from "../../components/public/FeaturedGardenScene";
import { FeaturedHero } from "../../components/public/FeaturedHero";
import { GalleryLevelSection } from "../../components/public/GalleryLevelSection";
import { GalleryTopicSection } from "../../components/public/GalleryTopicSection";
import { GallerySearch } from "../../components/public/GallerySearch";
import { PublicLightbox } from "../../components/public/PublicLightbox";
import type { ArtworkWithMeta, GradeLevel } from "../../lib/artworkApi";
import { fetchGradeLevels } from "../../lib/artworkApi";
import { fetchTopicCategories, type TopicCategory } from "../../lib/topicCategoryApi";

type EduLevel = "primary" | "secondary";

const LEVELS: { key: EduLevel; title: string; subtitle: string }[] = [
  {
    key: "primary",
    title: "Khối Tiểu học",
    subtitle: "Lớp 1 đến Lớp 5 — nơi màu nào cũng đúng và trí tưởng tượng chưa từng biết sợ",
  },
  {
    key: "secondary",
    title: "Khối Trung học",
    subtitle: "Lớp 6 đến Lớp 12 — những nét vẽ đã biết mình muốn nói điều gì",
  },
];

/**
 * Trang /phong-trien-lam — đúng 2 section (Tiểu học / Trung học).
 * Mỗi section: tiêu đề, node chọn lớp, slide tranh cuộn ngang.
 * Query "?tim=" lọc theo tên tranh / học sinh, chia sẻ được.
 */
export default function GalleryPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const sharedArtworkId = Number(searchParams.get("tranh"));
  const searchQuery = (searchParams.get("tim") ?? "").trim();
  const [searchInput, setSearchInput] = useState(searchQuery);
  const [grades, setGrades] = useState<GradeLevel[]>([]);
  const [selectedGradeId, setSelectedGradeId] = useState<number | null>(null);
  const [topics, setTopics] = useState<TopicCategory[]>([]);
  const [lightbox, setLightbox] = useState<{ items: ArtworkWithMeta[]; index: number } | null>(null);
  const didInitFromUrl = useRef(false);

  useEffect(() => {
    fetchGradeLevels().then(setGrades).catch(() => setGrades([]));
    fetchTopicCategories(false)
      .then((list) => setTopics([...(list ?? [])].sort((a, b) => a.display_order - b.display_order)))
      .catch(() => setTopics([]));
  }, []);

  function patchParams(patch: Record<string, string | null>, options?: { replace?: boolean }) {
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev);
      for (const [key, value] of Object.entries(patch)) {
        if (value === null) next.delete(key);
        else next.set(key, value);
      }
      return next;
    }, { replace: options?.replace ?? false });
  }

  useEffect(() => {
    setSearchInput((current) => (current.trim() === searchQuery ? current : searchQuery));
  }, [searchQuery]);

  useEffect(() => {
    const trimmed = searchInput.trim();
    if (trimmed === searchQuery) return;
    const timer = window.setTimeout(() => {
      setLightbox(null);
      patchParams({ tim: trimmed || null, tranh: null }, { replace: true });
    }, 320);
    return () => window.clearTimeout(timer);
  }, [searchInput, searchQuery]);

  useEffect(() => {
    if (didInitFromUrl.current || grades.length === 0) return;
    didInitFromUrl.current = true;
    const fromUrl = Number(searchParams.get("khoi_lop"));
    if (Number.isInteger(fromUrl) && fromUrl > 0 && grades.some((g) => g.id === fromUrl)) {
      setSelectedGradeId(fromUrl);
    }
    const targetLevel =
      grades.find((g) => g.id === fromUrl)?.education_level ??
      (searchParams.get("khoi") === "secondary" ? "secondary" : "primary");
    window.setTimeout(() => {
      document.getElementById(`khoi-${targetLevel}`)?.scrollIntoView({ behavior: "smooth", block: "start" });
    }, 120);
  }, [grades, searchParams]);

  function handleSelectGrade(gradeId: number | null) {
    setSelectedGradeId(gradeId);
    setLightbox(null);
    const grade = grades.find((g) => g.id === gradeId);
    patchParams({
      khoi: grade?.education_level ?? null,
      khoi_lop: gradeId === null ? null : String(gradeId),
      trang: null,
      tranh: null,
    });
  }

  const handleOpenArtwork = useCallback((items: ArtworkWithMeta[], index: number) => {
    setLightbox({ items, index });
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev);
      next.set("tranh", String(items[index].id));
      return next;
    });
  }, [setSearchParams]);

  function handleCloseArtwork() {
    setLightbox(null);
    patchParams({ tranh: null });
  }

  return (
    <div className="featured-page">
      <FeaturedHero
        kicker="Phòng triển lãm"
        title="Ở đây có tranh của con"
        description="Không một bức nào bị bỏ lại phía sau. Chọn khối, chọn lớp, hoặc gõ tên con vào ô tìm kiếm — bức tranh em gửi về đang được treo ở đây, ngay ngắn như mọi bức tranh khác."
      />

      <section className="featured-gallery-section gallery-explorer-section">
        <FeaturedGardenScene />
        <GallerySearch value={searchInput} onChange={setSearchInput} />

        <div className="gallery-hall">
          {LEVELS.map(({ key, title, subtitle }) => (
            <GalleryLevelSection
              key={key}
              level={key}
              title={title}
              subtitle={subtitle}
              search={searchQuery}
              grades={grades.filter((g) => g.education_level === key)}
              selectedGradeId={
                grades.find((g) => g.id === selectedGradeId)?.education_level === key ? selectedGradeId : null
              }
              onSelectGrade={(gradeId) => {
                if (
                  gradeId === null &&
                  grades.find((g) => g.id === selectedGradeId)?.education_level !== key
                ) {
                  return;
                }
                handleSelectGrade(gradeId);
              }}
              sharedArtworkId={Number.isInteger(sharedArtworkId) ? sharedArtworkId : undefined}
              onOpenArtwork={handleOpenArtwork}
            />
          ))}
        </div>

        {!searchQuery && topics.length > 0 && (
          <div className="gallery-hall gallery-hall--topics">
            {topics.map((topic) => (
              <GalleryTopicSection
                key={topic.id}
                topic={topic}
                sharedArtworkId={Number.isInteger(sharedArtworkId) ? sharedArtworkId : undefined}
                onOpenArtwork={handleOpenArtwork}
              />
            ))}
          </div>
        )}

        {lightbox && (
          <PublicLightbox
            items={lightbox.items}
            activeIndex={lightbox.index}
            onNavigate={(index) => setLightbox((prev) => (prev ? { ...prev, index } : prev))}
            onClose={handleCloseArtwork}
          />
        )}
      </section>
    </div>
  );
}
