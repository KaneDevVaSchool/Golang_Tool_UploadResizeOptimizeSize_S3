import { ArrowRight, Images } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { JsonLd } from "../../components/JsonLd";
import { FeaturedArtworkCard } from "../../components/public/FeaturedArtworkCard";
import { FeaturedGardenScene } from "../../components/public/FeaturedGardenScene";
import { FeaturedHero } from "../../components/public/FeaturedHero";
import { GalleryPagination } from "../../components/public/GalleryPagination";
import { PublicLightbox } from "../../components/public/PublicLightbox";
import { RegionTabs, type Region } from "../../components/public/RegionTabs";
import { usePageMeta } from "../../hooks/usePageMeta";
import type { ArtworkWithMeta } from "../../lib/artworkApi";
import { fetchFeaturedArtworks } from "../../lib/publicApi";

const PAGE_TITLE = "Tác phẩm tiêu biểu — Khu vườn nghệ thuật VA Schools";
const PAGE_DESCRIPTION =
  "Những bức tranh nổi bật được chọn giới thiệu từ hội thi vẽ tranh 20 năm Trường Việt Mỹ, theo từng khu vực Sài Gòn, Cần Thơ, Vũng Tàu.";

const VALID_REGIONS: Region[] = ["all", "saigon", "cantho", "vungtau"];
const PAGE_SIZE = 18;
const PODIUM_FALLBACK = 99;
const REGION_LABEL: Record<Region, string> = {
  all: "Tất cả",
  saigon: "Sài Gòn",
  cantho: "Cần Thơ",
  vungtau: "Vũng Tàu",
};

function foldAwardKey(value: string): string {
  return value
    .toLowerCase()
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/g, "");
}

/**
 * 1/2/3 = Giải Nhất/Nhì/Ba. Không thuộc podium thì 99 để xếp sau.
 *
 * rank_order lưu 0-based (vị trí kéo-thả ở trang admin/awards: giải đầu
 * danh sách rank_order=0) nên phải +1 mới ra đúng bậc 1..3 - xem giải thích
 * đầy đủ ở tierOf() trong HallOfFamePage.tsx, nơi cùng dữ liệu award được
 * xếp bậc cho trang /bang-vang.
 */
function podiumRank(item: ArtworkWithMeta): number {
  let best = PODIUM_FALLBACK;
  for (const award of item.awards ?? []) {
    const position = award.rank_order + 1;
    if (position >= 1 && position <= 3) {
      best = Math.min(best, position);
      continue;
    }
    const key = foldAwardKey(`${award.slug} ${award.name}`);
    if (/\bnhat\b|giai-nhat/.test(key)) best = Math.min(best, 1);
    else if (/\bnhi\b|giai-nhi/.test(key)) best = Math.min(best, 2);
    else if (/\bgiai ba\b|giai-ba\b/.test(key)) best = Math.min(best, 3);
  }
  return best;
}

/**
 * Trang /tac-pham-tieu-bieu - banner khu vườn ôm chữ + tab khu vực + lưới
 * khung gỗ trang trọng vừa phải (dày hơn phòng triển lãm, nhẹ hơn bảng vàng).
 *
 * Khu vực khởi tạo đọc từ query "?khu-vuc=" (điều hướng từ Hero ở trang
 * chủ); đổi tab cập nhật lại query để có thể chia sẻ/back-forward được.
 *
 * API trả về toàn bộ danh sách tác phẩm tiêu biểu trong 1 lần (không có
 * tham số phân trang phía server), nên việc chia trang thực hiện ở client:
 * cắt mảng theo PAGE_SIZE. Cách này giữ nguyên hợp đồng API hiện có và
 * tránh thêm round-trip mỗi lần đổi trang, đổi lại phải tự lo reset trang
 * khi đổi khu vực (xem effect bên dưới).
 */
