import { useEffect, useRef, useState, type CSSProperties } from "react";
import type { ArtworkWithMeta } from "../../lib/artworkApi";
import type { TopicCategory } from "../../lib/topicCategoryApi";
import { fetchPublicArtworks } from "../../lib/publicApi";
import { useDeviceTier } from "../../hooks/useDeviceTier";
import { ArtworkRail } from "./ArtworkRail";

const PAGE_SIZE_BY_TIER: Record<string, number> = {
  full: 36,
  light: 24,
  minimal: 12,
};

/**
 * Một nhóm chủ đề sáng tạo = một section, đặt sau 2 section cấp học
 * (GalleryLevelSection) ở /phong-trien-lam. Khác section cấp học: không có
 * node chọn khối lớp - nhóm chủ đề có thể dùng chung cả 2 cấp học, lọc thêm
 * theo khối sẽ vụn nội dung một section vốn đã hẹp hơn.
 *
 * Tự ẩn hoàn toàn (trả về null) nếu nhóm không có tác phẩm nào đã duyệt -
 * tránh hàng chục section trống trải dài trang khi nhóm chủ đề mới tạo chưa
 * có tác phẩm gán vào, hoặc nhóm cũ đã hết tác phẩm hiển thị.
 */
export function GalleryTopicSection({
  topic,
  sharedArtworkId,
  onOpenArtwork,
}: {
  topic: TopicCategory;
  sharedArtworkId?: number;
  onOpenArtwork: (items: ArtworkWithMeta[], index: number) => void;
}) {
  const [items, setItems] = useState<ArtworkWithMeta[]>([]);
  const [totalCount, setTotalCount] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);
  // Phân biệt "chưa tải xong lần đầu" với "đã tải xong, rỗng" - chỉ ẩn hẳn
  // section ở trường hợp sau, tránh nháy ẩn/hiện lúc đang chờ response.
  const [loadedOnce, setLoadedOnce] = useState(false);
  const tier = useDeviceTier();
  const pageSize = PAGE_SIZE_BY_TIER[tier] ?? PAGE_SIZE_BY_TIER.full;

  function buildFilter(nextPage: number) {
    return { topic_category_id: topic.id, page: nextPage, page_size: pageSize };
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
        if (!controller.signal.aborted) {
          setLoading(false);
          setLoadedOnce(true);
        }
      });
    return () => controller.abort();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [topic.id, pageSize]);

  const openedShareRef = useRef<number | null>(null);
  useEffect(() => {
    if (loading || !sharedArtworkId || sharedArtworkId <= 0) return;
    if (openedShareRef.current === sharedArtworkId) return;
    const index = items.findIndex((item) => item.id === sharedArtworkId);
    if (index < 0) return;
    openedShareRef.current = sharedArtworkId;
    onOpenArtwork(items, index);
  }, [loading, items, sharedArtworkId, onOpenArtwork]);

  const loadMoreRef = useRef<AbortController | null>(null);
  useEffect(() => () => loadMoreRef.current?.abort(), []);

  function handleLoadMore() {
    loadMoreRef.current?.abort();
    const controller = new AbortController();
    loadMoreRef.current = controller;

    const next = page + 1;
    setLoading(true);
    fetchPublicArtworks(buildFilter(next), controller.signal)
      .then((res) => {
        if (controller.signal.aborted) return;
        setItems((prev) => [...prev, ...(res.items ?? [])]);
        setTotalCount(res.total_count ?? 0);
        setPage(next);
      })
      .catch(() => {
        // Huỷ chủ động không phải lỗi cần báo - danh sách hiện tại vẫn đúng.
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false);
      });
  }

  if (loadedOnce && totalCount === 0) return null;

  return (
    <article
      id={`chu-de-${topic.slug}`}
      className="gallery-hall-section gallery-hall-section--topic"
      aria-labelledby={`chu-de-${topic.slug}-title`}
      style={{ "--topic-color": topic.color_hex } as CSSProperties}
    >
      <div className="gallery-hall-toolbar gallery-hall-toolbar--topic">
        <header className="gallery-hall-heading">
          <h2 id={`chu-de-${topic.slug}-title`}>{topic.name}</h2>
          {!loading && (
            <p className="gallery-hall-count">
              <svg width="13" height="13" viewBox="0 0 24 24" fill="none" aria-hidden="true">
                <rect x="3" y="3" width="18" height="18" rx="3" stroke="currentColor" strokeWidth="1.8" />
                <circle cx="9" cy="9.5" r="1.7" fill="currentColor" />
                <path
                  d="M4.5 16.5L9 12.2C9.6 11.6 10.5 11.6 11.1 12.2L13.4 14.4"
                  stroke="currentColor"
                  strokeWidth="1.8"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                />
                <path
                  d="M13 15.8L15.7 13.2C16.3 12.6 17.2 12.6 17.8 13.2L19.5 14.9"
                  stroke="currentColor"
                  strokeWidth="1.8"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                />
              </svg>
              {totalCount} tác phẩm
            </p>
          )}
        </header>
      </div>

      <ArtworkRail
        items={items}
        loading={loading && items.length === 0}
        hasMore={items.length < totalCount}
        onLoadMore={handleLoadMore}
        onOpen={(index) => onOpenArtwork(items, index)}
      />
    </article>
  );
}
