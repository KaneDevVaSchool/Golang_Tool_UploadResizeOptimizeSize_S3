import { AnimatePresence, motion } from "framer-motion";
import { useCallback, useEffect, useRef } from "react";
import { createPortal } from "react-dom";
import { PreviewImage } from "./PreviewImage";
import { ZoomViewport, type ZoomControls } from "./ZoomViewport";

type ImageLightboxProps = {
  open: boolean;
  src: string;
  alt: string;
  /** CSS rotate/flip string for the image */
  transformStyle?: string;
  rotation?: number;
  onClose: () => void;
};

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

function IconClose() {
  return (
    <svg viewBox="0 0 24 24" width="20" height="20" fill="none" aria-hidden>
      <path d="M6 6l12 12M18 6L6 18" stroke="currentColor" strokeWidth="1.75" strokeLinecap="round" />
    </svg>
  );
}

function LightboxToolbar({ controls }: { controls: ZoomControls }) {
  return (
    <div
      className="preview-toolbar lightbox-toolbar"
      role="toolbar"
      aria-label="Zoom ảnh"
      onPointerDown={(e) => e.stopPropagation()}
    >
      <div className="preview-tool-group">
        <button
          type="button"
          className="preview-tool"
          aria-label="Giảm phóng"
          title="Giảm phóng (−)"
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
          title="Phóng to (+)"
          disabled={!controls.canZoomIn}
          onClick={controls.zoomIn}
        >
          <IconZoomIn />
        </button>
        <button
          type="button"
          className="preview-tool"
          aria-label="Vừa khung"
          title="Vừa khung (0)"
          disabled={!controls.canFit}
          onClick={controls.fit}
        >
          <IconFit />
        </button>
      </div>
      <p className="lightbox-keys" aria-hidden>
        Esc đóng · +/− zoom · 0 vừa khung · chạm đôi phóng
      </p>
    </div>
  );
}

export function ImageLightbox({
  open,
  src,
  alt,
  transformStyle,
  rotation = 0,
  onClose,
}: ImageLightboxProps) {
  const controlsRef = useRef<ZoomControls | null>(null);
  const onControlsChange = useCallback((controls: ZoomControls) => {
    controlsRef.current = controls;
  }, []);

  useEffect(() => {
    if (!open) return;
    // Xoá inline style khi đóng, không khôi phục snapshot - xem giải thích
    // ở PublicLightbox (StrictMode chạy effect hai lần, snapshot lần hai sẽ
    // là "hidden" và khoá cuộn vĩnh viễn).
    document.body.style.overflow = "hidden";
    return () => {
      document.body.style.removeProperty("overflow");
    };
  }, [open]);

  useEffect(() => {
    if (!open) return;
    function onKey(e: KeyboardEvent) {
      const c = controlsRef.current;
      if (e.key === "Escape") {
        e.preventDefault();
        onClose();
        return;
      }
      if (!c) return;
      if (e.key === "+" || e.key === "=") {
        e.preventDefault();
        c.zoomIn();
      } else if (e.key === "-" || e.key === "_") {
        e.preventDefault();
        c.zoomOut();
      } else if (e.key === "0") {
        e.preventDefault();
        c.fit();
      }
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open, onClose]);

  if (typeof document === "undefined") return null;

  return createPortal(
    <AnimatePresence>
      {open && (
        <motion.div
          className="lightbox-root"
          role="dialog"
          aria-modal="true"
          aria-label={`Xem ảnh toàn màn hình: ${alt}`}
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          transition={{ duration: 0.22 }}
        >
          <button type="button" className="lightbox-backdrop" aria-label="Đóng" onClick={onClose} />

          <motion.div
            className="lightbox-stage"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.22, ease: [0.22, 1, 0.36, 1] }}
          >
            <header className="lightbox-topbar">
              <p className="lightbox-title" title={alt}>
                {alt}
              </p>
              <button
                type="button"
                className="lightbox-close"
                aria-label="Đóng toàn màn hình"
                onClick={onClose}
              >
                <IconClose />
              </button>
            </header>

            <ZoomViewport
              enabled
              resetKey={`${src}|${rotation}|${transformStyle ?? ""}`}
              className="lightbox-viewport"
              hintFit="Kéo để xem · chạm đôi để vừa khung"
              hintZoom="Kéo hoặc cuộn để xem · chạm đôi phóng tới 300%"
              onControlsChange={onControlsChange}
              chrome={(controls) => <LightboxToolbar controls={controls} />}
            >
              <div className="preview-media">
                <PreviewImage
                  key={`${src}|${rotation}|${transformStyle ?? ""}`}
                  src={src}
                  alt={alt}
                  transformStyle={transformStyle}
                  rotation={rotation}
                />
              </div>
            </ZoomViewport>
          </motion.div>
        </motion.div>
      )}
    </AnimatePresence>,
    document.body,
  );
}
