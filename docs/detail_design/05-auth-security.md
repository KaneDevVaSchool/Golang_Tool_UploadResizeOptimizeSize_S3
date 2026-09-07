# 05 — Xác thực và bảo mật

**Code chính**: `internal/auth/`, `internal/handlers/admin_auth_handler.go`,
`internal/middleware/{admin_auth,csrf,apikey,ratelimit,clientip,botguard,security_headers,bodylimit,cors,concurrency}.go`

## 1. Ba mô hình xác thực song song

Hệ thống dùng ba cơ chế khác nhau cho ba nhóm người dùng — không phải sự thiếu nhất quán mà
là kết quả của ba yêu cầu khác nhau:

| Khu vực | Cơ chế | Vì sao chọn |
|---|---|---|
| `/api/v1/admin/*` | Session cookie (Google OAuth) | Admin là người thật, cần thu hồi được quyền ngay |
| `/api/v1/public/*` | Không xác thực + rate limit | Yêu cầu nghiệp vụ: tương tác không rào cản |
| `/api/v1/upload*` | API key (tuỳ chọn) | Client máy-với-máy, không có trình duyệt để giữ session |

## 2. Luồng đăng nhập admin

```text
① Người dùng bấm "Đăng nhập với Google"
   GET /auth/google/login
     ├─ chưa cấu hình Client ID/Secret → 503 với thông báo rõ ràng
     ├─ sinh state ngẫu nhiên 24 byte
     ├─ đặt cookie vas_oauth_state (HttpOnly, TTL 5 phút, SameSite=Lax)
     └─ redirect sang Google

② Google xác thực người dùng, gọi về
   GET /auth/google/callback?code=...&state=...
     ├─ đọc + xoá ngay cookie state, so khớp với query    ← chống CSRF
     ├─ đổi code lấy access token
     ├─ gọi Google UserInfo API
     ├─ kiểm tra email_verified == true                   ← bắt buộc
     ├─ kiểm tra email nằm trong danh sách cho phép
     ├─ tìm admin_users theo google_sub
     │     chưa có → tạo mới (chỉ khi đã qua kiểm tra ở trên)
     ├─ tạo session trong DB, đặt cookie HttpOnly
     └─ redirect về /admin

③ Mọi request admin sau đó
   cookie session → AdminAuthMiddleware → ValidateSession → gắn user vào context
```

### Vì sao bắt buộc `email_verified`

Whitelist so khớp theo **chuỗi email**. Nếu không kiểm tra `email_verified`, một người có
thể tạo tài khoản Google khai email trùng danh sách cho phép mà chưa xác minh, và đi thẳng
vào admin. Kiểm tra này (`admin_auth_handler.go:186-191`) đóng lối đó.

### Hai tầng danh sách cho phép

```text
ADMIN_ALLOWED_EMAILS không rỗng?
   ├─ Có  → CHỈ các email trong danh sách được vào (bỏ qua hoàn toàn kiểm tra domain)
   └─ Không → xét ADMIN_ALLOWED_EMAIL_DOMAIN
                ├─ có domain → email phải thuộc một trong các domain
                └─ rỗng     → KHÔNG GIỚI HẠN  ⚠️ chỉ dùng khi dev
```

Whitelist email cụ thể **ưu tiên cao hơn** domain (`admin_auth_handler.go:84-88`). Chủ đích:
domain trường có hàng nghìn tài khoản (giáo viên, học sinh), nhưng chỉ chín người trong ban
tổ chức được vào admin.

Khi `.env` để trống `ADMIN_ALLOWED_EMAILS`, code dùng danh sách mặc định gồm 9 tài khoản
ban quản trị, khai báo tại `config/builder.go:20` (`defaultAllowedAdminEmails`). Nghĩa là
**mặc định đã an toàn** — quên cấu hình không dẫn tới mở toang admin.

⚠️ Chỉ khi cả hai biến đều rỗng *và* danh sách mặc định bị sửa thành rỗng thì mới thành
"không giới hạn". Đừng làm vậy ở production.

### Tự tạo tài khoản khi đăng nhập lần đầu

Không có màn hình "mời admin". Người trong danh sách cho phép đăng nhập lần đầu sẽ được tạo
bản ghi `admin_users` tự động (`admin_auth_handler.go:206-217`). An toàn vì việc kiểm tra
danh sách đã diễn ra **trước** đó — danh sách cho phép chính là cơ chế mời.

## 3. Session

