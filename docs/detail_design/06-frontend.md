# 06 — Frontend

**Công nghệ**: React 19 · TypeScript 5.8 · Vite 7 · React Router 7 · Framer Motion 12
**Vị trí**: `web/`, build ra `web/dist`, được binary Go phục vụ tĩnh

## 1. Một SPA, ba khu vực

```text
web/src/
├── App.tsx                 router gốc
├── pages/
│   ├── public/             ◀── khu vực công khai (không đăng nhập)
│   │   ├── PublicLayout    navbar + footer dùng chung
│   │   ├── HomePage              /
│   │   ├── FeaturedArtworksPage  /tac-pham-tieu-bieu
│   │   ├── GalleryPage           /phong-trien-lam
│   │   ├── HallOfFamePage        /bang-vang
│   │   ├── OpenLetterPage        /thu-ngo
│   │   └── NotFoundPage          * (mọi đường dẫn lạ dưới "/")
│   ├── admin/              ◀── khu vực quản trị (session)
│   │   ├── AdminLayout     bảo vệ route + sidebar
│   │   ├── Login                 /admin/login
│   │   ├── Dashboard             /admin
│   │   ├── ArtworksListPage      /admin/artworks
│   │   ├── ArtworksUploadPage    /admin/artworks/upload
│   │   ├── AwardsPage            /admin/awards
│   │   ├── TopicCategoriesPage   /admin/topic-categories
│   │   └── NotFoundPage           * (mọi đường dẫn lạ dưới "/admin")
├── components/{public,admin,...}
├── hooks/       useAdminAuth, useParallaxScroll, useRailScroll, useScrollableBody
└── lib/         api, adminApi, artworkApi, awardApi, dashboardApi, publicApi, ...
```

Bảng đường dẫn:

| Đường dẫn | Trang | Ghi chú |
|---|---|---|
| `/` | Trang chủ triển lãm | **Nạp tĩnh** — điểm vào của hầu hết khách |
| `/trien-lam` | → chuyển hướng về `/` | Giữ tương thích link cũ đã phát ra ngoài |
| `/tac-pham-tieu-bieu` | Tác phẩm tiêu biểu | |
| `/phong-trien-lam` | Phòng triển lãm (tìm/lọc) | |
| `/bang-vang` | Bảng vàng | |
| `/thu-ngo` | Thư ngỏ Chủ tịch HĐQT | Bố cục một khung nhìn — xem mục riêng bên dưới |
| `/admin/login` | Đăng nhập Google | |
| `/admin`, `/admin/*` | Khu quản trị | Bảo vệ bởi `AdminLayout` |
| Đường dẫn lạ dưới `/` | 404 (theme khu vườn) | `NotFoundPage` public |
| Đường dẫn lạ dưới `/admin` | 404 (theme quản trị) | `NotFoundPage` admin, giữ sidebar/header |

## 2. Tách bundle theo route

`App.tsx` nạp tĩnh **chỉ** `PublicLayout` + `HomePage`; mọi trang khác qua `React.lazy`.

Lý do rất cụ thể: trước khi tách, một phụ huynh vào xem tranh phải tải kèm **toàn bộ**
dashboard quản trị và công cụ upload S3 — những thứ họ không bao giờ mở được. Nay các chunk
đó chỉ tải khi thực sự vào `/admin`.

Một `<Suspense>` duy nhất bọc ngoài `<Routes>` là đủ, vì mỗi lần chỉ có một route khớp nên
không bao giờ có hai chunk cùng treo fallback.

Song song đó, `vite.config.ts` tách **theo thư viện**:

| Chunk | Chứa | Vì sao tách |
|---|---|---|
| `vendor-motion` | framer-motion, motion-dom, motion-utils | Dùng nhiều ở trang public |
| `vendor-router` | react-router | Ổn định, hiếm đổi |
| `vendor-react` | react, react-dom, scheduler | Ổn định nhất, cache lâu nhất |

Trước đây có thêm nhánh `vendor-charts` (recharts + d3-*) cho biểu đồ ở Dashboard admin.
Dashboard đã đổi sang card thống kê theo khu vực (ảnh + số liệu, không còn biểu đồ), nên
recharts không còn được dùng ở đâu trong toàn bộ frontend - đã gỡ khỏi `package.json` và
nhánh `vendor-charts` khỏi `manualChunks` thay vì để một chunk ~400KB không ai tải tới.

Mục tiêu: sửa một dòng trong `src/` không làm đổi hash của các chunk thư viện, nên trình
duyệt khách giữ nguyên cache qua nhiều lần deploy.

⚠️ Thứ tự kiểm tra trong `manualChunks` **quan trọng**: `react-dom` và `react-router` đều
chứa chuỗi `"react"`, nên các nhánh riêng phải đứng trước nhánh `react` chung. Ghi chú này
có trong chính file cấu hình.

### Trạng thái chờ khi chuyển trang

Tách chunk theo route (mục trên) đổi lấy một khoảng chờ mạng mỗi lần vào trang chưa từng
tải. React Router 7 điều hướng bên trong `startTransition`, nên khi chunk đích còn đang tải
React **giữ nguyên trang cũ** trên màn hình thay vì thay bằng fallback của `<Suspense>` —
đúng hành vi, nhưng nó để lại một khoảng im lặng: bấm link xong không thấy gì đổi.

Ba lớp xử lý, mỗi lớp cho một quãng chờ khác nhau:

| Quãng chờ | Component | Cơ chế |
|---|---|---|
| Chuyển trang (route cũ còn hiện) | `RouteProgress.tsx` | Thanh 3px ở đỉnh màn hình |
| `<Suspense>` thật sự phải treo (mở thẳng URL con, mạng rất chậm) | `RouteFallback.tsx` | Khung xám kiểu bố cục trang đích |
| Tải bundle lần đầu (trước khi React mount) | `index.html` (`#app-splash`) + `public/splash.js` | Splash toàn màn hình, HTML/CSS thuần |

**`RouteProgress`** phải phát hiện "đang chờ" mà không dùng `useLocation()` bình thường —
giá trị đó cũng bị giữ lại cùng cây cũ trong lúc transition treo, không đổi cho tới khi
trang mới commit. Cách phát hiện đúng: so **URL thật của trình duyệt** (đọc qua
`useSyncExternalStore`, vá `pushState`/`replaceState` để bắn sự kiện — hai hàm này không tự
bắn) với `location` mà cây React đã commit. Hai giá trị lệch nhau đúng bằng quãng transition
đang treo.

Chuyển động của thanh do CSS `@keyframes` chạy trên compositor, React chỉ đổi 3 trạng thái
(`idle` / `running` / `finishing`) qua class. Bản đầu tiên dùng `setInterval` cập nhật phần
trăm mỗi 90ms bị giật rõ — mỗi nhịp là một lần render React, cắt ngang transition CSS của
nhịp trước nên không nhịp nào chạy trọn. Tiến trình hiển thị là **giả** (trình duyệt không
cho biết chunk tải được bao nhiêu %): thanh chạy nhanh dần chậm rồi gần đứng lại quanh 94%,
khi trang mới commit mới chạy nốt lên 100% và tan — không bao giờ tự chạm đích rồi kẹt.

`RouteFallback` giờ vẽ khung bố cục (dải tiêu đề + lưới ô vuông kiểu khung tranh) thay vì
spinner giữa khoảng trắng, để không có cú nhảy layout khi nội dung thật thay vào. Nó cố ý
**không phụ thuộc** `public.css`/`admin.css` — style để inline vì fallback có thể hiện ra
trước khi stylesheet của route đích kịp tới.

Splash trong `index.html` xử lý quãng mà React **chưa tồn tại** — không thể làm bằng
component. Markup và CSS nằm trong `index.html`, còn script gỡ splash ở **`public/splash.js`**
nạp bằng `<script src defer>`. Gỡ bằng `MutationObserver` theo dõi `#root` có con hay chưa
(không dùng sự kiện `load`, vì đó là lúc tài nguyên tải xong chứ không phải lúc màn hình có
nội dung), kèm chốt timeout 8s để không kẹt vĩnh viễn nếu bundle lỗi.

