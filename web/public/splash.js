/*
  Gỡ splash khi React đã vẽ xong nội dung đầu tiên.

  Vì sao là file riêng chứ không nội tuyến trong index.html: CSP của app đặt
  script-src 'self' và cố ý KHÔNG nới 'unsafe-inline' (xem
  internal/middleware/security_headers.go) - script nội tuyến chính là hướng
  tấn công XSS đáng lo nhất. Cách còn lại là băm hash sha256 đoạn script rồi
  nhúng vào CSP, nhưng như thế mỗi lần sửa một ký tự ở đây là phải tính lại
  hash trong code Go; quên một lần thì splash kẹt vĩnh viễn trên production
  mà không ai thấy lỗi trong log server. Một request cùng origin rẻ hơn nhiều.

  Dùng MutationObserver chứ không phải sự kiện load: load bắn khi tài nguyên
  tải xong, còn cái ta cần là lúc MÀN HÌNH có nội dung - hai mốc này lệch
  nhau đúng bằng thời gian React mount.

  Có chốt chặn 8 giây: nếu bundle lỗi và React không bao giờ mount thì splash
  phải tự nhường chỗ, để người dùng còn thấy được trang trắng (và trình duyệt
  báo lỗi) thay vì kẹt vĩnh viễn ở màn hình chờ.
*/
(function () {
  var splash = document.getElementById("app-splash");
  var root = document.getElementById("root");
  if (!splash || !root) return;

  var removed = false;
  function dismiss() {
    if (removed) return;
    removed = true;
    observer.disconnect();
    clearTimeout(bailout);
    splash.classList.add("is-hidden");
    setTimeout(function () {
      splash.remove();
    }, 400);
  }

  var observer = new MutationObserver(function () {
    if (root.childElementCount > 0) dismiss();
  });
  observer.observe(root, { childList: true });

  var bailout = setTimeout(dismiss, 8000);
  if (root.childElementCount > 0) dismiss();
})();