| Thuộc tính | Giá trị | Lý do |
|---|---|---|
| Nơi lưu | Bảng `admin_sessions` | Thu hồi được ngay; JWT thì không |
| Token | 32 byte `crypto/rand`, base64 url-safe | Không đoán được |
| TTL | `SESSION_TTL_HOURS`, mặc định 168 (7 ngày) | Cân bằng tiện dụng và rủi ro |
| Cookie | `HttpOnly` | JavaScript không đọc được → XSS không lấy được session |
| `Secure` | Theo `APP_ENV=production` | Chỉ gửi qua HTTPS ở production |
| `SameSite` | `Lax` | **Bắt buộc** — callback OAuth là điều hướng cross-site |

**Vì sao `SameSite=Lax` chứ không `Strict`.** Sau khi Google chuyển hướng về, trình duyệt
coi đó là điều hướng từ site khác. `Strict` sẽ **không gửi cookie** trong request đó, và
người dùng vừa đăng nhập xong lại thấy mình chưa đăng nhập. `Lax` gửi cookie cho điều hướng
GET cấp cao nhất — vừa đủ cho luồng này, vẫn chặn CSRF qua form POST.

**Dọn session hết hạn**: goroutine chạy mỗi giờ (`Container.startSessionCleanup`), dừng
sạch qua channel khi tắt máy. Việc kiểm tra hết hạn vẫn diễn ra ở mỗi lần xác thực, nên goroutine
này chỉ để bảng không phình — không phải cơ chế bảo mật.

**Đăng xuất luôn thành công**: endpoint logout đặt ngoài `AdminAuthMiddleware`. Session đã
chết phía server thì request vẫn đi lọt để xoá cookie phía client.

## 4. CSRF

Áp dụng **toàn cục** (mọi route, không riêng admin) theo mẫu double-submit cookie:

```text
GET/HEAD/OPTIONS   → bỏ qua kiểm tra; nếu chưa có cookie thì sinh token và đặt vào
POST/PUT/PATCH/DELETE → so cookie csrf_token với header X-CSRF-Token (hoặc field form)
                         không khớp / thiếu → 403 INVALID_CSRF
```

| Thuộc tính | Giá trị | Lưu ý |
|---|---|---|
| Cookie | `csrf_token` | `HttpOnly=false` — **bắt buộc**, JS phải đọc được để gửi lại |
| Header | `X-CSRF-Token` | |
| `SameSite` | `Strict` | Khác cookie session (`Lax`) vì token này không cần sống sót qua điều hướng cross-site |
| TTL | 3600 giây | |

Việc `HttpOnly=false` **không phải lỗ hổng**: bản chất double-submit là client phải đọc
được token để gửi lại. Bảo vệ đến từ chỗ site khác không đọc được cookie của domain này
(same-origin policy), chứ không phải từ việc giấu token khỏi JavaScript.

Hai chi tiết trong cách so khớp token:

- **So sánh constant-time** (`crypto/subtle`). So sánh chuỗi thường thoát ra ở ký tự lệch
  đầu tiên, để lộ độ dài tiền tố đúng qua thời gian phản hồi và cho phép dò dần từng ký tự.
- **Không đọc form với body multipart.** `r.FormValue` parse toàn bộ body, nghĩa là một
  file 200MB được đọc và ghi ra đĩa tạm **trước** khi biết token có hợp lệ không — biến
  chính lớp chống CSRF thành đường làm cạn tài nguyên. Đường upload gửi token qua header
  nên nhánh form chỉ cần cho `application/x-www-form-urlencoded`.

⚠️ **Client không phải trình duyệt** (curl, script tích hợp) sẽ bị 403 khi gọi `POST`. Hoặc
lấy token qua một `GET` trước rồi gửi kèm, hoặc đặt `CSRF_ENABLED=false` nếu triển khai
thuần API. Xem [API.md](../API.md).

## 5. API key

```text
API_REQUIRE_KEY=true  →  mọi /api/* phải có header X-API-Key
```

Hai điểm về thiết kế:

- **Không bao giờ nhận key qua query string** — query string bị ghi vào log Nginx, log
  ứng dụng và lịch sử trình duyệt.
- `/api/v1/health` **luôn miễn xác thực**, để load balancer và giám sát thăm dò được.