⚠️ Script này **không được** đưa trở lại thành inline. CSP đặt `script-src 'self'` không có
`'unsafe-inline'`, nên script nội tuyến bị trình duyệt chặn thẳng và splash sẽ không bao giờ
tan. Cách còn lại — băm `sha256` đoạn script rồi nhúng hash vào CSP — buộc phải tính lại hash
trong code Go mỗi lần sửa một ký tự ở đây; quên một lần là splash kẹt vĩnh viễn trên
production mà log server hoàn toàn im lặng.

### Font tự phục vụ

Be Vietnam Pro và Fraunces nằm trong `web/public/fonts/` (định dạng `woff2`), khai `@font-face`
ở `src/styles/fonts.css` — **không** nạp từ `fonts.googleapis.com`.

Lý do đầu tiên là CSP: `style-src 'self' 'unsafe-inline'` chặn stylesheet từ origin ngoài, nên
link Google Fonts trong `index.html` trước đây bị chặn và cả trang tụt về font hệ thống. Nới
CSP cho Google là sửa được, nhưng đổi lại mỗi lượt xem trang đều gọi sang máy chủ Google —
tức là địa chỉ IP của từng học sinh và phụ huynh xem tranh đều đi ra ngoài. Self-host bỏ luôn
cả hai vấn đề, và tiết kiệm hai lần bắt tay DNS/TLS ở đường tải quan trọng nhất.

`fonts.css` **sinh bằng `scripts/fetch-fonts.sh`**, đừng sửa tay: script tải lại từ CSS gốc của
Google và giữ nguyên `unicode-range`, nên trình duyệt vẫn chỉ tải subset nó cần. Tổng số file
trên đĩa lớn hơn nhiều so với lượng thật sự truyền — một người xem trang tiếng Việt chỉ chạm
vào subset `vietnamese` + `latin` của những weight thực sự xuất hiện. Muốn thêm/bớt weight thì
sửa biến `URL` trong script rồi chạy lại.

`index.html` `preload` riêng subset `vietnamese` + `latin` của weight 400: các `@font-face` nằm
sau một lớp `@import` nên trình duyệt phát hiện khá muộn, mà splash cần đúng font đó ngay từ
khung hình đầu. Chỉ preload hai file — nhiều hơn thì tự cạnh tranh băng thông với bundle JS.

## 3. Bốn client API tách theo khu vực

| File | Phục vụ | Xác thực |
|---|---|---|
| `lib/api.ts` | Nền chung: CSRF token, dựng URL, xử lý envelope | — |
| `lib/publicApi.ts` | `/api/v1/public/*` | Không — chỉ CSRF + `visitor_token` |
| `lib/adminApi.ts` | `/api/v1/admin/*` | Cookie session; ném `UnauthorizedError` khi 401 |
| `lib/artworkApi.ts`, `awardApi.ts`, `dashboardApi.ts` | Bọc theo miền, dùng lại `adminApi` | Session |
| `lib/artworkImage.ts` | Chọn cỡ ảnh hiển thị (xem bên dưới) | — |
| `lib/chartTheme.ts` | Màu định danh theo khu vực (`REGION_COLOR`) + định dạng số kiểu Việt Nam dùng chung cho Dashboard | — |

### Chọn ảnh: `lib/artworkImage.ts`

Backend sinh sẵn biến thể `thumb`/`medium`/`large` × WebP/JPEG lúc upload, nhưng **không
phải tác phẩm nào cũng có**: ảnh upload trước khi có tính năng, ảnh gốc vốn nhỏ hơn cỡ
đích, hoặc khâu sinh biến thể lỗi.

Vì vậy mọi chỗ hiển thị đều cần đường lui — và toàn bộ được gom vào một file để **không mỗi
nơi lui một kiểu**:

```text
artworkImageURL(item, size)   → variants[size_jpg] → thumbnail_url → image_url
artworkPictureSources(item)   → <source> WebP trước, JPEG sau; rỗng thì dùng <img> trần
```

`src` **luôn** là JPEG hoặc ảnh gốc để trình duyệt nào cũng đọc được; WebP chỉ đi qua
`<source>` trong `<picture>`. Helper nhận cả DTO rút gọn (dashboard, billboard) chứ không
riêng `ArtworkWithMeta` đầy đủ.

Tách như vậy để **không lẫn ngữ cảnh xác thực**: một hàm public không vô tình gửi kèm thứ
chỉ dành cho admin, và ngược lại.

### Envelope dùng chung

Mọi response backend theo một dạng, nên client parse một chỗ:

```ts
{ success: true,  data: T }
{ success: false, error: { code, message } }
```

`parseEnvelope()` xử lý riêng vài trường hợp:

- **429** → thông điệp tiếng Việt thân thiện ("Bạn thao tác hơi nhanh…") thay vì mã lỗi thô.
- **Body không phải JSON** (ví dụ trang lỗi của Nginx) → cắt 200 ký tự đầu làm thông điệp,
  không để `JSON.parse` ném lỗi khó hiểu.
- **401** ở `adminApi` → ném `UnauthorizedError` riêng để `AdminLayout` bắt và chuyển hướng.

### CSRF ở phía client

`publicWrite()` đọc cookie `csrf_token` và gửi lại qua header `X-CSRF-Token` cho mọi
`POST`/`DELETE`. Vì thế cookie đó **phải** `HttpOnly=false` — xem
[05-auth-security.md](./05-auth-security.md).

## 4. Bảo vệ route admin

```text
AdminLayout mount
   └─ useAdminAuth() → GET /api/v1/admin/auth/me
         ├─ loading         → hiện trạng thái chờ
         ├─ 200 + user      → render sidebar + <Outlet/>
         ├─ UnauthorizedError → user = null → chuyển hướng /admin/login
         └─ lỗi khác        → hiện thông báo lỗi (không chuyển hướng)
```

Phân biệt **401** với **lỗi mạng/lỗi server** là có chủ đích: 401 nghĩa là chưa đăng nhập,
nên chuyển hướng; còn mất mạng thì đá người dùng ra trang login sẽ gây hiểu nhầm là bị đăng
xuất.

Đây là bảo vệ **giao diện**, không phải bảo vệ **dữ liệu** — mọi endpoint admin đều tự kiểm
tra session ở backend. Người dùng có thể mở DevTools và render component admin, nhưng sẽ
không lấy được dữ liệu nào.

## 5. `visitor_token` phía client

`lib/visitorToken.ts` sinh và lưu UUID trong `localStorage`:

```ts
crypto.randomUUID()            // đường chính
`${Date.now()}-${random}-...`  // dự phòng cho môi trường không có crypto
```

Toàn bộ thao tác `localStorage` bọc trong `try/catch`: chế độ ẩn danh của một số trình duyệt
chặn hoàn toàn. Khi đó dùng token tạm cho phiên hiện tại — mất khi tải lại trang, nhưng
**không làm hỏng chức năng**, chỉ ảnh hưởng việc chống đếm trùng.

Tên hiển thị của người bình luận cũng lưu cùng chỗ (`vas_visitor_display_name`) để không
phải gõ lại mỗi lần.

## 6. Thành phần trang public

Component trong `components/public/` (đếm bằng `ls web/src/components/public/ | wc -l`),
chia theo vai trò:

| Nhóm | Thành phần |
|---|---|
| Bố cục | `PublicNavbar`, `PublicFooter` (khung `PublicLayout` nằm ở `pages/public/`) |
| Trang chủ | `HeroSection`, `HeroParallaxHills`, `HillDivider`, `EducationLevelGate`, `EducationLevelCard`, `GradeNode` |
| Tiêu biểu | `FeaturedHero`, `FeaturedGardenScene`, `FeaturedArtworkFrame`, `FeaturedArtworkCard`, `ArtworkRail`, `RegionTabs` |
| Triển lãm | `GalleryLevelSection`, `GalleryTopicSection`, `GallerySearch`, `GalleryPagination` |
| Bảng vàng | `HallArtworkCard`, `HallRail`, `HallFireworks` |
| Tương tác | `PublicLightbox`, `ReactionPicker`, `CommentBox` |
| Trợ lý | `MascotAssistant` (xem mục riêng dưới đây) |

