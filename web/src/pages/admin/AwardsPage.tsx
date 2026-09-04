import { Award as AwardIcon, Medal, Pencil, Plus, Star, Trash2, Trophy } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { ConfirmDialog } from "../../components/ConfirmDialog";
import { createAward, deleteAward, fetchAwards, updateAward, type Award } from "../../lib/awardApi";
import { toast } from "../../lib/toastBus";

const ICON_OPTIONS: { key: string; label: string; Icon: typeof Trophy }[] = [
  { key: "trophy", label: "Cúp", Icon: Trophy },
  { key: "medal", label: "Huy chương", Icon: Medal },
  { key: "star", label: "Ngôi sao", Icon: Star },
  { key: "award", label: "Huy hiệu", Icon: AwardIcon },
];

function iconFor(key?: string) {
  return ICON_OPTIONS.find((o) => o.key === key)?.Icon ?? Trophy;
}

function slugify(name: string): string {
  return name
    .toLowerCase()
    .normalize("NFD")
    .replace(/[̀-ͯ]/g, "")
    .replace(/đ/g, "d")
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
}

type FormState = {
  id: number | null;
  name: string;
  rankOrder: number;
  colorHex: string;
  iconKey: string;
  isActive: boolean;
};

const EMPTY_FORM: FormState = { id: null, name: "", rankOrder: 0, colorHex: "#c49c57", iconKey: "trophy", isActive: true };

/**
 * Trang quản lý giải thưởng - danh sách card màu+icon+tên+thứ tự, form
 * thêm/sửa inline (không cần modal riêng vì form đơn giản), xoá có
 * ConfirmDialog và hiển thị rõ lỗi 409 khi giải đang gắn cho tác phẩm.
 */
