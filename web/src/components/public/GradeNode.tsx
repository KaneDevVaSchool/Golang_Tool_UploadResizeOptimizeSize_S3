import { motion } from "framer-motion";

export function GradeNode({
  label,
  active,
  onClick,
}: {
  label: string;
  active: boolean;
  onClick: () => void;
}) {
  return (
    <motion.button
      type="button"
      className={`grade-node${active ? " grade-node--active" : ""}`}
      onClick={onClick}
      whileHover={{ scale: 1.08, y: -4 }}
      whileTap={{ scale: 0.96 }}
      transition={{ type: "spring", stiffness: 320, damping: 18 }}
    >
      {label}
    </motion.button>
  );
}
