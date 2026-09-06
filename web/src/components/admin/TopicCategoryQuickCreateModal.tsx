import { AnimatePresence, motion } from "framer-motion";
import { Layers, X } from "lucide-react";
import { useEffect, useState } from "react";
import { createPortal } from "react-dom";
import { createTopicCategory, type TopicCategory } from "../../lib/topicCategoryApi";
import { toast } from "../../lib/toastBus";
import { RequiredMark } from "./ArtworkMetaForm";

function slugify(name: string): string {
  return name
    .toLowerCase()
    .normalize("NFD")
    .replace(/[̀-ͯ]/g, "")
    .replace(/đ/g, "d")
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
}

/**
 * Modal tạo nhanh 1 nhóm chủ đề, mở từ nút "+" cạnh dropdown "Nhóm chủ đề
 * sáng tạo" trong ArtworkMetaForm - tránh việc admin phải rời trang đang
 * nhập tác phẩm để sang /admin/topic-categories chỉ vì thiếu 1 nhóm.
 *
 * Chỉ hỏi đúng thông tin cần để dùng ngay: tên + cấp học áp dụng (mặc định
 * lấy theo cấp học đang chọn trong form tác phẩm, sửa được). Thứ tự hiển thị
 * và trạng thái ẩn/hiện không hỏi ở đây - nhóm mới luôn active, xếp cuối
 * cùng cấp học; muốn tinh chỉnh thêm thì vào /admin/topic-categories.
 */
export function TopicCategoryQuickCreateModal({
  open,
  defaultEducationLevel,
  onCreated,
  onClose,
}: {
  open: boolean;
  defaultEducationLevel: "primary" | "secondary" | "";
  onCreated: (category: TopicCategory) => void;
  onClose: () => void;
}) {
  const [name, setName] = useState("");
  const [educationLevel, setEducationLevel] = useState<"" | "primary" | "secondary">("");
  const [touched, setTouched] = useState(false);
  const [saving, setSaving] = useState(false);

  // Mở lại modal (open chuyển false -> true) thì nạp lại cấp học mặc định
  // theo form cha đang chọn lúc đó, không giữ giá trị lần mở trước.
  useEffect(() => {
    if (open) {
      setName("");
      setEducationLevel(defaultEducationLevel);
      setTouched(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  useEffect(() => {
    if (!open) return;
    function onKey(e: KeyboardEvent) {
      if (e.key === "Escape" && !saving) onClose();
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open, saving, onClose]);

  const nameError = name.trim() ? undefined : "Bắt buộc nhập tên nhóm chủ đề.";

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (nameError) {
      setTouched(true);
      return;
    }
    setSaving(true);
    try {
      const created = await createTopicCategory({
        name: name.trim(),
        slug: slugify(name),
        education_level: educationLevel === "" ? null : educationLevel,
        // Xếp cuối là lựa chọn an toàn nhất ở đây - modal này không biết số
        // lượng nhóm đã có trong cùng cấp học (không tải sẵn danh sách đầy
        // đủ), nên để backend/DB mặc định 0 rồi admin tự sắp lại thứ tự sau
        // ở trang /admin/topic-categories nếu cần.
        display_order: 0,
        is_active: true,
      });
      toast.success("Đã thêm nhóm chủ đề mới.");
      onCreated(created);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Không tạo được nhóm chủ đề");
    } finally {
      setSaving(false);
    }
  }

  // Portal ra document.body - cùng lý do các modal khác trong dự án
  // (ConfirmDialog, ArtworkEditModal): tránh bị nhốt trong khung cuộn
  // .admin-content-inner khi nó đang có will-change: transform lúc chuyển route.
  return createPortal(
    <AnimatePresence>
      {open && (
        <div className="topic-quick-modal-root" role="presentation">
          <motion.button
            type="button"
            className="topic-quick-modal-backdrop"
            aria-label="Đóng"
            disabled={saving}
            onClick={onClose}
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
          />
          <motion.form
            className="award-form topic-quick-modal-card"
            onSubmit={handleSubmit}
            role="dialog"
            aria-modal="true"
            aria-labelledby="topic-quick-create-title"
            initial={{ opacity: 0, y: 16, scale: 0.96 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: 8, scale: 0.97 }}
            transition={{ duration: 0.22, ease: [0.22, 1, 0.36, 1] }}
          >
            <div className="award-form-head">
              <h2 id="topic-quick-create-title">Thêm nhanh nhóm chủ đề</h2>
              <button type="button" className="award-form-close" title="Đóng" aria-label="Đóng" onClick={onClose} disabled={saving}>
                <X size={16} />
              </button>
            </div>

            <div className="award-form-preview">
              <span className="award-card-icon award-card-icon--lg" style={{ background: "#725139" }}>
                <Layers size={24} color="#fff" />
              </span>
              <div>
                <strong>{name.trim() || "Tên nhóm chủ đề"}</strong>
                <span>{name.trim() ? `/${slugify(name)}` : "đường dẫn tự sinh từ tên"}</span>
              </div>
            </div>

            <div className="artwork-meta-form">
              <div className={`form-field form-field--full${touched && nameError ? " form-field--error" : ""}`}>
                <label htmlFor="topic-quick-create-name">
                  Tên nhóm chủ đề
                  <RequiredMark />
                </label>
                <input
                  id="topic-quick-create-name"
                  type="text"
                  placeholder="Ví dụ: Trí tưởng tượng &amp; thế giới thần tiên"
                  value={name}
                  aria-required="true"
                  aria-invalid={touched && nameError ? "true" : undefined}
                  onChange={(e) => setName(e.target.value)}
                  onBlur={() => setTouched(true)}
                  autoFocus
                  disabled={saving}
                />
                {touched && nameError && <p className="form-field-error-text">{nameError}</p>}
              </div>

              <div className="form-field form-field--full">
                <label htmlFor="topic-quick-create-level">Cấp học áp dụng</label>
                <select
                  id="topic-quick-create-level"
                  value={educationLevel}
                  disabled={saving}
                  onChange={(e) => setEducationLevel(e.target.value as "" | "primary" | "secondary")}
                >
                  <option value="">Mọi cấp học (không tách riêng)</option>
                  <option value="primary">Tiểu học</option>
                  <option value="secondary">Trung học (THCS &amp; THPT)</option>
                </select>
              </div>
            </div>

            <div className="award-form-actions">
              <button type="button" className="btn btn-ghost" onClick={onClose} disabled={saving}>
                Huỷ
              </button>
              <button type="submit" className="btn btn-primary" disabled={saving}>
                {saving ? "Đang tạo…" : "Tạo và dùng ngay"}
              </button>
            </div>
          </motion.form>
        </div>
      )}
    </AnimatePresence>,
    document.body,
  );
}
