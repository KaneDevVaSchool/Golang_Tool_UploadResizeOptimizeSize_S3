import { motion } from "framer-motion";
import type { UploadMode } from "../lib/api";

export type TimelineStep = 1 | 2 | 3;

type StepTimelineProps = {
  current: TimelineStep;
  done?: boolean;
  mode?: UploadMode;
};

export function StepTimeline({ current, done = false, mode = "s3" }: StepTimelineProps) {
  const active = done ? 3 : current;
  const lastLabel = mode === "wp" ? "Thu nhỏ" : "Lưu";
  const steps = [
    { id: 1 as const, label: "Chọn" },
    { id: 2 as const, label: "Xử lý" },
    { id: 3 as const, label: lastLabel },
  ];

  return (
    <motion.ol
      className="timeline"
      initial={{ opacity: 0, y: 10 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ delay: 0.2, duration: 0.45, ease: [0.22, 1, 0.36, 1] }}
      aria-label={mode === "wp" ? "Các bước thu nhỏ ảnh" : "Các bước lưu ảnh"}
    >
      {steps.map((step, index) => {
        const state = done || step.id < active ? "done" : step.id === active ? "active" : "todo";
        return (
          <li key={step.id} className="timeline-step" data-state={state}>
            <div className="timeline-node">
              <span className="timeline-dot">
                {state === "done" ? (
                  <svg viewBox="0 0 24 24" aria-hidden>
                    <path d="M5 12.5l4.5 4.5L19 7" />
                  </svg>
                ) : (
                  step.id
                )}
              </span>
              <span className="timeline-label">{step.label}</span>
            </div>
            {index < steps.length - 1 && (
              <span className="timeline-connector" aria-hidden data-filled={step.id < active || done}>
                <span className="timeline-connector-fill" />
              </span>
            )}
          </li>
        );
      })}
    </motion.ol>
  );
}