⚠️ Khi bật API key, **toàn bộ** `/api/*` bị áp — kể cả `/api/v1/public/*`. Nghĩa là trang
public sẽ ngừng hoạt động với người xem ẩn danh. Nếu vừa muốn public mở vừa muốn bảo vệ
upload, hãy giữ `API_REQUIRE_KEY=false` và dựa vào session cho admin + rate limit cho public.

## 6. Xác định IP client — nền móng của mọi giới hạn

**Đọc mục này trước mục rate limit.** Mọi bộ đếm bên dưới đều tính theo IP, nên chúng chỉ
có giá trị đúng bằng độ tin cậy của việc xác định IP.

Bản đầu tiên đọc thẳng `X-Forwarded-For` do client gửi. Header đó là **do người gọi tự
đặt** — bất kỳ ai cũng chỉ cần thêm một giá trị ngẫu nhiên vào mỗi request là có một "IP"
mới, và toàn bộ rate limit theo IP trở thành trang trí. Với một trang cho phép bình luận
ẩn danh mà người dùng là học sinh, đó là lỗ hổng nghiêm trọng nhất trong hệ thống.

`middleware/clientip.go` áp nguyên tắc chuẩn của reverse proxy:

```text
RemoteAddr (chặng kết nối trực tiếp) có nằm trong TRUSTED_PROXIES không?
   ├─ Không → dùng thẳng RemoteAddr, BỎ QUA mọi header chuyển tiếp
   └─ Có    → đọc X-Forwarded-For, duyệt từ PHẢI sang TRÁI,
              lấy IP đầu tiên không thuộc dải tin cậy
```

**Vì sao duyệt từ phải sang trái.** Mỗi proxy *nối thêm* vào cuối chuỗi. Phần bên trái là
thứ client tự gửi lên — bịa được tuỳ ý; phần bên phải do các proxy tin cậy ghi — mới đáng
tin. Lấy phần tử đầu tiên (như bản cũ) là lấy đúng phần kẻ tấn công kiểm soát.

**Gom IPv6 về khối `/64`** (`ClientIPKey`). Một thuê bao IPv6 được cấp cả khối `/64` trở
lên, nên đếm theo địa chỉ đầy đủ là vô nghĩa — đổi sang địa chỉ khác trong cùng khối là có
bộ đếm mới. IPv4 giữ nguyên địa chỉ đầy đủ.

⚠️ `TRUSTED_PROXIES` cấu hình sai theo **cả hai hướng** đều nguy hiểm: khai quá rộng thì
mở lại lỗ hổng giả mạo; khai thiếu khi thật sự đứng sau proxy thì mọi khách bị gom vào một
bộ đếm và cả trang tự khoá lúc đông người nhất (R6). Xem
[deploys/02-configuration.md](../deploys/02-configuration.md).

## 7. Rate limit

Bốn bộ đếm độc lập, mỗi bộ có `map[khoá IP]counter` riêng:

| Phạm vi | Giới hạn | Áp cho |
|---|---|---|
| Toàn cục | `RATE_LIMIT_REQUESTS`/`WINDOW` (mặc định 100/phút) | Mọi request |
| `/api/v1/metrics` | 10/phút, **cố định trong code** | Chống dò thông tin vận hành |
| Ghi dữ liệu public | 20/phút | **Chỉ** `POST`/`DELETE` dưới `/api/v1/public/*` |
| Tải ảnh gốc public | `RATE_LIMIT_DOWNLOAD_REQUESTS` (mặc định 30/phút) | **Chỉ** `GET .../download` |

Các bộ sau lồng trong bộ thứ nhất: một request POST bình luận tính vào **cả hai** bộ đếm.

Bộ đếm tải ảnh tách riêng vì đó là thao tác đắt nhất trên trang public — đọc trọn object từ
S3 rồi ghi một dòng `artwork_downloads` — và là đích ngắm chính khi ai đó muốn gom toàn bộ
tranh. Người xem thật hiếm khi tải quá vài tấm một phút.

Mọi phản hồi 429 đều kèm `Retry-After` và thân JSON cùng khuôn
`{success,error:{code,message}}` như phần còn lại của API.

## 8. Chống tải trọn site (`botguard.go`)

Rate limit theo số request **không** chặn được trình tải site: WinHTTrack, `wget -r` và
tương tự đều cho hẹn nhịp chậm hơn ngưỡng, rồi kiên nhẫn kéo hết ảnh trong nhiều giờ.

Ba lớp, xếp theo mức độ chắc chắn giảm dần — chắc thì chặn thẳng, mơ hồ thì chỉ siết nhịp:

| Lớp | Dấu hiệu | Xử lý |
|---|---|---|
| 1 | User-Agent tự khai là công cụ tải hàng loạt (`httrack`, `wget`, `curl`, `scrapy`, `python-requests`…) | 403 `AUTOMATED_ACCESS_BLOCKED` |
| 2 | Không có User-Agent, **trên đường HTML** | 403 |
| 3 | Vượt `BOT_GUARD_MAX_REQUESTS_PER_MINUTE` **hoặc** `BOT_GUARD_MAX_PATHS_PER_MINUTE` | 429, tự hết sau `BOT_GUARD_BLOCK_MINUTES` |

**Số đường dẫn khác nhau mới là dấu hiệu quyết định.** Người thật xem đi xem lại vài trang,
tải lại, mở lightbox nhiều lần — số URL *khác nhau* vẫn thấp. Máy quét thì đi qua mỗi URL
đúng một lần rồi chuyển sang URL mới. Đếm theo đó phân biệt được hai bên mà không phạt
người xem hăng hái.

### Vì sao KHÔNG chặn theo "có chữ bot trong User-Agent"

Googlebot, bingbot, `facebookexternalhit`, coccocbot đều chứa các chuỗi đó. Chặn theo kiểu
chung chung sẽ **xoá sổ toàn bộ SEO** vừa dựng ở P2.15 và làm hỏng ảnh preview khi chia sẻ
link lên Facebook/Zalo. Danh sách `scraperAgentMarkers` cố tình chỉ liệt kê tên công cụ cụ
thể; `legitBotMarkers` liệt kê bot được miễn hoàn toàn lớp 3.

Bot hợp lệ **không** xác minh ngược DNS: một truy vấn DNS trên đường phục vụ mỗi request là
chi phí không đáng ở quy mô này, và hệ quả xấu nhất của việc giả mạo chỉ là được miễn giới
hạn nhịp — kẻ giả mạo vẫn dính lớp 1 nếu dùng công cụ tải hàng loạt, vẫn dính rate limit
chung theo IP.

### Miễn trừ

`/robots.txt`, `/sitemap.xml` (chính là thứ để phục vụ crawler), `/api/v1/health` (giám sát
phải luôn thăm dò được), `/api/v1/admin/*` và `/auth/*` (đã có session; admin chạy script
đối soát dữ liệu là việc bình thường).

### Điều cố ý KHÔNG làm

Không chặn theo "thiếu `Referer`" hay "thiếu `Accept-Language`". Trình duyệt thật vẫn thiếu
các header đó trong nhiều tình huống hợp lệ — mở thẳng link, thiết lập riêng tư — nên chặn
theo đó là chặn nhầm người xem thật.

⚠️ Bot guard đặt **ngoài** rate limit chung trong chuỗi middleware, để máy quét bị loại
trước khi kịp tiêu tốn hạn mức chung của những người khác cùng đi ra từ một IP NAT — trường
học dùng chung một IP ra ngoài (R6).

Mỗi `RateLimiter` và cả `BotGuard` đều chạy goroutine dọn bộ nhớ riêng. Container giữ danh
sách mọi limiter đã tạo và `Stop()` tất cả (kèm `BotGuard.Stop()`) khi tắt máy — không rò
rỉ goroutine.

⚠️ Bộ đếm nằm **trong bộ nhớ tiến trình**. Restart là mất, và nhiều instance sẽ không dùng
chung. Chấp nhận được với một instance; nếu scale ngang phải chuyển sang Redis.

## 9. Header bảo mật (`security_headers.go`)

Đặt ở **tầng ứng dụng**, không chỉ ở Nginx: vhost mẫu trong repo chỉ là điểm khởi đầu, còn
file thật trên VPS do Certbot sửa và người vận hành chỉnh tay — repo không kiểm soát được
nội dung đó. Đặt ở đây thì mọi lần triển khai đều có, kể cả khi dựng vhost mới. Header nào
Nginx đã đặt thì không ghi đè (`setIfAbsent`), nên không gửi trùng.

| Header | Giá trị | Ghi chú |
|---|---|---|
| `Content-Security-Policy` | `script-src 'self'`, `object-src 'none'`, `frame-ancestors 'self'`… | **Chỉ** đặt trên tài liệu HTML |
| `Permissions-Policy` | camera/mic/geolocation/payment/usb đều `()` | Khoá sẵn API trình duyệt trang này không dùng |
| `Cross-Origin-Resource-Policy` | `same-origin` | |
| `Cross-Origin-Opener-Policy` | `same-origin` | |
| `Strict-Transport-Security` | `max-age=31536000; includeSubDomains` | Chỉ khi `SECURITY_HSTS_ENABLED` **và** request thật sự là HTTPS |
| `X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy` | như Nginx | Lưới an toàn khi chạy không qua proxy |