Trang `/thu-ngo` không có component riêng trong bảng này: toàn bộ nằm trong
`pages/public/OpenLetterPage.tsx` (xem mục "Trang Thư ngỏ" bên dưới).

### `GalleryTopicSection` — section theo nhóm chủ đề, dưới 2 phòng cấp học

`/phong-trien-lam` (`GalleryPage`) giữ nguyên 2 section cấp học đầu trang
(`GalleryLevelSection` — Tiểu học/Trung học, có bộ lọc khối lớp), rồi thêm một
`GalleryTopicSection` cho **mỗi** nhóm chủ đề đang active (`fetchTopicCategories(false)`),
đặt sau trong cùng `.gallery-hall` nhưng bọc riêng `.gallery-hall--topics`.

Khác `GalleryLevelSection`: không có node chọn khối lớp (nhóm chủ đề có thể dùng chung cả 2
cấp học, lọc thêm theo khối sẽ vụn một section vốn đã hẹp), và **tự ẩn hoàn toàn** (return
`null`) nếu nhóm không có tác phẩm nào đã duyệt — tránh hàng chục section trống khi nhóm chủ
đề mới tạo chưa gán tác phẩm. Cờ `loadedOnce` phân biệt "chưa tải xong" với "đã tải xong,
rỗng" để không ẩn/hiện nháy trong lúc chờ response đầu tiên.

Ẩn toàn bộ khối section chủ đề khi đang tìm kiếm (`searchQuery` khác rỗng) — component không
nhận `search` prop, hiện cùng lúc dễ gây hiểu nhầm đó là kết quả tìm kiếm.

Backend: `PublicHandler.HandleListArtworks` (`internal/handlers/public_handler.go`) parse
thêm `topic_category_id` từ query — tham số này đã có sẵn trong `ArtworkFilter` và đã dùng ở
`/api/v1/admin/artworks`, chỉ thiếu ở nhánh public trước khi có section này.

### Trang Thư ngỏ (`/thu-ngo`) — bố cục một khung nhìn

`OpenLetterPage.tsx` là trang public duy nhất **không thiết kế để cuộn**: phần nội dung thư
nằm trọn trong một khung nhìn.

Bản trước xếp chồng ba section, mỗi section cao gần cả màn hình (banner wordmark → dải 5 huy
hiệu giá trị → thân thư hai cột) — phải cuộn ba lần mới đọc hết một lá thư chỉ có ba đoạn.
Bản hiện tại gộp cả ba vào một tờ giấy: wordmark + tiêu đề thành đầu thư, 5 giá trị thu từ
huy hiệu tròn xuống hàng nhãn chữ có chấm màu ở chân thư.

Đầu thư chỉ còn wordmark và tiêu đề — con dấu tròn "20 năm" ở góc phải đã bỏ. Hai dấu nhận
diện đặt cạnh nhau trên một hàng hẹp thì tranh nhau sự chú ý, mà mốc 20 năm vốn đã có trong
dòng kicker ngay dưới wordmark. Bỏ con dấu cũng trả lại chỗ để phóng `.letter-wordmark` lên
`clamp(2.9rem, 5vw, 4.1rem)`, gần gấp đôi bản trước.

Cơ chế giữ đúng một khung nhìn:

- `.letter-stage` cao `100dvh` — **không** trừ `--vas-navbar-h`. `padding-top` của nó đã
  chừa chỗ cho navbar, mà `box-sizing: border-box` tính padding nằm trong `height`; trừ
  thêm ở `min-height` là trừ hai lần, tổng chiều cao dôi ra đúng một nhịp navbar và sinh
  thanh cuộn thừa. Đây chính là lỗi của bản đầu tiên.
- Dùng `dvh` chứ không `vh`: trên iOS Safari và Chrome Android, thanh địa chỉ co lại khi
  cuộn nên `100vh` lớn hơn vùng nhìn thấy thật — cũng sinh thanh cuộn thừa. Có fallback
  `vh` ở dòng ngay trên cho trình duyệt cũ.
- `.letter-sheet` dùng `max-height: 100%` chứ không `height`: thư ngắn thì tờ giấy ôm sát
  chữ, thư dài thì kịch trần khung nhìn.
- `.letter-body` là chỗ **duy nhất** được cuộn (`flex: 1; min-height: 0; overflow-y: auto`).
  `min-height: 0` là bắt buộc — không có nó, flex item không co xuống dưới kích thước nội
  dung và tờ giấy sẽ tràn khỏi khung nhìn.
- Đệm dọc mỏng hơn đệm ngang ở mọi khổ: chiều dọc quyết định có phải cuộn hay không, chiều
  ngang thì dư — và lề ngang rộng mới ra dáng tờ thư.
- Breakpoint `max-height: 850px` (laptop màn thấp — khổ hay phải cuộn nhất) siết khoảng đệm
  và cỡ chữ **trang trí**, giữ nguyên cỡ chữ thân thư: phần phải đọc thì không được nhỏ đi.

### Bề rộng cột chữ và cách căn lề

`.letter-scene` rộng `min(84rem, 100%)`, `.letter-body` giới hạn `max-width: 68rem` (~90 ký
tự/dòng). Nới cột chữ là cách giảm chiều cao thân thư mà **không** phải thu nhỏ cỡ chữ: mỗi
dòng chứa nhiều chữ hơn thì cùng một đoạn văn chiếm ít dòng hơn. Bù lại bằng
`line-height: 1.8` — dòng càng dài thì khoảng cách dòng càng phải nới, nếu không mắt nhảy
nhầm hàng khi vắt sang dòng mới.

Thân thư căn **trái**, không `justify`, và `hyphens: none`. Tiếng Việt nhiều từ ghép hai âm
tiết và không ngắt từ được, nên căn đều hai biên phải giãn khoảng trắng rất thô — sinh ra
"dòng sông" trắng chạy dọc đoạn văn, đúng thứ gây rối mắt. Trình duyệt cũng không có từ điển
ngắt âm tiết tiếng Việt nên `hyphens: auto` cắt sai chỗ.

`PublicFooter` vẫn render bình thường bên dưới (địa chỉ hội sở, liên hệ — không bỏ được), nên
trang vẫn cuộn được xuống footer. `.letter-scroll-cue` là mũi tên nhấp nháy ở đáy báo còn nội
dung phía dưới; thiếu nó thì đáy màn hình trông như hết trang.

Đệm đáy của `.letter-stage` dày hơn đệm đỉnh (2.5rem so với 0.75rem) để tờ thư hở ra một
quãng trước khi chạm dải teal của footer — dính sát nhau thì hai khối đọc thành một mảng
liền, mất ranh giới giữa lá thư và phần chân trang.

Hoạ tiết giấy vẽ hoàn toàn bằng CSS, không tải ảnh texture nào: hai lớp
`repeating-linear-gradient` rất mảnh cộng lại thành hạt nhiễu, `radial-gradient` cho sắc ngả
vàng dồn về mép, `clip-path` đa giác biên độ ~4px cho mép răng cưa giấy thủ công,
`.letter-sheet::before` là đường kẻ lề dọc mép trái (dấu hiệu quen thuộc nhất của giấy viết
thư — đặt bằng pseudo-element nên không thêm phần tử vào DOM, và nằm gọn trong phần đệm ngang
nên không bao giờ chạm chữ), hai `.letter-fold` giả nếp gấp làm ba (một vệt sáng kề một vệt
tối — một đường đơn chỉ trông như kẻ ngang).

Nếp gấp cố ý rất nhạt và `mask-image` cho mờ dần về hai đầu: nó cắt **ngang** dòng chữ, đậm
một chút là mắt vấp phải giữa câu. Nguyên tắc chung cho cả trang này — mọi hoạ tiết nằm dưới
vùng chữ đều phải nhạt tới mức chỉ cảm nhận được chứ không nhìn thấy rõ.

