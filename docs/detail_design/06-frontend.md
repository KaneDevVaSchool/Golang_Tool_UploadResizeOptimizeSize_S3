# 06 — Frontend

**Công nghệ**: React 19 · TypeScript 5.8 · Vite 7 · React Router 7 · Framer Motion 12 · Recharts 3
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
│   ├── admin/              ◀── khu vực quản trị (session)
│   │   ├── AdminLayout     bảo vệ route + sidebar
│   │   ├── Login                 /admin/login
│   │   ├── Dashboard             /admin
│   │   ├── ArtworksListPage      /admin/artworks
│   │   ├── ArtworksUploadPage    /admin/artworks/upload
│   │   └── AwardsPage            /admin/awards
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

## 2. Tách bundle theo route

`App.tsx` nạp tĩnh **chỉ** `PublicLayout` + `HomePage`; mọi trang khác qua `React.lazy`.

Lý do rất cụ thể: trước khi tách, một phụ huynh vào xem tranh phải tải kèm **toàn bộ**
dashboard quản trị, thư viện biểu đồ Recharts (~400KB) và công cụ upload S3 — những thứ họ
không bao giờ mở được. Nay các chunk đó chỉ tải khi thực sự vào `/admin`.

Một `<Suspense>` duy nhất bọc ngoài `<Routes>` là đủ, vì mỗi lần chỉ có một route khớp nên
không bao giờ có hai chunk cùng treo fallback.

Song song đó, `vite.config.ts` tách **theo thư viện**:

| Chunk | Chứa | Vì sao tách |
|---|---|---|
| `vendor-charts` | recharts, d3-*, victory-vendor | Rất nặng, chỉ dùng ở Dashboard admin |
| `vendor-motion` | framer-motion, motion-dom, motion-utils | Dùng nhiều ở trang public |
| `vendor-router` | react-router | Ổn định, hiếm đổi |
| `vendor-react` | react, react-dom, scheduler | Ổn định nhất, cache lâu nhất |

Mục tiêu: sửa một dòng trong `src/` không làm đổi hash của các chunk thư viện, nên trình
duyệt khách giữ nguyên cache qua nhiều lần deploy.

⚠️ Thứ tự kiểm tra trong `manualChunks` **quan trọng**: `react-dom` và `react-router` đều
chứa chuỗi `"react"`, nên các nhánh riêng phải đứng trước nhánh `react` chung. Ghi chú này
có trong chính file cấu hình.

## 3. Bốn client API tách theo khu vực

| File | Phục vụ | Xác thực |
|---|---|---|
| `lib/api.ts` | Nền chung: CSRF token, dựng URL, xử lý envelope | — |
| `lib/publicApi.ts` | `/api/v1/public/*` | Không — chỉ CSRF + `visitor_token` |
| `lib/adminApi.ts` | `/api/v1/admin/*` | Cookie session; ném `UnauthorizedError` khi 401 |
| `lib/artworkApi.ts`, `awardApi.ts`, `dashboardApi.ts` | Bọc theo miền, dùng lại `adminApi` | Session |
| `lib/artworkImage.ts` | Chọn cỡ ảnh hiển thị (xem bên dưới) | — |

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

22 component trong `components/public/`, chia theo vai trò:

| Nhóm | Thành phần |
|---|---|
| Bố cục | `PublicNavbar`, `PublicFooter`, `PublicLayout` |
| Trang chủ | `HeroSection`, `HeroParallaxHills`, `HillDivider`, `EducationLevelGate`, `EducationLevelCard`, `GradeNode` |
| Tiêu biểu | `FeaturedHero`, `FeaturedGardenScene`, `FeaturedArtworkFrame`, `ArtworkRail`, `RegionTabs` |
| Triển lãm | `GalleryLevelSection`, `GallerySearch`, `GalleryPagination` |
| Bảng vàng | `HallArtworkCard`, `HallRail`, `HallFireworks` |
| Tương tác | `PublicLightbox`, `ReactionPicker`, `CommentBox` |

### Hiệu ứng và hiệu năng

`useParallaxScroll` là hook dùng chung cho hiệu ứng cuộn nhiều lớp ở trang chủ. Trang này
nặng về hình ảnh động (đồi parallax, pháo hoa, cảnh vườn), nên các nguyên tắc sau được giữ:

- Hiệu ứng cuộn đi qua `requestAnimationFrame`, không gắn trực tiếp vào sự kiện `scroll`.
- Ảnh nền dùng SVG khi có thể (`garden-butterfly.svg`, `garden-fern-cluster.svg`) — nhẹ và
  sắc nét ở mọi độ phân giải.
- Framer Motion nằm ở chunk riêng, không kéo theo khi vào khu admin.

## 7. Chạy ở chế độ phát triển

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

## 8. Cách frontend được phục vụ ở production

Không có web server riêng cho frontend. Binary Go phục vụ `web/dist` qua `spaFileServer`:

```text
GET /bat-ky-duong-dan-nao
   ├─ có file thật trong dist/  → trả file đó (JS, CSS, ảnh)
   └─ không có                  → trả index.html  ← React Router xử lý tiếp
   ⚠️ đường dẫn thoát khỏi dist/ → cũng trả index.html (chống path traversal)
```

Nếu `web/dist` chưa tồn tại, `/` trả JSON thông tin API thay vì lỗi — tiện khi phát triển
backend độc lập.

## 9. Điểm cần lưu ý khi sửa frontend

- **Kiểu dữ liệu phải khớp backend.** `ArtworkWithMeta` trong `lib/artworkApi.ts` phản chiếu
  struct Go cùng tên. Lưu ý `s3_url` ra JSON dưới tên **`image_url`**, và `s3_key` không bao
  giờ được trả về.
- **Không đọc `visitor_token` ngoài `lib/visitorToken.ts`.** Mọi chỗ dùng phải qua
  `getOrCreateVisitorToken()` để xử lý fallback nhất quán.
- **Thêm route mới**: nhớ thêm `lazy()` nếu đó không phải trang chủ, nếu không sẽ kéo ngược
  vào bundle chính.
- **Giữ chunk thư viện ổn định**: thêm thư viện nặng thì cân nhắc bổ sung nhánh trong
  `manualChunks`.
