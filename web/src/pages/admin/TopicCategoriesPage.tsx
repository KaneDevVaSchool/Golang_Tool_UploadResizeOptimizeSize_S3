import { AnimatePresence, Reorder, motion, useDragControls } from "framer-motion";
import { Check, GripVertical, Images, Layers, Pencil, Plus, Trash2, Users, X } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { AdminPageHeader } from "../../components/admin/AdminPageHeader";
import { RegionSummaryStrip } from "../../components/admin/RegionSummaryStrip";
import { ConfirmDialog } from "../../components/ConfirmDialog";
import { RequiredMark } from "../../components/admin/ArtworkMetaForm";
import { useRegionSummary } from "../../hooks/useRegionSummary";
import { TOPIC_CATEGORY_COLOR_PRESETS, TOPIC_CATEGORY_DEFAULT_COLOR } from "../../components/admin/topicCategoryColors";
import {
  createTopicCategory,
  deleteTopicCategory,
  fetchTopicCategories,
  updateTopicCategory,
  type TopicCategory,
} from "../../lib/topicCategoryApi";
import { toast } from "../../lib/toastBus";

function slugify(name: string): string {
  return name
    .toLowerCase()
    .normalize("NFD")
    .replace(/[̀-ͯ]/g, "")
    .replace(/đ/g, "d")
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
}

type LevelGroup = "" | "primary" | "secondary";

const LEVEL_GROUPS: { key: LevelGroup; label: string }[] = [
  { key: "", label: "Mọi cấp học" },
  { key: "primary", label: "Tiểu học" },
  { key: "secondary", label: "Trung học" },
];

type FormState = {
  id: number | null;
  name: string;
  colorHex: string;
  educationLevel: LevelGroup;
  isActive: boolean;
};

const EMPTY_FORM: FormState = {
  id: null,
  name: "",
  colorHex: TOPIC_CATEGORY_DEFAULT_COLOR,
  educationLevel: "",
  isActive: true,
};

/**
 * Trang quản lý nhóm chủ đề sáng tạo - cùng mẫu AwardsPage (danh sách kéo-thả
 * bên trái, form bên phải). Khác Award ở chỗ danh sách tách thành 3 khối theo
 * cấp học (Mọi cấp học / Tiểu học / Trung học) và kéo-thả ĐỘC LẬP trong từng
 * khối: display_order chỉ có ý nghĩa so sánh giữa các nhóm CÙNG cấp học, vì
 * `ArtworkMetaForm` lọc theo education_level trước rồi mới sắp theo thứ tự -
 * gộp chung 1 danh sách kéo-thả sẽ cho ra con số không phản ánh đúng thứ tự
 * thực tế hiển thị ở dropdown.
 */
