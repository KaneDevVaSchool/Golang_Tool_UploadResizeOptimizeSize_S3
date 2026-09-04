import { AnimatePresence, motion } from "framer-motion";
import { useState } from "react";
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
      onMouseEnter={() => setOpen(true)}
      onMouseLeave={() => setOpen(false)}
    >
      <button
        type="button"
        className={`reaction-picker-trigger${localReaction ? " reaction-picker-trigger--active" : ""}`}
        onClick={() => handlePick(localReaction ?? "like")}
        aria-label="Bày tỏ cảm xúc"
      >
        <span className="reaction-picker-emoji">{activeReaction?.emoji ?? "🤍"}</span>
        <span>{totalCount > 0 ? totalCount : "Thích"}</span>
      </button>

      <AnimatePresence>
        {open && (
          <motion.div
            className="reaction-picker-menu"
            initial={{ opacity: 0, y: 8, scale: 0.9 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: 8, scale: 0.9 }}
            transition={{ duration: 0.16 }}
          >
            {REACTIONS.map((r) => (
              <button
                key={r.key}
                type="button"
                className="reaction-picker-option"
                title={r.label}
                onClick={() => handlePick(r.key)}
              >
                <motion.span whileHover={{ scale: 1.3, y: -4 }} transition={{ type: "spring", stiffness: 400, damping: 12 }}>
                  {r.emoji}
                </motion.span>
              </button>
            ))}
          </motion.div>
        )}
      </AnimatePresence>

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