export default function FeaturedArtworksPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const initialRegion = (searchParams.get("khu-vuc") as Region) ?? "all";
  const sharedArtworkId = Number(searchParams.get("tranh"));
  const [region, setRegion] = useState<Region>(VALID_REGIONS.includes(initialRegion) ? initialRegion : "all");
  const [items, setItems] = useState<ArtworkWithMeta[]>([]);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(1);
  const [activeIndex, setActiveIndex] = useState<number | null>(null);

  // canonicalPath cố định về path gốc, không kèm ?khu-vuc=/?tranh= - các
  // biến thể lọc theo khu vực hay mở modal tác phẩm đều cùng một nội dung
  // cơ bản, không nên bị Google index như những trang riêng biệt.
  usePageMeta({ title: PAGE_TITLE, description: PAGE_DESCRIPTION, canonicalPath: "/tac-pham-tieu-bieu" });

  function handleChangeRegion(next: Region) {
    setRegion(next);
    setActiveIndex(null);
    setSearchParams(next === "all" ? {} : { "khu-vuc": next });
  }

  useEffect(() => {
    const controller = new AbortController();
    setLoading(true);
    fetchFeaturedArtworks(region === "all" ? undefined : region, controller.signal)
      .then((res) => {
        setItems(res.items ?? []);
        // Khu vực mới = danh sách mới, số trang mới - luôn quay về trang 1,
        // nếu không người dùng đang ở trang 5 của "Tất cả" chuyển sang một
        // khu vực chỉ có 2 trang sẽ rơi vào trang trống.
        setPage(1);
      })
      .catch(() => {
        if (!controller.signal.aborted) setItems([]);
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false);
      });
    return () => controller.abort();
  }, [region]);

  const rankedItems = useMemo(
    () => [...items].sort((a, b) => podiumRank(a) - podiumRank(b)),
    [items],
  );
  const totalPages = Math.max(1, Math.ceil(rankedItems.length / PAGE_SIZE));
  // Nhất → Nhì → Ba lên đầu (rồi mới tới tác phẩm không có podium), sau đó
  // mới cắt trang để giải không bị kẹt ở trang sau. CSS order trên ô lưới
  // giữ cùng thứ tự nếu trình duyệt xếp lại grid.
  const pageItems = useMemo(
    () => rankedItems.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE),
    [rankedItems, page],
  );

  // Link chia sẻ có ?tranh=<id>: tự tìm đúng trang rồi mở đúng tác phẩm.
  // Nhờ vậy Facebook/copy link trong lightbox là deep-link thực sự.
  useEffect(() => {
    if (loading || !Number.isInteger(sharedArtworkId) || sharedArtworkId <= 0) return;
    const rankedIndex = rankedItems.findIndex((item) => item.id === sharedArtworkId);
    if (rankedIndex < 0) return;
    const targetPage = Math.floor(rankedIndex / PAGE_SIZE) + 1;
    setPage(targetPage);
    setActiveIndex(rankedIndex % PAGE_SIZE);
  }, [loading, rankedItems, sharedArtworkId]);

  function setArtworkInUrl(id: number | null) {
    const next = new URLSearchParams(searchParams);
    if (id === null) next.delete("tranh");
    else next.set("tranh", String(id));
    setSearchParams(next);
  }

  function handleOpenArtwork(index: number) {
    setActiveIndex(index);
    setArtworkInUrl(pageItems[index].id);
  }

  function handleNavigateArtwork(index: number) {
    setActiveIndex(index);
    setArtworkInUrl(pageItems[index].id);
  }

  function handleCloseArtwork() {
    setActiveIndex(null);
    setArtworkInUrl(null);
  }

  function handleChangePage(next: number) {
    setPage(next);
    setActiveIndex(null);
    setArtworkInUrl(null);
    // Đổi trang mà giữ nguyên vị trí cuộn sẽ ném người dùng vào giữa lưới
    // ảnh mới - đưa về đầu khu gallery (không phải đầu trang, để không phải
    // cuộn lại qua Hero).
    document.querySelector(".featured-gallery-section")?.scrollIntoView({ behavior: "smooth", block: "start" });
  }

  return (
    <div className="featured-page">
      <JsonLd
        data={{
          "@context": "https://schema.org",
          "@type": "CollectionPage",
          name: PAGE_TITLE,
          description: PAGE_DESCRIPTION,
          url: window.location.origin + "/tac-pham-tieu-bieu",
        }}
      />
      <JsonLd
        data={{
          "@context": "https://schema.org",
          "@type": "BreadcrumbList",
          itemListElement: [
            { "@type": "ListItem", position: 1, name: "Trang chủ", item: window.location.origin + "/" },
            {
              "@type": "ListItem",
              position: 2,
              name: "Tác phẩm tiêu biểu",
              item: window.location.origin + "/tac-pham-tieu-bieu",
            },
          ],
        }}
      />
      <FeaturedHero
        kicker="Tác phẩm tiêu biểu"
        title="Những bức tranh khiến ta dừng lại"
        description="Tuyển chọn từ khắp các cơ sở VASchools — mỗi bức được đóng khung và treo trang trọng, đúng như cách một tác phẩm xứng đáng được nhìn ngắm."
      />
      <section className="featured-gallery-section">
        <FeaturedGardenScene />
        <RegionTabs value={region} onChange={handleChangeRegion} />

        {loading ? (
          <p className="featured-loading-note">Đang treo tranh lên tường…</p>
        ) : items.length === 0 ? (
          <div className="featured-empty-state">
            <span className="featured-empty-icon" aria-hidden>
              <Images strokeWidth={1.5} />
            </span>
            <p className="featured-empty-title">
              {region === "all"
                ? "Chưa có tác phẩm tiêu biểu nào"
                : `Khu vực ${REGION_LABEL[region]} chưa có tác phẩm tiêu biểu`}
            </p>
            <p className="featured-empty-note">
              Hãy thử chọn một khu vực khác, hoặc ghé Phòng triển lãm để xem toàn bộ tranh dự thi.
            </p>
            <Link className="featured-empty-cta" to="/phong-trien-lam">
              Vào phòng triển lãm
              <ArrowRight size={16} strokeWidth={2.4} aria-hidden />
            </Link>
          </div>
        ) : (
          <>
            {/* Mọi card một cỡ, mỗi ô 1 cột - lỗ ảnh chữ nhật 4:3 nên hàng
                không so le dù tranh gốc ngang hay dọc. */}
            <div className="featured-gallery-grid">
              {pageItems.map((item, index) => {
                const rank = podiumRank(item);
                return (
                  <div
                    key={item.id}
                    className="featured-gallery-cell"
                    data-podium={rank <= 3 ? rank : undefined}
                  >
                    <FeaturedArtworkCard
                      item={item}
                      award={item.awards?.[0]}
                      index={index}
                      onClick={() => handleOpenArtwork(index)}
                    />
                  </div>
                );
              })}
            </div>
            <GalleryPagination page={page} totalPages={totalPages} onChange={handleChangePage} />

            {/* Tuyển chọn nào cũng phải bỏ lại rất nhiều bức đáng xem - dẫn
                người xem sang phòng triển lãm để không ai ra về tay không. */}
            <aside className="hall-closing">
              <span className="hall-closing-mark" aria-hidden>
                🖼️
              </span>
              <p className="hall-closing-lead">
                Đằng sau mỗi bức tranh treo ở đây là một buổi chiều ngồi lì bên bàn vẽ, một hộp màu dùng đến
                cạn, và một em nhỏ tin rằng điều mình tưởng tượng đáng được người khác nhìn thấy.
              </p>
              <p className="hall-closing-note">
                Đây mới chỉ là những tác phẩm tiêu biểu. Còn rất nhiều nét vẽ đáng yêu khác đang đợi trong
                phòng triển lãm.
              </p>
              <Link className="hall-closing-cta" to="/phong-trien-lam">
                Vào phòng triển lãm
                <ArrowRight size={16} strokeWidth={2.4} aria-hidden />
              </Link>
            </aside>
          </>
        )}

        {activeIndex !== null && (
          <PublicLightbox
            items={pageItems}
            activeIndex={activeIndex}
            onNavigate={handleNavigateArtwork}
            onClose={handleCloseArtwork}
          />
        )}
      </section>
    </div>
  );
}
