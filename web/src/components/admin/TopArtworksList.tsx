import { Eye, Heart } from "lucide-react";
import type { TopArtwork } from "../../lib/dashboardApi";

/**
 * Danh sách "Top 10 tác phẩm hàng đầu" - sắp theo engagement_score (view +
 * reaction) từ backend, chỉ hiển thị lại thứ tự đó (không tự sort lại ở FE).
 */
export function TopArtworksList({ items }: { items: TopArtwork[] }) {
  if (items.length === 0) {
    return <p className="admin-empty-note">Chưa có tác phẩm nào để xếp hạng.</p>;
  }

  return (
    <ol className="top-artworks-list">
      {items.map((item, index) => (
        <li key={item.id} className="top-artworks-item">
          <span className="top-artworks-rank">{index + 1}</span>
          <img
            className="top-artworks-thumb"
            src={item.thumbnail_url || item.image_url}
            alt=""
            loading="lazy"
          />
          <div className="top-artworks-info">
            <strong>{item.title}</strong>
            <span>{item.student_name}</span>
          </div>
          <div className="top-artworks-stats">
            <span title="Lượt xem">
              <Eye size={14} /> {item.view_count}
            </span>
            <span title="Lượt cảm xúc">
              <Heart size={14} /> {item.reaction_count}
            </span>
          </div>
        </li>
      ))}
    </ol>
  );
}
