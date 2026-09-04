import { Send } from "lucide-react";
import { useEffect, useState } from "react";
import { createComment, fetchComments, type PublicComment } from "../../lib/publicApi";
import { getSavedDisplayName, saveDisplayName } from "../../lib/visitorToken";
import { toast } from "../../lib/toastBus";

const MAX_CONTENT_LENGTH = 1000;

/**
 * Ô bình luận ẩn danh cho 1 tác phẩm - tên hiển thị lưu localStorage để
 * không phải nhập lại mỗi lần comment, nội dung giới hạn 1000 ký tự (khớp
 * cột artwork_comments.content ở backend).
 */
export function CommentBox({ artworkId }: { artworkId: number }) {
  const [comments, setComments] = useState<PublicComment[]>([]);
  const [loading, setLoading] = useState(true);
  const [displayName, setDisplayName] = useState(() => getSavedDisplayName());
  const [content, setContent] = useState("");
  const [submitting, setSubmitting] = useState(false);

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
      toast.success("Đã gửi bình luận.");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Không gửi được bình luận, thử lại sau.");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="comment-box">
      <form className="comment-form" onSubmit={handleSubmit}>
        <input
          type="text"
          className="comment-name-input"
          placeholder="Tên hiển thị của bạn"
          maxLength={100}
          value={displayName}
          onChange={(e) => setDisplayName(e.target.value)}
          disabled={submitting}
        />
        <div className="comment-content-row">
          <textarea
            className="comment-content-input"
            placeholder="Viết cảm nhận về tác phẩm này…"
            maxLength={MAX_CONTENT_LENGTH}
            value={content}
            onChange={(e) => setContent(e.target.value)}
            disabled={submitting}
            rows={2}
          />
          <button type="submit" className="comment-submit-btn" disabled={submitting} aria-label="Gửi bình luận">
            <Send size={16} />
          </button>
        </div>
      </form>

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
                <strong>{c.display_name}</strong>
                <p>{c.content}</p>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
}
