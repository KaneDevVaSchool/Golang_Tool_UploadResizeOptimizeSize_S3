import { AnimatePresence, motion } from "framer-motion";
import { useEffect, useRef, useState } from "react";
import { formatBytes } from "../lib/api";
import {
  cssImageTransform,
  DEFAULT_TRANSFORM,
  hasTransform,
  normalizeRotation,
  type ImageTransform,
} from "../lib/imageTransform";
import { ImageLightbox } from "./ImageLightbox";
import { PreviewImage } from "./PreviewImage";
import { ProgressRing } from "./ProgressRing";
import { ZoomViewport, type ZoomControls } from "./ZoomViewport";

export type PreviewItem = {
  id: string;
  file: File;
  previewUrl: string | null;
  /** Downscaled JPEG for the thumb strip (optional until ready) */
  thumbUrl: string | null;
  progress: number;
  status: "ready" | "uploading" | "done" | "error";
  error?: string;
  transform: ImageTransform;
};

type PreviewPanelProps = {
  items: PreviewItem[];
  activeId: string;
  busy: boolean;
  /** "lưu" | "thu nhỏ" — used in meta / progress / CTA */
  modeLabel: string;
  onSelectItem: (id: string) => void;
  onRemoveItem: (id: string) => void;
  onAddMore: () => void;
  onClearAll: () => void;
  onRequestSend: () => void;
  onCancelUpload?: () => void;
  onTransformChange: (id: string, transform: ImageTransform) => void;
};

function modeVerbCapitalized(modeLabel: string) {
  if (modeLabel === "thu nhỏ") return "Thu nhỏ";
  return "Lưu";
}

function IconRotateLeft() {
  return (
    <svg viewBox="0 0 24 24" width="18" height="18" fill="none" aria-hidden>
      <path
        d="M9.5 4.5A7.5 7.5 0 1 0 17 12"
        stroke="currentColor"
        strokeWidth="1.75"
        strokeLinecap="round"
      />
      <path d="M9.5 4.5V8M9.5 4.5H6" stroke="currentColor" strokeWidth="1.75" strokeLinecap="round" />
    </svg>
  );
}

function IconRotateRight() {
  return (
    <svg viewBox="0 0 24 24" width="18" height="18" fill="none" aria-hidden>
      <path
        d="M14.5 4.5A7.5 7.5 0 1 1 7 12"
        stroke="currentColor"
        strokeWidth="1.75"
        strokeLinecap="round"
      />
      <path
        d="M14.5 4.5V8M14.5 4.5H18"
        stroke="currentColor"
        strokeWidth="1.75"
        strokeLinecap="round"
      />
    </svg>
  );
}

function IconFlipH() {
  return (
    <svg viewBox="0 0 24 24" width="18" height="18" fill="none" aria-hidden>
      <path d="M12 3v18" stroke="currentColor" strokeWidth="1.75" strokeLinecap="round" />
      <path d="M4 7l6 2v6l-6 2V7z" stroke="currentColor" strokeWidth="1.75" strokeLinejoin="round" />
      <path
        d="M20 7l-6 2v6l6 2V7z"
        stroke="currentColor"
        strokeWidth="1.75"
        strokeLinejoin="round"
        opacity="0.55"
      />
    </svg>
  );
}

function IconFlipV() {
  return (
    <svg viewBox="0 0 24 24" width="18" height="18" fill="none" aria-hidden>
      <path d="M3 12h18" stroke="currentColor" strokeWidth="1.75" strokeLinecap="round" />
      <path d="M7 4l2 6h6l2-6H7z" stroke="currentColor" strokeWidth="1.75" strokeLinejoin="round" />
      <path
        d="M7 20l2-6h6l2 6H7z"
        stroke="currentColor"
        strokeWidth="1.75"
        strokeLinejoin="round"
        opacity="0.55"
      />
    </svg>
  );
}

