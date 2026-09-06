import { ArrowRight, Award as AwardIcon, Crown } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { JsonLd } from "../../components/JsonLd";
import { FeaturedGardenScene } from "../../components/public/FeaturedGardenScene";
import { FeaturedHero } from "../../components/public/FeaturedHero";
import { HallArtworkCard } from "../../components/public/HallArtworkCard";
import { HallFireworks } from "../../components/public/HallFireworks";
import { HallRail } from "../../components/public/HallRail";
import { PublicLightbox } from "../../components/public/PublicLightbox";
import { usePageMeta } from "../../hooks/usePageMeta";
import { fetchBillboard, type BillboardEntry } from "../../lib/publicApi";

const PAGE_TITLE = "Bảng vàng — Khu vườn nghệ thuật VA Schools";
const PAGE_DESCRIPTION =
  "Bảng vinh danh và giải thưởng của hội thi vẽ tranh kỷ niệm 20 năm Trường Việt Mỹ.";

/** Bậc ngoài podium (chuyên đề / khuyến khích) gom chung vào một nhóm. */
const OTHER_TIER = 99;

/**
 * Nhãn huy chương + banner + crest trên bục, theo từng hạng.
 *
 * Banner phải thật ngắn: bệ chỉ rộng khoảng 1/3 bục và mọi dòng chữ trên bệ
 * đều bị khoá một hàng, chữ dài sẽ bị cắt bằng "…" chứ không xuống dòng.
 *
 * `crest` dùng nhãn hạng chuẩn hoá ("Hạng Nhất/Nhì/Ba"), KHÔNG phải tên
 * giải admin đặt (`entry.award.name`) - hai chỗ khác nguồn dữ liệu từng
 * hiện chữ khác nhau trên cùng một ô nếu admin đặt tên giải không chứa
 * "Nhất/Nhì/Ba" (vd "Giải Xuất Sắc"). Tên giải thật vẫn hiển thị đầy đủ
 * trong hall-podium-award ở bệ, chỉ là không còn lặp lại ở crest.
 */
const TIER_META: Record<number, { medal: string; banner: string; crest: string }> = {
  1: { medal: "Vàng", banner: "★ Giải Nhất ★", crest: "Hạng Nhất" },
  2: { medal: "Bạc", banner: "Giải Nhì", crest: "Hạng Nhì" },
  3: { medal: "Đồng", banner: "Giải Ba", crest: "Hạng Ba" },
};

function foldAwardKey(value: string): string {
  return value
    .toLowerCase()
    .normalize("NFD")
    .replace(/[̀-ͯ]/g, "");
}

/** Nhãn hạng chuẩn ("Hạng Nhất"...) theo từng bậc, dùng để lọc trùng với tên giải thật. */
const TIER_CANONICAL_KEY: Record<number, string> = {
  1: "nhat",
  2: "nhi",
  3: "ba",
};

/**
 * Tên giải admin đặt có đáng hiển thị thêm không, hay chỉ lặp lại đúng
 * nghĩa "Nhất/Nhì/Ba" đã có ở banner rồi. Ví dụ admin đặt tên giải đúng là
 * "Giải Nhất" thì không cần dòng phụ nhắc lại - nhưng "Giải Xuất Sắc Nhất
 * Khối 5" thì có, vì mang thêm thông tin.
 */
function isCanonicalAwardName(rank: number, awardName: string): boolean {
  const key = foldAwardKey(awardName);
  const canonical = TIER_CANONICAL_KEY[rank];
  if (!canonical) return false;
  return new RegExp(`^giai\\s+${canonical}$`).test(key.trim());
}

/**
 * Bậc của một tác phẩm. Ưu tiên rank_order do admin đặt; chỉ khi rank nằm
 * ngoài 1..3 mới đoán theo slug/tên - dữ liệu cũ có award chưa gán
 * rank_order chuẩn nhưng tên vẫn là "Giải Nhất/Nhì/Ba".
 *
 * rank_order lưu 0-based (đúng vị trí kéo-thả ở trang admin/awards: giải
 * đầu danh sách rank_order=0) nên phải +1 mới ra bậc 1..3 tương ứng
 * Nhất/Nhì/Ba - thiếu bước này từng khiến giải Nhì (rank_order=1) bị coi là
 * Nhất và giải Ba (rank_order=2) bị coi là Nhì.
 */
function tierOf(entry: BillboardEntry): number {
  const { rank_order: rank, slug, name } = entry.award;
  const position = rank + 1;
  if (position >= 1 && position <= 3) return position;
  const key = foldAwardKey(`${slug} ${name}`);
  if (/\bnhat\b|giai-nhat/.test(key)) return 1;
  if (/\bnhi\b|giai-nhi/.test(key)) return 2;
  if (/\bgiai ba\b|giai-ba\b/.test(key)) return 3;
  return OTHER_TIER;
}

