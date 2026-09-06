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
│   │   └── HallOfFamePage        /bang-vang
│   │   ├── HallOfFamePage         /bang-vang
│   │   └── NotFoundPage           * (mọi đường dẫn lạ dưới "/")
│   ├── admin/              ◀── khu vực quản trị (session)
│   │   ├── AdminLayout     bảo vệ route + sidebar
│   │   ├── Login                 /admin/login
│   │   ├── Dashboard             /admin
│   │   ├── ArtworksListPage      /admin/artworks
│   │   ├── ArtworksUploadPage    /admin/artworks/upload
│   │   ├── AwardsPage            /admin/awards
│   │   ├── TopicCategoriesPage   /admin/topic-categories
│   │   └── NotFoundPage           * (mọi đường dẫn lạ dưới "/admin")
│   └── UploadTool.tsx      ◀── công cụ nội bộ   /upload
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
| `/upload` | Công cụ upload nội bộ | Trước đây ở `/`, đã nhường chỗ |
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
| Tải bundle lần đầu (trước khi React mount) | `index.html` (`#app-splash`) | Splash toàn màn hình, HTML/CSS thuần |

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
component. Gỡ bằng `MutationObserver` theo dõi `#root` có con hay chưa (không dùng sự kiện
`load`, vì đó là lúc tài nguyên tải xong chứ không phải lúc màn hình có nội dung), kèm chốt
timeout 8s để không kẹt vĩnh viễn nếu bundle lỗi.

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

23 component trong `components/public/`, chia theo vai trò:

| Nhóm | Thành phần |
|---|---|
| Bố cục | `PublicNavbar`, `PublicFooter` (khung `PublicLayout` nằm ở `pages/public/`) |
| Trang chủ | `HeroSection`, `HeroParallaxHills`, `HillDivider`, `EducationLevelGate`, `EducationLevelCard`, `GradeNode` |
| Tiêu biểu | `FeaturedHero`, `FeaturedGardenScene`, `FeaturedArtworkFrame`, `ArtworkRail`, `RegionTabs` |
| Triển lãm | `GalleryLevelSection`, `GalleryTopicSection`, `GallerySearch`, `GalleryPagination` |
| Bảng vàng | `HallArtworkCard`, `HallRail`, `HallFireworks` |
| Tương tác | `PublicLightbox`, `ReactionPicker`, `CommentBox` |

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

### Hiệu ứng và hiệu năng

`useParallaxScroll` là hook dùng chung cho hiệu ứng cuộn nhiều lớp ở trang chủ. Trang này
nặng về hình ảnh động (đồi parallax, pháo hoa, cảnh vườn), nên các nguyên tắc sau được giữ:

- Hiệu ứng cuộn đi qua `requestAnimationFrame`, không gắn trực tiếp vào sự kiện `scroll`.
- Ảnh nền dùng SVG khi có thể (`garden-butterfly.svg`, `garden-fern-cluster.svg`) — nhẹ và
  sắc nét ở mọi độ phân giải.
- Framer Motion nằm ở chunk riêng, không kéo theo khi vào khu admin.

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
bình luận - vẫn dùng sparkline SVG tự vẽ ở `StatCard`, không phải Recharts) và 3 **card khu
vực** (Sài Gòn/Cần Thơ/Vũng Tàu), mỗi card có:

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

### Dải card "theo khu vực" ở đầu trang Tác phẩm/Giải thưởng/Nhóm chủ đề

Ba trang quản trị `ArtworksListPage`, `AwardsPage`, `TopicCategoriesPage` đều mở đầu bằng dải
6 ô nhỏ (3 khu vực × tác phẩm/học sinh), lấy qua
`GET /api/v1/admin/dashboard/region-summary` (xem [API.md](../API.md)). Đặt ngay sau
`<AdminPageHeader />` trong JSX của cả 3 trang — lưu ý `AdminPageHeader` render qua
`createPortal` nên bản thân nó không chiếm chỗ tại vị trí gọi, phần tử JSX theo sau nó mới là
nơi hiển thị thật.

Component `components/admin/RegionSummaryStrip.tsx` **tái dùng nguyên `StatCard.tsx`**
(không viết component nhân bản) — `StatCard` vốn chỉ có 6 `tone` cố định
(`tri-thuc`/`khai-phong`/`nhan-ai`/`trach-nhiem`/`ban-linh`/`neutral`), mở rộng thêm 3 tone
`saigon`/`cantho`/`vungtau` map sang đúng `REGION_COLOR` (`lib/chartTheme.ts`) trong
`TONE_INK`. Đây là thay đổi an toàn cho `StatCard` (thêm entry vào map, không đổi hành vi
tone cũ) nên `Dashboard.tsx` (nơi `StatCard` gốc phục vụ) không bị ảnh hưởng.

Ba trang dùng chung hook `hooks/useRegionSummary.ts` (state + `useEffect` + `AbortController`,
gọi 1 lần khi mount) thay vì mỗi trang tự viết lại - tiền lệ đã có ở `useAdminAuth.ts` dùng
chung cho `AdminLayout`/`AdminHeader`. Lỗi tải dải card này **không toast** (khác
`fetchDashboardStats` ở Dashboard) - đây là dải phụ trợ ở 3 trang có chức năng chính khác,
lỗi chỉ ẩn dải (`data.length === 0` → `null`), không làm phiền thao tác chính.

"Học sinh" đếm `COUNT(*)` bản ghi `students` JOIN `schools` theo `region`, **không dedupe
theo tên** - đúng quy ước ở [01-database.md](./01-database.md) mục `students`.

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

**`AdminDropzone` (`components/admin/AdminDropzone.tsx`) — component kéo-thả RIÊNG cho khu
quản trị, không dùng chung với `Dropzone.tsx`.** `Dropzone.tsx` phục vụ trang `/upload` độc
lập (`UploadTool.tsx`) — theme tối kiểu glassmorphism định nghĩa trong `index.css` (nạp
global cho toàn app qua `main.tsx`), có hiệu ứng xoay 3D theo con trỏ và viền chạy vô hạn
bằng framer-motion. Nhét thẳng component đó vào trang admin nền sáng là nguyên nhân giao diện
"lệch tông" ở bản trước. `AdminDropzone` viết lại từ đầu: phẳng, không framer-motion, style
qua `admin.css`/token `--color-*`, giữ nguyên hành vi (kéo-thả, dán Ctrl/Cmd+V, chọn file,
disabled khi đang lưu) nhưng bớt hẳn phần trình diễn — đây là công cụ quản trị, không phải
trang giới thiệu. **Sửa `Dropzone.tsx` gốc không ảnh hưởng trang này và ngược lại** — hai
component độc lập hoàn toàn dù cùng vai trò kéo-thả.

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

## 13. Điểm cần lưu ý khi sửa frontend

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
