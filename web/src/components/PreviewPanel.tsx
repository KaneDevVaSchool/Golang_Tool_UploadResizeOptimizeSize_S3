import { AnimatePresence, motion } from "framer-motion";
import {
  useEffect,
  useRef,
  useState,
  type PointerEvent as ReactPointerEvent,
} from "react";
import { formatBytes } from "../lib/api";
import {
  cssImageTransform,
  DEFAULT_TRANSFORM,
  normalizeRotation,
  type ImageTransform,
} from "../lib/imageTransform";
import { ProgressRing } from "./ProgressRing";

export type PreviewItem = {
  id: string;
  file: File;
  previewUrl: string | null;
  progress: number;
  status: "ready" | "uploading" | "done" | "error";
  error?: string;
  transform: ImageTransform;
};

type PreviewPanelProps = {
  items: PreviewItem[];
  activeId: string;
  busy: boolean;
  modeLabel: string;
  onSelectItem: (id: string) => void;
  onRemoveItem: (id: string) => void;
  onAddMore: () => void;
  onClearAll: () => void;
  onRequestSend: () => void;
  onCancelUpload?: () => void;
  onTransformChange: (id: string, transform: ImageTransform) => void;
};

const ZOOM_MIN = 1;
const ZOOM_MAX = 4;
const ZOOM_STEP = 0.25;

type ViewState = { zoom: number; x: number; y: number };

const FIT_VIEW: ViewState = { zoom: 1, x: 0, y: 0 };

