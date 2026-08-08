import { motion, useMotionValue, useSpring, useTransform } from "framer-motion";
import { useRef, type DragEvent, type KeyboardEvent, type PointerEvent } from "react";
import { ProgressRing } from "./ProgressRing";

type DropzoneProps = {
  busy: boolean;
  progress: number;
  onSelect: (file: File) => void;
};

export function Dropzone({ busy, progress, onSelect }: DropzoneProps) {
  const inputRef = useRef<HTMLInputElement>(null);
  const zoneRef = useRef<HTMLDivElement>(null);
  const x = useMotionValue(0);
  const y = useMotionValue(0);
  const springX = useSpring(x, { stiffness: 180, damping: 18 });
  const springY = useSpring(y, { stiffness: 180, damping: 18 });
  const rotateX = useTransform(springY, [-40, 40], [5, -5]);
  const rotateY = useTransform(springX, [-40, 40], [-5, 5]);

  function pickFile(fileList: FileList | null) {
    const file = fileList?.[0];
    if (file) onSelect(file);
  }

  function onDragOver(e: DragEvent) {
    e.preventDefault();
    zoneRef.current?.setAttribute("data-active", "true");
  }

  function onDragLeave() {
    zoneRef.current?.setAttribute("data-active", "false");
  }

  function onDrop(e: DragEvent) {
    e.preventDefault();
    zoneRef.current?.setAttribute("data-active", "false");
    if (!busy) pickFile(e.dataTransfer.files);
  }

  function onKeyDown(e: KeyboardEvent) {
    if (busy) return;
    if (e.key === "Enter" || e.key === " ") {
      e.preventDefault();
      inputRef.current?.click();
    }
  }

  function onPointerMove(e: PointerEvent<HTMLDivElement>) {
    const rect = e.currentTarget.getBoundingClientRect();
    x.set(e.clientX - rect.left - rect.width / 2);
    y.set(e.clientY - rect.top - rect.height / 2);
  }

  function onPointerLeave() {
    x.set(0);
    y.set(0);
    onDragLeave();
  }

  return (
    <motion.div
      className="stage"
      style={{ perspective: 1200 }}
      initial={{ opacity: 0, y: 36 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.7, delay: 0.35, ease: [0.22, 1, 0.36, 1] }}
    >
      <motion.div
        ref={zoneRef}
        className="dropzone"
        role="button"
        tabIndex={0}
        aria-label="Upload file dropzone"
        data-active="false"
        style={{ rotateX, rotateY, transformStyle: "preserve-3d" }}
        whileHover={busy ? undefined : { scale: 1.01 }}
        whileTap={busy ? undefined : { scale: 0.995 }}
        animate={
          busy
            ? { scale: 1 }
            : {
                borderColor: ["rgba(20,33,43,0.22)", "rgba(31,111,120,0.45)", "rgba(20,33,43,0.22)"],
              }
        }
        transition={
          busy
            ? undefined
            : { borderColor: { duration: 4.5, repeat: Infinity, ease: "easeInOut" } }
        }
        onClick={() => !busy && inputRef.current?.click()}
        onKeyDown={onKeyDown}
        onDragOver={onDragOver}
        onDragLeave={onDragLeave}
        onDrop={onDrop}
        onPointerMove={onPointerMove}
        onPointerLeave={onPointerLeave}
      >
        <div className="dropzone-orbit" aria-hidden />
        <input
          ref={inputRef}
          className="file-input"
          type="file"
          accept="image/*,.pdf,.zip,.txt,.csv,.json"
          onChange={(e) => {
            pickFile(e.target.files);
            e.target.value = "";
          }}
        />

        {busy ? (
          <ProgressRing percent={progress} />
        ) : (
          <motion.div
            className="dropzone-inner"
            initial={{ opacity: 0, scale: 0.96 }}
            animate={{ opacity: 1, scale: 1 }}
            transition={{ duration: 0.45 }}
          >
            <motion.div
              className="dropzone-glyph"
              animate={{ y: [0, -6, 0] }}
              transition={{ duration: 2.8, repeat: Infinity, ease: "easeInOut" }}
            >
              <svg viewBox="0 0 24 24" aria-hidden>
                <path d="M12 16V4" />
                <path d="M7 9l5-5 5 5" />
                <path d="M4 16v2a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-2" />
              </svg>
            </motion.div>
            <h2>Drop media into the light</h2>
            <p>Drag a file here, or click to browse. Images resize and optimize through the API pipeline.</p>
          </motion.div>
        )}
      </motion.div>
    </motion.div>
  );
}
