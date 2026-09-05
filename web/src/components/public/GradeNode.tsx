import { motion } from "framer-motion";

export function GradeNode({
  label,
  active,
  onClick,
  variant = "circle",
  ariaLabel,
}: {
  label: string;
  active: boolean;
  onClick: () => void;
  variant?: "circle" | "chip";
  ariaLabel?: string;
}) {
  return (
    <motion.button
      type="button"
      className={`grade-node${variant === "chip" ? " grade-node--chip" : ""}${active ? " grade-node--active" : ""}`}
      onClick={onClick}
      aria-pressed={active}
      aria-label={ariaLabel}
      whileTap={{ scale: 0.94 }}
      transition={{ type: "spring", stiffness: 420, damping: 26 }}
    >
      {label}
    </motion.button>
  );
}
