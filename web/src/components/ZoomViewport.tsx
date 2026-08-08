import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type PointerEvent as ReactPointerEvent,
  type ReactNode,
} from "react";

export const ZOOM_MIN = 0.5;
export const ZOOM_MAX = 12;
export const ZOOM_STEP = 0.5;
export const ZOOM_DOUBLE_TAP = 3;

export type ViewState = { zoom: number; x: number; y: number };

export const FIT_VIEW: ViewState = { zoom: 1, x: 0, y: 0 };

export type ZoomControls = {
  zoom: number;
  isZoomed: boolean;
  canZoomIn: boolean;
  canZoomOut: boolean;
  canFit: boolean;
  zoomIn: () => void;
  zoomOut: () => void;
  fit: () => void;
  setZoomPercent: (percent: number) => void;
};

function clamp(n: number, min: number, max: number) {
  return Math.min(max, Math.max(min, n));
}

/** Keep the scaled frame explorable — generous bounds so edges stay reachable. */
function clampPan(view: ViewState, width: number, height: number): ViewState {
  if (view.zoom <= 1.001) return { ...view, x: 0, y: 0 };
  const maxX = (width * (view.zoom - 1)) / 2 + Math.max(48, width * 0.25);
  const maxY = (height * (view.zoom - 1)) / 2 + Math.max(48, height * 0.25);
  return {
    zoom: view.zoom,
    x: clamp(view.x, -maxX, maxX),
    y: clamp(view.y, -maxY, maxY),
  };
}

function roundZoom(z: number) {
  return Math.round(z * 100) / 100;
}

type ZoomViewportProps = {
  enabled: boolean;
  resetKey?: string;
  className?: string;
  children: ReactNode;
  chrome?: (controls: ZoomControls) => ReactNode;
  /** Called when zoom/pan controls change — safe for keyboard shortcuts */
  onControlsChange?: (controls: ZoomControls) => void;
  hintFit?: string;
  hintZoom?: string;
};

