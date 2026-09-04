import { Pencil, Plus, Search, Star, Trash2 } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { ArtworkEditModal } from "../../components/admin/ArtworkEditModal";
import { ReactionIcons } from "../../components/admin/ReactionIcons";
import { ConfirmDialog } from "../../components/ConfirmDialog";
import {
  deleteArtwork,
  fetchArtworks,
  fetchGradeLevels,
  fetchSchools,
  toggleFeatured,
  updateArtwork,
  type ArtworkWithMeta,
  type GradeLevel,
  type School,
} from "../../lib/artworkApi";
import { fetchAwards } from "../../lib/awardApi";
import type { Award } from "../../lib/artworkApi";
import type { ArtworkMetaFormValues } from "../../components/admin/ArtworkMetaForm";
import { toast } from "../../lib/toastBus";

const REGION_LABEL: Record<string, string> = { saigon: "Sài Gòn", cantho: "Cần Thơ", vungtau: "Vũng Tàu" };

export default function ArtworksListPage() {
  const [items, setItems] = useState<ArtworkWithMeta[]>([]);
  const [totalCount, setTotalCount] = useState(0);
  const [page, setPage] = useState(1);
  const pageSize = 20;
  const [loading, setLoading] = useState(true);

  const [search, setSearch] = useState("");
  const [schoolFilter, setSchoolFilter] = useState<number | "">("");
  const [gradeFilter, setGradeFilter] = useState<number | "">("");
  const [awardFilter, setAwardFilter] = useState<number | "">("");
  const [featuredOnly, setFeaturedOnly] = useState(false);

  const [schools, setSchools] = useState<School[]>([]);
  const [gradeLevels, setGradeLevels] = useState<GradeLevel[]>([]);
  const [awards, setAwards] = useState<Award[]>([]);

  const [editing, setEditing] = useState<ArtworkWithMeta | null>(null);
  const [saving, setSaving] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<ArtworkWithMeta | null>(null);
  const [deleting, setDeleting] = useState(false);

  useEffect(() => {
    Promise.all([fetchSchools(), fetchGradeLevels(), fetchAwards(true)])
      .then(([s, g, a]) => {
        setSchools(s ?? []);
        setGradeLevels(g ?? []);
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
        })
        .catch((err: unknown) => {
          if (controller?.signal.aborted) return;
          toast.error(err instanceof Error ? err.message : "Không tải được danh sách tác phẩm");
        })
        .finally(() => {
          if (!controller?.signal.aborted) setLoading(false);
        });
    },
    [search, schoolFilter, gradeFilter, awardFilter, featuredOnly, page],
  );

  useEffect(() => {
    const controller = new AbortController();
    load(controller);
    return () => controller.abort();
  }, [load]);

  async function handleSave(values: ArtworkMetaFormValues) {
    if (!editing) return;
    setSaving(true);
    try {
      await updateArtwork(editing.id, {
        title: values.title,
        student_id: editing.student_id,
        school_id: Number(values.schoolId),
        grade_level_id: Number(values.gradeLevelId),
        is_featured: editing.is_featured,
        is_published: editing.is_published,
        award_id: values.awardId ? Number(values.awardId) : 0,
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

  const totalPages = Math.max(1, Math.ceil(totalCount / pageSize));

  return (
    <div className="artworks-list-page">
      <div className="artworks-list-header">
        <h1>Quản lý tác phẩm</h1>
        <Link to="/admin/artworks/upload" className="btn btn-primary">
          <Plus size={16} /> Tải tác phẩm mới
        </Link>
      </div>

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
      </div>

      {loading ? (
        <div className="admin-page-placeholder">Đang tải danh sách…</div>
      ) : items.length === 0 ? (
        <div className="admin-page-placeholder">Không tìm thấy tác phẩm nào phù hợp.</div>
      ) : (
        <div className="artworks-table" role="table">
          <div className="artworks-table-head" role="row">
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
            <div className="artworks-table-row" role="row" key={item.id}>
              <span className="artworks-cell-thumb" role="cell">
                <img src={item.thumbnail_url || item.image_url} alt={item.title} loading="lazy" />
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
