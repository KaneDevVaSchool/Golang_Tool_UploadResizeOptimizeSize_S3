import { AnimatePresence, motion } from "framer-motion";
import { useEffect, useRef, useState, type CSSProperties } from "react";
import { createPortal } from "react-dom";
import { addReaction, removeReaction, type ReactionCounts } from "../../lib/publicApi";
import { playReactionSound } from "../../lib/sound";
import { toast } from "../../lib/toastBus";

const REACTIONS: { key: string; emoji: string; freq: number; label: string }[] = [
  { key: "like", emoji: "👍", freq: 520, label: "Thích" },
  { key: "love", emoji: "❤️", freq: 660, label: "Yêu thích" },
  { key: "haha", emoji: "😂", freq: 740, label: "Haha" },
  { key: "wow", emoji: "😮", freq: 600, label: "Wow" },
  { key: "sad", emoji: "😢", freq: 420, label: "Buồn" },
  { key: "angry", emoji: "😡", freq: 350, label: "Giận" },
];

/**
 * Bảng chọn cảm xúc ẩn danh - hover/tap hiện picker, click gửi reaction,
 * animated burst (framer-motion scale spring) + âm thanh riêng mỗi loại.
 * localReaction đánh dấu cảm xúc visitor này đã chọn (toggle: click lại để
 * gỡ) - chỉ lưu tạm ở state, không đọc lại từ server (server không trả biết
 * visitor đã react gì, chỉ trả tổng count).
 */
export function ReactionPicker({
  artworkId,
  counts,
  onCountsChange,
}: {
  artworkId: number;
  counts: ReactionCounts | null | undefined;
  onCountsChange: (counts: ReactionCounts) => void;
}) {
  const [open, setOpen] = useState(false);
  const [localReaction, setLocalReaction] = useState<string | null>(null);
  const [burstKey, setBurstKey] = useState<string | null>(null);
  const [pending, setPending] = useState(false);
  const [menuPosition, setMenuPosition] = useState({ left: 8, top: 8 });
  const triggerRef = useRef<HTMLButtonElement>(null);
  const closeTimerRef = useRef<number | null>(null);

  function cancelClose() {
    if (closeTimerRef.current !== null) {
      window.clearTimeout(closeTimerRef.current);
      closeTimerRef.current = null;
    }
  }

  function scheduleClose() {
    cancelClose();
    closeTimerRef.current = window.setTimeout(() => setOpen(false), 160);
  }

  function updateMenuPosition() {
    const trigger = triggerRef.current;
    if (!trigger) return;
    const rect = trigger.getBoundingClientRect();
    const menuWidth = Math.min(280, window.innerWidth - 16);
    const menuHeight = 58;
    const left = Math.min(Math.max(8, rect.right - menuWidth), window.innerWidth - menuWidth - 8);
    const top = rect.top >= menuHeight + 12 ? rect.top - menuHeight - 8 : rect.bottom + 8;
    setMenuPosition({ left, top });
  }

  function showMenu() {
    cancelClose();
    updateMenuPosition();
    setOpen(true);
  }

  useEffect(() => {
    if (!open) return;

    // Gom về mỗi khung hình một lần. Listener này bắt scroll ở pha capture
    // nên nó nhận mọi sự kiện cuộn của mọi phần tử lồng nhau trên trang, mà
    // updateMenuPosition lại gọi getBoundingClientRect - đọc layout, tức là
    // ép trình duyệt tính lại bố cục ngay giữa lúc đang cuộn.
    let frame = 0;
    const reposition = () => {
      if (frame) return;
      frame = requestAnimationFrame(() => {
        frame = 0;
        updateMenuPosition();
      });
    };

    window.addEventListener("resize", reposition, { passive: true });
    window.addEventListener("scroll", reposition, { capture: true, passive: true });
    return () => {
      if (frame) cancelAnimationFrame(frame);
      window.removeEventListener("resize", reposition);
      window.removeEventListener("scroll", reposition, true);
    };
  }, [open]);

  useEffect(() => () => cancelClose(), []);

  async function handlePick(key: string) {
    if (pending) return;
    setOpen(false);
    setPending(true);
    playReactionSound(REACTIONS.find((r) => r.key === key)?.freq ?? 500);
    setBurstKey(key);
    window.setTimeout(() => setBurstKey(null), 700);

    try {
      if (localReaction === key) {
        const result = await removeReaction(artworkId, key);
        setLocalReaction(null);
        onCountsChange(result.reaction_counts);
      } else {
        const result = await addReaction(artworkId, key);
        setLocalReaction(key);
        onCountsChange(result.reaction_counts);
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Không gửi được cảm xúc, thử lại sau.");
    } finally {
      setPending(false);
    }
  }

  const totalCount = Object.values(counts ?? {}).reduce((sum, n) => sum + n, 0);
  const activeReaction = REACTIONS.find((r) => r.key === localReaction);

  return (
    <div
      className="reaction-picker"
      onMouseEnter={showMenu}
      onMouseLeave={scheduleClose}
    >
      <button
        ref={triggerRef}
        type="button"
        className={`reaction-picker-trigger${localReaction ? " reaction-picker-trigger--active" : ""}`}
        onClick={() => {
          if (open) setOpen(false);
          else showMenu();
        }}
        aria-label="Bày tỏ cảm xúc"
        aria-expanded={open}
      >
        <span className="reaction-picker-emoji">{activeReaction?.emoji ?? "🤍"}</span>
        <span>{totalCount > 0 ? totalCount : "Thích"}</span>
      </button>

      {typeof document !== "undefined" &&
        createPortal(
          <AnimatePresence>
            {open && (
              <motion.div
                className="reaction-picker-menu reaction-picker-menu--portal"
                style={{ left: menuPosition.left, top: menuPosition.top } as CSSProperties}
                initial={{ opacity: 0, y: 8, scale: 0.9 }}
                animate={{ opacity: 1, y: 0, scale: 1 }}
                exit={{ opacity: 0, y: 8, scale: 0.9 }}
                transition={{ duration: 0.16 }}
                onMouseEnter={cancelClose}
                onMouseLeave={scheduleClose}
              >
                {REACTIONS.map((r) => (
                  <button
                    key={r.key}
                    type="button"
                    className="reaction-picker-option"
                    title={r.label}
                    aria-label={r.label}
                    onClick={() => handlePick(r.key)}
                  >
                    <motion.span whileHover={{ scale: 1.3, y: -4 }} transition={{ type: "spring", stiffness: 400, damping: 12 }}>
                      {r.emoji}
                    </motion.span>
                  </button>
                ))}
              </motion.div>
            )}
          </AnimatePresence>,
          document.body,
        )}

      <AnimatePresence>
        {burstKey && (
          <motion.span
            key={burstKey}
            className="reaction-burst"
            initial={{ opacity: 1, scale: 0.6, y: 0 }}
            animate={{ opacity: 0, scale: 1.8, y: -36 }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.65, ease: "easeOut" }}
          >
            {REACTIONS.find((r) => r.key === burstKey)?.emoji}
          </motion.span>
        )}
      </AnimatePresence>
    </div>
  );
}
