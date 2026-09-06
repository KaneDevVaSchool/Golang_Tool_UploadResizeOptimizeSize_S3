/**
 * Fallback hiển thị khi Suspense thật sự phải treo trong lúc chunk của route
 * đang tải (React.lazy trong App.tsx).
 *
 * Khi nào thấy nó: điều hướng bằng <Link> chạy trong startTransition nên
 * React giữ trang cũ và người dùng chỉ thấy RouteProgress ở đỉnh - fallback
 * này KHÔNG hiện. Nó dành cho lúc không có trang cũ để giữ: mở thẳng một URL
 * con (dán link, bấm bookmark, tải lại trang), hoặc mạng quá chậm.
 *
 * Vì vậy nó không phải một spinner giữa khoảng trắng nữa mà là bộ khung của
 * trang sắp tới - dải tiêu đề, lưới khung tranh. Người xem thấy ngay bố cục
 * mình sắp nhận, và lúc nội dung thật thay vào thì không có cú nhảy layout.
 *
 * Cố ý giữ nhẹ và KHÔNG phụ thuộc stylesheet nào: fallback phải render được
 * cả ở route public (public.css) lẫn admin (admin.css), mà hai file này chỉ
 * được nạp kèm chunk tương ứng - tức là ngay lúc fallback hiện ra thì
 * stylesheet của trang đích có thể còn chưa tới. Nên style để inline.
 *
 * Trễ 180ms trước khi hiện: chunk thường về trong khoảng đó trên mạng bình
 * thường, chớp một khung xám rồi tắt ngay còn khó chịu hơn một khoảng lặng
 * ngắn. Dùng animation-delay chứ không phải setTimeout + state để không tốn
 * một vòng render.
 */
export function RouteFallback() {
  return (
    <div
      role="status"
      aria-live="polite"
      aria-busy="true"
      aria-label="Đang tải trang"
      style={{ padding: "clamp(1.5rem, 5vw, 3.5rem) 1rem", minHeight: "60vh" }}
    >
      <style>{`
        @keyframes route-skeleton-in {
          to { opacity: 1; }
        }
        /* Vệt sáng quét ngang - đủ để nói "đang chạy, chưa đứng hình", nhưng
           chậm và mờ để không kéo mắt khỏi nội dung thật khi nó thay vào. */
        @keyframes route-skeleton-sheen {
          to { background-position: 200% 0; }
        }
        .route-skeleton {
          opacity: 0;
          animation: route-skeleton-in 0.3s ease-out 0.18s forwards;
          max-width: 1180px;
          margin: 0 auto;
        }
        .route-skeleton-block {
          border-radius: 0.75rem;
          background: linear-gradient(
            100deg,
            rgba(120, 100, 88, 0.09) 30%,
            rgba(120, 100, 88, 0.16) 48%,
            rgba(120, 100, 88, 0.09) 66%
          );
          background-size: 200% 100%;
          animation: route-skeleton-sheen 1.5s linear infinite;
        }
        .route-skeleton-head {
          display: grid;
          justify-items: center;
          gap: 0.85rem;
          margin-bottom: clamp(1.75rem, 4vw, 2.75rem);
        }
        .route-skeleton-grid {
          display: grid;
          grid-template-columns: repeat(auto-fill, minmax(230px, 1fr));
          gap: clamp(1rem, 2.5vw, 1.75rem);
        }
        /* Khung tranh giả: ô vuông đúng như .featured-gallery-cell, kèm hai
           dòng chú thích bên dưới (tên tranh + tên học sinh). */
        .route-skeleton-frame {
          aspect-ratio: 1 / 1;
          border-radius: 0.9rem;
        }
        .route-skeleton-caption {
          display: grid;
          gap: 0.45rem;
          margin-top: 0.7rem;
        }
        /* Các ô lệch pha nhau để vệt sáng không quét đồng loạt như một khối
           - đồng loạt trông giả và ồn hơn nhiều. */
        .route-skeleton-cell:nth-child(2n) .route-skeleton-block {
          animation-delay: 0.18s;
        }
        .route-skeleton-cell:nth-child(3n) .route-skeleton-block {
          animation-delay: 0.36s;
        }
        @media (max-width: 480px) {
          .route-skeleton-grid {
            grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
          }
        }
        /* Người đã tắt chuyển động: bỏ vệt quét, giữ khung tĩnh. */
        @media (prefers-reduced-motion: reduce) {
          .route-skeleton { opacity: 1; animation: none; }
          .route-skeleton-block { animation: none; background-position: 0 0; }
        }
      `}</style>

      <div className="route-skeleton">
        <div className="route-skeleton-head">
          <div className="route-skeleton-block" style={{ width: "min(360px, 70%)", height: "2.1rem" }} />
          <div className="route-skeleton-block" style={{ width: "min(520px, 88%)", height: "0.9rem" }} />
        </div>

        <div className="route-skeleton-grid">
          {Array.from({ length: 6 }, (_, i) => (
            <div key={i} className="route-skeleton-cell">
              <div className="route-skeleton-block route-skeleton-frame" />
              <div className="route-skeleton-caption">
                <div className="route-skeleton-block" style={{ width: "72%", height: "0.7rem" }} />
                <div className="route-skeleton-block" style={{ width: "48%", height: "0.62rem" }} />
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
