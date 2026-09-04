import { motion } from "framer-motion";
import { EducationLevelCard } from "./EducationLevelCard";

/**
 * Section 2 - Cổng thông tin chọn khối: 2 card lớn side-by-side (chồng dọc
 * ở mobile), mỗi khối 1 theme riêng.
 */
export function EducationLevelGate({
  primaryCount,
  secondaryCount,
  onSelectLevel,
}: {
  primaryCount: number;
  secondaryCount: number;
  onSelectLevel: (level: "primary" | "secondary") => void;
}) {
  return (
    <section className="edu-gate-section">
      <motion.div
        className="section-heading"
        initial={{ opacity: 0, y: 20 }}
        whileInView={{ opacity: 1, y: 0 }}
        viewport={{ once: true, margin: "-80px" }}
        transition={{ duration: 0.5 }}
      >
        <span className="section-kicker">Cổng thông tin</span>
        <h2>Chọn khối để khám phá</h2>
      </motion.div>

      <div className="edu-gate-cards">
        <EducationLevelCard level="primary" artworkCount={primaryCount} onSelect={() => onSelectLevel("primary")} />
        <EducationLevelCard level="secondary" artworkCount={secondaryCount} onSelect={() => onSelectLevel("secondary")} />
      </div>
    </section>
  );
}
