import { AnimatePresence, motion } from "framer-motion";
import { EyeOff, Eye as EyeIcon, Loader2 } from "lucide-react";
import { useEffect, useState } from "react";
import { createPortal } from "react-dom";
import { fetchArtworkComments, setCommentHidden, type ArtworkComment, type ArtworkWithMeta } from "../../lib/artworkApi";
import { toast } from "../../lib/toastBus";

type ArtworkCommentsModalProps = {
  open: boolean;
  artwork: ArtworkWithMeta | null;
  onClose: () => void;
  /** Báo cho trang danh sách cập nhật lại comment_count sau khi ẩn/hiện. */
  onCountChange?: (artworkId: number, delta: number) => void;
};

function formatCommentDate(iso: string): string {
  try {
    return new Date(iso).toLocaleString("vi-VN", { dateStyle: "short", timeStyle: "short" });
  } catch {
    return iso;
  }
}

/**
 * Modal kiểm duyệt bình luận của 1 tác phẩm - liệt kê TOÀN BỘ bình luận (kể
 * cả đã ẩn) và cho ẩn/hiện từng cái. Dùng chung khung .artwork-modal-* với
 * ArtworkEditModal (portal ra body, cùng lý do tránh bị nhốt trong khung
 * cuộn .admin-content-inner), chỉ thêm CSS riêng cho phần danh sách.
 */
export function ArtworkCommentsModal({ open, artwork, onClose, onCountChange }: ArtworkCommentsModalProps) {
  const [comments, setComments] = useState<ArtworkComment[]>([]);
  const [loading, setLoading] = useState(false);
  // Theo dõi artwork nào đã tải, để phát hiện đổi tác phẩm khi component
  // không unmount giữa các lần mở (giống syncedId trong ArtworkEditModal).
  const [loadedId, setLoadedId] = useState<number | null>(null);
  const [pendingId, setPendingId] = useState<number | null>(null);

  useEffect(() => {
    if (!open || !artwork) return;
    if (artwork.id === loadedId) return;
    const controller = new AbortController();
    setLoading(true);
    fetchArtworkComments(artwork.id, controller.signal)
      .then((list) => {
        setComments(list ?? []);
        setLoadedId(artwork.id);
      })
      .catch((err: unknown) => {
        if (controller.signal.aborted) return;
        toast.error(err instanceof Error ? err.message : "Không tải được bình luận");
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false);
      });
    return () => controller.abort();
  }, [open, artwork, loadedId]);

  // Đóng modal thì quên artwork đã tải, lần mở kế tiếp (kể cả cùng tác phẩm)
  // sẽ tải lại - tránh hiện dữ liệu cũ nếu có bình luận mới từ trang public.
  useEffect(() => {
    if (!open) setLoadedId(null);
  }, [open]);

  useEffect(() => {
    if (!open) return;
    function onKey(e: KeyboardEvent) {
      if (e.key === "Escape") onClose();
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open, onClose]);

  async function handleToggleHidden(comment: ArtworkComment) {
    if (!artwork) return;
    const nextHidden = !comment.is_hidden;
    setPendingId(comment.id);
    try {
      await setCommentHidden(artwork.id, comment.id, nextHidden);
      setComments((prev) => prev.map((c) => (c.id === comment.id ? { ...c, is_hidden: nextHidden } : c)));
      onCountChange?.(artwork.id, nextHidden ? -1 : 1);
      toast.success(nextHidden ? "Đã ẩn bình luận." : "Đã hiện lại bình luận.");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Không cập nhật được bình luận");
    } finally {
      setPendingId(null);
    }
  }

  if (!artwork) return null;

  return createPortal(
    <AnimatePresence>
      {open && (
        <div className="artwork-modal-root" role="presentation">
          <motion.button
            type="button"
            className="artwork-modal-backdrop"
            aria-label="Đóng"
            onClick={onClose}
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
          />
          <motion.div
            className="artwork-modal-panel comments-modal-panel"
            role="dialog"
            aria-modal="true"
            aria-labelledby="comments-modal-title"
            initial={{ opacity: 0, y: 24, scale: 0.97 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: 16, scale: 0.97 }}
            transition={{ duration: 0.26, ease: [0.22, 1, 0.36, 1] }}
          >
            <header className="artwork-modal-header">
              <h2 id="comments-modal-title">
                Bình luận · <span title={artwork.title}>{artwork.title}</span>
              </h2>
              <button type="button" className="artwork-modal-close" aria-label="Đóng" onClick={onClose}>
                ×
              </button>
            </header>

            <div className="artwork-modal-body">
              {loading ? (
                <div className="comments-modal-placeholder">
                  <Loader2 size={18} className="spin" /> Đang tải bình luận…
                </div>
              ) : comments.length === 0 ? (
                <div className="comments-modal-placeholder">Tác phẩm này chưa có bình luận nào.</div>
              ) : (
                <ul className="comments-modal-list">
                  {comments.map((comment) => (
                    <li
                      key={comment.id}
                      className={`comments-modal-item${comment.is_hidden ? " is-hidden" : ""}`}
                    >
                      <div className="comments-modal-item-body">
                        <div className="comments-modal-item-head">
                          <strong>{comment.display_name}</strong>
                          <span className="comments-modal-item-date">{formatCommentDate(comment.created_at)}</span>
                          {comment.is_hidden && <span className="comments-modal-item-badge">Đã ẩn</span>}
                        </div>
                        <p>{comment.content}</p>
                      </div>
                      <button
                        type="button"
                        className={`comments-modal-toggle-btn${comment.is_hidden ? " is-hidden" : ""}`}
                        disabled={pendingId === comment.id}
                        onClick={() => handleToggleHidden(comment)}
                        title={comment.is_hidden ? "Hiện lại bình luận này" : "Ẩn bình luận này khỏi trang public"}
                      >
                        {pendingId === comment.id ? (
                          <Loader2 size={14} className="spin" />
                        ) : comment.is_hidden ? (
                          <EyeIcon size={14} />
                        ) : (
                          <EyeOff size={14} />
                        )}
                        {comment.is_hidden ? "Hiện" : "Ẩn"}
                      </button>
                    </li>
                  ))}
                </ul>
              )}
            </div>

            <footer className="artwork-modal-footer">
              <button type="button" className="btn btn-ghost" onClick={onClose}>
                Đóng
              </button>
            </footer>
          </motion.div>
        </div>
      )}
    </AnimatePresence>,
    document.body,
  );
}
