import { useEffect, useRef, useState } from "react";
import type { ArtworkWithMeta, GradeLevel } from "../../lib/artworkApi";
import { fetchPublicArtworks } from "../../lib/publicApi";
import { ArtworkRail } from "./ArtworkRail";
import { GradeNode } from "./GradeNode";

const PAGE_SIZE = 36;

type EduLevel = "primary" | "secondary";

/**
 * Một cấp học = một section: tiêu đề, node chọn lớp, slide tranh ngang.
 */
export function GalleryLevelSection({
  level,
  title,
  subtitle,
  search = "",
  grades,
  selectedGradeId,
  onSelectGrade,
  sharedArtworkId,
  onOpenArtwork,
}: {
  level: EduLevel;
  title: string;
  /** Dòng giới thiệu ngắn dưới tiêu đề khối - ẩn khi đang lọc/tìm, lúc đó
   *  dòng đếm kết quả mới là thông tin người dùng cần. */
  subtitle?: string;
  search?: string;
  grades: GradeLevel[];
  selectedGradeId: number | null;
  onSelectGrade: (gradeId: number | null) => void;
  sharedArtworkId?: number;
  onOpenArtwork: (items: ArtworkWithMeta[], index: number) => void;
}) {
  const [items, setItems] = useState<ArtworkWithMeta[]>([]);
  const [totalCount, setTotalCount] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);
  const selectedGrade = grades.find((g) => g.id === selectedGradeId) ?? null;

  function buildFilter(nextPage: number) {
    return {
      ...(selectedGradeId ? { grade_level_id: selectedGradeId } : { education_level: level }),
      ...(search ? { search } : {}),
      page: nextPage,
      page_size: PAGE_SIZE,
    };
  }

  useEffect(() => {
    const controller = new AbortController();
    setLoading(true);
    fetchPublicArtworks(buildFilter(1), controller.signal)
      .then((res) => {
        setItems(res.items ?? []);
        setTotalCount(res.total_count ?? 0);
        setPage(1);
      })
      .catch(() => {
        if (!controller.signal.aborted) {
          setItems([]);
          setTotalCount(0);
        }
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false);
      });
    return () => controller.abort();
  }, [level, selectedGradeId, search]);

  const openedShareRef = useRef<number | null>(null);
  useEffect(() => {
    if (loading || !sharedArtworkId || sharedArtworkId <= 0) return;
    if (openedShareRef.current === sharedArtworkId) return;
    const index = items.findIndex((item) => item.id === sharedArtworkId);
    if (index < 0) return;
    openedShareRef.current = sharedArtworkId;
    onOpenArtwork(items, index);
  }, [loading, items, sharedArtworkId, onOpenArtwork]);

  function handleLoadMore() {
    const next = page + 1;
    setLoading(true);
    fetchPublicArtworks(buildFilter(next))
      .then((res) => {
        setItems((prev) => [...prev, ...(res.items ?? [])]);
        setTotalCount(res.total_count ?? 0);
        setPage(next);
      })
      .finally(() => setLoading(false));
  }

  const emptyLabel = selectedGrade ? selectedGrade.label : title;

  return (
    <article id={`khoi-${level}`} className="gallery-hall-section" aria-labelledby={`khoi-${level}-title`}>
      <div className="gallery-hall-toolbar">
        <header className="gallery-hall-heading">
          <h2 id={`khoi-${level}-title`}>{title}</h2>
          {subtitle && !search && !selectedGrade && <p className="gallery-hall-sub">{subtitle}</p>}
          {!loading && (
            <p className="gallery-hall-count">
              {totalCount} tác phẩm
              {selectedGrade ? ` · ${selectedGrade.label}` : ""}
              {search ? ` · “${search}”` : ""}
            </p>
          )}
        </header>

        <div className="grade-node-grid gallery-hall-nodes" role="group" aria-label={`Chọn lớp ${title}`}>
          <GradeNode
            label="Tất cả"
            variant="chip"
            active={selectedGradeId === null}
            onClick={() => onSelectGrade(null)}
          />
          {grades.map((g) => (
            <GradeNode
              key={g.id}
              label={String(g.grade_number)}
              ariaLabel={g.label}
              active={g.id === selectedGradeId}
              onClick={() => onSelectGrade(g.id === selectedGradeId ? null : g.id)}
            />
          ))}
        </div>
      </div>

      <ArtworkRail
        items={items}
        loading={loading && items.length === 0}
        emptyText={
          search
            ? `Chưa tìm thấy “${search}” trong ${emptyLabel}. Thử tìm bằng tên tác phẩm, hoặc chỉ một phần tên học sinh.`
            : `${emptyLabel} chưa có tác phẩm nào được trưng bày. Tranh sẽ xuất hiện tại đây ngay khi được duyệt.`
        }
        hasMore={items.length < totalCount}
        onLoadMore={handleLoadMore}
        onOpen={(index) => onOpenArtwork(items, index)}
      />
    </article>
  );
}