Hoạt cảnh mở đầu: phong bì lật nắp (`rotateX(-172deg)` quanh `transform-origin: top center`,
cần `perspective` trên `.letter-scene`) rồi tờ thư trượt lên. Khi `useReducedMotion()` bật,
phong bì **không render** — nó chỉ tồn tại để chạy hoạt cảnh, đứng yên thì là một hình thù lạ
nằm sau tờ thư.

Ở `max-width: 480px` bỏ `clip-path` và drop cap (ở khổ đó chúng chỉ còn là nhiễu), ở
`max-height: 560px` (điện thoại nằm ngang) ẩn hàng giá trị và mũi tên — giữ cam kết một khung
nhìn bằng cách hy sinh phần trang trí, không phải bằng cách cho trang cuộn.

### Hiệu ứng và hiệu năng

`useParallaxScroll` là hook dùng chung cho hiệu ứng cuộn nhiều lớp ở trang chủ. Trang này
nặng về hình ảnh động (đồi parallax, pháo hoa, cảnh vườn), nên các nguyên tắc sau được giữ:

- Hiệu ứng cuộn đi qua `requestAnimationFrame`, không gắn trực tiếp vào sự kiện `scroll`.
- Ảnh nền dùng SVG khi có thể (`garden-butterfly.svg`, `garden-fern-cluster.svg`) — nhẹ và
  sắc nét ở mọi độ phân giải.
- Framer Motion nằm ở chunk riêng, không kéo theo khi vào khu admin.
- Ảnh PNG tĩnh dùng ở critical path (đồi parallax, mascot, wordmark) đều có bản `.webp`
  cùng thư mục, phục vụ qua `<picture><source>` — xem `lib/staticImage.ts` và mục "Ảnh tĩnh
  frontend sang WebP" trong [01-current-state.md](../plan/01-current-state.md).

### `MascotAssistant` — trợ lý tìm kiếm nổi, chỉ desktop

Mascot rồng nổi góc dưới-phải toàn trang public (`≥1024px` — ẩn hẳn trên tablet/mobile qua
CSS, tránh che nội dung màn hình nhỏ). Render qua `createPortal(..., document.body)` như
`AdminPageHeader`/`ConfirmDialog` để không phụ thuộc vị trí gọi trong cây component.

Click mascot mở panel tìm kiếm nội bộ. Logic diễn giải câu gõ tách hẳn khỏi component vào
`lib/mascotSearch.ts` để test độc lập UI:

- `detectIntent()` so khớp từ khoá tiếng Việt đã bỏ dấu (`giai nhat`, `bang vang`, `ai dat
  giai`, …) để nhận ra câu hỏi về **giải thưởng** và chuyển sang tra `fetchBillboard()`
  thay vì tìm tác phẩm thường.
- Ý định "search" gọi thẳng `fetchPublicArtworks({ search })` — đúng hợp đồng LIKE đã dùng
  ở `GallerySearch`, không phải API riêng.
- Cả hai nhánh có bước "gõ gần đúng" bằng Levenshtein khoảng cách ngắn (`isCloseMatch`) khi
  không khớp chính xác, để chịu được lỗi chính tả nhẹ — **không gọi AI ngoài**, thuần so
  khớp chuỗi phía client trên dữ liệu đã tải từ API public sẵn có.

Trạng thái ẩn/hiện lưu `localStorage` (`vas_mascot_hidden`), có API đọc/ghi bọc try/catch vì
trình duyệt ẩn danh có thể chặn `localStorage`. Đóng panel qua Escape, click ra ngoài dock,
hoặc click lại mascot — dùng `AnimatePresence mode="wait"` để trạng thái "ẩn hẳn" và "đang
đóng panel" không unmount chồng lên nhau giữa chừng animation.

## 7. Khung quản trị: header và sidebar

`AdminLayout` dựng khung cao đúng một màn hình (`100dvh`), chỉ `.admin-content` cuộn nội bộ —
nhờ vậy header và sidebar luôn đứng yên, và trên mobile không bị nhảy khi bàn phím ảo bật lên.

### Chia vai: mỗi thứ chỉ nằm ở một chỗ

Header và sidebar **không** được lặp nội dung của nhau. Người dùng gặp cùng một lối tắt ở hai
chỗ sẽ phải dừng lại tự hỏi "hai cái này có khác nhau không".

| Vùng | Chịu trách nhiệm |
|---|---|
| Sidebar | Điều hướng giữa các trang, và lối tắt sang trang triển lãm công khai (thẻ promo ở chân menu) |
| Header | Trạng thái của **trang đang xem** (tiêu đề, phụ đề, hành động chính) + giờ/ngày + tài khoản |

Vì thế header **không** có nút "Xem triển lãm": sidebar đã giữ lối tắt đó.

### `AdminPageHeader` portal lên thanh header

Từng trang admin khai báo tiêu đề và hành động chính của mình bằng `AdminPageHeader`, nhưng
nội dung được `createPortal` lên ô `#admin-content-header` nằm **trong** `AdminHeader` (xem
`pageHeaderPortal.tsx`). Đổi lại là header không tốn thêm chiều dọc cho một hàng tiêu đề
riêng — đáng kể trên màn hình điện thoại.

Tiêu đề chạy hiệu ứng dồn chữ mỗi lần đổi trang. Chuyển trang trong SPA không có phản hồi
"đã tải xong trang mới" như trình duyệt tải lại, nên hiệu ứng này đóng vai trò báo hiệu.
`key={title}` buộc React dựng lại nhánh để animation chạy lại — thiếu nó thì CSS animation
chỉ chạy đúng lần gắn đầu tiên. Bản chữ đầy đủ vẫn nằm trong DOM cho trình đọc màn hình
(`.admin-page-header-title-sr`), phần tách ký tự đã `aria-hidden` để không bị đọc rời rạc.

### Màu trong khu quản trị

Không dùng mảng màu đặc cho nền và nút. Header phủ ba quầng gradient rất nhạt lấy từ bảng
màu giá trị cốt lõi VASchools; nút hành động chính và avatar chữ cái đầu dùng gradient nhiều chặng
thay vì một khối đỏ phẳng. Dải 5 màu mảnh 2px chạy sát **mép dưới** header kiêm luôn vai trò
đường phân cách với vùng nội dung.

Mọi hiệu ứng nói trên đều tắt dưới `prefers-reduced-motion: reduce`. Lưu ý phần tiêu đề phải
trả lại `opacity`/`transform` bằng tay khi tắt: trạng thái đầu của keyframe nằm ở chính rule
(`opacity: 0`), nên `animation: none` không tự hoàn tác.

### Dashboard: ô chỉ số + card khu vực

Dashboard (`pages/admin/Dashboard.tsx`) từng dùng Recharts cho ba biểu đồ (nhịp hoạt động
14 ngày, phân bổ khối lớp, top tác phẩm dạng bảng). Đã bỏ toàn bộ để đổi sang bố cục gọn
hơn, vừa khít một viewport không cần cuộn: 4 ô chỉ số dẫn dắt (tác phẩm, lượt xem, cảm xúc,
tác phẩm nổi bật - ba ô đầu vẫn dùng sparkline SVG tự vẽ ở `StatCard`, không phải Recharts;
riêng "tác phẩm nổi bật" không có chuỗi theo ngày nên không vẽ sparkline/delta) và 3
**card khu vực** (Sài Gòn/Cần Thơ/Vũng Tàu), mỗi card có:

- Ảnh hero cắt clip-path (tác phẩm đứng đầu bảng tương tác *của khu vực đó*, suy ra bằng
  cách đối chiếu `top_artworks[].school_id` với `school_coverage[].region` - cả hai danh
  sách backend đã trả sẵn trong 1 lần gọi `/api/v1/admin/dashboard/stats`, nên đây thuần là
  suy diễn ở FE, **không** thêm endpoint mới). Khu vực không có tác phẩm nào lọt top hiển
  thị icon thay ảnh, không để trống lặng.
- 3 chỉ số tổng hợp: số tác phẩm, số đã trao giải, % phủ khối lớp.
- Danh sách cơ sở thuộc khu vực kèm số bài, cuộn nội bộ nếu dài.

