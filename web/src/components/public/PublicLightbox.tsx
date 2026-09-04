import { AnimatePresence, motion } from "framer-motion";
import { ChevronLeft, ChevronRight, X } from "lucide-react";
import { useEffect, useState } from "react";
import { createPortal } from "react-dom";
import type { ArtworkWithMeta } from "../../lib/artworkApi";
import type { ReactionCounts } from "../../lib/publicApi";
import { CommentBox } from "./CommentBox";
import { ReactionPicker } from "./ReactionPicker";

/**
 * Lightbox trang public - tham khảo cấu trúc ImageLightbox.tsx (Portal +
 * AnimatePresence + Escape/Arrow key), nhưng thêm prev/next chevron, counter
 * "x/y", và nhúng ReactionPicker + CommentBox ngay trong lightbox (khác
 * ImageLightbox gốc chỉ dùng để zoom ảnh upload, không có tương tác xã hội).
 */
export function PublicLightbox({
  items,
  activeIndex,
  onNavigate,
  onClose,
}: {
  items: ArtworkWithMeta[];
  activeIndex: number;
  onNavigate: (index: number) => void;
  onClose: () => void;
}) {
  const artwork = items[activeIndex];
  const open = Boolean(artwork);
  const [reactionCounts, setReactionCounts] = useState<ReactionCounts | null>(null);

  useEffect(() => {
    setReactionCounts(artwork?.reaction_counts ?? null);
  }, [artwork?.id]);

  useEffect(() => {
    if (!open) return;
    const prevOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    return () => {
      document.body.style.overflow = prevOverflow;
    };
  }, [open]);

  useEffect(() => {
    if (!open) return;
    function onKey(e: KeyboardEvent) {
      if (e.key === "Escape") {
        onClose();
      } else if (e.key === "ArrowLeft" && items.length > 1) {
        onNavigate((activeIndex - 1 + items.length) % items.length);
      } else if (e.key === "ArrowRight" && items.length > 1) {
        onNavigate((activeIndex + 1) % items.length);
      }
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open, activeIndex, items.length, onNavigate, onClose]);

  if (typeof document === "undefined" || !artwork) return null;

  return createPortal(
    <AnimatePresence>
      {open && (
        <motion.div
          className="public-lightbox-root"
          role="dialog"
          aria-modal="true"
          aria-label={`Xem tác phẩm: ${artwork.title}`}
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          transition={{ duration: 0.22 }}
        >
          <button type="button" className="public-lightbox-backdrop" aria-label="Đóng" onClick={onClose} />

          <motion.div
            className="public-lightbox-panel"
            initial={{ opacity: 0, scale: 0.96, y: 16 }}
            animate={{ opacity: 1, scale: 1, y: 0 }}
            exit={{ opacity: 0, scale: 0.96, y: 12 }}
            transition={{ duration: 0.26, ease: [0.22, 1, 0.36, 1] }}
          >
            <button type="button" className="public-lightbox-close" aria-label="Đóng" onClick={onClose}>
              <X size={20} />
            </button>

            {items.length > 1 && (
              <>
                <button
                  type="button"
                  className="public-lightbox-nav public-lightbox-nav--prev"
                  aria-label="Tác phẩm trước"
                  onClick={() => onNavigate((activeIndex - 1 + items.length) % items.length)}
                >
                  <ChevronLeft size={22} />
                </button>
                <button
                  type="button"
                  className="public-lightbox-nav public-lightbox-nav--next"
                  aria-label="Tác phẩm sau"
                  onClick={() => onNavigate((activeIndex + 1) % items.length)}
                >
                  <ChevronRight size={22} />
                </button>
              </>
            )}

            <div className="public-lightbox-image-wrap">
              <img src={artwork.image_url} alt={artwork.title} />
              {items.length > 1 && (
                <span className="public-lightbox-counter">
                  {activeIndex + 1} / {items.length}
                </span>
              )}
            </div>

            <div className="public-lightbox-sidebar">
              <div className="public-lightbox-meta">
                <h3>{artwork.title}</h3>
                <p>{artwork.student_name}</p>
                <p className="public-lightbox-meta-sub">
                  {artwork.school_name} · {artwork.grade_label}
                </p>
                {artwork.awards && artwork.awards.length > 0 && (
                  <div className="public-lightbox-awards">
                    {artwork.awards.map((a) => (
                      <span key={a.id} className="award-badge" style={{ background: a.color_hex }}>
                        {a.name}
                      </span>
                    ))}
                  </div>
                )}
                <ReactionPicker artworkId={artwork.id} counts={reactionCounts} onCountsChange={setReactionCounts} />
              </div>

              <div className="public-lightbox-comments">
                <CommentBox artworkId={artwork.id} />
              </div>
            </div>
          </motion.div>
        </motion.div>
      )}
    </AnimatePresence>,
    document.body,
  );
}
