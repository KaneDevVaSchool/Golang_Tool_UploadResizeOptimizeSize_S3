import { AnimatePresence, motion } from "framer-motion";
import { useEffect, useState } from "react";
import { ArtworkMetaForm, type ArtworkMetaFormValues, isMetaFormValid } from "./ArtworkMetaForm";
import type { ArtworkWithMeta, Award, GradeLevel, School } from "../../lib/artworkApi";

type ArtworkEditModalProps = {
  open: boolean;
  artwork: ArtworkWithMeta | null;
  schools: School[];
  gradeLevels: GradeLevel[];
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
    className: artwork.class_name ?? "",
    awardId: artwork.awards?.[0]?.id ?? "",
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
  awards,
  busy,
  onSave,
  onClose,
}: ArtworkEditModalProps) {
  const [values, setValues] = useState<ArtworkMetaFormValues>(() =>
    artwork ? toFormValues(artwork, gradeLevels) : { ...({} as ArtworkMetaFormValues) },
  );

  useEffect(() => {
    if (artwork) setValues(toFormValues(artwork, gradeLevels));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [artwork?.id]);

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

  return (
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
                  <img src={artwork.thumbnail_url || artwork.image_url} alt={artwork.title} />
                </div>
                <div className="artwork-modal-fields">
                  <ArtworkMetaForm
                    values={values}
                    onChange={setValues}
                    schools={schools}
                    gradeLevels={gradeLevels}
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
    </AnimatePresence>
  );
}