export default function TopicCategoriesPage() {
  const [categories, setCategories] = useState<TopicCategory[]>([]);
  const [loading, setLoading] = useState(true);
  const [reorderingGroup, setReorderingGroup] = useState<LevelGroup | null>(null);
  const [form, setForm] = useState<FormState>(EMPTY_FORM);
  const [formOpen, setFormOpen] = useState(false);
  const [nameTouched, setNameTouched] = useState(false);
  const [saving, setSaving] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<TopicCategory | null>(null);
  const [deleting, setDeleting] = useState(false);

  const regionSummary = useRegionSummary();

  // Bản nháp thứ tự đang kéo, theo từng nhóm cấp học - cùng lý do tách khỏi
  // `categories` như AwardsPage: chỉ ghi API khi buông tay và thứ tự thực sự đổi.
  const [groupOrders, setGroupOrders] = useState<Record<LevelGroup, TopicCategory[]>>({
    "": [],
    primary: [],
    secondary: [],
  });
  const savedOrderIds = useRef<Record<LevelGroup, string>>({ "": "", primary: "", secondary: "" });
  const groupOrdersRef = useRef(groupOrders);
  groupOrdersRef.current = groupOrders;

  const load = useMemo(
    () => (signal?: AbortSignal) => {
      setLoading(true);
      fetchTopicCategories(true, signal)
        .then((list) => {
          const sorted = [...(list ?? [])].sort((a, b) => a.display_order - b.display_order);
          setCategories(sorted);
          const grouped: Record<LevelGroup, TopicCategory[]> = { "": [], primary: [], secondary: [] };
          for (const c of sorted) grouped[(c.education_level ?? "") as LevelGroup].push(c);
          setGroupOrders(grouped);
          for (const key of Object.keys(grouped) as LevelGroup[]) {
            savedOrderIds.current[key] = grouped[key].map((c) => c.id).join(",");
          }
        })
        .catch((err: unknown) => {
          if (signal?.aborted) return;
          toast.error(err instanceof Error ? err.message : "Không tải được danh sách nhóm chủ đề");
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

  function startEdit(category: TopicCategory) {
    setForm({
      id: category.id,
      name: category.name,
      colorHex: category.color_hex || TOPIC_CATEGORY_DEFAULT_COLOR,
      educationLevel: (category.education_level ?? "") as LevelGroup,
      isActive: category.is_active,
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

  const nameError = form.name.trim() ? undefined : "Bắt buộc nhập tên nhóm chủ đề.";

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (nameError) {
      setNameTouched(true);
      toast.error(nameError);
      return;
    }

    setSaving(true);
    const group = form.educationLevel;
    try {
      if (form.id) {
        const existing = categories.find((c) => c.id === form.id);
        const existingGroup = (existing?.education_level ?? "") as LevelGroup;
        // Đổi sang cấp học khác thì xếp cuối nhóm mới - display_order cũ
        // thuộc về nhóm cũ, giữ nguyên số dễ trùng vị trí với nhóm đang
        // chuyển đến (mỗi nhóm cấp học đánh số 0..n độc lập).
        const displayOrder =
          existingGroup === group ? existing?.display_order ?? 0 : groupOrders[group].length;
        await updateTopicCategory(form.id, {
          name: form.name.trim(),
          slug: slugify(form.name),
          color_hex: form.colorHex,
          education_level: group === "" ? null : group,
          display_order: displayOrder,
          is_active: form.isActive,
        });
        toast.success("Đã cập nhật nhóm chủ đề.");
      } else {
        // Nhóm mới xếp cuối khối cấp học tương ứng - kéo-thả lại được ngay
        // sau khi tạo nếu muốn thứ tự khác.
        await createTopicCategory({
          name: form.name.trim(),
          slug: slugify(form.name),
          color_hex: form.colorHex,
          education_level: group === "" ? null : group,
          display_order: groupOrders[group].length,
          is_active: form.isActive,
        });
        toast.success("Đã thêm nhóm chủ đề mới.");
      }
      closeForm();
      load();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Không lưu được nhóm chủ đề");
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete() {
    if (!deleteTarget) return;
    setDeleting(true);
    try {
      await deleteTopicCategory(deleteTarget.id);
      toast.success("Đã xoá nhóm chủ đề.");
      setDeleteTarget(null);
      if (form.id === deleteTarget.id) closeForm();
      load();
    } catch (err) {
      const message = err instanceof Error ? err.message : "Không xoá được nhóm chủ đề";
      toast.error(message);
    } finally {
      setDeleting(false);
    }
  }

  async function commitOrder(group: LevelGroup, next: TopicCategory[]) {
    setGroupOrders((prev) => ({ ...prev, [group]: next }));
    const nextIds = next.map((c) => c.id).join(",");
    if (nextIds === savedOrderIds.current[group]) return;

    const changed = next
      .map((category, index) => ({ category, index }))
      .filter(({ category, index }) => category.display_order !== index);
    if (changed.length === 0) {
      savedOrderIds.current[group] = nextIds;
      return;
    }

    setReorderingGroup(group);
    savedOrderIds.current[group] = nextIds;
    try {
      await Promise.all(
        changed.map(({ category, index }) =>
          updateTopicCategory(category.id, {
            name: category.name,
            slug: category.slug,
            color_hex: category.color_hex,
            education_level: category.education_level ?? null,
            display_order: index,
            is_active: category.is_active,
          }),
        ),
      );
      setCategories((prev) =>
        prev.map((c) => {
          const idx = next.findIndex((n) => n.id === c.id);
          return idx >= 0 && (c.education_level ?? "") === group ? { ...c, display_order: idx } : c;
        }),
      );
      toast.success("Đã cập nhật thứ tự hiển thị.");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Không lưu được thứ tự mới");
      load();
    } finally {
      setReorderingGroup(null);
    }
  }

  function moveByKeyboard(group: LevelGroup, index: number, direction: -1 | 1) {
    const current = groupOrdersRef.current[group];
    const target = index + direction;
    if (target < 0 || target >= current.length) return;
    const next = [...current];
    [next[index], next[target]] = [next[target], next[index]];
    commitOrder(group, next);
  }

  return (
    <div className="awards-page">
      <AdminPageHeader
        title="Nhóm chủ đề sáng tạo"
        subtitle={categories.length > 0 ? `${categories.length} nhóm chủ đề theo thể lệ hội thi · kéo để đổi thứ tự` : undefined}
        primaryAction={{ label: "Thêm nhóm chủ đề", icon: Plus, onClick: startCreate, iconOnly: true }}
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
          ) : categories.length === 0 ? (
            <div className="admin-page-placeholder awards-empty">
              <Layers size={28} strokeWidth={1.5} />
              <p>Chưa có nhóm chủ đề nào.</p>
              <button type="button" className="btn btn-primary" onClick={startCreate}>
                <Plus size={16} /> Tạo nhóm chủ đề đầu tiên
              </button>
            </div>
          ) : (
            <div className="topic-category-groups">
              {LEVEL_GROUPS.map(({ key, label }) => {
                const order = groupOrders[key];
                if (order.length === 0) return null;
                return (
                  <div key={key || "all"} className="topic-category-group">
                    <h3 className="topic-category-group-title">{label}</h3>
                    <Reorder.Group
                      as="div"
                      axis="y"
                      values={order}
                      onReorder={(next) => setGroupOrders((prev) => ({ ...prev, [key]: next }))}
                      className={`awards-list${reorderingGroup === key ? " awards-list--busy" : ""}`}
                    >
                      {order.map((category, index) => (
                        <TopicCategoryRow
                          key={category.id}
                          category={category}
                          isFirst={index === 0}
                          isLast={index === order.length - 1}
                          onDragEnd={() => commitOrder(key, groupOrdersRef.current[key])}
                          onMove={(dir) => moveByKeyboard(key, index, dir)}
                          onEdit={() => startEdit(category)}
                          onDelete={() => setDeleteTarget(category)}
                        />
                      ))}
                    </Reorder.Group>
                  </div>
                );
              })}
            </div>
          )}
        </div>

        <AnimatePresence>
          {formOpen && (
            <TopicCategoryFormPanel
              form={form}
              setForm={setForm}
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
        title="Xoá nhóm chủ đề này?"
        message={`"${deleteTarget?.name ?? ""}" sẽ bị xoá. Nếu nhóm đang được gắn cho tác phẩm nào, thao tác sẽ bị từ chối.`}
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

function TopicCategoryRow({
  category,
  isFirst,
  isLast,
  onDragEnd,
  onMove,
  onEdit,
  onDelete,
}: {
  category: TopicCategory;
  isFirst: boolean;
  isLast: boolean;
  onDragEnd: () => void;
  onMove: (direction: -1 | 1) => void;
  onEdit: () => void;
  onDelete: () => void;
}) {
  const controls = useDragControls();

  return (
    <Reorder.Item
      value={category}
      dragListener={false}
      dragControls={controls}
      onDragEnd={onDragEnd}
      className={`award-card${category.is_active ? "" : " award-card--inactive"}`}
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

      <span className="award-card-icon" style={{ background: category.color_hex || TOPIC_CATEGORY_DEFAULT_COLOR }}>
        <Layers size={20} color="#fff" />
      </span>

      <div className="award-card-info">
        <strong>{category.name}</strong>
        <span>
          {!category.is_active && <span className="award-hidden-tag">Đã ẩn</span>}
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

function TopicCategoryFormPanel({
  form,
  setForm,
  saving,
  nameError,
  onNameBlur,
  onSubmit,
  onCancel,
}: {
  form: FormState;
  setForm: React.Dispatch<React.SetStateAction<FormState>>;
  saving: boolean;
  nameError?: string;
  onNameBlur: () => void;
  onSubmit: (e: React.FormEvent) => void;
  onCancel: () => void;
}) {
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
        <h2>{form.id ? "Sửa nhóm chủ đề" : "Thêm nhóm chủ đề"}</h2>
        <button type="button" className="award-form-close" title="Đóng" aria-label="Đóng" onClick={onCancel}>
          <X size={16} />
        </button>
      </div>

      <div className="award-form-preview">
        <span className="award-card-icon award-card-icon--lg" style={{ background: form.colorHex }}>
          <Layers size={24} color="#fff" />
        </span>
        <div>
          <strong>{form.name.trim() || "Tên nhóm chủ đề"}</strong>
          <span>{form.name.trim() ? `/${slugify(form.name)}` : "đường dẫn tự sinh từ tên"}</span>
        </div>
      </div>

      <div className="artwork-meta-form">
        <div className={`form-field form-field--full${nameError ? " form-field--error" : ""}`}>
          <label htmlFor="topic-category-name">
            Tên nhóm chủ đề
            <RequiredMark />
          </label>
          <input
            id="topic-category-name"
            type="text"
            placeholder="Ví dụ: Trí tưởng tượng &amp; thế giới thần tiên"
            value={form.name}
            aria-required="true"
            aria-invalid={nameError ? "true" : undefined}
            aria-describedby={nameError ? "topic-category-name-error" : undefined}
            onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
            onBlur={onNameBlur}
            autoFocus
          />
          {nameError && (
            <p className="form-field-error-text" id="topic-category-name-error">
              {nameError}
            </p>
          )}
        </div>

        <div className="form-field form-field--full">
          <label htmlFor="topic-category-color">Màu sắc</label>
          <div className="award-color-row">
            {TOPIC_CATEGORY_COLOR_PRESETS.map((hex) => (
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
              id="topic-category-color"
              type="color"
              className="award-color-input"
              value={form.colorHex}
              onChange={(e) => setForm((f) => ({ ...f, colorHex: e.target.value }))}
              title="Chọn màu tuỳ ý"
            />
          </div>
        </div>

        <div className="form-field form-field--full">
          <label htmlFor="topic-category-level">Cấp học áp dụng</label>
          <select
            id="topic-category-level"
            value={form.educationLevel}
            onChange={(e) => setForm((f) => ({ ...f, educationLevel: e.target.value as LevelGroup }))}
          >
            <option value="">Mọi cấp học (không tách riêng)</option>
            <option value="primary">Tiểu học</option>
            <option value="secondary">Trung học (THCS &amp; THPT)</option>
          </select>
          <p className="form-field-hint">
            Thứ tự hiển thị điều khiển bằng kéo-thả trong danh sách, riêng theo từng cấp học đã chọn ở đây.
          </p>
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
          {saving ? "Đang lưu…" : form.id ? "Cập nhật" : "Thêm nhóm chủ đề"}
        </button>
      </div>
    </motion.form>
  );
}