`lib/chartTheme.ts` giờ chỉ còn `REGION_COLOR`/`REGION_LABEL` (màu định danh cố định theo
khu vực, không đổi theo thứ hạng - để người đã quen "Sài Gòn màu đỏ" không bị đánh lừa) và
`formatNumber` (định dạng số kiểu Việt Nam). Các export chỉ phục vụ biểu đồ (`CHART_*`,
`LEVEL_*`, `formatDayLabel`, `formatCompact`) đã gỡ theo, cùng `recharts` khỏi
`package.json` và nhánh `vendor-charts` khỏi `vite.config.ts`.

### Nhịp hoạt động: bộ lọc theo tháng/khoảng ngày

`ActivityChart` (trong `Dashboard.tsx`) không còn cố định "14 ngày gần nhất" - thêm
`ActivityRangePicker`, segmented control 3 chế độ đặt ngay dưới tiêu đề biểu đồ:

- **14 ngày gần nhất** (mặc định khi vào trang) - không gửi `from`/`to` lên API, giữ nguyên
  hành vi cũ.
- **Theo tháng** - `<input type="month">`, quy về `from` = ngày 1, `to` = ngày cuối tháng đó
  (`lastDayOfMonth()` dùng "ngày 0 của tháng sau" để tự đúng cả tháng 2 năm nhuận, không cần
  bảng tra số ngày/tháng).
- **Khoảng ngày** - 2 `<input type="date">` from/to tuỳ ý, `max`/`min` ràng buộc lẫn nhau và
  chặn chọn ngày tương lai.

State `ActivityRangeValue` (union 3 nhánh theo chế độ) sống ở `Dashboard.tsx`, đổi thì
`useEffect` gọi lại `fetchDashboardStats(signal, range)` - tương tự luồng fetch đã có, chỉ
thêm tham số.

