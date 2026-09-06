import { useEffect, useRef, useState, useSyncExternalStore } from "react";
import { useLocation } from "react-router-dom";

/**
 * Thanh tiến trình mảnh ở đỉnh màn hình trong lúc chuyển trang (kiểu
 * YouTube/GitHub).
 *
 * Vì sao không phải spinner giữa màn hình: React Router 7 điều hướng bên
 * trong startTransition, nên khi chunk React.lazy của trang đích còn đang
 * tải, React GIỮ NGUYÊN trang cũ trên màn hình thay vì thay bằng fallback
 * của Suspense. Đó là hành vi tốt - mắt người xem không mất điểm bám - nhưng
 * nó để lại một khoảng im lặng: bấm vào link xong không thấy gì đổi, trên
 * mạng chậm thì tưởng app không nhận cú bấm. Thanh này lấp đúng khoảng đó.
 *
 * Cách phát hiện "đang chờ": so URL THẬT của trình duyệt với location mà cây
 * React đã commit. Hai giá trị lệch nhau đúng bằng quãng transition đang
 * treo - URL đổi ngay khi bấm link, còn useLocation chỉ đổi khi trang mới
 * commit xong.
 *
 * Phải đọc URL qua useSyncExternalStore chứ không phải state thường: giá trị
 * lấy từ external store không bị React "giữ lại" cùng cây cũ trong lúc
 * transition treo, nên nó là thứ duy nhất trong cây này còn nhìn thấy được
 * địa chỉ mới. (useNavigation của RR7 làm sẵn việc này nhưng chỉ chạy với
 * data router; app đang dùng BrowserRouter.)
 *
 * Tiến trình là GIẢ, và cố ý như vậy: trình duyệt không cho biết chunk đã
 * tải bao nhiêu phần trăm. Thanh bò nhanh dần chậm rồi gần như đứng lại ở
 * quãng 90%; khi trang mới commit thì chạy nốt lên 100% và tan. Thà nằm im
 * gần đích một lúc còn hơn chạm 100% rồi kẹt - cái sau trông như hỏng.
 *
 * Chuyển động do CSS animation lo trọn vẹn, React chỉ đổi trạng thái (đang
 * chạy / đang đóng / ẩn). Bản đầu tiên nhích phần trăm bằng setInterval mỗi
 * 90ms và bị giật rõ: mỗi nhịp là một lần render, và transition của nhịp
 * trước luôn bị nhịp sau cắt ngang giữa chừng nên không nhịp nào chạy hết.
 * Giao cho compositor thì hoạt cảnh chạy liền một mạch, không phụ thuộc
 * React render kịp hay không.
 */

/**
 * Trễ trước khi hiện. Phần lớn chunk về trong khoảng này trên mạng bình
 * thường; chớp một thanh rồi tắt ngay còn khó chịu hơn là không có gì.
 */
const SHOW_DELAY_MS = 160;
/** Thời gian chạy nốt từ chỗ đang dở tới 100% - khớp CSS. */
const FINISH_MS = 320;
/** Thời gian tan sau khi chạm 100% - khớp CSS. */
const FADE_MS = 260;

/* ------------------------------------------------------------------ */
/* Store URL thật của trình duyệt                                      */
/* ------------------------------------------------------------------ */

const urlListeners = new Set<() => void>();

function notifyUrlChange() {
  urlListeners.forEach((fn) => fn());
}

/**
 * pushState/replaceState không bắn sự kiện nào - popstate chỉ nổ khi người
 * dùng bấm nút quay lại. Mà điều hướng trong SPA đi qua đúng hai hàm đó, tức
 * là phần lớn chuyển trang sẽ không được nghe thấy nếu không vá.
 *
 * Vá một lần ở module scope (không phải trong effect): nhiều lần vá chồng
 * lên nhau sẽ nhân bản thông báo, và hàm gốc phải được giữ nguyên hành vi -
 * chỉ thêm tiếng gọi phía sau, không đổi giá trị trả về.
 */
if (typeof window !== "undefined") {
  for (const method of ["pushState", "replaceState"] as const) {
    const original = history[method];
    history[method] = function patched(this: History, ...args: Parameters<History["pushState"]>) {
      const result = original.apply(this, args);
      notifyUrlChange();
      return result;
    };
  }
  window.addEventListener("popstate", notifyUrlChange);
}

function subscribeUrl(onChange: () => void) {
  urlListeners.add(onChange);
  return () => {
    urlListeners.delete(onChange);
  };
}

function getUrlSnapshot() {
  return typeof window === "undefined" ? "/" : window.location.pathname + window.location.search;
}

/** Ba trạng thái nhìn thấy được của thanh. */
type Phase = "idle" | "running" | "finishing";

export function RouteProgress() {
  // URL thật của trình duyệt - đổi ngay khi bấm link.
  const browserUrl = useSyncExternalStore(subscribeUrl, getUrlSnapshot, getUrlSnapshot);
  // Địa chỉ mà cây React đã commit - chỉ đổi khi trang mới thực sự lên
  // màn hình. Chừng nào còn lệch với browserUrl là còn đang chờ.
  const location = useLocation();
  const committedUrl = location.pathname + location.search;

  const [phase, setPhase] = useState<Phase>("idle");
  const phaseRef = useRef<Phase>("idle");
  phaseRef.current = phase;

  const pending = browserUrl !== committedUrl;

  useEffect(() => {
    if (!pending) {
      // Chưa từng hiện thì không có gì để kết thúc - chunk về trước cả khi
      // hết SHOW_DELAY_MS, im lặng là đúng.
      if (phaseRef.current !== "running") return;

      // Đang chạy dở mà xong: chuyển sang giai đoạn đóng. CSS lo phần chạy
      // nốt tới 100% rồi tan; ở đây chỉ cần hẹn giờ dọn khỏi DOM.
      setPhase("finishing");
      const t = window.setTimeout(() => setPhase("idle"), FINISH_MS + FADE_MS);
      return () => window.clearTimeout(t);
    }

    const showTimer = window.setTimeout(() => setPhase("running"), SHOW_DELAY_MS);
    return () => window.clearTimeout(showTimer);
  }, [pending]);

  if (phase === "idle") return null;

  return (
    <div className={`route-progress is-${phase}`} role="presentation">
      <div className="route-progress-bar" />
    </div>
  );
}