export default function AwardsPage() {
  const [awards, setAwards] = useState<Award[]>([]);
  const [loading, setLoading] = useState(true);
  const [form, setForm] = useState<FormState>(EMPTY_FORM);
  const [saving, setSaving] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<Award | null>(null);
  const [deleting, setDeleting] = useState(false);

  const load = useMemo(
    () => (signal?: AbortSignal) => {
      setLoading(true);
      fetchAwards(true, signal)
        .then((list) => setAwards(list ?? []))
        .catch((err: unknown) => {
          if (signal?.aborted) return;
          toast.error(err instanceof Error ? err.message : "Không tải được danh sách giải thưởng");
        })
        .finally(() => {
          if (!signal?.aborted) setLoading(false);
        });
    },
    [],
  );

  useEffect(() => {
    const controller = new AbortController();
    load(controller.signal);
    return () => controller.abort();
  }, [load]);

  function startEdit(award: Award) {
    setForm({
      id: award.id,
      name: award.name,
      rankOrder: award.rank_order,
      colorHex: award.color_hex,
      iconKey: award.icon_key ?? "trophy",
      isActive: award.is_active,
    });
  }

  function resetForm() {
    setForm(EMPTY_FORM);
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!form.name.trim()) {
      toast.error("Vui lòng nhập tên giải thưởng.");
      return;
    }

    setSaving(true);
    try {
      const payload = {
        name: form.name.trim(),
        slug: slugify(form.name),
        rank_order: form.rankOrder,
        color_hex: form.colorHex,
        icon_key: form.iconKey,
        is_active: form.isActive,
      };
      if (form.id) {
        await updateAward(form.id, payload);
        toast.success("Đã cập nhật giải thưởng.");
      } else {
        await createAward(payload);
        toast.success("Đã thêm giải thưởng mới.");
      }
      resetForm();
      load();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Không lưu được giải thưởng");
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete() {
    if (!deleteTarget) return;
    setDeleting(true);
    try {
      await deleteAward(deleteTarget.id);
      toast.success("Đã xoá giải thưởng.");
      setDeleteTarget(null);
      if (form.id === deleteTarget.id) resetForm();
      load();
    } catch (err) {
      const message = err instanceof Error ? err.message : "Không xoá được giải thưởng";
      toast.error(message);
    } finally {
      setDeleting(false);
    }
  }

  const sorted = [...awards].sort((a, b) => a.rank_order - b.rank_order);

  return (
    <div className="awards-page">
      <div className="artworks-list-header">
        <h1>Quản lý giải thưởng</h1>
      </div>

      <div className="awards-layout">
        <div className="awards-list">
          {loading ? (
            <div className="admin-page-placeholder">Đang tải…</div>
          ) : sorted.length === 0 ? (
            <div className="admin-page-placeholder">Chưa có giải thưởng nào. Thêm giải đầu tiên ở form bên phải.</div>
          ) : (
            sorted.map((award) => {
              const Icon = iconFor(award.icon_key);
              return (
                <div key={award.id} className={`award-card${award.is_active ? "" : " award-card--inactive"}`}>
                  <span className="award-card-icon" style={{ background: award.color_hex }}>
                    <Icon size={20} color="#fff" />
                  </span>
                  <div className="award-card-info">
                    <strong>{award.name}</strong>
                    <span>Thứ tự #{award.rank_order}{!award.is_active ? " · Đã ẩn" : ""}</span>
                  </div>
                  <div className="award-card-actions">
                    <button type="button" className="artworks-icon-btn" title="Sửa" onClick={() => startEdit(award)}>
                      <Pencil size={16} />
                    </button>
                    <button
                      type="button"
                      className="artworks-icon-btn artworks-icon-btn--danger"
                      title="Xoá"
                      onClick={() => setDeleteTarget(award)}
                    >
                      <Trash2 size={16} />
                    </button>
                  </div>
                </div>
              );
            })
          )}
        </div>

        <form className="award-form" onSubmit={handleSubmit}>
          <h2>{form.id ? "Sửa giải thưởng" : "Thêm giải thưởng"}</h2>

          <div className="artwork-meta-form">
            <div className="form-field form-field--full">
              <label htmlFor="award-name">Tên giải</label>
              <input
                id="award-name"
                type="text"
                placeholder="Ví dụ: Giải Nhất"
                value={form.name}
                onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
              />
            </div>

            <div className="form-field">
              <label htmlFor="award-rank">Thứ tự hiển thị</label>
              <input
                id="award-rank"
                type="number"
                min={0}
                value={form.rankOrder}
                onChange={(e) => setForm((f) => ({ ...f, rankOrder: Number(e.target.value) }))}
              />
            </div>

            <div className="form-field">
              <label htmlFor="award-color">Màu sắc</label>
              <input
                id="award-color"
                type="color"
                className="award-color-input"
                value={form.colorHex}
                onChange={(e) => setForm((f) => ({ ...f, colorHex: e.target.value }))}
              />
            </div>

            <div className="form-field form-field--full">
              <label>Biểu tượng</label>
              <div className="award-icon-picker">
                {ICON_OPTIONS.map(({ key, label, Icon }) => (
                  <button
                    key={key}
                    type="button"
                    className={`award-icon-option${form.iconKey === key ? " award-icon-option--active" : ""}`}
                    title={label}
                    onClick={() => setForm((f) => ({ ...f, iconKey: key }))}
                  >
                    <Icon size={18} />
                  </button>
                ))}
              </div>
            </div>

            <div className="form-field form-field--full">
              <label className="award-active-toggle">
                <input
                  type="checkbox"
                  checked={form.isActive}
                  onChange={(e) => setForm((f) => ({ ...f, isActive: e.target.checked }))}
                />
                Hiển thị công khai
              </label>
            </div>
          </div>

          <div className="award-form-actions">
            {form.id && (
              <button type="button" className="btn btn-ghost" onClick={resetForm} disabled={saving}>
                Huỷ sửa
              </button>
            )}
            <button type="submit" className="btn btn-primary" disabled={saving}>
              <Plus size={16} /> {saving ? "Đang lưu…" : form.id ? "Cập nhật" : "Thêm giải"}
            </button>
          </div>
        </form>
      </div>

      <ConfirmDialog
        open={Boolean(deleteTarget)}
        title="Xoá giải thưởng này?"
        message={`"${deleteTarget?.name ?? ""}" sẽ bị xoá. Nếu giải đang gắn cho tác phẩm nào, thao tác sẽ bị từ chối.`}
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
