import {
  useLayoutEffect,
  useRef,
  useState,
  type CSSProperties,
} from "react";
import { normalizeRotation } from "../lib/imageTransform";

type PreviewImageProps = {
  src: string;
  alt: string;
  className?: string;
  /** CSS rotate/flip — applied after sizing so the full frame stays visible */
  transformStyle?: string;
  /** Degrees used for contain math (90/270 swap bounds) */
  rotation?: number;
  draggable?: boolean;
  onReady?: () => void;
};

type LoadState = "loading" | "ready" | "error";
type NaturalSize = { w: number; h: number };
type BoxSize = { w: number; h: number };

function containBox(
  containerW: number,
  containerH: number,
  naturalW: number,
  naturalH: number,
  rotationDeg: number,
): BoxSize {
  if (containerW <= 0 || containerH <= 0 || naturalW <= 0 || naturalH <= 0) {
    return { w: 0, h: 0 };
  }
  const rot = normalizeRotation(rotationDeg);
  const swap = rot === 90 || rot === 270;
  // Bounds after CSS rotate — size the pre-rotate box so the rotated AABB fits.
  const boundW = swap ? naturalH : naturalW;
  const boundH = swap ? naturalW : naturalH;
  const scale = Math.min(containerW / boundW, containerH / boundH);
  return {
    w: Math.max(1, naturalW * scale),
    h: Math.max(1, naturalH * scale),
  };
}

export function PreviewImage({
  src,
  alt,
  className,
  transformStyle,
  rotation = 0,
  draggable = false,
  onReady,
}: PreviewImageProps) {
  const wrapRef = useRef<HTMLDivElement>(null);
  const imgRef = useRef<HTMLImageElement>(null);
  const onReadyRef = useRef(onReady);
  onReadyRef.current = onReady;
  const readyOnceRef = useRef(false);
  const [state, setState] = useState<LoadState>("loading");
  const [natural, setNatural] = useState<NaturalSize | null>(null);
  const [box, setBox] = useState<BoxSize>({ w: 0, h: 0 });

  const markReady = (img: HTMLImageElement) => {
    if (!img.naturalWidth || !img.naturalHeight) return;
    setNatural({ w: img.naturalWidth, h: img.naturalHeight });
    setState("ready");
    if (!readyOnceRef.current) {
      readyOnceRef.current = true;
      onReadyRef.current?.();
    }
  };

  // Reset + adopt cached decode in the same layout pass (lightbox reuses blob URLs).
  useLayoutEffect(() => {
    setState("loading");
    setNatural(null);
    setBox({ w: 0, h: 0 });
    readyOnceRef.current = false;

    const img = imgRef.current;
    if (img?.complete && img.naturalWidth > 0) {
      markReady(img);
    }
  }, [src]);

  useLayoutEffect(() => {
    const el = wrapRef.current;
    if (!el || !natural) return;

    const measure = () => {
      const next = containBox(el.clientWidth, el.clientHeight, natural.w, natural.h, rotation);
      setBox((prev) => (prev.w === next.w && prev.h === next.h ? prev : next));
    };

    measure();
    const ro = new ResizeObserver(measure);
    ro.observe(el);
    return () => ro.disconnect();
  }, [natural, rotation]);

  const imgStyle: CSSProperties = {
    width: box.w > 0 ? box.w : undefined,
    height: box.h > 0 ? box.h : undefined,
    transform: transformStyle,
  };

  return (
    <div
      ref={wrapRef}
      className={className ? `preview-img-wrap ${className}` : "preview-img-wrap"}
      data-state={state}
      aria-busy={state === "loading" ? true : undefined}
    >
      {state === "loading" && <div className="preview-skeleton" aria-hidden />}
      {state === "error" ? (
        <p className="preview-fallback preview-img-error">Không xem trước được ảnh này.</p>
      ) : (
        <img
          ref={imgRef}
          src={src}
          alt={alt}
          draggable={draggable}
          decoding="async"
          style={imgStyle}
          className="preview-img"
          data-ready={state === "ready" && box.w > 0 ? "true" : "false"}
          onLoad={(e) => markReady(e.currentTarget)}
          onError={() => setState("error")}
        />
      )}
    </div>
  );
}
