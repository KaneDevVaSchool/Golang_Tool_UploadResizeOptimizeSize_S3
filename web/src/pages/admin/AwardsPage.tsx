import { AnimatePresence, motion, Reorder, useDragControls } from "framer-motion";
import {
  Award as AwardIcon,
  Check,
  EyeOff,
  GripVertical,
  Images,
  Medal,
  Pencil,
  Plus,
  Star,
  Trash2,
  Trophy,
  Users,
  X,
} from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { AdminPageHeader } from "../../components/admin/AdminPageHeader";
import { RegionSummaryStrip } from "../../components/admin/RegionSummaryStrip";
import { ConfirmDialog } from "../../components/ConfirmDialog";
import { RequiredMark } from "../../components/admin/ArtworkMetaForm";
import { useRegionSummary } from "../../hooks/useRegionSummary";
import { createAward, deleteAward, fetchAwards, updateAward, type Award } from "../../lib/awardApi";
import { fetchGradeLevels, type GradeLevel } from "../../lib/artworkApi";
import { toast } from "../../lib/toastBus";

const ICON_OPTIONS: { key: string; label: string; Icon: typeof Trophy }[] = [
  { key: "trophy", label: "Cúp", Icon: Trophy },
  { key: "medal", label: "Huy chương", Icon: Medal },
  { key: "star", label: "Ngôi sao", Icon: Star },
  { key: "award", label: "Huy hiệu", Icon: AwardIcon },
];

