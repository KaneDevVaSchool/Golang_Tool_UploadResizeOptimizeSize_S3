import { useEffect } from "react";

/**
 * index.css khoá html/body/#root ở height:100%+overflow:hidden - thiết kế
 * riêng cho UploadTool (1 màn hình cố định, không cuộn). Các trang mới
 * (admin, public) có nội dung dài hơn viewport nên cần cuộn được - hook này
 * gắn class "scrollable-page" lên <body> khi mount, gỡ khi unmount.
 *
 * Rule CSS đi kèm được khai báo ở CẢ admin.css và public.css - mỗi khu vực
 * chỉ import stylesheet của mình, nên khu nào cũng phải tự có rule mở khoá,
 * không được dựa vào stylesheet của khu kia đã tình cờ được nạp.
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
