import { Images, LayoutGrid, List, Pencil, Plus, Search, Star, Trash2, Users } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { AdminPageHeader } from "../../components/admin/AdminPageHeader";
import { ArtworkEditModal } from "../../components/admin/ArtworkEditModal";
import { ReactionIcons } from "../../components/admin/ReactionIcons";
import { RegionSummaryStrip } from "../../components/admin/RegionSummaryStrip";
import { ConfirmDialog } from "../../components/ConfirmDialog";
import { useRegionSummary } from "../../hooks/useRegionSummary";
import { artworkImageURL } from "../../lib/artworkImage";
import {
  deleteArtwork,
  fetchArtworks,
  fetchGradeLevels,
  fetchSchools,
  setFeaturedBatch,
  toggleFeatured,
  updateArtwork,
  type ArtworkWithMeta,
  type GradeLevel,
  type School,
} from "../../lib/artworkApi";
import { fetchAwards } from "../../lib/awardApi";
import type { Award } from "../../lib/artworkApi";
import { REGION_LABEL } from "../../lib/chartTheme";
import { fetchTopicCategories, type TopicCategory } from "../../lib/topicCategoryApi";
import type { ArtworkMetaFormValues } from "../../components/admin/ArtworkMetaForm";
import { toast } from "../../lib/toastBus";

// Chế độ xem được nhớ qua localStorage: admin quay lại trang vẫn giữ đúng
// chế độ đã chọn lần trước, không phải bấm lại mỗi lần vào trang.
type ViewMode = "list" | "grid";
const VIEW_MODE_KEY = "vas_admin_artworks_view_mode";

function loadViewMode(): ViewMode {
  try {
    const saved = localStorage.getItem(VIEW_MODE_KEY);
    return saved === "grid" ? "grid" : "list";
  } catch {
    // Chế độ ẩn danh của một số trình duyệt chặn localStorage - mặc định
    // list là đủ, không làm hỏng chức năng chính của trang.
    return "list";
  }
}

