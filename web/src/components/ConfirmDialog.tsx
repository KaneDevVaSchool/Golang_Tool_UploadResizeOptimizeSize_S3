import { AnimatePresence, motion } from "framer-motion";
import { useEffect } from "react";
import { createPortal } from "react-dom";
import { webpOf } from "../lib/staticImage";

type ConfirmDialogProps = {
  open: boolean;
  title: string;
  message: string;
  confirmLabel?: string;
  cancelLabel?: string;
  busyLabel?: string;
  busy?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
};

export function ConfirmDialog({
  open,
  title,
  message,
  confirmLabel = "Có, lưu",
  cancelLabel = "Chưa",
  busyLabel = "Đang lưu…",
  busy = false,
  onConfirm,
  onCancel,
}: ConfirmDialogProps) {
  useEffect(() => {
    if (!open) return;
    function onKey(e: KeyboardEvent) {
      if (busy) return;
      if (e.key === "Escape") onCancel();
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open, busy, onCancel]);

  // Portal ra document.body - xem ghi chú trong ArtworkEditModal.tsx: modal
  // render bên trong .admin-content-inner (class "route-enter" khi chuyển
  // trang dùng will-change: transform, tạo containing block mới cho
  // position:fixed) sẽ bị nhốt trong khung cuộn nội dung thay vì phủ toàn
  // viewport nếu không portal.
  return createPortal(
    <AnimatePresence>
      {open && (
        <div className="confirm-root" role="presentation">
          <motion.button
            type="button"
            className="confirm-backdrop"
            aria-label="Đóng"
            disabled={busy}
            onClick={onCancel}
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
          />
          <motion.div
            className="confirm-card"
            role="dialog"
            aria-modal="true"
            aria-labelledby="confirm-title"
            initial={{ opacity: 0, y: 24, scale: 0.94 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: 12, scale: 0.96 }}
            transition={{ duration: 0.28, ease: [0.22, 1, 0.36, 1] }}
          >
            <picture>
              <source srcSet={webpOf("/images/vas-mascot-wave.png")} type="image/webp" />
              <motion.img
                className="confirm-mascot"
                src="/images/vas-mascot-wave.png"
                alt=""
                aria-hidden
                animate={{ y: [0, -8, 0], rotate: [-4, 4, -4] }}
                transition={{ duration: 2.6, repeat: Infinity, ease: "easeInOut" }}
              />
            </picture>
            <h2 id="confirm-title">{title}</h2>
            <p>{message}</p>
            <div className="confirm-actions">
              <button type="button" className="btn btn-ghost" onClick={onCancel} disabled={busy}>
                {cancelLabel}
              </button>
              <button type="button" className="btn btn-primary" onClick={onConfirm} disabled={busy}>
                {busy ? busyLabel : confirmLabel}
              </button>
            </div>
          </motion.div>
        </div>
      )}
    </AnimatePresence>,
    document.body,
  );
}
