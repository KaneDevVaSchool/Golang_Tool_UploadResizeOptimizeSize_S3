import { AnimatePresence, motion, useMotionValue, useSpring, useTransform } from "framer-motion";
import {
  useEffect,
  useRef,
  useState,
  type DragEvent,
  type KeyboardEvent,
  type PointerEvent,
  type RefObject,
} from "react";
import type { UploadMode } from "../lib/api";

type DropzoneProps = {
  busy: boolean;
  mode: UploadMode;
  onSelect: (files: File[]) => void;
  accept?: string;
  inputRef?: RefObject<HTMLInputElement | null>;
};

const CHIPS_S3 = ["Ảnh", "PDF", "ZIP", "TXT", "CSV", "JSON"];
const CHIPS_WP = ["JPG", "PNG", "WebP", "GIF"];

export function Dropzone({
  busy,
  mode,
  onSelect,
  accept = "image/*,.pdf,.zip,.txt,.csv,.json",
  inputRef: externalRef,
}: DropzoneProps) {
  const localRef = useRef<HTMLInputElement>(null);
  const inputRef = externalRef ?? localRef;
  const zoneRef = useRef<HTMLDivElement>(null);
  const [dragging, setDragging] = useState(false);
  const x = useMotionValue(0);
  const y = useMotionValue(0);
  const springX = useSpring(x, { stiffness: 180, damping: 18 });
  const springY = useSpring(y, { stiffness: 180, damping: 18 });
  const rotateX = useTransform(springY, [-40, 40], [4, -4]);
  const rotateY = useTransform(springX, [-40, 40], [-4, 4]);

  const chips = mode === "wp" ? CHIPS_WP : CHIPS_S3;
  const title = dragging
    ? "Thả để chọn"
    : mode === "wp"
      ? "Kéo ảnh vào đây"
      : "Kéo file vào đây";
  const subtitle =
    mode === "wp"
      ? "Chỉ ảnh — hệ thống thu nhỏ, tối ưu và tạo nhiều kích thước. Lấy link dùng ngay."
      : "Ảnh hoặc tài liệu — lưu tập trung, hỗ trợ file lớn. Lấy link dùng ngay.";

  function pickFiles(fileList: FileList | null) {
    if (!fileList?.length) return;
    onSelect(Array.from(fileList));
  }

  function openPicker() {
    if (!busy) inputRef.current?.click();
  }

  function onDragOver(e: DragEvent) {
    e.preventDefault();
    setDragging(true);
    zoneRef.current?.setAttribute("data-active", "true");
  }

  function onDragLeave() {
    setDragging(false);
    zoneRef.current?.setAttribute("data-active", "false");
  }

  function onDrop(e: DragEvent) {
    e.preventDefault();
    setDragging(false);
    zoneRef.current?.setAttribute("data-active", "false");
    if (!busy) pickFiles(e.dataTransfer.files);
  }

  function onKeyDown(e: KeyboardEvent) {
    if (busy) return;
    if (e.key === "Enter" || e.key === " ") {
      e.preventDefault();
      openPicker();
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

  const onSelectRef = useRef(onSelect);
  onSelectRef.current = onSelect;

  useEffect(() => {
    function onPaste(e: ClipboardEvent) {
      if (busy) return;
      const clipItems = e.clipboardData?.items;
      if (!clipItems?.length) return;
      const files: File[] = [];
      for (const item of Array.from(clipItems)) {
        if (item.kind === "file" && item.type.startsWith("image/")) {
          const file = item.getAsFile();
          if (file) files.push(file);
        }
      }
      if (!files.length) return;
      e.preventDefault();
      onSelectRef.current(files);
    }
    window.addEventListener("paste", onPaste);
    return () => window.removeEventListener("paste", onPaste);
  }, [busy]);

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
        aria-label={
          mode === "wp"
            ? "Kéo ảnh vào đây, dán từ clipboard, hoặc bấm để chọn"
            : "Kéo file vào đây, dán ảnh từ clipboard, hoặc bấm để chọn"
        }
        data-active={dragging ? "true" : "false"}
        data-mode={mode}
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
        onClick={openPicker}
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
            key={dragging ? "drag" : mode}
            initial={{ opacity: 0, scale: 0.96, y: 8 }}
            animate={{ opacity: 1, scale: 1, y: 0 }}
            exit={{ opacity: 0, scale: 0.96, y: -8 }}
            transition={{ duration: 0.35 }}
          >
            <motion.div
              className="dropzone-glyph"
              animate={{ y: dragging ? 0 : [0, -8, 0] }}
              transition={
                dragging
                  ? { duration: 0.2 }
                  : { duration: 3.2, repeat: Infinity, ease: "easeInOut" }
              }
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
            <h2>{title}</h2>
            <p>{subtitle}</p>
            <ul className="dropzone-chips" aria-label="Định dạng hỗ trợ">
              {chips.map((chip) => (
                <li key={chip}>{chip}</li>
              ))}
            </ul>
            <p className="dropzone-paste-hint">Hoặc Ctrl/Cmd+V để dán ảnh</p>
            <button
              type="button"
              className="btn btn-ghost dropzone-pick-btn"
              disabled={busy}
              onClick={(e) => {
                e.stopPropagation();
                openPicker();
              }}
            >
              Chọn từ máy
            </button>
          </motion.div>
        </AnimatePresence>
      </motion.div>
    </motion.div>
  );
}