const COLOR_PRESETS = [
  "#c49c57",
  "#9a0036",
  "#1750b5",
  "#009082",
  "#725139",
  "#a67f42",
  "#2e7d32",
  "#d81b60",
  "#5e35b1",
  "#ef6c00",
  "#00838f",
  "#455a64",
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

/** Nhãn hạng theo vị trí (1-based) - không dùng "#N" trần trụi. */
function rankLabel(position: number): string {
  if (position === 1) return "Hạng nhất";
  if (position === 2) return "Hạng nhì";
  if (position === 3) return "Hạng ba";
  return `Hạng ${position}`;
}

type FormState = {
  id: number | null;
  name: string;
  colorHex: string;
  iconKey: string;
  isActive: boolean;
  /** Khối lớp riêng của giải (vd Tiểu học chia giải theo từng khối 1-5).
   * "" = giải dùng chung toàn hệ thống, không tách khối. */
  gradeLevelId: string;
};

const EMPTY_FORM: FormState = {
  id: null,
  name: "",
  colorHex: "#c49c57",
  iconKey: "trophy",
  isActive: true,
  gradeLevelId: "",
};

/**
 * Trang quản lý giải thưởng. Thứ tự hiển thị điều khiển bằng kéo-thả
 * (framer-motion `Reorder`) thay vì ô nhập số: vị trí trong danh sách MỚI
 * là dữ liệu, `rank_order` chỉ là hệ quả tính ra khi lưu. Người dùng không
 * còn phải tự nghĩ ra một con số rồi tránh trùng với giải khác.
 *
 * Card không còn hiển thị "#rank_order" - thay bằng nhãn hạng ("Hạng nhất"/
 * "Hạng 4") và một huy hiệu số thứ tự đã tạo dáng thay vì ký tự "#".
 */
export default function AwardsPage() {
  const [awards, setAwards] = useState<Award[]>([]);
  const [gradeLevels, setGradeLevels] = useState<GradeLevel[]>([]);
  const [loading, setLoading] = useState(true);
  const [reordering, setReordering] = useState(false);
  const [form, setForm] = useState<FormState>(EMPTY_FORM);
  const [formOpen, setFormOpen] = useState(false);
  const [nameTouched, setNameTouched] = useState(false);
  const [saving, setSaving] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<Award | null>(null);
  const [deleting, setDeleting] = useState(false);

  const regionSummary = useRegionSummary();

  // Bản nháp thứ tự đang kéo - chỉ ghi lên server khi người dùng buông và
  // thứ tự thực sự đổi, tránh gọi API dồn dập theo từng khung hình kéo.
  const [order, setOrder] = useState<Award[]>([]);
  const savedOrderIds = useRef<string>("");
  // onReorder bắn liên tục trong lúc kéo nên `order` ở closure của
  // onDragEnd (tạo lúc render trước khi kéo bắt đầu) đã cũ - đọc qua ref
  // để luôn lấy đúng vị trí tại thời điểm buông tay.
  const orderRef = useRef<Award[]>([]);
  orderRef.current = order;

  const load = useMemo(
    () => (signal?: AbortSignal) => {
      setLoading(true);
      fetchAwards(true, signal)
        .then((list) => {
          const sorted = [...(list ?? [])].sort((a, b) => a.rank_order - b.rank_order);
          setAwards(sorted);
          setOrder(sorted);
          savedOrderIds.current = sorted.map((a) => a.id).join(",");
        })
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
    fetchGradeLevels(controller.signal)
      .then(setGradeLevels)
      .catch((err: unknown) => {
        if (controller.signal.aborted) return;
        toast.error(err instanceof Error ? err.message : "Không tải được danh sách khối lớp");
      });
    return () => controller.abort();
  }, [load]);

  function startEdit(award: Award) {
    setForm({
      id: award.id,
      name: award.name,
      colorHex: award.color_hex,
      iconKey: award.icon_key ?? "trophy",
      isActive: award.is_active,
      gradeLevelId: award.grade_level_id ? String(award.grade_level_id) : "",
    });
    setNameTouched(false);
    setFormOpen(true);
  }

  function startCreate() {
    setForm(EMPTY_FORM);
    setNameTouched(false);
    setFormOpen(true);
  }

  function closeForm() {
    setFormOpen(false);
    setForm(EMPTY_FORM);
    setNameTouched(false);
  }

  const nameError = form.name.trim() ? undefined : "Bắt buộc nhập tên giải thưởng.";

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (nameError) {
      setNameTouched(true);
      toast.error(nameError);
      return;
    }

    setSaving(true);
    const gradeLevelId = form.gradeLevelId ? Number(form.gradeLevelId) : null;
    try {
      if (form.id) {
        const existing = awards.find((a) => a.id === form.id);
        await updateAward(form.id, {
          name: form.name.trim(),
          slug: slugify(form.name),
          grade_level_id: gradeLevelId,
          rank_order: existing?.rank_order ?? 0,
          color_hex: form.colorHex,
          icon_key: form.iconKey,
          is_active: form.isActive,
        });
        toast.success("Đã cập nhật giải thưởng.");
      } else {
        // Giải mới xếp cuối danh sách hiện tại theo mặc định - vẫn kéo thả
        // lại được ngay sau khi tạo nếu muốn xếp hạng khác.
        await createAward({
          name: form.name.trim(),
          slug: slugify(form.name),
          grade_level_id: gradeLevelId,
          rank_order: awards.length,
          color_hex: form.colorHex,
          icon_key: form.iconKey,
          is_active: form.isActive,
        });
        toast.success("Đã thêm giải thưởng mới.");
      }
      closeForm();
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
      if (form.id === deleteTarget.id) closeForm();
      load();
    } catch (err) {
      const message = err instanceof Error ? err.message : "Không xoá được giải thưởng";
      toast.error(message);
    } finally {
      setDeleting(false);
    }
  }

  // Sau khi buông kéo: nếu thứ tự thực sự đổi so với lần lưu gần nhất, ghi
  // rank_order mới (0-based theo vị trí) cho từng giải bị dịch chuyển.
  async function commitOrder(next: Award[]) {
    setOrder(next);
    const nextIds = next.map((a) => a.id).join(",");
    if (nextIds === savedOrderIds.current) return;

    const changed = next
      .map((award, index) => ({ award, index }))
      .filter(({ award, index }) => award.rank_order !== index);
    if (changed.length === 0) {
      savedOrderIds.current = nextIds;
      return;
    }

    setReordering(true);
    savedOrderIds.current = nextIds;
    try {
      await Promise.all(
        changed.map(({ award, index }) =>
          updateAward(award.id, {
            name: award.name,
            slug: award.slug,
            grade_level_id: award.grade_level_id ?? null,
            rank_order: index,
            color_hex: award.color_hex,
            icon_key: award.icon_key,
            is_active: award.is_active,
          }),
        ),
      );
      setAwards(next.map((award, index) => ({ ...award, rank_order: index })));
      toast.success("Đã cập nhật thứ tự hiển thị.");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Không lưu được thứ tự mới");
      load();
    } finally {
      setReordering(false);
    }
  }

  /** Dịch một giải lên/xuống bằng bàn phím - không bắt buộc phải kéo chuột. */
  function moveByKeyboard(index: number, direction: -1 | 1) {
    const target = index + direction;
    if (target < 0 || target >= order.length) return;
    const next = [...order];
    [next[index], next[target]] = [next[target], next[index]];
    commitOrder(next);
  }

  return (
    <div className="awards-page">
      <AdminPageHeader
        title="Giải thưởng"
        subtitle={awards.length > 0 ? `${awards.length} hạng mục vinh danh · kéo để đổi thứ tự` : undefined}
        primaryAction={{ label: "Thêm giải thưởng", icon: Plus, onClick: startCreate, iconOnly: true }}
      />

      <RegionSummaryStrip
        data={regionSummary.data}
        loading={regionSummary.loading}
        artworkIcon={Images}
        studentIcon={Users}
      />

      <div className="awards-layout">
        <div className="awards-list-panel">
          {loading ? (
            <div className="awards-skeleton">
              {[0, 1, 2].map((i) => (
                <div key={i} className="award-card-skeleton" />
              ))}
            </div>
          ) : order.length === 0 ? (
            <div className="admin-page-placeholder awards-empty">
              <Trophy size={28} strokeWidth={1.5} />
              <p>Chưa có hạng mục nào.</p>
              <button type="button" className="btn btn-primary" onClick={startCreate}>
                <Plus size={16} /> Tạo giải thưởng đầu tiên
              </button>
            </div>
          ) : (
            <Reorder.Group
              as="div"
              axis="y"
              values={order}
              onReorder={setOrder}
              className={`awards-list${reordering ? " awards-list--busy" : ""}`}
            >
              {order.map((award, index) => (
                <AwardRow
                  key={award.id}
                  award={award}
                  gradeLevels={gradeLevels}
                  position={index + 1}
                  isFirst={index === 0}
                  isLast={index === order.length - 1}
                  onDragEnd={() => commitOrder(orderRef.current)}
                  onMove={(dir) => moveByKeyboard(index, dir)}
                  onEdit={() => startEdit(award)}
                  onDelete={() => setDeleteTarget(award)}
                />
              ))}
            </Reorder.Group>
          )}
        </div>

        <AnimatePresence>
          {formOpen && (
            <AwardFormPanel
              form={form}
              setForm={setForm}
              gradeLevels={gradeLevels}
              saving={saving}
              nameError={nameTouched ? nameError : undefined}
              onNameBlur={() => setNameTouched(true)}
              onSubmit={handleSubmit}
              onCancel={closeForm}
            />
          )}
        </AnimatePresence>
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

function AwardRow({
  award,
  gradeLevels,
  position,
  isFirst,
  isLast,
  onDragEnd,
  onMove,
  onEdit,
  onDelete,
}: {
  award: Award;
  gradeLevels: GradeLevel[];
  position: number;
  isFirst: boolean;
  isLast: boolean;
  onDragEnd: () => void;
  onMove: (direction: -1 | 1) => void;
  onEdit: () => void;
  onDelete: () => void;
}) {
  const Icon = iconFor(award.icon_key);
  const controls = useDragControls();
  const gradeLabel = award.grade_level_id
    ? gradeLevels.find((g) => g.id === award.grade_level_id)?.label
    : undefined;

  return (
    <Reorder.Item
      value={award}
      dragListener={false}
      dragControls={controls}
      onDragEnd={onDragEnd}
      className={`award-card${award.is_active ? "" : " award-card--inactive"}`}
    >
      <button
        type="button"
        className="award-card-handle"
        title="Kéo để đổi thứ tự"
        aria-label="Kéo để đổi thứ tự"
        onPointerDown={(e) => controls.start(e)}
      >
        <GripVertical size={16} />
      </button>

      <span className={`award-rank-badge${position <= 3 ? ` award-rank-badge--top${position}` : ""}`}>
        {position}
      </span>

      <span className="award-card-icon" style={{ background: award.color_hex }}>
        <Icon size={20} color="#fff" />
      </span>

      <div className="award-card-info">
        <strong>{award.name}</strong>
        <span>
          {rankLabel(position)}
          {gradeLabel && <span className="award-grade-tag">{gradeLabel}</span>}
          {!award.is_active && (
            <span className="award-hidden-tag">
              <EyeOff size={11} /> Đã ẩn
            </span>
          )}
        </span>
      </div>

      <div className="award-card-reorder">
        <button
          type="button"
          className="artworks-icon-btn"
          title="Đưa lên trên"
          disabled={isFirst}
          onClick={() => onMove(-1)}
        >
          ▲
        </button>
        <button
          type="button"
          className="artworks-icon-btn"
          title="Đưa xuống dưới"
          disabled={isLast}
          onClick={() => onMove(1)}
        >
          ▼
        </button>
      </div>

      <div className="award-card-actions">
        <button type="button" className="artworks-icon-btn" title="Sửa" onClick={onEdit}>
          <Pencil size={16} />
        </button>
        <button type="button" className="artworks-icon-btn artworks-icon-btn--danger" title="Xoá" onClick={onDelete}>
          <Trash2 size={16} />
        </button>
      </div>
    </Reorder.Item>
  );
}

function AwardFormPanel({
  form,
  setForm,
  gradeLevels,
  saving,
  nameError,
  onNameBlur,
  onSubmit,
  onCancel,
}: {
  form: FormState;
  setForm: React.Dispatch<React.SetStateAction<FormState>>;
  gradeLevels: GradeLevel[];
  saving: boolean;
  nameError?: string;
  onNameBlur: () => void;
  onSubmit: (e: React.FormEvent) => void;
  onCancel: () => void;
}) {
  const PreviewIcon = iconFor(form.iconKey);

  return (
    <motion.form
      className="award-form"
      onSubmit={onSubmit}
      initial={{ opacity: 0, y: 12, scale: 0.98 }}
      animate={{ opacity: 1, y: 0, scale: 1 }}
      exit={{ opacity: 0, y: 8, scale: 0.98 }}
      transition={{ duration: 0.2, ease: [0.22, 1, 0.36, 1] }}
    >
      <div className="award-form-head">
        <h2>{form.id ? "Sửa giải thưởng" : "Thêm giải thưởng"}</h2>
        <button type="button" className="award-form-close" title="Đóng" aria-label="Đóng" onClick={onCancel}>
          <X size={16} />
        </button>
      </div>

      <div className="award-form-preview">
        <span className="award-card-icon award-card-icon--lg" style={{ background: form.colorHex }}>
          <PreviewIcon size={24} color="#fff" />
        </span>
        <div>
          <strong>{form.name.trim() || "Tên giải thưởng"}</strong>
          <span>{form.name.trim() ? `/${slugify(form.name)}` : "đường dẫn tự sinh từ tên"}</span>
        </div>
      </div>

      <div className="artwork-meta-form">
        <div className={`form-field form-field--full${nameError ? " form-field--error" : ""}`}>
          <label htmlFor="award-name">
            Tên giải
            <RequiredMark />
          </label>
          <input
            id="award-name"
            type="text"
            placeholder="Ví dụ: Giải Nhất"
            value={form.name}
            aria-required="true"
            aria-invalid={nameError ? "true" : undefined}
            aria-describedby={nameError ? "award-name-error" : undefined}
            onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
            onBlur={onNameBlur}
            autoFocus
          />
          {nameError && (
            <p className="form-field-error-text" id="award-name-error">
              {nameError}
            </p>
          )}
        </div>

        <div className="form-field form-field--full">
          <label htmlFor="award-color">Màu sắc</label>
          <div className="award-color-row">
            {COLOR_PRESETS.map((hex) => (
              <button
                key={hex}
                type="button"
                className={`award-color-swatch${form.colorHex.toLowerCase() === hex ? " award-color-swatch--active" : ""}`}
                style={{ background: hex }}
                title={hex}
                aria-label={`Chọn màu ${hex}`}
                onClick={() => setForm((f) => ({ ...f, colorHex: hex }))}
              >
                {form.colorHex.toLowerCase() === hex && <Check size={14} color="#fff" />}
              </button>
            ))}
            <input
              id="award-color"
              type="color"
              className="award-color-input"
              value={form.colorHex}
              onChange={(e) => setForm((f) => ({ ...f, colorHex: e.target.value }))}
              title="Chọn màu tuỳ ý"
            />
          </div>
        </div>

        <div className="form-field form-field--full">
          <label htmlFor="award-grade-level">Khối lớp áp dụng</label>
          <select
            id="award-grade-level"
            value={form.gradeLevelId}
            onChange={(e) => setForm((f) => ({ ...f, gradeLevelId: e.target.value }))}
          >
            <option value="">Toàn hệ thống (không tách khối)</option>
            {gradeLevels.map((g) => (
              <option key={g.id} value={g.id}>
                {g.label}
              </option>
            ))}
          </select>
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
        <button type="button" className="btn btn-ghost" onClick={onCancel} disabled={saving}>
          Huỷ
        </button>
        <button type="submit" className="btn btn-primary" disabled={saving}>
          {saving ? "Đang lưu…" : form.id ? "Cập nhật" : "Thêm giải"}
        </button>
      </div>
    </motion.form>
  );
}
