import { AnimatePresence, motion } from "framer-motion";
import { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { playErrorSound, playInfoSound, playSuccessSound } from "../lib/sound";
import { subscribeToast, type ToastPayload, type ToastVariant } from "../lib/toastBus";

const MAX_VISIBLE = 4;

const VARIANT_ICON: Record<ToastVariant, string> = {
  success: "✓",
  error: "✕",
  info: "ℹ",
  warning: "!",
};

function playSoundFor(variant: ToastVariant) {
  if (variant === "success") playSuccessSound();
  else if (variant === "error") playErrorSound();
  else playInfoSound();
}

type LiveToast = ToastPayload & {
  remaining: number; // ms còn lại - cập nhật khi pause/resume qua hover
  startedAt: number;
  paused: boolean;
};

/**
 * Toast host: lắng nghe toastBus, render qua Portal lên document.body (nổi
 * trên mọi layout kể cả modal), progress-bar tự đóng (pause khi hover), tối
 * đa MAX_VISIBLE toast cùng lúc (thêm mới sẽ đẩy toast cũ nhất ra nếu đầy).
 * Mount 1 lần duy nhất trong App.tsx, dùng chung cho UploadTool cũ + admin +
 * public (không phân biệt theo route).
 */
export function ToastHost() {
  const [toasts, setToasts] = useState<LiveToast[]>([]);
  const timers = useRef<Map<string, number>>(new Map());

  useEffect(() => {
    const unsubscribe = subscribeToast((payload) => {
      playSoundFor(payload.variant);
      setToasts((prev) => {
        const next: LiveToast[] = [
          ...prev,
          { ...payload, remaining: payload.duration, startedAt: Date.now(), paused: false },
        ];
        return next.length > MAX_VISIBLE ? next.slice(next.length - MAX_VISIBLE) : next;
      });
    });
    return unsubscribe;
  }, []);

  function dismiss(id: string) {
    const t = timers.current.get(id);
    if (t) {
      window.clearTimeout(t);
      timers.current.delete(id);
    }
    setToasts((prev) => prev.filter((x) => x.id !== id));
  }

  function scheduleDismiss(t: LiveToast) {
    if (!Number.isFinite(t.remaining)) return; // error: không tự đóng
    const existing = timers.current.get(t.id);
    if (existing) window.clearTimeout(existing);
    const handle = window.setTimeout(() => dismiss(t.id), t.remaining);
    timers.current.set(t.id, handle);
  }

  useEffect(() => {
    toasts.forEach((t) => {
      if (!t.paused && !timers.current.has(t.id)) scheduleDismiss(t);
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [toasts]);

  function pause(id: string) {
    setToasts((prev) =>
      prev.map((t) => {
        if (t.id !== id || t.paused) return t;
        const elapsed = Date.now() - t.startedAt;
        const remaining = Math.max(0, t.remaining - elapsed);
        const timer = timers.current.get(id);
        if (timer) {
          window.clearTimeout(timer);
          timers.current.delete(id);
        }
        return { ...t, paused: true, remaining };
      }),
    );
  }

  function resume(id: string) {
    setToasts((prev) =>
      prev.map((t) => (t.id === id ? { ...t, paused: false, startedAt: Date.now() } : t)),
    );
  }

  useEffect(() => {
    return () => {
      timers.current.forEach((handle) => window.clearTimeout(handle));
      timers.current.clear();
    };
  }, []);

  return createPortal(
    <div className="toast-host" aria-live="polite" aria-atomic="false">
      <AnimatePresence initial={false}>
        {toasts.map((t) => (
          <motion.div
            key={t.id}
            className={`toast-item toast-${t.variant}`}
            role={t.variant === "error" ? "alert" : "status"}
            layout
            initial={{ opacity: 0, x: 40, scale: 0.95 }}
            animate={{ opacity: 1, x: 0, scale: 1 }}
            exit={{ opacity: 0, x: 40, scale: 0.95, transition: { duration: 0.2 } }}
            transition={{ duration: 0.32, ease: [0.22, 1, 0.36, 1] }}
            onMouseEnter={() => pause(t.id)}
            onMouseLeave={() => resume(t.id)}
          >
            <span className="toast-icon" aria-hidden>
              {VARIANT_ICON[t.variant]}
            </span>
            <p className="toast-message">{t.message}</p>
            <button type="button" className="toast-close" aria-label="Đóng thông báo" onClick={() => dismiss(t.id)}>
              ×
            </button>
            {Number.isFinite(t.duration) && (
              <motion.span
                className="toast-progress"
                initial={{ scaleX: 1 }}
                animate={{ scaleX: t.paused ? undefined : 0 }}
                transition={t.paused ? { duration: 0 } : { duration: t.remaining / 1000, ease: "linear" }}
              />
            )}
          </motion.div>
        ))}
      </AnimatePresence>
    </div>,
    document.body,
  );
}
