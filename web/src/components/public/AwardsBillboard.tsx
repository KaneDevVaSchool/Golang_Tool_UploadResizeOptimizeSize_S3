import { motion } from "framer-motion";
import { useEffect, useState } from "react";
import { fetchBillboard, type BillboardEntry } from "../../lib/publicApi";

// clip-path riêng theo hạng: Nhất = khiên hexagon nổi bật, Nhì/Ba = bo góc
// bất đối xứng nhẹ hơn, còn lại = ribbon cắt góc dưới. Đơn giản, thuần CSS,
// không cần thư viện.
function clipPathForRank(rank: number): string {
  if (rank === 1) return "polygon(50% 0%, 100% 20%, 100% 80%, 50% 100%, 0% 80%, 0% 20%)";
  if (rank === 2) return "polygon(0% 0%, 100% 0%, 100% 88%, 85% 100%, 0% 100%)";
  if (rank === 3) return "polygon(15% 0%, 100% 0%, 100% 100%, 0% 100%, 0% 15%)";
  return "polygon(0% 0%, 100% 0%, 100% 100%, 8% 100%, 0% 85%)";
}

/**
 * Section 5 - Billboard vinh danh: khung ảnh dùng clip-path riêng theo
 * hạng giải (Nhất nổi bật nhất), border/glow theo award.color_hex.
 */
export function AwardsBillboard() {
  const [entries, setEntries] = useState<BillboardEntry[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const controller = new AbortController();
    fetchBillboard(controller.signal)
      .then((list) => setEntries(list ?? []))
      .catch(() => {
        if (!controller.signal.aborted) setEntries([]);
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false);
      });
    return () => controller.abort();
  }, []);

  if (!loading && entries.length === 0) return null;

  return (
    <section className="billboard-section">
      <motion.div
        className="section-heading section-heading--light"
        initial={{ opacity: 0, y: 20 }}
        whileInView={{ opacity: 1, y: 0 }}
        viewport={{ once: true, margin: "-80px" }}
        transition={{ duration: 0.5 }}
      >
        <span className="section-kicker">Bảng vàng</span>
        <h2>Billboard vinh danh</h2>
      </motion.div>

      {loading ? (
        <p className="admin-empty-note admin-empty-note--light">Đang tải…</p>
      ) : (
        <div className="billboard-grid">
          {entries.map((entry, index) => (
            <motion.div
              key={`${entry.id}-${entry.award.id}`}
              className="billboard-card"
              initial={{ opacity: 0, y: 24 }}
              whileInView={{ opacity: 1, y: 0 }}
              viewport={{ once: true, margin: "-60px" }}
              transition={{ duration: 0.45, delay: Math.min(index, 8) * 0.05, ease: [0.22, 1, 0.36, 1] }}
              style={{ ["--billboard-color" as string]: entry.award.color_hex }}
            >
              <div className="billboard-frame" style={{ clipPath: clipPathForRank(entry.award.rank_order) }}>
                <img src={entry.image_url} alt={entry.title} loading="lazy" />
              </div>
              <span className="billboard-award-badge" style={{ background: entry.award.color_hex }}>
                {entry.award.name}
              </span>
              <strong className="billboard-title">{entry.title}</strong>
              <span className="billboard-student">{entry.student_name}</span>
              <span className="billboard-school">{entry.school_name}</span>
            </motion.div>
          ))}
        </div>
      )}
    </section>
  );
}
