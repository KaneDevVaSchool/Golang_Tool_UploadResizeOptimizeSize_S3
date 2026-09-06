import { AnimatePresence, motion } from "framer-motion";
import { useEffect, useState } from "react";
import { createPortal } from "react-dom";
import { ArtworkMetaForm, EMPTY_META_FORM_VALUES, type ArtworkMetaFormValues, isMetaFormValid } from "./ArtworkMetaForm";
import type { ArtworkWithMeta, Award, GradeLevel, School } from "../../lib/artworkApi";
import type { TopicCategory } from "../../lib/topicCategoryApi";
import { artworkImageURL } from "../../lib/artworkImage";

type ArtworkEditModalProps = {
  open: boolean;
  artwork: ArtworkWithMeta | null;
  schools: School[];
  gradeLevels: GradeLevel[];
  topicCategories: TopicCategory[];
  onTopicCategoryCreated?: (category: TopicCategory) => void;
  awards: Award[];
  busy?: boolean;
  onSave: (values: ArtworkMetaFormValues) => void;
  onClose: () => void;
};

function toFormValues(artwork: ArtworkWithMeta, gradeLevels: GradeLevel[]): ArtworkMetaFormValues {
  const grade = gradeLevels.find((g) => g.id === artwork.grade_level_id);
  return {
    title: artwork.title,
    studentName: artwork.student_name,
    schoolId: artwork.school_id,
    educationLevel: grade?.education_level ?? "",
    gradeLevelId: artwork.grade_level_id,
    topicCategoryId: artwork.topic_category_id ?? "",
    className: artwork.class_name ?? "",
    awardIds: artwork.awards?.map((a) => a.id) ?? [],
  };
}

/**
 * Modal edit tác phẩm - form ngang 2 cột (ảnh preview trái full-height,
 * fields phải), tham khảo pattern SocialGroupFormModal.vue (va-workspace):
 * CSS Grid grid-template-areas, collapse 1 cột dưới 768px.
 */
export function ArtworkEditModal({
  open,
  artwork,
  schools,
  gradeLevels,
  topicCategories,
  onTopicCategoryCreated,
  awards,
  busy,
  onSave,
  onClose,
}: ArtworkEditModalProps) {
  const [values, setValues] = useState<ArtworkMetaFormValues>(EMPTY_META_FORM_VALUES);
  // Theo dõi artwork nào đã đồng bộ vào `values`, để phát hiện đổi tác phẩm
  // (đóng modal rồi bấm Sửa tác phẩm khác - component không unmount vì luôn
  // render, chỉ ẩn hiện qua `open`).
  const [syncedId, setSyncedId] = useState<number | null>(null);

  // Đồng bộ `values` NGAY TRONG lúc render (không phải useEffect) khi
  // artwork đổi: nếu chờ effect chạy sau render, có một nhịp render với
  // `values` cũ (rỗng hoặc của tác phẩm trước) trong khi `artwork` đã là
  // tác phẩm mới - isMetaFormValid() bên dưới đọc `values.title.trim()` lúc
  // đó sẽ vỡ vì field rỗng không phải string. Đây là pattern React chính
  // thức cho "đổi state theo prop" (adjusting state during render).
  if (artwork && artwork.id !== syncedId) {
    setSyncedId(artwork.id);
    setValues(toFormValues(artwork, gradeLevels));
  }

  useEffect(() => {
    if (!open) return;
    function onKey(e: KeyboardEvent) {
      if (e.key === "Escape" && !busy) onClose();
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open, busy, onClose]);

  if (!artwork) return null;

  const valid = isMetaFormValid(values);

  // Portal ra document.body: modal render bên trong .admin-content-inner,
  // vùng này mang class "route-enter" (will-change: transform khi chuyển
  // trang) - will-change: transform biến ancestor thành containing block
  // mới cho position:fixed, nên nếu không portal, modal bị "nhốt" trong
  // khung cuộn nội dung thay vì phủ toàn viewport (tràn đáy, bị cắt).
  return createPortal(
    <AnimatePresence>
      {open && (
        <div className="artwork-modal-root" role="presentation">
          <motion.button
            type="button"
            className="artwork-modal-backdrop"
            aria-label="Đóng"
            disabled={busy}
            onClick={onClose}
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
          />
          <motion.div
            className="artwork-modal-panel"
            role="dialog"
            aria-modal="true"
            aria-labelledby="artwork-modal-title"
            initial={{ opacity: 0, y: 24, scale: 0.97 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: 16, scale: 0.97 }}
            transition={{ duration: 0.26, ease: [0.22, 1, 0.36, 1] }}
          >
            <header className="artwork-modal-header">
              <h2 id="artwork-modal-title">Chỉnh sửa tác phẩm</h2>
              <button type="button" className="artwork-modal-close" aria-label="Đóng" onClick={onClose} disabled={busy}>
                ×
              </button>
            </header>

            <div className="artwork-modal-body">
              <div className="artwork-modal-grid">
                <div className="artwork-modal-preview">
                  <img src={artworkImageURL(artwork, "medium")} alt={artwork.title} />
                </div>
                <div className="artwork-modal-fields">
                  <ArtworkMetaForm
                    values={values}
                    onChange={setValues}
                    schools={schools}
                    gradeLevels={gradeLevels}
                    topicCategories={topicCategories}
                    onTopicCategoryCreated={onTopicCategoryCreated}
                    awards={awards}
                    disabled={busy}
                  />
                </div>
              </div>
            </div>

            <footer className="artwork-modal-footer">
              <button type="button" className="btn btn-ghost" onClick={onClose} disabled={busy}>
                Huỷ
              </button>
              <button
                type="button"
                className="btn btn-primary"
                disabled={busy || !valid}
                onClick={() => onSave(values)}
              >
                {busy ? "Đang lưu…" : "Lưu thay đổi"}
              </button>
            </footer>
          </motion.div>
        </div>
      )}
    </AnimatePresence>,
    document.body,
  );
}