**Khoảng trống khi tháng/khoảng chọn còn ít dữ liệu.** Repository luôn trả đủ ngày liên tục
trong `[from, to]` (nguyên tắc cũ, không đổi - xem
[API.md](../API.md#get-apiv1admindashboardstats)), nhưng `DashboardService.GetStats` cắt bớt
điểm 0 hoạt động ở **đầu và cuối** chuỗi trước khi trả ra
(`trimLeadingTrailingZero` trong `dashboard_service.go`) - tháng đang chọn còn dở dang (vd hôm
nay là ngày 7, 23 ngày còn lại chưa có gì để đếm) sẽ không còn vẽ một đoạn thẳng nằm ngang vô
nghĩa chiếm hết chỗ trống. Ngày 0 hoạt động nằm **xen giữa** hai ngày có dữ liệu vẫn giữ
nguyên - đó là tín hiệu thật (ngày đó không ai thao tác gì), khác về bản chất với phần đầu/
cuối chưa/không còn gì để đếm.

**Mật độ nhãn trục hoành tự thích ứng.** `labelStepFor()` thay cho ngưỡng cứng "> 8 điểm thì
nhảy 2" (chỉ đúng cho 14 ngày) - tính bước nhảy theo đích ~60px/nhãn trên biểu đồ rộng 640
điểm ảo, nên một tháng (~30 điểm) hay một khoảng ngày dài vài tháng đều không bị chồng chữ.

**So sánh biến động (`deltas`)** trước đây neo cứng "đủ 14 điểm mới so sánh 7 ngày/7 ngày";
giờ tổng quát thành "đủ ≥ 4 điểm thì cắt đôi chuỗi đang có mà so sánh nửa/nửa" - áp dụng được
cho mọi độ dài khoảng, không riêng 14 ngày.

Backend: xem [dashboard/stats](../API.md#get-apiv1admindashboardstats) — tham số `from`/`to`,
và cách `ActivityTrend` ở `dashboard_repository.go` nhận khoảng `[from, to]` tường minh thay
vì "N ngày gần nhất".

## 8. Chạy ở chế độ phát triển

```bash
cd web && npm run dev     # Vite tại :5173
go run ./cmd/server       # API tại :8080
```

Vite proxy hai nhánh sang backend:

| Nhánh | Vì sao cần |
|---|---|
| `/api` | Mọi lệnh gọi API |
| `/auth` | Luồng redirect OAuth nằm **ngoài** `/api` — thiếu proxy này thì nút đăng nhập Google không chạy ở dev |

`npm run build` chạy `tsc --noEmit` **trước** khi Vite build — lỗi kiểu dữ liệu chặn được
build, không lọt ra bản phát hành.

## 9. Cách frontend được phục vụ ở production

Không có web server riêng cho frontend. Binary Go phục vụ `web/dist` qua `spaFileServer`:

```text
GET /bat-ky-duong-dan-nao
   ├─ có file thật trong dist/  → trả file đó (JS, CSS, ảnh)
   └─ không có                  → trả index.html  ← React Router xử lý tiếp
   ⚠️ đường dẫn thoát khỏi dist/ → cũng trả index.html (chống path traversal)
```

Nếu `web/dist` chưa tồn tại, `/` trả JSON thông tin API thay vì lỗi — tiện khi phát triển
backend độc lập.

## 10. Trang danh sách tác phẩm quản trị: bulk-featured + chế độ xem Grid

`ArtworksListPage` (`/admin/artworks`) có thêm ba việc ngoài CRUD cơ bản:

**Nút hành động chính chỉ còn icon.** `AdminPageHeader` giờ nhận thêm
`primaryAction.iconOnly` - ẩn phần chữ, chỉ còn icon `+`, giữ nguyên
`title`/`aria-label` để không mất khả năng tiếp cận. Cờ này dùng chung được
cho mọi trang, không riêng danh sách tác phẩm.

**Hai chế độ xem, nhớ lựa chọn qua `localStorage`.** Nút chuyển List ⇄ Grid
lưu key `vas_admin_artworks_view_mode`; đọc lỗi (chế độ ẩn danh chặn
`localStorage`) thì lặng lẽ về `list`, không làm hỏng trang. Grid dùng
`CSS Grid` với `repeat(auto-fill, minmax(190px, 1fr))` - tự co giãn số cột
theo bề ngang màn hình, không cần breakpoint riêng cho tablet/desktop.

**Chọn nhiều + bật/tắt tiêu biểu hàng loạt.** Checkbox trên từng dòng/thẻ dồn
vào state `Set<number>`; đổi trang hoặc đổi bộ lọc thì tự xoá lựa chọn (tránh
áp nhầm hành động lên tác phẩm không còn hiển thị). Thanh `.artworks-bulk-bar`
chỉ hiện khi có ít nhất 1 tác phẩm được chọn, gọi
`PATCH /api/v1/admin/artworks/bulk-featured` (xem
[03-artwork-domain.md](./03-artwork-domain.md) và [API.md](../API.md)) - một
request cho cả lô thay vì lặp `toggleFeatured` cho từng id.

**Xoá hàng loạt** dùng cùng thanh `.artworks-bulk-bar`, gọi
`deleteArtworkBatch()` (`DELETE /api/v1/admin/artworks/bulk-delete`) sau khi
xác nhận qua `ConfirmDialog` riêng (khác dialog xoá 1 tác phẩm). Response trả
`deleted < requested` (một vài ảnh lỗi xoá S3) thì hiện toast báo rõ số lượng
thay vì chỉ nói "thành công" chung chung - tác phẩm chưa xoá được vẫn còn
nguyên trong danh sách sau khi `load()` lại.

**Tải ảnh xuống hàng loạt (.zip).** `lib/artworkDownload.ts` -
`buildArtworkZip()` tải tuần tự từng ảnh gốc qua
`GET /api/v1/admin/artworks/{id}/download` (không watermark, xem
[03-artwork-domain.md](./03-artwork-domain.md)) rồi nén bằng `zipSync` của
`fflate` ngay tại trình duyệt. Bắt buộc tải qua backend rồi tự đóng gói ở
client - không thể lặp thẻ `<a download>` trỏ thẳng URL S3, vì thuộc tính
`download` chỉ có hiệu lực same-origin và ảnh nằm trên domain S3 khác. Ảnh nào
tải lỗi bị loại khỏi file zip (không làm hỏng cả lô), tên trùng nhau tự thêm
hậu tố `(2)`, `(3)`... `triggerBlobDownload()` kích hoạt tải file zip qua thẻ
`<a>` tạm trỏ `blob:` URL (same-origin nên `download` hoạt động), dùng chung
cho cả tải zip lẫn tải ảnh gốc đơn lẻ trong `ArtworkEditModal`.

**Dòng dung lượng ảnh.** `formatBytes()` (`lib/api.ts`, dùng lại từ công cụ
upload gốc) hiện `artworks.file_size` dưới tên tác phẩm ở cả bảng List
(`.artworks-cell-title`) và thẻ Grid (`.artworks-grid-body`) - trường này đã
có sẵn trong `ArtworkWithMeta`, chỉ thêm hiển thị.

**Toggle "Hiển thị công khai" trong modal sửa.** `ArtworkEditModal` thêm
checkbox `is_published` cạnh ảnh preview, tách khỏi `ArtworkMetaForm` (form đó
dùng chung cho cả lúc *tạo* tác phẩm, khi "công khai" luôn mặc định `true` -
đặt toggle trong đó sẽ sai ngữ cảnh). Cùng nguyên tắc đã áp dụng cho
`is_featured` (đặt riêng ở nút icon ngoài `ArtworkMetaForm`, không phải
field trong form): đây là quyết định **trạng thái** của tác phẩm, không phải
**metadata** biên tập nội dung. Tắt toggle này dùng lại đúng cột
`artworks.is_published` có sẵn (không thêm cột `is_active` mới) - cột này vốn
đã đúng nghĩa "tắt thì ẩn khỏi mọi trang public, không xoá dữ liệu" (xem
[03-artwork-domain.md §1](./03-artwork-domain.md)).

**Hover phóng ảnh tại chỗ.** `transform: scale()` thuần CSS, không dùng thư
viện lightbox: bảng List phóng ảnh tràn ra ngoài ô (`overflow: visible` trên
ô chứa) vì thumbnail quá nhỏ để nhìn rõ nếu chỉ phóng gọn trong khung; Grid
phóng nhẹ hơn và bị cắt bởi `overflow: hidden` của card vì ảnh grid đã đủ lớn.
Tắt hẳn dưới `prefers-reduced-motion: reduce` và trên thiết bị cảm ứng
(`hover: none`) - chạm màn hình không có sự kiện hover thật, ảnh phóng to chỉ
gây khó chịu khi "dính" lại sau khi chạm.

**Modal sửa tác phẩm (`ArtworkEditModal`) portal ra `document.body`.** Ban đầu
modal render tại chỗ trong cây `<Outlet />`, tức bên trong `.admin-content-inner`.
Mỗi lần chuyển route, khối này mang class `route-enter` (`styles/tokens.css`),
dùng `will-change: transform` cho hoạt cảnh trượt vào - và `will-change:
transform` biến chính ancestor đó thành **containing block mới cho
`position: fixed`** (ngang hàng với việc đặt `transform` thật). Hệ quả: modal
`position: fixed; inset: 0` bị "nhốt" trong khung `.admin-content` (vùng tự
cuộn nội bộ, xem mục 7) thay vì phủ toàn viewport - tràn xuống đáy, bị cắt bởi
`overflow-y: auto` của khung cuộn. Khắc phục bằng `createPortal(..., document.body)`
- cùng nguyên nhân, cùng cách sửa đã áp dụng cho `ConfirmDialog`
(`components/ConfirmDialog.tsx`). Modal/dialog toàn màn hình mới thêm sau này
**phải** portal ra `document.body`, không phụ thuộc vị trí render trong cây.

**Kiểm duyệt bình luận (`ArtworkCommentsModal`).** Icon bình luận (số `comment_count` ở cột
"Tương tác" trong List, icon riêng ở Grid) mở modal liệt kê **toàn bộ** bình luận của tác
phẩm — kể cả đã ẩn — qua `GET /api/v1/admin/artworks/{id}/comments`, mỗi dòng có nút Ẩn/Hiện
gọi `PATCH .../comments/{commentID}` (xem [API.md](../API.md) và
[04-public-engagement.md](./04-public-engagement.md)). Component dùng chung khung CSS
`.artwork-modal-*` với `ArtworkEditModal` (cùng lý do portal ra `document.body`, xem dưới),
chỉ thêm class riêng `.comments-modal-*` cho phần danh sách. Ẩn/hiện thành công cập nhật
`comment_count` ngay trên state `items` của trang (không phải gọi lại `load()` cả danh sách)
qua callback `onCountChange`, và tải lại danh sách bình luận mỗi lần mở modal (kể cả cùng
tác phẩm) để không hiện dữ liệu cũ nếu có bình luận mới từ trang public trong lúc admin đang
xem trang khác.

### Dải card "theo khu vực" ở đầu trang Tác phẩm/Giải thưởng/Nhóm chủ đề — đã gỡ bỏ

Ba trang quản trị `ArtworksListPage`, `AwardsPage`, `TopicCategoriesPage` từng mở đầu bằng
dải 6 ô nhỏ (3 khu vực × tác phẩm/học sinh) qua `component/admin/RegionSummaryStrip.tsx` +
`hooks/useRegionSummary.ts`, lấy dữ liệu từ `GET /api/v1/admin/dashboard/region-summary`.
Component và hook đã bị xoá cùng phần gọi trong cả 3 trang — endpoint
`region-summary` ở backend vẫn còn (xem [API.md](../API.md)) nhưng hiện không có nơi nào ở
frontend gọi tới nó. Bộ lọc `region=` trực tiếp trên `ArtworkFilter` (xem
[03-artwork-domain.md](./03-artwork-domain.md)) đã đảm nhiệm việc lọc theo khu vực khi cần.

## 11. Trang tải tác phẩm lên: một luồng, lưới thẻ ảnh + panel sửa

`ArtworksUploadPage` (`/admin/artworks/upload`) từng tách "Tải 1 ảnh"/"Tải nhiều ảnh" bằng 2
tab dùng chung route, chế độ nhiều ảnh dùng `ArtworkBulkTable` — bảng HTML mỗi ảnh 1 hàng,
input/select nhồi trong ô hẹp, phải cuộn ngang trên mobile. Đã bỏ hẳn cách này (không remix)
để đổi sang **một luồng duy nhất**: chọn 1 hay nhiều ảnh đều vào chung 1 giao diện.

```text
AdminDropzone (kéo-thả/dán/chọn file - luôn hiện, kể cả khi đã có ảnh)
   │
   ▼
artwork-picker-layout (2 cột, ≤768px xếp dọc)
   ├─ ArtworkPickerGrid   lưới thẻ ảnh, mỗi thẻ vuông đủ lớn để nhận diện
   └─ panel sửa bên phải  form cho ảnh đang chọn
```

Dữ liệu nền (`schools`, `gradeLevels`, `topicCategories`, `awards`) vẫn tải một lần như cũ.
State ảnh gộp về một mảng `rows: BulkRow[]` duy nhất (kiểu `BulkRow` chuyển sang
`components/admin/artworkUploadTypes.ts` vì không còn gắn với 1 component bảng cụ thể), thêm
`selectedIds: Set<string>` cho việc chọn thẻ.

**Chọn thẻ, không phải "click để mở form riêng".** Click vào thân thẻ = chọn **riêng** đúng
thẻ đó (thay thế toàn bộ lựa chọn hiện có), giống file-manager thông thường. Click ô tick góc
trái = bật/tắt thẻ đó trong lựa chọn hiện có (multi-select), tách biệt khỏi việc click thân
thẻ để không xung đột hai kiểu chọn trên cùng 1 vùng bấm.

Panel bên phải đổi theo số thẻ đang chọn (`ArtworksUploadPage.tsx`):

| Số thẻ chọn | Panel hiện gì |
|---|---|
| 0 | Thông báo "Chọn một ảnh trong lưới" |
| 1 | `ArtworkMetaForm` đầy đủ, validate bắt buộc như cũ (`showAllErrors` bật khi bấm "Lưu tất cả" mà thẻ đó còn thiếu) |
| ≥2 | Form "áp dụng cho N ảnh" — field nào điền thì ghi đè lên **mọi** thẻ đang chọn, field để trống giữ nguyên giá trị riêng từng ảnh |

**Sửa hàng loạt tái dùng nguyên `ArtworkMetaForm`, không thêm prop mới vào nó.** Panel ≥2 thẻ
render `ArtworkMetaForm` với `values` là một `ArtworkMetaFormValues` rỗng tách biệt
(`EMPTY_META_FORM_VALUES`), luôn truyền `showAllErrors={false}` và bỏ qua kết quả validate —
bulk-edit không bắt buộc field nào. Nút "Áp dụng cho N ảnh" gọi `applyPatch()` (hàm thuần
trong `ArtworksUploadPage.tsx`) merge từng field không rỗng vào tất cả thẻ trong
`selectedIds`; `awardIds` ghi đè toàn bộ danh sách nếu patch có chọn ít nhất 1 giải (không
cộng dồn — dễ đoán hơn "hợp nhất 2 mảng"). Cân nhắc đã loại: không sửa `ArtworkMetaForm` để
nhận cờ "tắt validate", vì component này đã dùng chung ở 3 nơi (xem đoạn dưới) — thêm field
mới ở đó phải rà đủ cả ba, thêm biến thể hành vi càng làm việc rà đó dễ sót.

**`ArtworkPickerGrid` (`components/admin/ArtworkPickerGrid.tsx`) thay `ArtworkBulkTable`.**
Mỗi thẻ: ảnh vuông cover, ô tick góc trái, badge trạng thái góc phải (chờ lưu/đang lưu/đã
lưu/lỗi — thay `RowStatusBadge` cũ), tên tác phẩm cắt ngắn dưới ảnh, nút xoá hiện khi
hover/focus (luôn hiện trên cảm ứng qua `@media (hover: none) and (pointer: coarse)`, cùng lý
do đã áp dụng ở `ArtworksListPage` mục 10). Không còn popover phóng to khi hover thumbnail —
thẻ trong lưới đã đủ lớn để nhận diện, khác thumbnail 3rem trong bảng cũ buộc phải phóng to
mới đọc được.

**`AdminDropzone` (`components/admin/AdminDropzone.tsx`) — component kéo-thả riêng cho khu
quản trị.** Ban đầu viết tách khỏi `Dropzone.tsx` (component của trang `/upload` độc lập
lúc đó — theme tối kiểu glassmorphism, hiệu ứng xoay 3D theo con trỏ và viền chạy vô hạn
bằng framer-motion) vì nhét thẳng component đó vào trang admin nền sáng từng gây giao diện
"lệch tông". `AdminDropzone`: phẳng, không framer-motion, style qua `admin.css`/token
`--color-*`, giữ hành vi kéo-thả/dán Ctrl+Cmd+V/chọn file/disabled khi đang lưu nhưng bớt
phần trình diễn — đây là công cụ quản trị, không phải trang giới thiệu. Trang `/upload` và
`Dropzone.tsx` đã bị xoá sau đó (xem mục 1); `AdminDropzone` không phụ thuộc nó nên không
bị ảnh hưởng.

**Ảnh chỉ chạm S3 lúc bấm Lưu, không phải lúc chọn file** — hành vi này KHÔNG đổi so với bản
trước. Khác với thiết kế bulk-upload gốc mô tả ở [03-artwork-domain.md](./03-artwork-domain.md)
(đẩy S3 trước, điền metadata sau, để lỗi mạng lúc điền form không làm mất ảnh đã lên), trang
này giữ ảnh dưới dạng `URL.createObjectURL(file)` thuần phía client cho tới khi người dùng
bấm "Lưu tất cả" — lúc đó mới gọi `bulkUploadArtworks` rồi `createArtwork` cho từng ảnh. Đánh
đổi có chủ đích: đổi lại việc phải nhập lại metadata nếu mạng lỗi giữa chừng lấy ưu tiên xem
toàn bộ ảnh đã chọn trước khi bất kỳ thứ gì rời khỏi máy. Xem lý do đầy đủ trong comment đầu
`ArtworksUploadPage.tsx`.

**Upload từng ảnh một khi lưu, không gộp 1 request nhiều file** — cũng KHÔNG đổi. Dù
`bulkUploadArtworks` nhận được mảng file, trang gọi nó với **1 file mỗi lần** trong vòng lặp
tuần tự khi bấm "Lưu tất cả" — để biết chính xác ảnh nào lỗi và cập nhật trạng thái đúng thẻ
đó (`pending` → `saving` → `done`/`error`), thay vì đợi cả lô xong mới biết kết quả tổng.

**`ArtworkMetaForm` dùng chung giữa 3 nơi**: panel sửa 1 ảnh, panel sửa hàng loạt (cùng dùng
`ArtworkMetaForm` như mô tả ở trên, khác nhau ở `values` truyền vào và có bật validate hay
không), và `ArtworkEditModal` ở trang danh sách. Sửa field mới trên `ArtworkMetaFormValues`
phải rà cả ba nơi.

## 12. Validate bắt buộc: dấu `*` + lỗi hiện ngay tại field

Mọi form admin dùng chung một cách đánh dấu bắt buộc và một cách hiện lỗi, để người dùng
không phải học lại quy ước mỗi trang.

**Nguồn sự thật duy nhất cho field bắt buộc**: `validateMetaForm()` trong
`components/admin/ArtworkMetaForm.tsx` — nhận `ArtworkMetaFormValues`, trả về
`MetaFormErrors` (map lỗi theo field, rỗng = hợp lệ). `isMetaFormValid()` chỉ là
`Object.keys(validateMetaForm(values)).length === 0`, giữ lại cho chỗ chỉ cần biết đúng/sai
(vd điều kiện `disabled` của nút Lưu). Trước đây hai hàm tách rời có thể lệch nhau khi thêm
field mới; giờ chỉ một danh sách field bắt buộc (`RequiredMetaField`) chi phối cả logic lẫn
UI (dấu `*` chỉ hiện đúng field có trong danh sách này).

**`RequiredMark`** (cùng file) là component `<span>*</span>` dùng chung — không tự viết
`<span style="color:red">*</span>` ở form khác, vì màu/khoảng cách phải giống nhau mọi nơi
(`.form-required-mark` trong `admin.css`).

**Lỗi chỉ hiện khi field đã "touched" (rời khỏi ô - `onBlur`), hoặc khi `showAllErrors`
bật.** Không hiện lỗi ngay lúc form vừa mount trống — nếu không, một form 5 trường bắt buộc
sẽ đỏ lòm ngay khi mở, gây hoảng chứ không giúp ích. `showAllErrors` được bật từ nơi gọi khi
người dùng bấm Lưu mà form còn thiếu, để không phải rà tay từng ô mới biết còn sai chỗ nào:

- **`ArtworksUploadPage`** (panel sửa 1 ảnh): giữ state `showErrors`, bật khi `saveAll()`
  phát hiện còn thẻ chưa hợp lệ (`!isMetaFormValid(...)`) — dùng chung 1 cờ cho toàn trang vì
  panel chỉ hiện form của đúng 1 ảnh tại một thời điểm, khác bản trước có `singleShowErrors`
  riêng cho chế độ 1 ảnh. Thẻ nào chưa điền đủ còn được viền đỏ ngay trong `ArtworkPickerGrid`
  qua `errorRowIds` (`Set<localId>`, điền vào lúc `saveAll()` phát hiện thẻ chưa hợp lệ) — độc
  lập với việc field trong panel có "touched" hay chưa, để nhìn lướt qua lưới cũng biết ảnh
  nào còn thiếu mà không cần click vào từng thẻ.
- **`AwardsPage`** (form giải thưởng, không dùng `ArtworkMetaForm`) tự áp cùng pattern tại
  chỗ: 1 field bắt buộc (`tên giải`), state `nameTouched`, tính lỗi trực tiếp thay vì gọi
  `validateMetaForm` (không cùng kiểu dữ liệu).
- **`TopicCategoriesPage`** (`/admin/topic-categories`, form quản lý nhóm chủ đề sáng tạo)
  cùng bố cục danh sách thẻ + form bên phải và cùng cơ chế kéo-thả (`Reorder.Group`) như
  `AwardsPage`, cùng pattern validate 1 field bắt buộc (`tên nhóm chủ đề`). Khác Award ở chỗ
  danh sách tách thành **3 khối kéo-thả độc lập** theo cấp học (Mọi cấp học / Tiểu học /
  Trung học, `.topic-category-groups`): `display_order` chỉ so sánh có ý nghĩa trong cùng
  một cấp học, vì `ArtworkMetaForm` lọc theo `education_level` trước rồi mới áp thứ tự — gộp
  chung 1 danh sách kéo-thả (như Award) sẽ cho ra con số không phản ánh đúng thứ tự hiển thị
  thực tế. Đổi cấp học áp dụng của một nhóm khi sửa thì nhóm đó tự xếp cuối khối mới, không
  giữ `display_order` cũ (thuộc khối khác, dễ trùng vị trí).

  Nút "+" cạnh nhãn "Nhóm chủ đề sáng tạo" trong `ArtworkMetaForm` mở
  `TopicCategoryQuickCreateModal` — tạo nhanh 1 nhóm (hỏi tên, màu sắc và cấp học, luôn
  active, `display_order = 0`) mà không rời form đang nhập tác phẩm. Màu dùng chung dải màu
  preset + input `type="color"` tuỳ ý với `TopicCategoriesPage`, tách vào
  `topicCategoryColors.ts` (`TOPIC_CATEGORY_COLOR_PRESETS`, `TOPIC_CATEGORY_DEFAULT_COLOR`) để
  2 nơi tạo/sửa nhóm chủ đề không lệch bảng màu. Modal nhận
  `onTopicCategoryCreated` từ nơi gọi `ArtworkMetaForm` (không bắt buộc — thiếu thì ẩn hẳn
  nút "+") để ghi nhóm mới vào đúng state `topicCategories` của trang cha
  (`ArtworksUploadPage`, `ArtworkEditModal`/`ArtworksListPage`), tránh giữ một bản sao lệch
  khỏi nguồn sự thật. Muốn sắp lại thứ tự hoặc đổi cấp học sau khi tạo nhanh thì vào hẳn
  `/admin/topic-categories`.

**CSS dùng chung** (`admin.css`, ngay sau block `.form-field`): `.form-field--error` (viền +
nền đỏ nhạt trên input/select, nhãn đổi màu), `.form-field-error-text` (dòng lỗi dưới ô).
`ArtworkPickerGrid` đánh dấu thẻ lỗi bằng viền đỏ trên card (`.artwork-picker-card--invalid`)
thay vì lặp lại kiểu `.form-field--error` — thẻ ảnh không có input/select bên trong để viền,
chỉ cần báo hiệu "thẻ này còn thiếu" rồi người dùng click vào xem chi tiết lỗi trong panel.

**Thêm form admin mới có field bắt buộc**: dùng lại `RequiredMark` + class `.form-field--error`/
`.form-field-error-text` thay vì tự nghĩ ra kiểu mới. Nếu field đó thuộc `ArtworkMetaFormValues`,
thêm vào `RequiredMetaField` và `validateMetaForm()` — không thêm điều kiện validate rời rạc ở
từng nơi gọi.

## 13. Meta tags động + JSON-LD cho SEO

Trước đây `index.html` chỉ có một `<title>` tĩnh duy nhất dùng chung cho mọi route — Googlebot
crawl trang chủ hay trang bảng vàng đều thấy cùng một tiêu đề. Xem quyết định kiến trúc đầy đủ
(vì sao không SSR/prerender) ở [docs/plan/02-roadmap.md P2.15](../plan/02-roadmap.md).

**`hooks/usePageMeta.ts`** — hook tự viết (~40 dòng), KHÔNG dùng `react-helmet-async`: nhu cầu
chỉ là title + description + canonical cho 4 trang public + `NotFoundPage`, mỗi route có đúng
một tầng gọi hook, không có component con nào cần ghi đè — không cần cơ chế merge theo cây
component mà các thư viện quản lý `<head>` giải quyết. Dùng DOM API trực tiếp
(`document.title`, `querySelector` + tạo/update thẻ `<meta>`/`<link>`), không cleanup khi
unmount (trang kế tiếp luôn tự gọi hook và ghi đè ngay).

Mỗi trang public tự gọi ở đầu component với `title`/`description` riêng:

| Route | Title |
|---|---|
| `/` (HomePage) | Khu vườn nghệ thuật VA Schools — 20 năm Trường Việt Mỹ |
| `/tac-pham-tieu-bieu` (FeaturedArtworksPage) | Tác phẩm tiêu biểu — Khu vườn nghệ thuật VA Schools |
| `/phong-trien-lam` (GalleryPage) | Phòng triển lãm — Khu vườn nghệ thuật VA Schools |
| `/bang-vang` (HallOfFamePage) | Bảng vàng — Khu vườn nghệ thuật VA Schools |
| `NotFoundPage` (public `*`) | Không tìm thấy trang — VA Schools |

**Canonical cố định, không kèm query param.** `FeaturedArtworksPage` và `GalleryPage` nhận
`canonicalPath` tường minh (vd `"/tac-pham-tieu-bieu"`) thay vì để hook tự suy từ
`location.pathname` — các biến thể `?tranh=`, `?khu-vuc=`, `?tim=`, `?khoi=` đều là cùng một
nội dung cơ bản (lọc/mở modal), không nên để Google index như những trang riêng biệt (duplicate
content). `/admin/*` KHÔNG dùng hook này (khu quản trị không cần SEO).

**`components/JsonLd.tsx`** — component dùng chung chèn `<script type="application/ld+json">`.
Không gộp vào `usePageMeta` vì khác bản chất (một `<script>` render trong cây React, không phải
thao tác DOM thủ công vào `<head>`). Dữ liệu đầu vào luôn dựng sẵn ở component gọi (tên trang,
mô tả, URL cố định), không lấy trực tiếp từ input người dùng chưa kiểm soát.

| Trang | Schema |
|---|---|
| HomePage | `WebSite` |
| FeaturedArtworksPage, GalleryPage, HallOfFamePage | `CollectionPage` + `BreadcrumbList` |

`CollectionPage` phù hợp hơn `ImageGallery` vì đây là trang danh sách/lọc, không phải một bộ
sưu tập ảnh cố định. `BreadcrumbList` không thêm cho `HomePage` (gốc, breadcrumb 1 phần tử vô
nghĩa).

## 14. Điểm cần lưu ý khi sửa frontend

- **Kiểu dữ liệu phải khớp backend.** `ArtworkWithMeta` trong `lib/artworkApi.ts` phản chiếu
  struct Go cùng tên. Lưu ý `s3_url` ra JSON dưới tên **`image_url`**, và `s3_key` không bao
  giờ được trả về.
- **Không đọc `visitor_token` ngoài `lib/visitorToken.ts`.** Mọi chỗ dùng phải qua
  `getOrCreateVisitorToken()` để xử lý fallback nhất quán.
- **Thêm route mới**: nhớ thêm `lazy()` nếu đó không phải trang chủ, nếu không sẽ kéo ngược
  vào bundle chính.
- **Giữ chunk thư viện ổn định**: thêm thư viện nặng thì cân nhắc bổ sung nhánh trong
  `manualChunks`.
- **Thêm lối tắt vào khu quản trị**: chọn **một** chỗ — header hoặc sidebar, không phải cả
  hai (xem mục 7). Điều hướng thuộc về sidebar; header chỉ nói về trang đang xem.
- **Thêm hiệu ứng chuyển động**: bổ sung selector tương ứng vào khối
  `prefers-reduced-motion: reduce` cuối `styles/admin.css`. Hiệu ứng có trạng thái đầu ẩn
  (`opacity: 0`) còn phải trả lại giá trị cuối bằng tay ở khối đó.
