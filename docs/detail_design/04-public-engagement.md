# 04 — Tương tác public ẩn danh

Trang public cho phép thả cảm xúc, bình luận và tính lượt xem **mà không cần đăng ký tài
khoản**. Tài liệu này mô tả cơ chế định danh nhẹ đứng sau và những gì nó bảo vệ được — cũng
như những gì nó không.

**Code chính**: `internal/handlers/public_handler.go`, `internal/repository/{reaction,comment,artwork_view}_repository.go`

## 1. Yêu cầu nghiệp vụ và hệ quả

Yêu cầu: *phụ huynh mở link con mình gửi là thả tim và khen được ngay, không đăng nhập.*

Hệ quả kỹ thuật: không có danh tính đáng tin. Mọi cơ chế "một người một phiếu" chỉ có thể
dựa vào tín hiệu phía trình duyệt — vốn giả mạo được. Thiết kế chấp nhận điều này và chọn
mục tiêu thực tế hơn:

| Mục tiêu | Có đạt được không |
|---|---|
| Chặn người dùng bình thường bấm trùng nhiều lần | ✅ Có |
| Chặn số liệu bị thổi phồng do tải lại trang | ✅ Có |
| Chặn spam bình luận hàng loạt | ✅ Phần lớn — rate limit theo IP |
| Chặn người cố tình gian lận có kỹ thuật | ❌ Không — và không đặt mục tiêu này |

## 2. `visitor_token`

Một UUID sinh ở trình duyệt lần đầu vào trang, lưu `localStorage`
(`web/src/lib/visitorToken.ts`), gửi kèm mọi thao tác tương tác.

```text
Lần đầu vào trang     →  sinh UUID  →  lưu localStorage
Mọi request sau đó    →  đọc lại và gửi kèm
                          - reaction/comment: trong JSON body
                          - view/xoá comment: trong query param
```

⚠️ **Đây không phải xác thực.** Cần hiểu rõ ba giới hạn:

1. Xoá dữ liệu duyệt web = mất quyền xoá bình luận mình đã viết.
2. Máy khác/trình duyệt khác = token khác = tính là người mới.
3. Người dùng tự sửa token trong localStorage để thả cảm xúc nhiều lần — làm được.

Điểm thứ ba là lý do rate limit theo IP tồn tại: nó chặn kịch bản lạm dụng tự động, còn
`visitor_token` chỉ lo phần "đừng đếm trùng cho người dùng thật thà".

## 3. Cảm xúc

### Sáu loại, cố định

`like`, `love`, `haha`, `wow`, `sad`, `angry` — khai báo ở `models/reaction.go` và ràng buộc
bằng `ENUM` trong MySQL. Handler kiểm tra qua `models.ValidReactionTypes` trước khi chạm DB,
nên giá trị lạ bị chặn ở tầng ứng dụng với thông báo rõ ràng thay vì để MySQL báo lỗi.

### Quy tắc: nhiều loại được, trùng loại thì không

Ràng buộc `UNIQUE(artwork_id, visitor_token, reaction_type)` cho phép **một người thả nhiều
loại cảm xúc khác nhau** trên cùng tác phẩm, nhưng mỗi loại chỉ tính một lần.

Repository dùng `INSERT IGNORE` (`reaction_repository.go:36`) nên bấm lại nhiều lần là vô
hại — không lỗi, không nhân bản. Idempotent theo đúng nghĩa: gửi lại request cho cùng kết
quả.

```text
POST   /api/v1/public/artworks/{id}/reactions          body: {reaction_type, visitor_token}
DELETE /api/v1/public/artworks/{id}/reactions/{type}?visitor_token=...
```

Cả hai đều trả về **bảng đếm mới nhất** sau thao tác, để frontend cập nhật giao diện ngay
mà không phải gọi thêm một vòng.

## 4. Bình luận

### Ràng buộc đầu vào

| Trường | Ràng buộc | Kiểm ở đâu |
|---|---|---|
| `display_name` | Bắt buộc, ≤ 100 **ký tự** | `public_handler.go:535-542` |
| `content` | Bắt buộc, ≤ 1000 **ký tự** | `public_handler.go:543-550` |
| `visitor_token` | Bắt buộc | `public_handler.go:551-554` |

Độ dài đếm bằng `len([]rune(...))` — **ký tự**, không phải byte. Quan trọng với tiếng Việt:
"Nguyễn" là 6 ký tự nhưng 8 byte. Đếm byte sẽ từ chối oan những cái tên hợp lệ.

Hằng số ở handler khớp đúng `VARCHAR` trong migration 011. Sửa một nơi phải sửa nơi kia.

### Chuyện escape HTML — một lỗi đã sửa

Code hiện tại **lưu văn bản nguyên bản**, không escape trước khi ghi DB. Trên đường trả về,
`html.UnescapeString()` được áp dụng (`public_handler.go:504-505`).

Nguyên do: một phiên bản trước từng escape HTML *trước khi lưu*, khiến React hiển thị
nguyên chuỗi `&amp;` `&quot;` cho người dùng — vì React vốn đã tự escape khi render, thành
ra escape hai lần. `UnescapeString` khi trả JSON là để **sửa dữ liệu cũ** đã bị hỏng theo
cách đó.

Điều này **không mở lại lỗ hổng XSS**: React escape ở đúng thời điểm render, và JSON encoder
escape ở đúng thời điểm truyền. Nguyên tắc chung là escape ở *output context*, không phải
lúc lưu trữ.

