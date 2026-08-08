import { useEffect, useState, type CSSProperties } from "react";

type PreviewImageProps = {
  src: string;
  alt: string;
  className?: string;
  style?: CSSProperties;
  /** Cover (thumbs) vs contain (stage) */
  fit?: "contain" | "cover";
  draggable?: boolean;
  onReady?: () => void;
};

type LoadState = "loading" | "ready" | "error";

export function PreviewImage({
  src,
  alt,
  className,
  style,
  fit = "contain",
  draggable = false,
  onReady,
}: PreviewImageProps) {
  const [state, setState] = useState<LoadState>("loading");

  useEffect(() => {
    setState("loading");
  }, [src]);

  return (
    <div
      className={className ? `preview-img-wrap ${className}` : "preview-img-wrap"}
      data-fit={fit}
      data-state={state}
      aria-busy={state === "loading" ? true : undefined}
    >
      {state === "loading" && <div className="preview-skeleton" aria-hidden />}
      {state === "error" ? (
        <p className="preview-fallback preview-img-error">Không xem trước được ảnh này.</p>
      ) : (
        <img
          src={src}
          alt={alt}
          draggable={draggable}
          decoding="async"
          style={style}
          className="preview-img"
          data-ready={state === "ready" ? "true" : "false"}
          onLoad={() => {
            setState("ready");
            onReady?.();
          }}
          onError={() => setState("error")}
        />
      )}
    </div>
  );
}