**`style-src` có `'unsafe-inline'`, `script-src` thì không.** React đặt style nội tuyến qua
thuộc tính `style`, và trang chia sẻ render bằng `html/template` cũng có style nội tuyến —
không nới thì giao diện vỡ. Với script thì không cần nới: Vite sinh file `.js` riêng, và
script nội tuyến mới là hướng tấn công XSS đáng lo. Nới cả hai cho "tiện" là vô hiệu hoá
phần có giá trị nhất của CSP.

Ràng buộc đó **có giá phải trả ở frontend**, và giá đó đã được trả chứ không phải né:

- Script gỡ splash chuyển từ nội tuyến trong `index.html` sang `web/public/splash.js`, nạp
  bằng `<script src defer>`. Phương án thay thế là băm `sha256` rồi nhúng hash vào CSP, nhưng
  như thế mỗi lần sửa script là phải tính lại hash trong code Go — quên một lần thì splash
  kẹt vĩnh viễn trên production mà log server im lặng hoàn toàn.
- Font chuyển sang **tự phục vụ** trong `web/public/fonts/` thay vì nạp từ
  `fonts.googleapis.com` — `style-src 'self'` chặn stylesheet của Google, và nới CSP cho
  Google đổi lấy việc mỗi lượt xem trang gửi IP người xem sang máy chủ bên thứ ba. Chi tiết ở
  [06-frontend.md](06-frontend.md).

Vì vậy **`font-src 'self' data:` là đủ** và không cần thêm origin ngoài nào. Nếu ai đó siết
`font-src` bỏ `'self'`, toàn bộ chữ trên trang tụt về font hệ thống —
`TestSecurityHeaders_FontTuPhucVuDuocPhep` khoá lại điều này.

CSP **không** đặt lên tài nguyên tĩnh (`/assets/`, `/images/`, `/fonts/`, `/splash.js`,
`robots.txt`, `sitemap.xml`, `favicon.ico`): chúng không phải tài liệu HTML nên header đó chỉ
tốn băng thông ở mọi request.

**Domain S3 tự suy** từ `S3_BUCKET_NAME`/`AWS_REGION`/`S3_ENDPOINT` (`deriveS3Origins`).
Bắt người vận hành khai lại domain trong một biến CSP riêng là mời gọi việc quên đồng bộ
hai chỗ — mà hậu quả là CSP chặn đúng ảnh tác phẩm, lỗi chỉ lộ trên trình duyệt người dùng
cuối chứ không xuất hiện trong log server.

⚠️ HSTS chỉ đặt khi request thật sự đến qua HTTPS. `X-Forwarded-Proto` chỉ được tin khi
request đến từ proxy tin cậy — cùng nguyên tắc ở mục 6.

## 10. Giới hạn kích thước body (`bodylimit.go`)

Nginx đặt `client_max_body_size 200m` ở mức **server** để đường upload ảnh đi lọt, nghĩa là
mọi endpoint đều nhận được body 200MB — kể cả `POST /api/v1/public/artworks/{id}/comments`.
Vài chục request bình luận với body khổng lồ ép server đọc và cấp phát bộ nhớ, hoàn toàn
hợp lệ dưới mắt rate limit vì số request vẫn thấp.

`http.MaxBytesReader` cắt việc đọc ngay khi vượt ngưỡng: `MAX_JSON_BODY_KB` (mặc định 1MB)
cho endpoint thường, `UPLOAD_ABSOLUTE_MAX_MB` cho đường upload ảnh.

Đặt **ngoài** CSRF trong chuỗi middleware để body quá khổ bị cắt trước khi bất kỳ lớp nào
đọc nó.

## 11. Giới hạn đồng thời

Semaphore toàn server (`golang.org/x/sync/semaphore`) giới hạn số request xử lý cùng lúc.
Vượt quá thì **chờ** tối đa `CONCURRENCY_ACQUIRE_TIMEOUT_SECONDS` rồi mới trả lỗi — xếp
hàng thay vì từ chối ngay, vì upload ảnh là thao tác người dùng chủ động và chờ vài giây
tốt hơn báo lỗi.