function clamp(n: number, min: number, max: number) {
  return Math.min(max, Math.max(min, n));
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
  const totalBytes = items.reduce((sum, i) => sum + i.file.size, 0);
  const uploading = items.find((i) => i.status === "uploading");

  const viewportRef = useRef<HTMLDivElement>(null);
  const [view, setView] = useState<ViewState>(FIT_VIEW);
  const viewRef = useRef(view);
  viewRef.current = view;

  const pointers = useRef(new Map<number, { x: number; y: number }>());
  const pinchRef = useRef<{ dist: number; zoom: number } | null>(null);
  const panRef = useRef<{ x: number; y: number; ox: number; oy: number } | null>(null);
  const lastTapRef = useRef(0);

  useEffect(() => {
    setView(FIT_VIEW);
    pointers.current.clear();
    pinchRef.current = null;
    panRef.current = null;
  }, [active?.id]);

  useEffect(() => {
    const el = viewportRef.current;
    if (!el) return;
    const onWheelNative = (e: WheelEvent) => {
      if (el.dataset.editable !== "true") return;
      e.preventDefault();
      const dir = e.deltaY > 0 ? -ZOOM_STEP : ZOOM_STEP;
      setView((prev) => {
        const zoom = clamp(prev.zoom + dir, ZOOM_MIN, ZOOM_MAX);
        if (zoom === 1) return FIT_VIEW;
        const rect = el.getBoundingClientRect();
        const cx = e.clientX - rect.left - rect.width / 2;
        const cy = e.clientY - rect.top - rect.height / 2;
        const ratio = zoom / prev.zoom;
        return {
          zoom,
          x: cx - (cx - prev.x) * ratio,
          y: cy - (cy - prev.y) * ratio,
        };
      });
    };
    el.addEventListener("wheel", onWheelNative, { passive: false });
    return () => el.removeEventListener("wheel", onWheelNative);
  }, [active?.id]);

  if (!active) return null;

  const transform = active.transform ?? DEFAULT_TRANSFORM;
  const canEdit = !busy && Boolean(active.previewUrl);

  function setZoom(next: number, origin?: { x: number; y: number }) {
    setView((prev) => {
      const zoom = clamp(next, ZOOM_MIN, ZOOM_MAX);
      if (zoom === 1) return FIT_VIEW;
      if (!origin || !viewportRef.current) return { ...prev, zoom };
      const rect = viewportRef.current.getBoundingClientRect();
      const cx = origin.x - rect.left - rect.width / 2;
      const cy = origin.y - rect.top - rect.height / 2;
      const ratio = zoom / prev.zoom;
      return {
        zoom,
        x: cx - (cx - prev.x) * ratio,
        y: cy - (cy - prev.y) * ratio,
      };
    });
  }

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

  function onPointerDown(e: ReactPointerEvent) {
    if (!canEdit) return;
    const el = viewportRef.current;
    if (!el) return;
    el.setPointerCapture(e.pointerId);
    pointers.current.set(e.pointerId, { x: e.clientX, y: e.clientY });

    if (pointers.current.size === 2) {
      const pts = [...pointers.current.values()];
      const dist = Math.hypot(pts[0].x - pts[1].x, pts[0].y - pts[1].y);
      pinchRef.current = { dist: Math.max(dist, 1), zoom: viewRef.current.zoom };
      panRef.current = null;
      return;
    }

    if (viewRef.current.zoom > 1) {
      panRef.current = {
        x: e.clientX,
        y: e.clientY,
        ox: viewRef.current.x,
        oy: viewRef.current.y,
      };
    }

    const now = Date.now();
    if (now - lastTapRef.current < 280) {
      if (viewRef.current.zoom > 1) setView(FIT_VIEW);
      else setZoom(2, { x: e.clientX, y: e.clientY });
      lastTapRef.current = 0;
    } else {
      lastTapRef.current = now;
    }
  }

  function onPointerMove(e: ReactPointerEvent) {
    if (!canEdit || !pointers.current.has(e.pointerId)) return;
    pointers.current.set(e.pointerId, { x: e.clientX, y: e.clientY });

    if (pointers.current.size === 2 && pinchRef.current) {
      const pts = [...pointers.current.values()];
      const dist = Math.hypot(pts[0].x - pts[1].x, pts[0].y - pts[1].y);
      const mid = { x: (pts[0].x + pts[1].x) / 2, y: (pts[0].y + pts[1].y) / 2 };
      const next = pinchRef.current.zoom * (dist / pinchRef.current.dist);
      setZoom(next, mid);
      return;
    }

    if (panRef.current && viewRef.current.zoom > 1) {
      const dx = e.clientX - panRef.current.x;
      const dy = e.clientY - panRef.current.y;
      setView((prev) => ({
        ...prev,
        x: panRef.current!.ox + dx,
        y: panRef.current!.oy + dy,
      }));
    }
  }

  function onPointerUp(e: ReactPointerEvent) {
    pointers.current.delete(e.pointerId);
    if (pointers.current.size < 2) pinchRef.current = null;
    if (pointers.current.size === 0) panRef.current = null;
    try {
      viewportRef.current?.releasePointerCapture(e.pointerId);
    } catch {
      /* ignore */
    }
  }

  const edited =
    transform.rotation !== 0 || transform.flipH || transform.flipV || view.zoom !== 1;

  return (
    <motion.div
      className="preview-panel"
      initial={{ opacity: 0, scale: 0.97, y: 12 }}
      animate={{ opacity: 1, scale: 1, y: 0 }}
      exit={{ opacity: 0, scale: 0.97, y: -10 }}
      transition={{ duration: 0.4, ease: [0.22, 1, 0.36, 1] }}
    >
      <div className="preview-stage">
        <div
          ref={viewportRef}
          className="preview-viewport"
          data-zoomed={view.zoom > 1 ? "true" : "false"}
          data-editable={canEdit ? "true" : "false"}
          onPointerDown={onPointerDown}
          onPointerMove={onPointerMove}
          onPointerUp={onPointerUp}
          onPointerCancel={onPointerUp}
        >
          <AnimatePresence mode="wait">
            <motion.div
              key={active.id}
              className="preview-media-shell"
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              transition={{ duration: 0.28 }}
            >
              <div
                className="preview-media-frame"
                style={{
                  transform: `translate3d(${view.x}px, ${view.y}px, 0) scale(${view.zoom})`,
                }}
              >
                {active.previewUrl ? (
                  <img
                    src={active.previewUrl}
                    alt={active.file.name}
                    draggable={false}
                    style={{ transform: cssImageTransform(transform) }}
                  />
                ) : (
                  <p className="preview-fallback">File này không xem trước được — vẫn gửi được.</p>
                )}
              </div>
            </motion.div>
          </AnimatePresence>

          {canEdit && (
            <div className="preview-hint" aria-hidden>
              {view.zoom > 1 ? "Kéo để xem · chạm đôi để vừa khung" : "Chạm đôi để phóng · pinch / cuộn để zoom"}
            </div>
          )}
        </div>

        {canEdit && (
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
                aria-label="Thu nhỏ"
                title="Thu nhỏ"
                disabled={view.zoom <= ZOOM_MIN}
                onClick={() => setZoom(view.zoom - ZOOM_STEP)}
              >
                <IconZoomOut />
              </button>
              <span className="preview-zoom-label">{Math.round(view.zoom * 100)}%</span>
              <button
                type="button"
                className="preview-tool"
                aria-label="Phóng to"
                title="Phóng to"
                disabled={view.zoom >= ZOOM_MAX}
                onClick={() => setZoom(view.zoom + ZOOM_STEP)}
              >
                <IconZoomIn />
              </button>
              <button
                type="button"
                className="preview-tool"
                aria-label="Vừa khung"
                title="Vừa khung"
                disabled={view.zoom === 1 && view.x === 0 && view.y === 0}
                onClick={() => setView(FIT_VIEW)}
              >
                <IconFit />
              </button>
              <button
                type="button"
                className="preview-tool"
                aria-label="Đặt lại xoay và lật"
                title="Đặt lại"
                disabled={
                  transform.rotation === 0 && !transform.flipH && !transform.flipV
                }
                onClick={() => {
                  onTransformChange(active.id, { ...DEFAULT_TRANSFORM });
                  setView(FIT_VIEW);
                }}
              >
                <IconReset />
              </button>
            </div>
          </div>
        )}

        {busy && uploading && (
          <motion.div
            className="preview-busy"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
          >
            <ProgressRing percent={uploading.progress} />
            <p className="progress-copy">
              Đang gửi {items.filter((i) => i.status === "done").length + 1}/{items.length}
            </p>
          </motion.div>
        )}
      </div>

      <div className="preview-thumbs" role="list">
        {items.map((item, index) => (
          <motion.div
            key={item.id}
            role="listitem"
            className="preview-thumb"
            data-active={item.id === active.id}
            data-status={item.status}
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: index * 0.04, duration: 0.3 }}
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
              {item.previewUrl ? (
                <img
                  src={item.previewUrl}
                  alt=""
                  style={{ transform: cssImageTransform(item.transform ?? DEFAULT_TRANSFORM) }}
                />
              ) : (
                <span className="preview-thumb-fallback">{item.file.name.slice(0, 1)}</span>
              )}
            </button>
            <span className="preview-thumb-badge">{index + 1}</span>
            {(item.transform?.rotation || item.transform?.flipH || item.transform?.flipV) && (
              <span className="preview-thumb-edit" aria-hidden>
                ✎
              </span>
            )}
            {!busy && items.length > 1 && (
              <button
                type="button"
                className="preview-thumb-remove"
                aria-label="Gỡ ảnh này"
                onClick={() => onRemoveItem(item.id)}
              >
                ×
              </button>
            )}
          </motion.div>
        ))}
        {!busy && (
          <button type="button" className="preview-thumb preview-thumb-add" onClick={onAddMore}>
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
            {items.length > 1 ? `${items.length} ảnh đã chọn` : active.file.name}
          </strong>
          <span>
            {formatBytes(totalBytes)}
            {edited ? " · đã chỉnh" : " · xem trước"} · chưa {modeLabel}
            {transform.rotation ? ` · xoay ${transform.rotation}°` : ""}
          </span>
        </div>
        <div className="preview-actions">
          {busy ? (
            <button type="button" className="btn btn-danger" onClick={onCancelUpload}>
              Hủy gửi
            </button>
          ) : (
            <button type="button" className="btn btn-danger" onClick={onClearAll}>
              Xóa hết
            </button>
          )}
          <button type="button" className="btn btn-primary" onClick={onRequestSend} disabled={busy}>
            {busy ? "Đang gửi…" : items.length > 1 ? `Gửi ${items.length} ảnh` : "Gửi đi"}
          </button>
        </div>
      </div>
    </motion.div>
  );
}