function IconZoomIn() {
  return (
    <svg viewBox="0 0 24 24" width="18" height="18" fill="none" aria-hidden>
      <circle cx="10.5" cy="10.5" r="6.25" stroke="currentColor" strokeWidth="1.75" />
      <path d="M15.2 15.2 20 20" stroke="currentColor" strokeWidth="1.75" strokeLinecap="round" />
      <path d="M10.5 8v5M8 10.5h5" stroke="currentColor" strokeWidth="1.75" strokeLinecap="round" />
    </svg>
  );
}

function IconZoomOut() {
  return (
    <svg viewBox="0 0 24 24" width="18" height="18" fill="none" aria-hidden>
      <circle cx="10.5" cy="10.5" r="6.25" stroke="currentColor" strokeWidth="1.75" />
      <path d="M15.2 15.2 20 20" stroke="currentColor" strokeWidth="1.75" strokeLinecap="round" />
      <path d="M8 10.5h5" stroke="currentColor" strokeWidth="1.75" strokeLinecap="round" />
    </svg>
  );
}

function IconFit() {
  return (
    <svg viewBox="0 0 24 24" width="18" height="18" fill="none" aria-hidden>
      <path
        d="M4 9V5h4M15 5h4v4M19 15v4h-4M9 19H5v-4"
        stroke="currentColor"
        strokeWidth="1.75"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}

function IconFullscreen() {
  return (
    <svg viewBox="0 0 24 24" width="18" height="18" fill="none" aria-hidden>
      <path
        d="M8 4H5v3M16 4h3v3M19 16v3h-3M8 20H5v-3"
        stroke="currentColor"
        strokeWidth="1.75"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <rect x="8.5" y="8.5" width="7" height="7" rx="1" stroke="currentColor" strokeWidth="1.5" />
    </svg>
  );
}

function IconReset() {
  return (
    <svg viewBox="0 0 24 24" width="18" height="18" fill="none" aria-hidden>
      <path
        d="M4.5 12a7.5 7.5 0 1 0 2.2-5.3"
        stroke="currentColor"
        strokeWidth="1.75"
        strokeLinecap="round"
      />
      <path d="M4.5 4.5v4h4" stroke="currentColor" strokeWidth="1.75" strokeLinecap="round" />
    </svg>
  );
}

function IconEdit() {
  return (
    <svg viewBox="0 0 16 16" width="10" height="10" fill="none" aria-hidden>
      <path
        d="M10.5 2.5l3 3L5 14H2v-3L10.5 2.5z"
        stroke="currentColor"
        strokeWidth="1.4"
        strokeLinejoin="round"
      />
    </svg>
  );
}

function IconRemove() {
  return (
    <svg viewBox="0 0 16 16" width="10" height="10" fill="none" aria-hidden>
      <path
        d="M4.25 4.25l7.5 7.5M11.75 4.25l-7.5 7.5"
        stroke="currentColor"
        strokeWidth="1.75"
        strokeLinecap="round"
      />
    </svg>
  );
}

function IconPrev() {
  return (
    <svg viewBox="0 0 24 24" width="18" height="18" fill="none" aria-hidden>
      <path d="M14.5 6L9 12l5.5 6" stroke="currentColor" strokeWidth="1.75" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}

function IconNext() {
  return (
    <svg viewBox="0 0 24 24" width="18" height="18" fill="none" aria-hidden>
      <path d="M9.5 6L15 12l-5.5 6" stroke="currentColor" strokeWidth="1.75" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}

export function PreviewPanel({
  items,
  activeId,
  busy,
  modeLabel,
  onSelectItem,
  onRemoveItem,
  onAddMore,
  onClearAll,
  onRequestSend,
  onCancelUpload,
  onTransformChange,
}: PreviewPanelProps) {
  const active = items.find((i) => i.id === activeId) ?? items[0];
  const activeIndex = items.findIndex((i) => i.id === active?.id);
  const totalBytes = items.reduce((sum, i) => sum + i.file.size, 0);
  const uploading = items.find((i) => i.status === "uploading");
  const [fullscreen, setFullscreen] = useState(false);
  const thumbsRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    setFullscreen(false);
  }, [active?.id]);

  useEffect(() => {
    if (busy) setFullscreen(false);
  }, [busy]);

  useEffect(() => {
    if (!active?.id || !thumbsRef.current) return;
    const el = thumbsRef.current.querySelector<HTMLElement>(`[data-thumb-id="${active.id}"]`);
    el?.scrollIntoView({ behavior: "smooth", block: "nearest", inline: "center" });
  }, [active?.id]);

  useEffect(() => {
    if (busy || !active) return;
    function onKey(e: KeyboardEvent) {
      const tag = (e.target as HTMLElement | null)?.tagName;
      if (tag === "INPUT" || tag === "TEXTAREA" || (e.target as HTMLElement | null)?.isContentEditable) {
        return;
      }
      if ((e.key === "f" || e.key === "F") && active.previewUrl) {
        e.preventDefault();
        setFullscreen(true);
        return;
      }
      if (items.length < 2) return;
      if (e.key === "ArrowLeft") {
        e.preventDefault();
        const prev = items[(activeIndex - 1 + items.length) % items.length];
        if (prev) onSelectItem(prev.id);
      } else if (e.key === "ArrowRight") {
        e.preventDefault();
        const next = items[(activeIndex + 1) % items.length];
        if (next) onSelectItem(next.id);
      }
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [busy, active, activeIndex, items, onSelectItem]);

  if (!active) return null;

  const transform = active.transform ?? DEFAULT_TRANSFORM;
  const canEdit = !busy && Boolean(active.previewUrl);
  const rotation = normalizeRotation(transform.rotation);
  const edited = hasTransform(transform);
  const canNav = items.length > 1 && !busy;

  function updateTransform(patch: Partial<ImageTransform>) {
    const next: ImageTransform = {
      rotation: normalizeRotation(patch.rotation ?? transform.rotation),
      flipH: patch.flipH ?? transform.flipH,
      flipV: patch.flipV ?? transform.flipV,
    };
    onTransformChange(active.id, next);
  }

  function rotateBy(delta: number) {
    updateTransform({ rotation: normalizeRotation(transform.rotation + delta) });
  }

  function selectRelative(delta: number) {
    if (!canNav) return;
    const next = items[(activeIndex + delta + items.length) % items.length];
    if (next) onSelectItem(next.id);
  }

  function renderToolbar(controls: ZoomControls) {
    return (
      <div
        className="preview-toolbar"
        role="toolbar"
        aria-label="Chỉnh ảnh"
        onPointerDown={(e) => e.stopPropagation()}
      >
        <div className="preview-tool-group">
          <button
            type="button"
            className="preview-tool"
            aria-label="Xoay trái 90°"
            title="Xoay trái"
            onClick={() => rotateBy(-90)}
          >
            <IconRotateLeft />
          </button>
          <button
            type="button"
            className="preview-tool"
            aria-label="Xoay phải 90°"
            title="Xoay phải"
            onClick={() => rotateBy(90)}
          >
            <IconRotateRight />
          </button>
          <button
            type="button"
            className="preview-tool"
            aria-label="Lật ngang"
            title="Lật ngang"
            data-active={transform.flipH ? "true" : undefined}
            onClick={() => updateTransform({ flipH: !transform.flipH })}
          >
            <IconFlipH />
          </button>
          <button
            type="button"
            className="preview-tool"
            aria-label="Lật dọc"
            title="Lật dọc"
            data-active={transform.flipV ? "true" : undefined}
            onClick={() => updateTransform({ flipV: !transform.flipV })}
          >
            <IconFlipV />
          </button>
        </div>

        <div className="preview-tool-group">
          <button
            type="button"
            className="preview-tool"
            aria-label="Giảm phóng"
            title="Giảm phóng"
            disabled={!controls.canZoomOut}
            onClick={controls.zoomOut}
          >
            <IconZoomOut />
          </button>
          <button
            type="button"
            className="preview-zoom-label lightbox-zoom-btn"
            title="Đặt 100%"
            onClick={() => controls.setZoomPercent(100)}
          >
            {Math.round(controls.zoom * 100)}%
          </button>
          <button
            type="button"
            className="preview-tool"
            aria-label="Phóng to"
            title="Phóng to"
            disabled={!controls.canZoomIn}
            onClick={controls.zoomIn}
          >
            <IconZoomIn />
          </button>
          <button
            type="button"
            className="preview-tool"
            aria-label="Vừa khung"
            title="Vừa khung"
            disabled={!controls.canFit}
            onClick={controls.fit}
          >
            <IconFit />
          </button>
          <button
            type="button"
            className="preview-tool"
            aria-label="Xem toàn màn hình"
            title="Toàn màn hình (F)"
            disabled={!active.previewUrl}
            onClick={() => setFullscreen(true)}
          >
            <IconFullscreen />
          </button>
          <button
            type="button"
            className="preview-tool"
            aria-label="Đặt lại xoay và lật"
            title="Đặt lại"
            disabled={!edited}
            onClick={() => {
              onTransformChange(active.id, { ...DEFAULT_TRANSFORM });
              controls.fit();
            }}
          >
            <IconReset />
          </button>
        </div>
      </div>
    );
  }

  return (
    <motion.div
      className="preview-panel"
      initial={{ opacity: 0, scale: 0.97, y: 12 }}
      animate={{ opacity: 1, scale: 1, y: 0 }}
      exit={{ opacity: 0, scale: 0.97, y: -10 }}
      transition={{ duration: 0.4, ease: [0.22, 1, 0.36, 1] }}
    >
      <div className="preview-stage">
        {canNav && (
          <>
            <button
              type="button"
              className="preview-nav preview-nav-prev"
              aria-label="Ảnh trước"
              title="Ảnh trước (←)"
              onClick={() => selectRelative(-1)}
            >
              <IconPrev />
            </button>
            <button
              type="button"
              className="preview-nav preview-nav-next"
              aria-label="Ảnh sau"
              title="Ảnh sau (→)"
              onClick={() => selectRelative(1)}
            >
              <IconNext />
            </button>
          </>
        )}

        <ZoomViewport
          enabled={canEdit}
          resetKey={active.id}
          className="preview-zoom"
          hintFit="Kéo để xem · chạm đôi để vừa khung"
          hintZoom="Kéo hoặc cuộn để xem · chạm đôi phóng · ← → đổi ảnh"
          chrome={canEdit ? renderToolbar : undefined}
        >
          <AnimatePresence mode="wait">
            <motion.div
              key={active.id}
              className="preview-media"
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              transition={{ duration: 0.22 }}
            >
              {active.previewUrl ? (
                <PreviewImage
                  src={active.previewUrl}
                  alt={active.file.name}
                  transformStyle={cssImageTransform(transform)}
                  rotation={rotation}
                />
              ) : (
                <p className="preview-fallback">
                  File này không xem trước được — vẫn {modeLabel} được.
                </p>
              )}
            </motion.div>
          </AnimatePresence>
        </ZoomViewport>

        <AnimatePresence>
          {busy && uploading && (
            <motion.div
              className="preview-busy"
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
            >
              <ProgressRing percent={uploading.progress} modeLabel={modeLabel} />
              <p className="progress-copy">
                Đang {modeLabel} {items.filter((i) => i.status === "done").length + 1}/{items.length}
              </p>
            </motion.div>
          )}
        </AnimatePresence>
      </div>

      <div className="preview-thumbs" role="list" ref={thumbsRef}>
        {items.map((item, index) => {
          const t = item.transform ?? DEFAULT_TRANSFORM;
          const rot = normalizeRotation(t.rotation);
          const thumbSwap = rot === 90 || rot === 270;
          const thumbEdited = hasTransform(t);
          return (
            <motion.div
              key={item.id}
              role="listitem"
              className="preview-thumb"
              data-thumb-id={item.id}
              data-active={item.id === active.id}
              data-status={item.status}
              data-edited={thumbEdited ? "true" : undefined}
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: Math.min(index, 8) * 0.03, duration: 0.28 }}
              title={item.file.name}
            >
              <button
                type="button"
                className="preview-thumb-select"
                disabled={busy}
                onClick={() => onSelectItem(item.id)}
                aria-label={`Chọn ảnh ${index + 1}: ${item.file.name}`}
                aria-current={item.id === active.id ? "true" : undefined}
              >
                {item.thumbUrl ? (
                  <span className="preview-thumb-frame" data-swap={thumbSwap ? "true" : "false"}>
                    <img
                      src={item.thumbUrl}
                      alt=""
                      decoding="async"
                      loading="lazy"
                      style={{ transform: cssImageTransform(t) }}
                    />
                  </span>
                ) : item.previewUrl ? (
                  <span className="preview-thumb-skeleton" aria-hidden />
                ) : (
                  <span className="preview-thumb-fallback">{item.file.name.slice(0, 1)}</span>
                )}
              </button>
              <span className="preview-thumb-badge">{index + 1}</span>
              {thumbEdited && (
                <span className="preview-thumb-edit" aria-hidden title="Đã chỉnh">
                  <IconEdit />
                </span>
              )}
              {!busy && items.length > 1 && (
                <button
                  type="button"
                  className="preview-thumb-remove"
                  aria-label={`Gỡ ảnh: ${item.file.name}`}
                  title="Gỡ ảnh"
                  onClick={(e) => {
                    e.stopPropagation();
                    onRemoveItem(item.id);
                  }}
                >
                  <IconRemove />
                </button>
              )}
            </motion.div>
          );
        })}
        {!busy && (
          <button type="button" className="preview-thumb preview-thumb-add" onClick={onAddMore} aria-label="Thêm ảnh">
            +
          </button>
        )}
      </div>

      {busy && (
        <div className="batch-progress" aria-hidden>
          <motion.span
            className="batch-progress-fill"
            initial={false}
            animate={{
              width: `${Math.round(
                ((items.filter((i) => i.status === "done").length +
                  (uploading ? uploading.progress / 100 : 0)) /
                  Math.max(items.length, 1)) *
                  100,
              )}%`,
            }}
            transition={{ duration: 0.2 }}
          />
        </div>
      )}

      <div className="preview-meta">
        <div className="file-info">
          <strong title={active.file.name}>
            {items.length > 1
              ? `${activeIndex + 1}/${items.length} · ${active.file.name}`
              : active.file.name}
          </strong>
          <span>
            {formatBytes(active.file.size)}
            {items.length > 1 ? ` · tổng ${formatBytes(totalBytes)}` : ""}
            {edited ? " · đã chỉnh" : " · xem trước"} · chưa {modeLabel}
            {transform.rotation ? ` · xoay ${transform.rotation}°` : ""}
          </span>
        </div>
        <div className="preview-actions">
          {busy ? (
            <button type="button" className="btn btn-danger" onClick={onCancelUpload}>
              {modeLabel === "thu nhỏ" ? "Hủy thu nhỏ" : "Hủy lưu"}
            </button>
          ) : (
            <button type="button" className="btn btn-danger" onClick={onClearAll}>
              Xóa hết
            </button>
          )}
          <button type="button" className="btn btn-primary" onClick={onRequestSend} disabled={busy}>
            {busy
              ? `Đang ${modeLabel}…`
              : items.length > 1
                ? `${modeVerbCapitalized(modeLabel)} ${items.length} ảnh`
                : `${modeVerbCapitalized(modeLabel)} ảnh`}
          </button>
        </div>
      </div>

      {active.previewUrl && (
        <ImageLightbox
          open={fullscreen}
          src={active.previewUrl}
          alt={active.file.name}
          transformStyle={cssImageTransform(transform)}
          rotation={rotation}
          onClose={() => setFullscreen(false)}
        />
      )}
    </motion.div>
  );
}
