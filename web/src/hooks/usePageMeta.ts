import { useEffect } from "react";

interface PageMetaOptions {
  title: string;
  description: string;
  /** Đường dẫn tương đối để dựng canonical, vd "/tac-pham-tieu-bieu". Mặc
   * định dùng location.pathname hiện tại nếu bỏ trống - truyền tường minh
   * cho trang có query param lọc/mở modal (?tranh=, ?khu-vuc=...) để
   * canonical luôn trỏ về path gốc, tránh Google coi mỗi biến thể query là
   * một trang riêng (duplicate content). */
  canonicalPath?: string;
}

/**
 * Cập nhật <title>, <meta name="description">, <link rel="canonical"> theo
 * trang hiện tại. Dùng DOM API trực tiếp thay vì thư viện quản lý <head>
 * (vd react-helmet-async) - mỗi trang public chỉ có đúng một tầng gọi hook
 * này, không có component con nào cần ghi đè, nên không cần cơ chế
 * merge/ưu tiên theo cây component mà các thư viện đó giải quyết.
 *
 * Dọn dẹp: KHÔNG trả title/description về giá trị cũ khi unmount - trang kế
 * tiếp luôn tự gọi hook này và ghi đè ngay lúc mount, nên tránh cleanup ở
 * đây chỉ để không có một nhịp "tên cũ rồi lại đổi" giữa hai lần set.
 */
export function usePageMeta({ title, description, canonicalPath }: PageMetaOptions): void {
  useEffect(() => {
    document.title = title;

    let metaDesc = document.querySelector<HTMLMetaElement>('meta[name="description"]');
    if (!metaDesc) {
      metaDesc = document.createElement("meta");
      metaDesc.setAttribute("name", "description");
      document.head.appendChild(metaDesc);
    }
    metaDesc.setAttribute("content", description);

    let canonical = document.querySelector<HTMLLinkElement>('link[rel="canonical"]');
    if (!canonical) {
      canonical = document.createElement("link");
      canonical.setAttribute("rel", "canonical");
      document.head.appendChild(canonical);
    }
    const path = canonicalPath ?? window.location.pathname;
    canonical.setAttribute("href", `${window.location.origin}${path}`);
  }, [title, description, canonicalPath]);
}