Mặc định `MAX_CONCURRENT_UPLOADS=500` là khá rộng; điều chỉnh theo RAM thực tế của VPS, vì
mỗi upload đang xử lý giữ một file tạm và bộ đệm.

## 12. Kiểm tra bắt buộc ở production

`config/builder.go:426-432` **chặn server khởi động** nếu `APP_ENV=production` mà:

| Điều kiện | Thông báo |
|---|---|
| `API_REQUIRE_KEY=false` hoặc `API_KEY` rỗng | `production requires API_REQUIRE_KEY=true and a non-empty API_KEY` |
| `CORS_ORIGINS=*` hoặc rỗng | `production requires an explicit CORS_ORIGINS allowlist (not *)` |

Fail-fast có chủ đích: cấu hình sai bị phát hiện lúc khởi động, không phải sau khi đã chạy
và để lộ dữ liệu. ⚠️ Lưu ý ràng buộc thứ nhất kéo theo cảnh báo ở mục 5 — bật API key ở
production sẽ ảnh hưởng trang public; xem [deploys/02-configuration.md](../deploys/02-configuration.md)
để biết cách xử lý.

## 13. Bảo vệ khác rải rác trong code

| Bảo vệ | Nơi thực hiện |
|---|---|
| Chống path traversal ở tên file | `utils.SanitizeFilename()` |
| Chống path traversal khi phục vụ SPA | `spaFileServer` kiểm tra đường dẫn tuyệt đối nằm trong `dist` |
| Chặn liệt kê thư mục | `secureFileServer` trả 404 cho đường dẫn kết thúc bằng `/` |
| Chống SQL injection | Toàn bộ truy vấn dùng tham số `?`, kể cả WHERE động |
| Không lộ dữ liệu nhạy cảm | `visitor_token`, `ip_address`, `s3_key` đánh `json:"-"` |
| Không lộ quyền sở hữu | Xoá bình luận trả cùng lỗi cho "không tồn tại" và "không phải của bạn" |
| Chống XSS | React escape khi render; `html/template` escape ở trang chia sẻ |
| Chuẩn hoá thông tin lỗi | `handlers/error_mapper.go` — `sanitizeError` không trả chi tiết nội bộ ra ngoài |

## 14. Những gì hệ thống **không** bảo vệ

Nêu rõ để không ai hiểu nhầm về mức bảo đảm:

- **Danh tính người tương tác public.** `visitor_token` giả mạo được. Xem
  [04-public-engagement.md](./04-public-engagement.md).
- **Phân quyền theo vai trò.** Cột `role` tồn tại nhưng mọi admin có quyền như nhau —
  không có phân biệt người duyệt/người xem.
- **Nhật ký thao tác admin — mới có một phần.** Bảng `artwork_downloads` (migration `018`)
  ghi lại ai **tải** tác phẩm nào lúc nào (xem [01-database.md](./01-database.md)), nhưng
  xoá/sửa tác phẩm vẫn **không** được ghi lại — chỉ có `artworks.created_by` cho việc tạo.
- **Mã hoá dữ liệu khi lưu.** Dựa vào mã hoá ở tầng đĩa/dịch vụ, không mã hoá ở tầng ứng dụng.
- **Chặn tấn công từ chối dịch vụ phân tán (DDoS thật sự).** Bot guard và rate limit chặn
  được **một nguồn** lạm dụng — kể cả nguồn kiên nhẫn chạy chậm. Chúng **không** chặn được
  hàng nghìn IP khác nhau cùng lúc: lưu lượng đó đã tiêu thụ băng thông và tài nguyên VPS
  *trước khi* đến được tầng ứng dụng. Chống DDoS phân tán phải đặt ở tuyến trước (Cloudflare
  hoặc tương đương) — không có cách nào làm được điều đó bên trong tiến trình Go.
- **Người quyết tâm sao chép nội dung.** Ai đó chạy trình duyệt thật, tốc độ người thường,
  vẫn lưu được từng tấm tranh. Mục tiêu của lớp chống scraping là làm việc **tải hàng loạt
  tự động** trở nên bất tiện, không phải làm nội dung công khai thành không sao chép được —
  điều đó bất khả thi với bất kỳ website nào.
- **Bot giả mạo User-Agent của Googlebot.** Không xác minh ngược DNS (xem mục 8) — chúng
  thoát được lớp giới hạn nhịp, nhưng vẫn chịu rate limit chung theo IP.
