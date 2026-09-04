// Pub/sub toast qua CustomEvent trên window - gọi được từ bất kỳ đâu (page,
// hook, lib) mà không cần Context/props drilling. ToastHost.tsx là subscriber
// duy nhất, mount 1 lần ở App.tsx.
export type ToastVariant = "success" | "error" | "info" | "warning";

export type ToastPayload = {
  id: string;
  variant: ToastVariant;
  message: string;
  duration: number;
};

const EVENT_NAME = "vas:toast";

// Duration mặc định theo variant - error đặt Infinity để KHÔNG tự đóng
// (người dùng phải tự tắt hoặc thao tác lại), khớp yêu cầu toast lỗi cần
// được chú ý rõ ràng hơn info/success.
const DEFAULT_DURATION: Record<ToastVariant, number> = {
  success: 3200,
  info: 3200,
  warning: 4200,
  error: Infinity,
};

function dispatch(variant: ToastVariant, message: string, durationMs?: number) {
  const payload: ToastPayload = {
    id: `${Date.now()}-${Math.random().toString(36).slice(2, 9)}`,
    variant,
    message,
    duration: durationMs ?? DEFAULT_DURATION[variant],
  };
  window.dispatchEvent(new CustomEvent<ToastPayload>(EVENT_NAME, { detail: payload }));
}

export const toast = {
  success: (message: string, durationMs?: number) => dispatch("success", message, durationMs),
  error: (message: string, durationMs?: number) => dispatch("error", message, durationMs),
  info: (message: string, durationMs?: number) => dispatch("info", message, durationMs),
  warning: (message: string, durationMs?: number) => dispatch("warning", message, durationMs),
};

export function subscribeToast(handler: (payload: ToastPayload) => void): () => void {
  const listener = (e: Event) => handler((e as CustomEvent<ToastPayload>).detail);
  window.addEventListener(EVENT_NAME, listener);
  return () => window.removeEventListener(EVENT_NAME, listener);
}
