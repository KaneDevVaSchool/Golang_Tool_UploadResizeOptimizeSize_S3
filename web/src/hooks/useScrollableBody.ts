import { useEffect } from "react";

/**
 * index.css khoá html/body/#root ở height:100%+overflow:hidden - thiết kế
 * riêng cho UploadTool (1 màn hình cố định, không cuộn). Các trang mới
 * (admin, public) có nội dung dài hơn viewport nên cần cuộn được - hook này
 * gắn class "scrollable-page" lên <body> khi mount, gỡ khi unmount (xem CSS
 * tương ứng trong styles/admin.css / styles/public.css).
 */
export function useScrollableBody() {
  useEffect(() => {
    document.documentElement.classList.add("scrollable-page");
    document.body.classList.add("scrollable-page");
    return () => {
      document.documentElement.classList.remove("scrollable-page");
      document.body.classList.remove("scrollable-page");
    };
  }, []);
}