### Quyền xoá

```text
DELETE /api/v1/public/artworks/{id}/comments/{commentID}?visitor_token=...
```

`DeleteOwned()` kiểm tra cả ba điều kiện trong một câu lệnh: đúng comment, đúng tác phẩm,
đúng visitor_token. Không khớp → không xoá.

Handler trả **cùng một lỗi 404** cho cả trường hợp "không tồn tại" và "không phải của bạn"
(`public_handler.go:599-601`). Chủ đích: không để ai dò xem bình luận nào thuộc về ai bằng
cách so sánh mã lỗi trả về.

Trường `can_delete` trong response danh sách được tính bằng cách so `visitor_token` gửi lên
với token của từng bình luận — frontend chỉ hiện nút xoá ở bình luận của chính người đó.

### Kiểm duyệt

Cột `is_hidden` tồn tại, và `ListByArtwork(ctx, id, false)` lọc bỏ bình luận bị ẩn khỏi API
public. Nhưng ⚠️ **chưa có endpoint admin nào bật/tắt cờ này** — hiện phải `UPDATE` bằng SQL
tay. Xem [plan/02-roadmap.md](../plan/02-roadmap.md).

## 5. Lượt xem — đếm mỗi lần mở, không chống trùng

`RecordView()` (`artwork_view_repository.go`) là nơi duy nhất `view_count` được tăng:

```text
① BEGIN
     INSERT artwork_views(...)
     UPDATE artworks SET view_count = view_count + 1
   COMMIT
② trả counted=true
```

Hai thao tác ở bước ① nằm trong **cùng một transaction**, nên `artwork_views` và
`view_count` không bao giờ lệch nhau. `counted` luôn `true` khi thành công — tham số này giữ
lại trong chữ ký hàm để tương thích lời gọi cũ, nhưng handler hiện không còn nhánh nào rẽ
theo giá trị của nó.

⚠️ **Đổi so với thiết kế ban đầu**: trước đây có chống trùng 24 giờ theo `visitor_token`
(một khách tải lại trang trong ngày không bị đếm thêm). Đã bỏ có chủ đích — **mỗi lần gọi
`GET /api/v1/public/artworks/{id}` kèm `visitor_token` đều +1 `view_count`**, kể cả tải lại
trang nhiều lần liên tiếp. Frontend gọi qua `recordArtworkView()`
(`web/src/lib/publicApi.ts`), có gộp request trùng lặp đang bay (`artworkViewInFlight`) để
một lần render không bắn nhiều request cùng lúc — nhưng không chặn việc gọi lại ở lần
render/mở trang sau.

⚠️ **Hạn chế còn lại**: bảng `artwork_views` chỉ tăng, không bao giờ được dọn — xem
[plan/03-risks.md](../plan/03-risks.md).

## 6. Trang chia sẻ có Open Graph

Vấn đề: SPA React trả về `index.html` trống cho mọi đường dẫn. Crawler của Facebook/Zalo
không chạy JavaScript, nên khi ai đó dán link tác phẩm sẽ không thấy ảnh preview.

Giải pháp: một đường dẫn riêng **render HTML phía server**.

```text
GET /chia-se/tac-pham/{id}
  │
  ├─ Crawler (Facebook, Zalo…)  → đọc thẻ og:title/og:description/og:image → hiện preview
  └─ Người dùng thật            → <meta http-equiv="refresh"> → /tac-pham-tieu-bieu?tranh={id}
```

Chi tiết đáng chú ý:

- **Nhận diện scheme qua `X-Forwarded-Proto`** (`public_handler.go:340-346`) — sau reverse
  proxy, `r.TLS` luôn `nil`, nên nếu không đọc header này thì `og:url` sẽ ra `http://` và
  Facebook có thể từ chối.
- **Template dùng `html/template`**, không phải `text/template` — tự escape mọi giá trị
  chèn vào, nên tên tác phẩm chứa ký tự đặc biệt không phá được HTML.
- **Cache 5 phút** (`Cache-Control: public, max-age=300`) — crawler gọi lại nhiều lần cho
  cùng một link.
- Tác phẩm chưa publish trả **404**, không lộ sự tồn tại.

## 7. Phòng vệ lạm dụng

Ba lớp, mỗi lớp lo một chuyện:

| Lớp | Cơ chế | Chặn được gì |
|---|---|---|
| 1 | Rate limit 20 req/phút/IP cho `POST`/`DELETE` | Spam tự động từ một nguồn |
| 2 | `visitor_token` + ràng buộc UNIQUE | Bấm trùng, tải lại trang |
| 3 | Giới hạn độ dài + validate kiểu | Payload rác, dữ liệu vượt cột DB |

Rate limit lớp 1 **chỉ áp cho method ghi**. Người xem lướt trang (toàn `GET`) không bao giờ
chạm giới hạn này (nhánh `publicHandlerChain` trong `GetServerHandler`) — một quyết định quan trọng, vì trang public có
thể có nhiều người xem cùng lúc từ cùng một mạng trường học chung IP.

⚠️ Đằng sau reverse proxy, rate limit chỉ đúng khi Nginx truyền `X-Forwarded-For` —
`middleware.GetClientIP()` đọc header này. Cấu hình Nginx trong repo đã có
(`deploy/nginx/*.conf`). Thiếu nó thì mọi request trông như đến từ `127.0.0.1` và **một
người spam sẽ khoá cả trang với mọi người**.