export function ZoomViewport({
  enabled,
  resetKey,
  className,
  children,
  chrome,
  onControlsChange,
  hintFit = "Kéo để xem · chạm đôi để vừa khung",
  hintZoom = "Kéo để xem · chạm đôi phóng · cuộn/pinch zoom",
}: ZoomViewportProps) {
  const viewportRef = useRef<HTMLDivElement>(null);
  const [view, setView] = useState<ViewState>(FIT_VIEW);
  const viewRef = useRef(view);
  viewRef.current = view;

  const pointers = useRef(new Map<number, { x: number; y: number }>());
  const pinchRef = useRef<{ dist: number; zoom: number } | null>(null);
  const panRef = useRef<{ x: number; y: number; ox: number; oy: number } | null>(null);
  const lastTapRef = useRef(0);
  const movedRef = useRef(false);
  /** Drag from fit → bump zoom once so pan can explore the photo */
  const dragZoomArmedRef = useRef(false);

  const applyZoom = useCallback((next: number, origin?: { x: number; y: number }) => {
    setView((prev) => {
      const el = viewportRef.current;
      const zoom = roundZoom(clamp(next, ZOOM_MIN, ZOOM_MAX));
      if (zoom <= 1.001) return FIT_VIEW;
      if (!origin || !el) {
        return clampPan({ ...prev, zoom }, el?.clientWidth ?? 0, el?.clientHeight ?? 0);
      }
      const rect = el.getBoundingClientRect();
      const cx = origin.x - rect.left - rect.width / 2;
      const cy = origin.y - rect.top - rect.height / 2;
      const ratio = zoom / prev.zoom;
      return clampPan(
        {
          zoom,
          x: cx - (cx - prev.x) * ratio,
          y: cy - (cy - prev.y) * ratio,
        },
        rect.width,
        rect.height,
      );
    });
  }, []);

  const fit = useCallback(() => setView(FIT_VIEW), []);
  const zoomIn = useCallback(() => {
    const el = viewportRef.current;
    if (!el) {
      applyZoom(viewRef.current.zoom + ZOOM_STEP);
      return;
    }
    const rect = el.getBoundingClientRect();
    applyZoom(viewRef.current.zoom + ZOOM_STEP, {
      x: rect.left + rect.width / 2,
      y: rect.top + rect.height / 2,
    });
  }, [applyZoom]);
  const zoomOut = useCallback(() => {
    const el = viewportRef.current;
    if (!el) {
      applyZoom(viewRef.current.zoom - ZOOM_STEP);
      return;
    }
    const rect = el.getBoundingClientRect();
    applyZoom(viewRef.current.zoom - ZOOM_STEP, {
      x: rect.left + rect.width / 2,
      y: rect.top + rect.height / 2,
    });
  }, [applyZoom]);

  useEffect(() => {
    setView(FIT_VIEW);
    pointers.current.clear();
    pinchRef.current = null;
    panRef.current = null;
    dragZoomArmedRef.current = false;
  }, [resetKey]);

  useEffect(() => {
    const el = viewportRef.current;
    if (!el) return;
    const onWheelNative = (e: WheelEvent) => {
      if (!enabled) return;
      e.preventDefault();
      const factor = Math.exp(-e.deltaY * 0.0016);
      const next = viewRef.current.zoom * factor;
      applyZoom(next, { x: e.clientX, y: e.clientY });
    };
    el.addEventListener("wheel", onWheelNative, { passive: false });
    return () => el.removeEventListener("wheel", onWheelNative);
  }, [enabled, applyZoom, resetKey]);

  function onPointerDown(e: ReactPointerEvent) {
    if (!enabled) return;
    // Ignore secondary buttons / toolbar chrome (stopPropagation there)
    if (e.button !== 0 && e.pointerType === "mouse") return;
    const el = viewportRef.current;
    if (!el) return;
    el.setPointerCapture(e.pointerId);
    pointers.current.set(e.pointerId, { x: e.clientX, y: e.clientY });
    movedRef.current = false;
    dragZoomArmedRef.current = viewRef.current.zoom <= 1.001;

    if (pointers.current.size === 2) {
      const pts = [...pointers.current.values()];
      const dist = Math.hypot(pts[0].x - pts[1].x, pts[0].y - pts[1].y);
      pinchRef.current = { dist: Math.max(dist, 1), zoom: viewRef.current.zoom };
      panRef.current = null;
      dragZoomArmedRef.current = false;
      return;
    }

    panRef.current = {
      x: e.clientX,
      y: e.clientY,
      ox: viewRef.current.x,
      oy: viewRef.current.y,
    };
  }

  function onPointerMove(e: ReactPointerEvent) {
    if (!enabled || !pointers.current.has(e.pointerId)) return;
    const prev = pointers.current.get(e.pointerId)!;
    const dist = Math.hypot(e.clientX - prev.x, e.clientY - prev.y);
    if (dist > 6) movedRef.current = true;
    pointers.current.set(e.pointerId, { x: e.clientX, y: e.clientY });

    if (pointers.current.size === 2 && pinchRef.current) {
      const pts = [...pointers.current.values()];
      const d = Math.hypot(pts[0].x - pts[1].x, pts[0].y - pts[1].y);
      const mid = { x: (pts[0].x + pts[1].x) / 2, y: (pts[0].y + pts[1].y) / 2 };
      const next = pinchRef.current.zoom * (d / pinchRef.current.dist);
      applyZoom(next, mid);
      return;
    }

    if (!panRef.current) return;

    const el = viewportRef.current;
    if (!el) return;

    // Drag from fit: bump zoom around the pointer so pan can explore
    if (dragZoomArmedRef.current && movedRef.current && viewRef.current.zoom <= 1.001) {
      dragZoomArmedRef.current = false;
      const rect = el.getBoundingClientRect();
      const zoom = 2;
      const cx = e.clientX - rect.left - rect.width / 2;
      const cy = e.clientY - rect.top - rect.height / 2;
      // From fit (zoom=1,x=0,y=0): keep the grab point under the cursor
      const next = clampPan(
        { zoom, x: cx - cx * zoom, y: cy - cy * zoom },
        rect.width,
        rect.height,
      );
      viewRef.current = next;
      setView(next);
      panRef.current = { x: e.clientX, y: e.clientY, ox: next.x, oy: next.y };
      return;
    }

    if (viewRef.current.zoom <= 1.001) return;

    const dx = e.clientX - panRef.current.x;
    const dy = e.clientY - panRef.current.y;
    setView((prev) =>
      clampPan(
        {
          ...prev,
          x: panRef.current!.ox + dx,
          y: panRef.current!.oy + dy,
        },
        el.clientWidth,
        el.clientHeight,
      ),
    );
  }

  function onPointerUp(e: ReactPointerEvent) {
    if (!enabled) return;
    const wasPinch = Boolean(pinchRef.current);
    pointers.current.delete(e.pointerId);
    if (pointers.current.size < 2) pinchRef.current = null;
    if (pointers.current.size === 0) {
      panRef.current = null;
      dragZoomArmedRef.current = false;
    }
    try {
      viewportRef.current?.releasePointerCapture(e.pointerId);
    } catch {
      /* ignore */
    }

    if (wasPinch || movedRef.current || pointers.current.size > 0) return;

    const now = Date.now();
    if (now - lastTapRef.current < 300) {
      if (viewRef.current.zoom > 1.05) fit();
      else applyZoom(ZOOM_DOUBLE_TAP, { x: e.clientX, y: e.clientY });
      lastTapRef.current = 0;
    } else {
      lastTapRef.current = now;
    }
  }

  const setZoomPercent = useCallback(
    (percent: number) => {
      const el = viewportRef.current;
      const rect = el?.getBoundingClientRect();
      applyZoom(
        percent / 100,
        rect ? { x: rect.left + rect.width / 2, y: rect.top + rect.height / 2 } : undefined,
      );
    },
    [applyZoom],
  );

  const controls: ZoomControls = {
    zoom: view.zoom,
    isZoomed: view.zoom > 1.01 || view.x !== 0 || view.y !== 0,
    canZoomIn: view.zoom < ZOOM_MAX - 0.001,
    canZoomOut: view.zoom > ZOOM_MIN + 0.001,
    canFit: view.zoom !== 1 || view.x !== 0 || view.y !== 0,
    zoomIn,
    zoomOut,
    fit,
    setZoomPercent,
  };

  useEffect(() => {
    onControlsChange?.(controls);
  }, [view.zoom, view.x, view.y, enabled, zoomIn, zoomOut, fit, setZoomPercent, onControlsChange]);

  return (
    <div className={className ? `zoom-viewport-wrap ${className}` : "zoom-viewport-wrap"}>
      <div
        ref={viewportRef}
        className="preview-viewport"
        data-zoomed={view.zoom > 1.01 ? "true" : "false"}
        data-editable={enabled ? "true" : "false"}
        onPointerDown={onPointerDown}
        onPointerMove={onPointerMove}
        onPointerUp={onPointerUp}
        onPointerCancel={onPointerUp}
      >
        <div
          className="preview-media-frame"
          style={{
            transform: `translate3d(${view.x}px, ${view.y}px, 0) scale(${view.zoom})`,
          }}
        >
          {children}
        </div>

        {enabled && (
          <div className="preview-hint" aria-hidden>
            {view.zoom > 1.01 ? hintFit : hintZoom}
          </div>
        )}
      </div>
      {chrome?.(controls)}
    </div>
  );
}