/** Thứ tự hiển thị trong bục: Nhì trái - Nhất giữa - Ba phải. */
const PODIUM_ORDER = [2, 1, 3];

/**
 * Nhánh nguyệt quế cho khung quán quân. Vẽ tay bằng path thay vì dùng icon
 * có sẵn: nhánh cần cong ôm theo cạnh khung, còn icon vòng nguyệt quế của
 * lucide là một vòng khép kín không tách đôi được.
 */
function LaurelBranch() {
  return (
    <svg viewBox="0 0 40 120" fill="none" xmlns="http://www.w3.org/2000/svg">
      <path d="M31 116C14 98 8 74 12 50C15 30 22 14 31 4" stroke="currentColor" strokeWidth="2.4" strokeLinecap="round" />
      {[
        { x: 27, y: 100, r: -28 },
        { x: 21, y: 86, r: -20 },
        { x: 17, y: 71, r: -12 },
        { x: 15, y: 56, r: -4 },
        { x: 16, y: 41, r: 6 },
        { x: 19, y: 27, r: 16 },
        { x: 24, y: 14, r: 26 },
      ].map((leaf, i) => (
        <ellipse
          key={i}
          cx={leaf.x}
          cy={leaf.y}
          rx="8.5"
          ry="4.6"
          fill="currentColor"
          transform={`rotate(${leaf.r} ${leaf.x} ${leaf.y})`}
        />
      ))}
    </svg>
  );
}

/**
 * Trang /bang-vang - bục vinh danh 3 vị trí + dải giải chuyên đề.
 *
 *  1. Bục podium: Nhì trái - Nhất giữa (cao hơn) - Ba phải, đúng một tác
 *     phẩm mỗi hạng theo giải admin đã gán.
 *  2. Dải giải chuyên đề: mọi giải nằm ngoài podium (Khuyến khích, giải
 *     theo chủ đề...).
 *
 * Card dùng chung HallArtworkCard (khung gỗ trang trọng) ở cả hai tầng.
 */
