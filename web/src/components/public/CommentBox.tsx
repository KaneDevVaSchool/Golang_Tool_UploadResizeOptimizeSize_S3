import { AnimatePresence, motion } from "framer-motion";
import { CalendarClock, Plus, Send, Trash2, X } from "lucide-react";
import { useEffect, useState } from "react";
import { createPortal } from "react-dom";
import { createComment, deleteComment, fetchComments, type PublicComment } from "../../lib/publicApi";
import { getSavedDisplayName, saveDisplayName } from "../../lib/visitorToken";
import { toast } from "../../lib/toastBus";

const MAX_CONTENT_LENGTH = 1000;

function formatCommentDate(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  return new Intl.DateTimeFormat("vi-VN", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }).format(date);
}

export function CommentBox({
  artworkId,
  composeOpen,
  onComposeOpenChange,
}: {
  artworkId: number;
  composeOpen: boolean;
  onComposeOpenChange: (open: boolean) => void;
}) {
  const [comments, setComments] = useState<PublicComment[]>([]);
  const [loading, setLoading] = useState(true);
  const [displayName, setDisplayName] = useState(() => getSavedDisplayName());
  const [content, setContent] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<PublicComment | null>(null);
  const [deleting, setDeleting] = useState(false);

  useEffect(() => {
    const controller = new AbortController();
    setLoading(true);
    fetchComments(artworkId, controller.signal)
      .then((list) => setComments(list ?? []))
      .catch(() => {
        // im lặng bỏ qua lỗi tải comment - không chặn xem tác phẩm
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false);
      });
    return () => controller.abort();
  }, [artworkId]);

  useEffect(() => {
    if (!composeOpen && !deleteTarget) return;
    function onKey(event: KeyboardEvent) {
      if (event.key !== "Escape" || submitting || deleting) return;
      if (deleteTarget) setDeleteTarget(null);
      else onComposeOpenChange(false);
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [composeOpen, deleteTarget, submitting, deleting, onComposeOpenChange]);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    const name = displayName.trim();
    const text = content.trim();
    if (!name) {
      toast.error("Vui lòng nhập tên hiển thị của bạn.");
      return;
    }
    if (!text) {
      toast.error("Vui lòng nhập nội dung bình luận.");
      return;
    }

    setSubmitting(true);
    try {
      const created = await createComment(artworkId, name, text);
      setComments((prev) => [created, ...prev]);
      setContent("");
      saveDisplayName(name);
      onComposeOpenChange(false);
      toast.success("Đã gửi bình luận.");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Không gửi được bình luận, thử lại sau.");
    } finally {
      setSubmitting(false);
    }
  }

  async function handleDelete() {
    if (!deleteTarget || deleting) return;
    setDeleting(true);
    try {
      await deleteComment(artworkId, deleteTarget.id);
      setComments((current) => current.filter((comment) => comment.id !== deleteTarget.id));
      setDeleteTarget(null);
      toast.success("Đã xoá bình luận của bạn.");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Không xoá được bình luận.");
    } finally {
      setDeleting(false);
    }
  }

  return (
    <div className="comment-box">
      <div className="comment-list">
        {loading ? (
          <p className="comment-empty">Đang tải bình luận…</p>
        ) : comments.length === 0 ? (
          <p className="comment-empty">Chưa có bình luận nào. Hãy là người đầu tiên!</p>
        ) : (
          comments.map((c) => (
            <div key={c.id} className="comment-item">
              <span className="comment-item-avatar">{c.display_name.charAt(0).toUpperCase()}</span>
              <div className="comment-item-body">
                <div className="comment-item-head">
                  <span>
                    <strong>{c.display_name}</strong>
                    <time dateTime={c.created_at}>
                      <CalendarClock size={11} />
                      {formatCommentDate(c.created_at)}
                    </time>
                  </span>
                  {c.can_delete && (
                    <button type="button" className="comment-delete-btn" onClick={() => setDeleteTarget(c)} title="Xoá bình luận">
                      <Trash2 size={14} />
                    </button>
                  )}
                </div>
                <p>{c.content}</p>
              </div>
            </div>
          ))
        )}
      </div>

      {typeof document !== "undefined" &&
        createPortal(
          <AnimatePresence>
            {composeOpen && (
              <motion.div className="comment-modal-root" role="presentation" initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}>
                <button type="button" className="comment-modal-backdrop" aria-label="Đóng" onClick={() => !submitting && onComposeOpenChange(false)} />
                <motion.div
                  className="comment-modal-card"
                  role="dialog"
                  aria-modal="true"
                  aria-labelledby="comment-compose-title"
                  initial={{ opacity: 0, y: 22, scale: 0.94 }}
                  animate={{ opacity: 1, y: 0, scale: 1 }}
                  exit={{ opacity: 0, y: 12, scale: 0.96 }}
                  transition={{ duration: 0.22, ease: [0.22, 1, 0.36, 1] }}
                >
                  <div className="comment-modal-header">
                    <span className="comment-modal-icon"><Plus size={20} /></span>
                    <span>
                      <h3 id="comment-compose-title">Gửi một lời yêu thương</h3>
                      <p>Lời động viên của bạn sẽ tiếp thêm tự tin cho họa sĩ nhí.</p>
                    </span>
                    <button type="button" onClick={() => onComposeOpenChange(false)} disabled={submitting} aria-label="Đóng">
                      <X size={18} />
                    </button>
                  </div>

                  <form className="comment-modal-form" onSubmit={handleSubmit}>
                    <label>
                      <span>Tên hiển thị</span>
                      <input
                        type="text"
                        className="comment-name-input"
                        placeholder="Ví dụ: Minh Anh"
                        maxLength={100}
                        value={displayName}
                        onChange={(event) => setDisplayName(event.target.value)}
                        disabled={submitting}
                        autoFocus
                      />
                    </label>
                    <label>
                      <span>Lời nhắn của bạn</span>
                      <textarea
                        className="comment-content-input"
                        placeholder="Ví dụ: Mình rất thích cách bạn kể câu chuyện bằng màu sắc…"
                        maxLength={MAX_CONTENT_LENGTH}
                        value={content}
                        onChange={(event) => setContent(event.target.value)}
                        disabled={submitting}
                        rows={5}
                      />
                    </label>
                    <div className="comment-modal-footer">
                      <small>{content.length}/{MAX_CONTENT_LENGTH} ký tự</small>
                      <div>
                        <button type="button" className="comment-modal-cancel" onClick={() => onComposeOpenChange(false)} disabled={submitting}>
                          Để sau
                        </button>
                        <button type="submit" className="comment-modal-submit" disabled={submitting}>
                          <Send size={15} />
                          {submitting ? "Đang gửi…" : "Gửi lời nhắn"}
                        </button>
                      </div>
                    </div>
                  </form>
                </motion.div>
              </motion.div>
            )}

            {deleteTarget && (
              <motion.div className="comment-modal-root" role="presentation" initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}>
                <button type="button" className="comment-modal-backdrop" aria-label="Huỷ xoá" onClick={() => !deleting && setDeleteTarget(null)} />
                <motion.div
                  className="comment-delete-confirm"
                  role="alertdialog"
                  aria-modal="true"
                  aria-labelledby="comment-delete-title"
                  initial={{ opacity: 0, y: 18, scale: 0.94 }}
                  animate={{ opacity: 1, y: 0, scale: 1 }}
                  exit={{ opacity: 0, y: 10, scale: 0.96 }}
                >
                  <span className="comment-delete-confirm-icon"><Trash2 size={21} /></span>
                  <h3 id="comment-delete-title">Xoá lời nhắn này?</h3>
                  <p>Bình luận sẽ biến mất và không thể khôi phục.</p>
                  <blockquote>“{deleteTarget.content}”</blockquote>
                  <div>
                    <button type="button" className="comment-modal-cancel" onClick={() => setDeleteTarget(null)} disabled={deleting}>Giữ lại</button>
                    <button type="button" className="comment-delete-confirm-btn" onClick={handleDelete} disabled={deleting}>
                      {deleting ? "Đang xoá…" : "Xoá bình luận"}
                    </button>
                  </div>
                </motion.div>
              </motion.div>
            )}
          </AnimatePresence>,
          document.body,
        )}
    </div>
  );
}
