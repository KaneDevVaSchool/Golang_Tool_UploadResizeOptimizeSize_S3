/**
 * Fallback hiển thị trong lúc chunk của route đang tải (React.lazy trong
 * App.tsx).
 *
 * Cố ý giữ cực nhẹ và không phụ thuộc stylesheet nào: nó phải render được cả
 * ở route public (public.css) lẫn route admin (admin.css), mà hai file này
 * chỉ được nạp kèm chunk tương ứng — tức là ngay lúc fallback hiện ra thì
 * stylesheet của trang đích có thể còn chưa tới. Vì vậy style để inline.
 *
 * Trễ 180ms trước khi hiện: chunk thường về trong khoảng đó trên mạng bình
 * thường, chớp một spinner rồi tắt ngay còn khó chịu hơn là một khoảng lặng
 * ngắn. Dùng CSS animation-delay chứ không phải setTimeout + state để không
 * tốn một vòng render.
 */
export function RouteFallback() {
  return (
    <div
      role="status"
      aria-live="polite"
      style={{
        minHeight: "60vh",
        display: "grid",
        placeItems: "center",
        gap: "0.9rem",
        padding: "3rem 1rem",
      }}
    >
      <style>{`
        @keyframes route-fallback-in {
          to { opacity: 1; }
        }
        @keyframes route-fallback-spin {
          to { transform: rotate(360deg); }
        }
        .route-fallback-inner {
          opacity: 0;
          animation: route-fallback-in 0.25s ease-out 0.18s forwards;
          display: grid;
          justify-items: center;
          gap: 0.9rem;
        }
        .route-fallback-ring {
          width: 34px;
          height: 34px;
          border-radius: 50%;
          border: 3px solid rgba(120, 100, 80, 0.18);
          border-top-color: rgba(120, 100, 80, 0.62);
          animation: route-fallback-spin 0.75s linear infinite;
        }
        .route-fallback-text {
          margin: 0;
          font-size: 0.9rem;
          color: rgba(70, 58, 48, 0.62);
        }
        /* Người đã tắt chuyển động: bỏ vòng xoay, chỉ để lại dòng chữ. */
        @media (prefers-reduced-motion: reduce) {
          .route-fallback-inner { opacity: 1; animation: none; }
          .route-fallback-ring { animation: none; }
        }
      `}</style>
      <div className="route-fallback-inner">
        <div className="route-fallback-ring" aria-hidden />
        <p className="route-fallback-text">Đang tải…</p>
      </div>
    </div>
  );
}