export default function HallOfFamePage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const sharedArtworkId = Number(searchParams.get("tranh"));
  const [entries, setEntries] = useState<BillboardEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [activeIndex, setActiveIndex] = useState<number | null>(null);

  usePageMeta({ title: PAGE_TITLE, description: PAGE_DESCRIPTION, canonicalPath: "/bang-vang" });

  useEffect(() => {
    const controller = new AbortController();
    fetchBillboard(controller.signal)
      .then((list) => setEntries(list ?? []))
      .catch(() => {
        if (!controller.signal.aborted) setEntries([]);
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false);
      });
    return () => controller.abort();
  }, []);

  const { podium, specials, ordered } = useMemo(() => {
    const byTier = new Map<number, BillboardEntry[]>();
    for (const entry of entries) {
      const tier = tierOf(entry);
      const bucket = byTier.get(tier);
      if (bucket) bucket.push(entry);
      else byTier.set(tier, [entry]);
    }

    // Bục lấy thẳng tác phẩm admin đã gán - cuộc thi chỉ có một giải Nhất,
    // một Nhì, một Ba nên mỗi hạng đúng một ô. Trang KHÔNG tự xếp hạng lại
    // (theo lượt thích hay bất cứ tiêu chí nào): ai đạt giải là quyết định
    // của ban giám khảo, trang chỉ trình bày.
    //
    // Phòng dữ liệu bất thường (admin lỡ gán 2 tác phẩm cùng hạng), lấy id
    // nhỏ nhất để bục luôn đúng 3 ô thay vì vỡ bố cục, và cảnh báo ở
    // console để admin biết mà sửa - im lặng nuốt mất một giải thì tệ hơn.
    const podiumList = PODIUM_ORDER.map((rank) => {
      const list = byTier.get(rank) ?? [];
      if (list.length === 0) return null;
      if (list.length > 1 && import.meta.env.DEV) {
        console.warn(
          `[Bảng vàng] Hạng ${rank} có ${list.length} tác phẩm được gán giải, chỉ hiện 1. Kiểm tra lại phần gán giải trong admin.`,
        );
      }
      const entry = [...list].sort((a, b) => a.id - b.id)[0];
      return { rank, entry };
    }).filter((slot): slot is { rank: number; entry: BillboardEntry } => slot !== null);

    // Mọi giải ngoài podium (Khuyến khích, giải chuyên đề...) vào dải dưới.
    const specialList = [...byTier.entries()]
      .filter(([rank]) => rank > 3)
      .sort(([a], [b]) => a - b)
      .flatMap(([, list]) => list);

    // Thứ tự lightbox = đúng thứ tự mắt đọc trang: bục (trái→phải) rồi
    // tới dải giải chuyên đề.
    const flat = [...podiumList.map((slot) => slot.entry), ...specialList];

    return { podium: podiumList, specials: specialList, ordered: flat };
  }, [entries]);

  // Link chia sẻ ?tranh=<id> mở đúng tác phẩm trong lightbox.
  useEffect(() => {
    if (loading || !Number.isInteger(sharedArtworkId) || sharedArtworkId <= 0) return;
    const index = ordered.findIndex((item) => item.id === sharedArtworkId);
    if (index >= 0) setActiveIndex(index);
  }, [loading, ordered, sharedArtworkId]);

  function setArtworkInUrl(id: number | null) {
    const next = new URLSearchParams(searchParams);
    if (id === null) next.delete("tranh");
    else next.set("tranh", String(id));
    setSearchParams(next);
  }

  function handleOpenArtwork(index: number) {
    setActiveIndex(index);
    setArtworkInUrl(ordered[index].id);
  }

  function handleCloseArtwork() {
    setActiveIndex(null);
    setArtworkInUrl(null);
  }

  /** Index trên mảng phẳng để lightbox duyệt xuyên suốt cả trang. */
  const flatIndexOf = (entry: BillboardEntry) => ordered.indexOf(entry);

  return (
    <div className="featured-page hall-page">
      <JsonLd
        data={{
          "@context": "https://schema.org",
          "@type": "CollectionPage",
          name: PAGE_TITLE,
          description: PAGE_DESCRIPTION,
          url: window.location.origin + "/bang-vang",
        }}
      />
      <JsonLd
        data={{
          "@context": "https://schema.org",
          "@type": "BreadcrumbList",
          itemListElement: [
            { "@type": "ListItem", position: 1, name: "Trang chủ", item: window.location.origin + "/" },
            { "@type": "ListItem", position: 2, name: "Bảng vàng", item: window.location.origin + "/bang-vang" },
          ],
        }}
      />
      {/* Không có thú/chim ở trang này: bục vinh danh đã có pháo hoa, thêm
          sóc thỏ chạy qua chỉ chia mắt người xem ra hai chỗ. */}
      <FeaturedHero
        critters={false}
        kicker="Bảng vàng"
        title="Nơi những nét vẽ được gọi tên"
        description="Mỗi bức tranh ở đây bắt đầu từ một trang giấy trắng và rất nhiều lần các em dám vẽ tiếp. Xin chúc mừng những tác phẩm được ban giám khảo xướng tên — và cảm ơn mọi bàn tay nhỏ đã góp màu cho mùa thi này."
      />

      <section className="featured-gallery-section hall-section">
        <FeaturedGardenScene critters={false} />

        {loading ? (
          <p className="admin-empty-note">Đang mở bảng vàng…</p>
        ) : ordered.length === 0 ? (
          <p className="admin-empty-note">
            Bảng vàng đang chờ được xướng tên. Kết quả sẽ hiện tại đây ngay khi ban giám khảo công bố.
          </p>
        ) : (
          <div className="hall-wrap">
            {/* --- Tầng 1: bục vinh danh --------------------------------- */}
            {podium.length > 0 && (
              <section className="hall-honour hall-panel hall-panel--honour" aria-labelledby="buc-vinh-danh">
                <HallFireworks />

                <header className="hall-honour-heading">
                  <span className="hall-honour-rule" aria-hidden />
                  <div className="hall-honour-titles">
                    <span className="hall-honour-eyebrow" aria-hidden>
                      Phần 1
                    </span>
                    <h2 id="buc-vinh-danh">Bục Vinh Danh</h2>
                    <p className="hall-honour-sub">Ba tác phẩm xuất sắc nhất mùa thi năm nay</p>
                  </div>
                  <span className="hall-honour-rule" aria-hidden />
                </header>

                <div className="hall-podium" role="list">
                  {podium.map(({ rank, entry }) => {
                    const meta = TIER_META[rank];
                    return (
                      <div key={rank} className="hall-podium-slot" data-rank={rank} role="listitem">
                        <span className="hall-podium-crest">
                          {rank === 1 ? (
                            <Crown size={15} strokeWidth={2.2} aria-hidden />
                          ) : (
                            <AwardIcon size={14} strokeWidth={2.2} aria-hidden />
                          )}
                          {meta.crest}
                        </span>

                        <div className="hall-podium-stage">
                          {/* Vòng nguyệt quế ôm hai bên khung quán quân. */}
                          {rank === 1 && (
                            <>
                              <span className="hall-laurel hall-laurel--left" aria-hidden>
                                <LaurelBranch />
                              </span>
                              <span className="hall-laurel hall-laurel--right" aria-hidden>
                                <LaurelBranch />
                              </span>
                            </>
                          )}

                          <HallArtworkCard
                            item={entry}
                            award={entry.award}
                            index={rank}
                            variant="podium"
                            medalLabel={meta.medal}
                            onClick={() => handleOpenArtwork(flatIndexOf(entry))}
                          />
                        </div>

                        <div className="hall-podium-base">
                          <span className="hall-podium-rank" aria-hidden>
                            {rank}
                          </span>
                          <span className="hall-podium-banner">{meta.banner}</span>
                          {!isCanonicalAwardName(rank, entry.award.name) && (
                            <span className="hall-podium-award" title={entry.award.name}>
                              {entry.award.name}
                            </span>
                          )}
                          <span className="hall-podium-name" title={entry.title}>{`“${entry.title}”`}</span>
                          <span className="hall-podium-student" title={entry.student_name}>
                            Họa sĩ nhí {entry.student_name}
                          </span>
                          <span className="hall-podium-school" title={entry.school_name}>
                            {entry.class_name ? `Lớp ${entry.class_name} · ` : ""}
                            {entry.school_name}
                          </span>
                        </div>
                      </div>
                    );
                  })}
                </div>
              </section>
            )}

            {/* --- Tầng 2: giải chuyên đề -------------------------------- */}
            {/* Hai tầng trước đây trôi cạnh nhau trên cùng nền vườn nên đọc
                như một khối card liền mạch. Vạch phân cách + khung riêng cho
                mỗi tầng để mắt biết đây là một hạng mục khác. */}
            {specials.length > 0 && (
              <>
                {podium.length > 0 && (
                  <div className="hall-divider" aria-hidden>
                    <span className="hall-divider-line" />
                    <span className="hall-divider-mark">✦</span>
                    <span className="hall-divider-line" />
                  </div>
                )}

                <article className="hall-tier hall-tier--special hall-panel hall-panel--special" aria-labelledby="giai-chuyen-de">
                  <header className="hall-tier-heading">
                    <span className="hall-tier-medal hall-tier-medal--special" aria-hidden>
                      <AwardIcon size={18} strokeWidth={2.2} />
                    </span>
                    <div className="hall-tier-titles">
                      <span className="hall-tier-eyebrow" aria-hidden>
                        Phần 2
                      </span>
                      <h2 id="giai-chuyen-de">Giải Thưởng Đặc Biệt &amp; Chuyên Đề</h2>
                      <p className="hall-tier-sub">
                        Những tác phẩm khiến ban giám khảo dừng lại thật lâu — mỗi bức một lý do rất riêng
                      </p>
                    </div>
                    <span className="hall-tier-count">{specials.length} tác phẩm</span>
                  </header>

                  <HallRail entries={specials} onOpen={(entry) => handleOpenArtwork(flatIndexOf(entry))} />
                </article>
              </>
            )}

            {/* --- Lời kết ---------------------------------------------- */}
            {/* Bảng vàng chỉ gọi tên được vài em, nhưng trang này được phụ
                huynh của tất cả các em mở ra. Khối kết dẫn họ sang phòng
                triển lãm - nơi tranh của con mình chắc chắn có mặt. */}
            <aside className="hall-closing">
              <span className="hall-closing-mark" aria-hidden>
                🌱
              </span>
              <p className="hall-closing-lead">
                Một mùa thi khép lại, nhưng điều ở lại không phải là thứ hạng — mà là buổi chiều các em ngồi
                pha màu, những lần tẩy đi vẽ lại, và ánh mắt khi bức tranh cuối cùng cũng xong.
              </p>
              <p className="hall-closing-note">
                Xin cảm ơn thầy cô, và cảm ơn quý phụ huynh đã ngồi cạnh các em suốt chặng đường ấy. Mỗi tác
                phẩm gửi về đều được trân trọng trưng bày.
              </p>
              <Link className="hall-closing-cta" to="/phong-trien-lam">
                Xem toàn bộ tác phẩm dự thi
                <ArrowRight size={16} strokeWidth={2.4} aria-hidden />
              </Link>
            </aside>
          </div>
        )}

        {activeIndex !== null && (
          <PublicLightbox
            items={ordered}
            activeIndex={activeIndex}
            onNavigate={handleOpenArtwork}
            onClose={handleCloseArtwork}
          />
        )}
      </section>
    </div>
  );
}
