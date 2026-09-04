const REACTION_EMOJI: Record<string, string> = {
  like: "👍",
  love: "❤️",
  haha: "😂",
  wow: "😮",
  sad: "😢",
  angry: "😡",
};

const REACTION_ORDER = ["like", "love", "haha", "wow", "sad", "angry"];

/**
 * Hiển thị 6 icon cảm xúc nhỏ + số đếm - dùng emoji Unicode trực tiếp
 * (zero-dependency, khớp bộ 6 loại Facebook-style dùng ở va-workspace).
 * Chỉ hiện loại có count > 0 để không chiếm chỗ vô ích trên danh sách.
 */
export function ReactionIcons({ counts }: { counts: Record<string, number> | null | undefined }) {
  const entries = REACTION_ORDER.map((key) => ({ key, count: counts?.[key] ?? 0 })).filter(
    (e) => e.count > 0,
  );

  if (entries.length === 0) {
    return <span className="reaction-icons reaction-icons--empty">—</span>;
  }

  return (
    <span className="reaction-icons">
      {entries.map((e) => (
        <span key={e.key} className="reaction-icon-item" title={e.key}>
          <span aria-hidden>{REACTION_EMOJI[e.key]}</span>
          {e.count}
        </span>
      ))}
    </span>
  );
}
