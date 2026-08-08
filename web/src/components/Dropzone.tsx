import { AnimatePresence, motion, useMotionValue, useSpring, useTransform } from "framer-motion";
import { useRef, type DragEvent, type KeyboardEvent, type PointerEvent } from "react";

type DropzoneProps = {
  busy: boolean;
  onSelect: (files: File[]) => void;
  accept?: string;
};

export function Dropzone({
  busy,
  onSelect,
  accept = "image/*,.pdf,.zip,.txt,.csv,.json",
}: DropzoneProps) {
  const inputRef = useRef<HTMLInputElement>(null);
  const zoneRef = useRef<HTMLDivElement>(null);
  const x = useMotionValue(0);
  const y = useMotionValue(0);
  const springX = useSpring(x, { stiffness: 180, damping: 18 });
  const springY = useSpring(y, { stiffness: 180, damping: 18 });
  const rotateX = useTransform(springY, [-40, 40], [4, -4]);
  const rotateY = useTransform(springX, [-40, 40], [-4, 4]);

  function pickFiles(fileList: FileList | null) {
    if (!fileList?.length) return;
    onSelect(Array.from(fileList));
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
    if (!busy) pickFiles(e.dataTransfer.files);
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
      style={{ height: "100%", perspective: 1200 }}
      initial={{ opacity: 0, y: 24, scale: 0.98 }}
      animate={{ opacity: 1, y: 0, scale: 1 }}
      exit={{ opacity: 0, y: -16, scale: 0.98 }}
      transition={{ duration: 0.5, ease: [0.22, 1, 0.36, 1] }}
    >
      <motion.div
        ref={zoneRef}
        className="dropzone"
        role="button"
        tabIndex={0}
        aria-label="Kéo nhiều ảnh vào đây hoặc bấm để chọn"
        data-active="false"
        style={{ rotateX, rotateY, transformStyle: "preserve-3d" }}
        whileHover={busy ? undefined : { scale: 1.01 }}
        whileTap={busy ? undefined : { scale: 0.995 }}
        animate={
          busy
            ? { scale: 1 }
            : {
                borderColor: [
                  "rgba(255,255,255,0.4)",
                  "rgba(255,143,114,0.85)",
                  "rgba(255,255,255,0.4)",
                ],
              }
        }
        transition={
          busy
            ? undefined
            : { borderColor: { duration: 3.5, repeat: Infinity, ease: "easeInOut" } }
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
          multiple
          accept={accept}
          onChange={(e) => {
            pickFiles(e.target.files);
            e.target.value = "";
          }}
        />
        <AnimatePresence mode="wait">
          <motion.div
            className="dropzone-inner"
            key="idle"
            initial={{ opacity: 0, scale: 0.96, y: 8 }}
            animate={{ opacity: 1, scale: 1, y: 0 }}
            exit={{ opacity: 0, scale: 0.96, y: -8 }}
            transition={{ duration: 0.35 }}
          >
            <motion.div
              className="dropzone-glyph"
              animate={{ y: [0, -8, 0] }}
              transition={{ duration: 3.2, repeat: Infinity, ease: "easeInOut" }}
            >
              <svg viewBox="0 0 48 48" fill="none" aria-hidden>
                <path
                  d="M24 32V14m0 0l-7 7m7-7l7 7"
                  stroke="currentColor"
                  strokeWidth="2.4"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                />
                <path
                  d="M12 30.5v3.2A4.3 4.3 0 0016.3 38h15.4A4.3 4.3 0 0036 33.7v-3.2"
                  stroke="currentColor"
                  strokeWidth="2.4"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                />
              </svg>
            </motion.div>
            <h2>Kéo nhiều ảnh vào đây</h2>
            <p>Hoặc bấm để chọn nhiều file cùng lúc. Bạn xem trước rồi mới gửi.</p>
          </motion.div>
        </AnimatePresence>
      </motion.div>
    </motion.div>
  );
}