export default function ArtworksListPage() {
  const [items, setItems] = useState<ArtworkWithMeta[]>([]);
  const [totalCount, setTotalCount] = useState(0);
  const [page, setPage] = useState(1);
  const pageSize = 20;
  const [loading, setLoading] = useState(true);

  const [search, setSearch] = useState("");
  const [schoolFilter, setSchoolFilter] = useState<number | "">("");
  const [gradeFilter, setGradeFilter] = useState<number | "">("");
  const [topicFilter, setTopicFilter] = useState<number | "">("");
  const [awardFilter, setAwardFilter] = useState<number | "">("");
  const [featuredOnly, setFeaturedOnly] = useState(false);

  const [schools, setSchools] = useState<School[]>([]);
  const [gradeLevels, setGradeLevels] = useState<GradeLevel[]>([]);
  const [topicCategories, setTopicCategories] = useState<TopicCategory[]>([]);
  const [awards, setAwards] = useState<Award[]>([]);

  const [viewMode, setViewMode] = useState<ViewMode>(loadViewMode);
  const [selected, setSelected] = useState<Set<number>>(new Set());
  const [bulkBusy, setBulkBusy] = useState(false);

  const [editing, setEditing] = useState<ArtworkWithMeta | null>(null);
  const [saving, setSaving] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<ArtworkWithMeta | null>(null);
  const [deleting, setDeleting] = useState(false);

  const regionSummary = useRegionSummary();

  useEffect(() => {
    Promise.all([fetchSchools(), fetchGradeLevels(), fetchTopicCategories(true), fetchAwards(true)])
      .then(([s, g, t, a]) => {
        setSchools(s ?? []);
        setGradeLevels(g ?? []);
        setTopicCategories(t ?? []);
        setAwards(a ?? []);
      })
      .catch((err: unknown) => toast.error(err instanceof Error ? err.message : "Không tải được dữ liệu nền"));
  }, []);

  const load = useCallback(
    (controller?: AbortController) => {
      setLoading(true);
      fetchArtworks(
        {
          search: search || undefined,
          school_id: schoolFilter || undefined,
          grade_level_id: gradeFilter || undefined,
          topic_category_id: topicFilter || undefined,
          award_id: awardFilter || undefined,
          featured: featuredOnly || undefined,
          page,
          page_size: pageSize,
        },
        controller?.signal,
      )
        .then((res) => {
          setItems(res.items ?? []);
          setTotalCount(res.total_count);
          // Trang đổi (lọc/phân trang) -> bỏ chọn, tránh áp bulk action nhầm
          // lên tác phẩm không còn hiển thị trên màn hình.
          setSelected(new Set());
        })
        .catch((err: unknown) => {
          if (controller?.signal.aborted) return;
          toast.error(err instanceof Error ? err.message : "Không tải được danh sách tác phẩm");
        })
        .finally(() => {
          if (!controller?.signal.aborted) setLoading(false);
        });
    },
    [search, schoolFilter, gradeFilter, topicFilter, awardFilter, featuredOnly, page],
  );

  useEffect(() => {
    const controller = new AbortController();
    load(controller);
    return () => controller.abort();
  }, [load]);

  function changeViewMode(mode: ViewMode) {
    setViewMode(mode);
    try {
      localStorage.setItem(VIEW_MODE_KEY, mode);
    } catch {
      // Không lưu được thì lần sau mặc định list - không ảnh hưởng phiên hiện tại.
    }
  }

  function toggleSelect(id: number) {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }

  function toggleSelectAll() {
    setSelected((prev) => (prev.size === items.length ? new Set() : new Set(items.map((i) => i.id))));
  }

  async function handleSave(values: ArtworkMetaFormValues) {
    if (!editing) return;
    setSaving(true);
    try {
      await updateArtwork(editing.id, {
        title: values.title,
        student_id: editing.student_id,
        school_id: Number(values.schoolId),
        grade_level_id: Number(values.gradeLevelId),
        topic_category_id: values.topicCategoryId ? Number(values.topicCategoryId) : null,
        is_featured: editing.is_featured,
        is_published: editing.is_published,
        award_ids: values.awardIds,
      });
      toast.success("Đã cập nhật tác phẩm.");
      setEditing(null);
      load();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Không cập nhật được tác phẩm");
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete() {
    if (!deleteTarget) return;
    setDeleting(true);
    try {
      await deleteArtwork(deleteTarget.id);
      toast.success("Đã xoá tác phẩm.");
      setDeleteTarget(null);
      load();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Không xoá được tác phẩm");
    } finally {
      setDeleting(false);
    }
  }

  async function handleToggleFeatured(item: ArtworkWithMeta) {
    try {
      await toggleFeatured(item.id, !item.is_featured);
      setItems((prev) => prev.map((a) => (a.id === item.id ? { ...a, is_featured: !a.is_featured } : a)));
      toast.success(!item.is_featured ? "Đã đánh dấu tiêu biểu." : "Đã bỏ đánh dấu tiêu biểu.");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Không cập nhật được trạng thái tiêu biểu");
    }
  }

  async function handleBulkSetFeatured(featured: boolean) {
    const ids = Array.from(selected);
    if (ids.length === 0) return;
    setBulkBusy(true);
    try {
      await setFeaturedBatch(ids, featured);
      setItems((prev) => prev.map((a) => (selected.has(a.id) ? { ...a, is_featured: featured } : a)));
      toast.success(
        featured
          ? `Đã đánh dấu tiêu biểu cho ${ids.length} tác phẩm.`
          : `Đã bỏ đánh dấu tiêu biểu cho ${ids.length} tác phẩm.`,
      );
      setSelected(new Set());
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Không cập nhật được trạng thái tiêu biểu hàng loạt");
    } finally {
      setBulkBusy(false);
    }
  }

  const totalPages = Math.max(1, Math.ceil(totalCount / pageSize));

  return (
    <div className="artworks-list-page">
      <AdminPageHeader
        title="Thư viện tác phẩm"
        subtitle={totalCount > 0 ? `${totalCount} tác phẩm đang được lưu giữ` : undefined}
        primaryAction={{ label: "Đưa tác phẩm lên", icon: Plus, to: "/admin/artworks/upload", iconOnly: true }}
      />

      <RegionSummaryStrip
        data={regionSummary.data}
        loading={regionSummary.loading}
        artworkIcon={Images}
        studentIcon={Users}
      />

      <div className="artworks-filter-bar">
        <div className="artworks-search-input">
          <Search size={16} />
          <input
            type="text"
            placeholder="Tìm theo tên tác phẩm hoặc học sinh…"
            value={search}
            onChange={(e) => {
              setPage(1);
              setSearch(e.target.value);
            }}
          />
        </div>

        <select
          value={schoolFilter}
          onChange={(e) => {
            setPage(1);
            setSchoolFilter(e.target.value ? Number(e.target.value) : "");
          }}
        >
          <option value="">Tất cả cơ sở</option>
          {schools.map((s) => (
            <option key={s.id} value={s.id}>
              {s.name}
            </option>
          ))}
        </select>

        <select
          value={gradeFilter}
          onChange={(e) => {
            setPage(1);
            setGradeFilter(e.target.value ? Number(e.target.value) : "");
          }}
        >
          <option value="">Tất cả khối</option>
          {gradeLevels.map((g) => (
            <option key={g.id} value={g.id}>
              {g.label}
            </option>
          ))}
        </select>

        <select
          value={topicFilter}
          onChange={(e) => {
            setPage(1);
            setTopicFilter(e.target.value ? Number(e.target.value) : "");
          }}
        >
          <option value="">Tất cả nhóm chủ đề</option>
          {topicCategories.map((t) => (
            <option key={t.id} value={t.id}>
              {t.name}
            </option>
          ))}
        </select>

        <select
          value={awardFilter}
          onChange={(e) => {
            setPage(1);
            setAwardFilter(e.target.value ? Number(e.target.value) : "");
          }}
        >
          <option value="">Tất cả giải thưởng</option>
          {awards.map((a) => (
            <option key={a.id} value={a.id}>
              {a.name}
            </option>
          ))}
        </select>

        <label className="artworks-featured-toggle">
          <input
            type="checkbox"
            checked={featuredOnly}
            onChange={(e) => {
              setPage(1);
              setFeaturedOnly(e.target.checked);
            }}
          />
          Chỉ tiêu biểu
        </label>

        <div className="artworks-view-switch" role="group" aria-label="Chế độ xem">
          <button
            type="button"
            className={viewMode === "list" ? "is-active" : ""}
            title="Xem dạng danh sách"
            aria-pressed={viewMode === "list"}
            onClick={() => changeViewMode("list")}
          >
            <List size={16} />
          </button>
          <button
            type="button"
            className={viewMode === "grid" ? "is-active" : ""}
            title="Xem dạng lưới ảnh"
            aria-pressed={viewMode === "grid"}
            onClick={() => changeViewMode("grid")}
          >
            <LayoutGrid size={16} />
          </button>
        </div>
      </div>

      {selected.size > 0 && (
        <div className="artworks-bulk-bar">
          <span>{selected.size} tác phẩm đã chọn</span>
          <div className="artworks-bulk-actions">
            <button type="button" disabled={bulkBusy} onClick={() => handleBulkSetFeatured(true)}>
              <Star size={14} /> Đánh dấu tiêu biểu
            </button>
            <button type="button" disabled={bulkBusy} onClick={() => handleBulkSetFeatured(false)}>
              <Star size={14} /> Bỏ tiêu biểu
            </button>
            <button type="button" className="artworks-bulk-clear" disabled={bulkBusy} onClick={() => setSelected(new Set())}>
              Bỏ chọn
            </button>
          </div>
        </div>
      )}

      {loading ? (
        <div className="admin-page-placeholder">Đang tải danh sách…</div>
      ) : items.length === 0 ? (
        <div className="admin-page-placeholder">Không tìm thấy tác phẩm nào phù hợp.</div>
      ) : viewMode === "grid" ? (
        <div className="artworks-grid">
          {items.map((item) => (
            <div className={`artworks-grid-card${selected.has(item.id) ? " is-selected" : ""}`} key={item.id}>
              <label className="artworks-grid-select">
                <input type="checkbox" checked={selected.has(item.id)} onChange={() => toggleSelect(item.id)} />
              </label>
              <div className="artworks-grid-thumb">
                <img src={artworkImageURL(item, "medium")} alt={item.title} loading="lazy" />
                {item.is_featured && (
                  <span className="artworks-grid-featured-badge" title="Đang là tác phẩm tiêu biểu">
                    <Star size={12} fill="currentColor" />
                  </span>
                )}
              </div>
              <div className="artworks-grid-body">
                <strong title={item.title}>{item.title}</strong>
                <span title={item.student_name}>{item.student_name}</span>
                <span className="artworks-grid-meta">
                  {item.school_name} · {item.grade_label}
                </span>
                {item.awards && item.awards.length > 0 && (
                  <span className="artworks-cell-award">
                    {item.awards.map((a) => (
                      <span key={a.id} className="award-badge" style={{ background: a.color_hex }}>
                        {a.name}
                      </span>
                    ))}
                  </span>
                )}
              </div>
              <div className="artworks-grid-actions">
                <button
                  type="button"
                  className={`artworks-icon-btn${item.is_featured ? " artworks-icon-btn--active" : ""}`}
                  title={item.is_featured ? "Bỏ đánh dấu tiêu biểu" : "Đánh dấu tiêu biểu"}
                  onClick={() => handleToggleFeatured(item)}
                >
                  <Star size={16} fill={item.is_featured ? "currentColor" : "none"} />
                </button>
                <button type="button" className="artworks-icon-btn" title="Chỉnh sửa" onClick={() => setEditing(item)}>
                  <Pencil size={16} />
                </button>
                <button
                  type="button"
                  className="artworks-icon-btn artworks-icon-btn--danger"
                  title="Xoá"
                  onClick={() => setDeleteTarget(item)}
                >
                  <Trash2 size={16} />
                </button>
              </div>
            </div>
          ))}
        </div>
      ) : (
        <div className="artworks-table" role="table">
          <div className="artworks-table-head" role="row">
            <span className="artworks-cell-select">
              <input
                type="checkbox"
                checked={items.length > 0 && selected.size === items.length}
                onChange={toggleSelectAll}
                aria-label="Chọn tất cả"
              />
            </span>
            <span>Ảnh</span>
            <span>Tác phẩm / Học sinh</span>
            <span>Cơ sở / Khối</span>
            <span>Giải</span>
            <span>Lượt xem</span>
            <span>Cảm xúc</span>
            <span>Bình luận</span>
            <span>Thao tác</span>
          </div>
          {items.map((item) => (
            <div className={`artworks-table-row${selected.has(item.id) ? " is-selected" : ""}`} role="row" key={item.id}>
              <span className="artworks-cell-select" role="cell">
                <input
                  type="checkbox"
                  checked={selected.has(item.id)}
                  onChange={() => toggleSelect(item.id)}
                  aria-label={`Chọn ${item.title}`}
                />
              </span>
              <span className="artworks-cell-thumb" role="cell">
                <img src={artworkImageURL(item, "thumb")} alt={item.title} loading="lazy" />
              </span>
              <span className="artworks-cell-title" role="cell">
                <strong title={item.title}>{item.title}</strong>
                <span title={item.student_name}>{item.student_name}</span>
              </span>
              <span className="artworks-cell-meta" role="cell">
                <span>{item.school_name}</span>
                <span className="artworks-cell-region">{REGION_LABEL[item.region] ?? item.region}</span>
                <span>{item.grade_label}</span>
              </span>
              <span className="artworks-cell-award" role="cell">
                {item.awards && item.awards.length > 0 ? (
                  item.awards.map((a) => (
                    <span key={a.id} className="award-badge" style={{ background: a.color_hex }}>
                      {a.name}
                    </span>
                  ))
                ) : (
                  <span className="artworks-cell-empty">—</span>
                )}
              </span>
              <span className="artworks-cell-views" role="cell">
                {item.view_count}
              </span>
              <span className="artworks-cell-reactions" role="cell">
                <ReactionIcons counts={item.reaction_counts} />
              </span>
              <span className="artworks-cell-comments" role="cell">
                {item.comment_count}
              </span>
              <span className="artworks-cell-actions" role="cell">
                <button
                  type="button"
                  className={`artworks-icon-btn${item.is_featured ? " artworks-icon-btn--active" : ""}`}
                  title={item.is_featured ? "Bỏ đánh dấu tiêu biểu" : "Đánh dấu tiêu biểu"}
                  onClick={() => handleToggleFeatured(item)}
                >
                  <Star size={16} fill={item.is_featured ? "currentColor" : "none"} />
                </button>
                <button
                  type="button"
                  className="artworks-icon-btn"
                  title="Chỉnh sửa"
                  onClick={() => setEditing(item)}
                >
                  <Pencil size={16} />
                </button>
                <button
                  type="button"
                  className="artworks-icon-btn artworks-icon-btn--danger"
                  title="Xoá"
                  onClick={() => setDeleteTarget(item)}
                >
                  <Trash2 size={16} />
                </button>
              </span>
            </div>
          ))}
        </div>
      )}

      {totalPages > 1 && (
        <div className="artworks-pagination">
          <button type="button" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>
            Trước
          </button>
          <span>
            Trang {page}/{totalPages}
          </span>
          <button type="button" disabled={page >= totalPages} onClick={() => setPage((p) => p + 1)}>
            Sau
          </button>
        </div>
      )}

      <ArtworkEditModal
        open={Boolean(editing)}
        artwork={editing}
        schools={schools}
        gradeLevels={gradeLevels}
        topicCategories={topicCategories}
        onTopicCategoryCreated={(category) => setTopicCategories((prev) => [...prev, category])}
        awards={awards}
        busy={saving}
        onSave={handleSave}
        onClose={() => setEditing(null)}
      />

      <ConfirmDialog
        open={Boolean(deleteTarget)}
        title="Xoá tác phẩm này?"
        message={`"${deleteTarget?.title ?? ""}" sẽ bị xoá khỏi hệ thống. Hành động này không thể hoàn tác.`}
        confirmLabel="Xoá"
        cancelLabel="Huỷ"
        busyLabel="Đang xoá…"
        busy={deleting}
        onConfirm={handleDelete}
        onCancel={() => setDeleteTarget(null)}
      />
    </div>
  );
}
