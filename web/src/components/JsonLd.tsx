interface JsonLdProps {
  data: Record<string, unknown>;
}

/**
 * Chèn 1 thẻ <script type="application/ld+json"> chứa structured data
 * schema.org cho SEO (WebSite/CollectionPage/BreadcrumbList tuỳ trang).
 *
 * Dữ liệu đầu vào luôn là object dựng sẵn ở component gọi (tên trang, mô tả,
 * URL cố định) - không lấy trực tiếp từ input người dùng chưa qua kiểm soát
 * (bình luận, tên hiển thị...), nên rủi ro injection qua JSON.stringify ở
 * đây thấp.
 */
export function JsonLd({ data }: JsonLdProps) {
  return (
    <script
      type="application/ld+json"
      // eslint-disable-next-line react/no-danger -- JSON-LD bắt buộc phải là text node thô trong <script>, không qua children thường
      dangerouslySetInnerHTML={{ __html: JSON.stringify(data) }}
    />
  );
}
