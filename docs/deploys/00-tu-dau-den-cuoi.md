# 00 — Từ số không đến production

Hướng dẫn **cực chi tiết**, dành cho người chưa từng dựng server Linux bao giờ. Mỗi lệnh
đều kèm: nó làm gì, kết quả đúng trông ra sao, và sai thì sửa thế nào.

> **Tài liệu này khác gì [01-vps-systemd.md](./01-vps-systemd.md)?**
> File 01 bắt đầu từ lúc VPS **đã có sẵn** Go, Node, MySQL, Nginx — nó là quy trình tra cứu
> nhanh cho người đã quen. File này bắt đầu từ lúc bạn **chưa mua VPS**, và giải thích cả
> những thứ file 01 coi là hiển nhiên: tạo user, mở firewall, cấu hình MySQL, tạo IAM user
> trên AWS. Deploy lần đầu thì đọc file này. Các lần sau dùng file 01.

**Thời gian dự kiến**: 90–120 phút cho lần đầu, trong đó ~20 phút là chờ (DNS, cài gói).

---

## Mục lục

| Giai đoạn | Nội dung | Thời gian |
|---|---|---|
| [A](#a--chuẩn-bị-trước-khi-chạm-vào-server) | Chuẩn bị: mua VPS, trỏ DNS, gom thông tin | 20 phút + chờ DNS |
| [B](#b--đăng-nhập-lần-đầu-và-làm-cứng-server) | Đăng nhập lần đầu, tạo user, khoá SSH, bật firewall | 25 phút |
| [C](#c--cài-phần-mềm-nền) | Cài Go, Node, MySQL, Nginx | 20 phút |
| [D](#d--cơ-sở-dữ-liệu) | Tạo database, user, kiểm tra bảng mã | 10 phút |
| [E](#e--amazon-s3) | Tạo bucket, IAM user, bucket policy | 20 phút |
| [F](#f--google-oauth) | Tạo OAuth client cho đăng nhập admin | 10 phút |
| [G](#g--lấy-mã-nguồn-và-cấu-hình) | Clone repo, viết `.env` | 15 phút |
| [H](#h--deploy-lần-đầu) | Chạy `deploy.sh`, bật HTTPS | 15 phút |
| [I](#i--nghiệm-thu) | Kiểm tra từng chức năng thật sự chạy | 15 phút |
| [J](#j--việc-phải-làm-ngay-sau-khi-chạy-được) | Backup, logrotate, giám sát | 20 phút |

---

## Quy ước đọc

Trong tài liệu này:

- Dòng bắt đầu bằng `$` là lệnh chạy trên **máy của bạn** (Windows/Mac).
- Dòng bắt đầu bằng `#` trong khối lệnh là **chú thích**, không phải lệnh.
- `<trong-ngoặc-nhọn>` là chỗ bạn phải thay bằng giá trị thật.
- Khối **"Đúng thì thấy"** mô tả kết quả mong đợi — nếu khác, xem khối **"Sai thì sửa"**.

⚠️ **Đừng copy cả khối lệnh dài rồi dán một lần.** Chạy từng lệnh, đọc kết quả, rồi mới
sang lệnh tiếp theo. Lỗi ở bước 2 mà phát hiện ở bước 9 thì rất khó lần ngược.

---

## A — Chuẩn bị trước khi chạm vào server

### A1. Mua VPS

Cấu hình tối thiểu cho hệ thống này:

| Thành phần | Tối thiểu | Khuyến nghị | Vì sao |
|---|---|---|---|
| RAM | 2 GB | **4 GB** | Sinh biến thể ảnh giải nén ảnh ra bitmap trong RAM. Ảnh 6000×4000 chiếm ~96MB, nhân số CPU chạy song song. 2GB sẽ bị OOM khi nhiều người upload cùng lúc. |
| CPU | 2 nhân | **2–4 nhân** | Encode WebP/JPEG chạy song song theo số nhân |
| Ổ đĩa | 40 GB | **60 GB** | Ảnh gốc nằm trên S3, nhưng VPS vẫn cần chỗ cho MySQL, log, và file tạm lúc upload |
| Băng thông | 2 TB/tháng | | Trang public phục vụ ảnh từ S3 nên VPS chủ yếu tải HTML/JSON |
| Hệ điều hành | **Ubuntu 22.04 LTS** hoặc 24.04 LTS | | Toàn bộ lệnh dưới đây viết cho Ubuntu. Debian 12 cũng chạy được, đổi vài tên gói. |

Chọn vùng đặt máy **gần người dùng**: hệ thống phục vụ trường học ở Việt Nam, nên chọn
Singapore (`ap-southeast-1`) — cùng vùng với bucket S3 để ảnh đi đường ngắn nhất.

Sau khi tạo xong, nhà cung cấp cho bạn:

- **IP công cộng** (ví dụ `103.x.x.x`) — ghi lại.
- **Mật khẩu root** hoặc **SSH key** — ghi lại, sẽ dùng ở bước B1.

### A2. Trỏ DNS — làm sớm nhất có thể

DNS cần thời gian lan truyền (vài phút đến vài giờ), và bước xin chứng chỉ HTTPS ở giai
đoạn H **bắt buộc** phải có DNS đúng. Nên làm ngay bây giờ.

Vào trang quản trị tên miền `vaschools.edu.vn`, thêm bản ghi:

| Loại | Tên | Giá trị | TTL |
|---|---|---|---|
| `A` | `pictures` | `<IP-VPS-của-bạn>` | 300 |

Kiểm tra từ máy của bạn:

```bash
$ nslookup pictures.vaschools.edu.vn
```

**Đúng thì thấy** dòng `Address:` chứa đúng IP VPS.

**Sai thì sửa**: chưa lan truyền xong — chờ 10 phút rồi thử lại. Nếu sau 1 giờ vẫn sai,
kiểm tra lại bản ghi đã lưu chưa và tên có bị thừa đuôi domain không (một số nhà cung cấp
yêu cầu gõ `pictures`, số khác yêu cầu `pictures.vaschools.edu.vn`).

### A3. Gom đủ thông tin trước khi bắt đầu

Mở một file tạm, điền dần bảng này. Thiếu bất kỳ dòng nào cũng sẽ phải dừng giữa chừng:

```text
IP VPS                    : ___________________
Tên miền                  : pictures.vaschools.edu.vn
Mật khẩu root VPS         : ___________________
Tài khoản AWS (có quyền IAM + S3)? : có / không
Tài khoản Google Cloud (tạo được OAuth client)? : có / không
Danh sách email ban tổ chức được vào admin : ___________________
```

⚠️ Hai dòng AWS và Google cần **quyền quản trị** trên hai nền tảng đó. Nếu bạn không có,
xin trước — đây là chỗ hay kẹt nhất và không tự giải quyết được.

---

## B — Đăng nhập lần đầu và làm cứng server

### B1. Đăng nhập

Từ máy Windows, mở PowerShell (hoặc Terminal trên Mac):

```bash
$ ssh root@<IP-VPS>
```

Lần đầu sẽ hỏi `Are you sure you want to continue connecting?` — gõ `yes`.

**Đúng thì thấy** dấu nhắc đổi thành `root@hostname:~#`.

**Sai thì sửa**:

| Lỗi | Nguyên nhân | Cách sửa |
|---|---|---|
| `Connection refused` | SSH chưa chạy, hoặc cổng khác 22 | Xem bảng điều khiển VPS, một số nhà cung cấp dùng cổng khác: `ssh -p <cổng> root@<IP>` |
| `Connection timed out` | Firewall của nhà cung cấp chặn | Mở cổng 22 trong bảng điều khiển VPS |
| `Permission denied` | Sai mật khẩu, hoặc VPS chỉ nhận SSH key | Dùng `ssh -i <đường-dẫn-key> root@<IP>` |

### B2. Cập nhật hệ thống

```bash
apt update && apt upgrade -y
```

Mất 2–5 phút. Nếu hiện màn hình xanh hỏi về file cấu hình, chọn **"keep the local
version currently installed"**.

Nếu cuối cùng có dòng `*** System restart required ***`:

```bash
reboot
```

Chờ 30 giây rồi `ssh` vào lại.

### B3. Tạo tài khoản quản trị riêng

Đăng nhập thẳng bằng `root` là thói quen xấu: mọi lệnh gõ nhầm đều có toàn quyền phá hệ
thống, và log không phân biệt được ai làm gì.

```bash
# Thay <tên-bạn> bằng tên không dấu, viết thường, ví dụ "khoana"
adduser <tên-bạn>
```

Lệnh hỏi mật khẩu (gõ 2 lần) rồi hỏi họ tên, phòng ban... — cứ Enter bỏ qua hết.

Cho tài khoản này quyền `sudo`:

```bash
usermod -aG sudo <tên-bạn>
```

**Kiểm tra ngay, đừng đóng cửa sổ hiện tại.** Mở **cửa sổ terminal thứ hai** và thử:

```bash
$ ssh <tên-bạn>@<IP-VPS>
# sau khi vào được:
sudo whoami
```

**Đúng thì thấy** chữ `root` in ra sau khi nhập mật khẩu.

⚠️ Chỉ khi cửa sổ thứ hai vào được mới đóng cửa sổ `root`. Nếu tự khoá mình ra ngoài,
cách duy nhất còn lại là console cứu hộ của nhà cung cấp.

### B4. Tạo tài khoản chạy ứng dụng

Khác với tài khoản ở B3 (dành cho **người**), tài khoản này dành cho **tiến trình**. Nó cố
ý không có shell và không có mật khẩu: nếu ứng dụng bị khai thác, kẻ tấn công không có sẵn
một shell để dùng.

```bash
sudo useradd --system --no-create-home --shell /usr/sbin/nologin appuser
```

Kiểm tra:

```bash
id appuser
```

**Đúng thì thấy** `uid=... (appuser) gid=... (appuser) groups=...`.

> Tên `appuser` được ghi cứng trong `deploy/systemd/s3-upload-tool.service`. Muốn đổi tên
> thì phải sửa cả file đó.

### B5. Bật firewall

Mặc định VPS mở toàn bộ cổng ra Internet. Chỉ nên mở đúng ba cổng cần thiết.

```bash
sudo ufw allow OpenSSH     # cổng 22 — KHÔNG được quên, quên là tự khoá mình ra ngoài
sudo ufw allow 80/tcp      # HTTP — Certbot cần để xác thực domain
sudo ufw allow 443/tcp     # HTTPS
sudo ufw enable            # hỏi xác nhận, gõ y
```

⚠️ **Lệnh `allow OpenSSH` phải chạy TRƯỚC `enable`.** Bật firewall khi chưa mở SSH sẽ ngắt
kết nối của chính bạn ngay lập tức.

```bash
sudo ufw status
```

**Đúng thì thấy**:

```text
Status: active
To                         Action      From
--                         ------      ----
OpenSSH                    ALLOW       Anywhere
80/tcp                     ALLOW       Anywhere
443/tcp                    ALLOW       Anywhere
```

Lưu ý MySQL (cổng 3306) **không** có trong danh sách — đúng như vậy. Ứng dụng và MySQL
chạy trên cùng máy, nói chuyện qua `localhost`, không cần mở ra ngoài.

### B6. Đổi múi giờ

Log ghi theo giờ hệ thống. Để giờ UTC thì mỗi lần đọc log lại phải cộng trừ 7 tiếng.

```bash
sudo timedatectl set-timezone Asia/Ho_Chi_Minh
date
```

**Đúng thì thấy** giờ hiện tại của Việt Nam, kèm `+07`.

---

## C — Cài phần mềm nền

### C1. Các gói cơ bản

```bash
sudo apt install -y git curl wget nginx mysql-server ufw
```

### C2. Go 1.24+

Kho `apt` của Ubuntu thường có Go cũ hơn yêu cầu của dự án, nên cài từ trang chính thức:

```bash
cd /tmp
wget https://go.dev/dl/go1.24.2.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.24.2.linux-amd64.tar.gz
```

Thêm vào `PATH` cho **mọi** người dùng — quan trọng, vì `deploy.sh` chạy dưới `sudo`:

```bash
echo 'export PATH=$PATH:/usr/local/go/bin' | sudo tee /etc/profile.d/go.sh
sudo chmod +x /etc/profile.d/go.sh
source /etc/profile.d/go.sh
```

```bash
go version
sudo env "PATH=$PATH" go version    # kiểm tra cả khi chạy qua sudo
```

**Đúng thì thấy** `go version go1.24.2 linux/amd64` ở **cả hai** lệnh.

**Sai thì sửa**: nếu lệnh thứ hai báo `command not found`, `deploy.sh` sẽ hỏng ở bước
build. Thêm đường dẫn tuyệt đối vào `secure_path`:

```bash
sudo visudo
# tìm dòng bắt đầu bằng: Defaults secure_path=
# thêm :/usr/local/go/bin vào cuối chuỗi, trước dấu nháy đóng
```

### C3. Node 20+

```bash
curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
sudo apt install -y nodejs
node -v && npm -v
```

**Đúng thì thấy** `v20.x.x` và một số phiên bản npm.

> Node chỉ dùng để **build** giao diện thành file tĩnh. Lúc chạy thật không có tiến trình
> Node nào cả — Go phục vụ luôn thư mục `web/dist`.

### C4. Kiểm tra Nginx

```bash
nginx -v
sudo systemctl status nginx
```

**Đúng thì thấy** `active (running)`. Mở trình duyệt vào `http://<IP-VPS>` sẽ thấy trang
"Welcome to nginx!". Trang này sẽ được thay ở giai đoạn H.

---

## D — Cơ sở dữ liệu

### D1. Làm cứng MySQL

```bash
sudo mysql_secure_installation
```

Trả lời theo thứ tự:

| Câu hỏi | Trả lời | Vì sao |
|---|---|---|
| Setup VALIDATE PASSWORD component? | `n` | Bật lên sẽ chặn cả mật khẩu ngẫu nhiên dài nếu thiếu ký tự đặc biệt, gây rối hơn là lợi |
| Set root password? | `y`, rồi nhập mật khẩu mạnh | Ghi lại vào nơi an toàn |
| Remove anonymous users? | `y` | |
| Disallow root login remotely? | `y` | root chỉ nên vào từ chính máy đó |
| Remove test database? | `y` | |
| Reload privilege tables now? | `y` | |

### D2. Tạo database và tài khoản riêng

Không dùng `root` cho ứng dụng: nếu lộ thông tin kết nối thì thiệt hại giới hạn trong đúng
một database.

```bash
sudo mysql
```

Trong dấu nhắc `mysql>`:

```sql
CREATE DATABASE va_stu_pic_db_prd
  CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

CREATE USER 'vasapp'@'localhost' IDENTIFIED BY '<mật-khẩu-mạnh>';
GRANT ALL PRIVILEGES ON va_stu_pic_db_prd.* TO 'vasapp'@'localhost';
FLUSH PRIVILEGES;
EXIT;
```

⚠️ **`utf8mb4` là bắt buộc, không phải tuỳ chọn.** Tên học sinh có dấu tiếng Việt và bình
luận có emoji. Bảng mã `utf8` của MySQL chỉ dùng 3 byte, không chứa nổi emoji — dữ liệu sẽ
bị cắt cụt hoặc lỗi khi ghi, và sửa sau thì phải chuyển đổi toàn bộ bảng.

⚠️ **Sinh mật khẩu ngẫu nhiên, đừng tự nghĩ**: `openssl rand -base64 24`. Nhưng **tránh ký
tự `@`** trong mật khẩu — chuỗi `DATABASE_URL` dùng `@` làm dấu phân cách nên mật khẩu chứa
`@` sẽ làm hỏng cú pháp. Sinh lại nếu gặp.

### D3. Kiểm tra kết nối

Đây là bước hay bị bỏ qua, và bỏ qua thì lỗi chỉ lộ ra lúc ứng dụng khởi động — khó đoán
hơn nhiều.

```bash
mysql -u vasapp -p va_stu_pic_db_prd -e "SELECT @@character_set_database, @@collation_database;"
```

**Đúng thì thấy**:

```text
+--------------------------+----------------------+
| @@character_set_database | @@collation_database |
+--------------------------+----------------------+
| utf8mb4                  | utf8mb4_unicode_ci   |
+--------------------------+----------------------+
```

**Sai thì sửa**: nếu thấy `utf8mb3`, xoá và tạo lại database đúng như D2 — làm ngay bây giờ
khi chưa có dữ liệu thì mất 10 giây.

> Không cần chạy migration thủ công. Ứng dụng tự tạo 13 bảng khi khởi động lần đầu
> (`DATABASE_AUTO_MIGRATE=true`), idempotent qua bảng `schema_migrations`.

---

## E — Amazon S3

Ảnh dự thi nằm trên S3, không nằm trên VPS. Cần ba thứ: một bucket, một IAM user để ứng
dụng ghi vào bucket, và một bucket policy cho phép **người xem** đọc ảnh.

### E1. Tạo bucket

Vào [console S3](https://s3.console.aws.amazon.com/), **Create bucket**:

| Trường | Giá trị | Vì sao |
|---|---|---|
| Bucket name | `<tên-bucket>` | Duy nhất toàn cầu, chỉ chữ thường/số/gạch ngang |
| Region | **Asia Pacific (Singapore) `ap-southeast-1`** | Gần Việt Nam nhất; phải khớp `AWS_REGION` trong `.env` |
| Block all public access | **BỎ TÍCH** | Ảnh phải đọc được công khai — xem cảnh báo dưới |
| Bucket Versioning | Disable | Ảnh không sửa, chỉ thêm/xoá; bật lên chỉ tốn tiền lưu trữ |

⚠️ AWS sẽ bắt tích vào ô xác nhận rằng bạn hiểu bucket sẽ public. Điều này **đúng và cần
thiết** ở đây: đây là triển lãm tranh công khai, ảnh sinh ra để mọi người xem. Nhưng nó
cũng có nghĩa **đừng bao giờ để dữ liệu riêng tư vào bucket này**.

### E2. Bucket policy cho quyền đọc công khai

Tab **Permissions** → **Bucket policy** → **Edit**, dán vào (thay `<tên-bucket>`):

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "PublicReadForArtworkImages",
      "Effect": "Allow",
      "Principal": "*",
      "Action": "s3:GetObject",
      "Resource": "arn:aws:s3:::<tên-bucket>/*"
    }
  ]
}
```

Chỉ cho phép `s3:GetObject` (đọc một object đã biết đường dẫn) — **không** cho
`s3:ListBucket`, nên không ai liệt kê được toàn bộ nội dung bucket.

Chi tiết thêm: [S3-PUBLIC-READ.md](../S3-PUBLIC-READ.md).

### E3. Tạo IAM user cho ứng dụng

**Đừng dùng access key của tài khoản root AWS.** Key đó có toàn quyền trên mọi dịch vụ; lộ
ra là mất cả tài khoản.

1. Vào [IAM](https://console.aws.amazon.com/iam/) → **Users** → **Create user**.
2. Tên: `vas-pictures-app`. **Không** tích "Provide user access to the AWS Management
   Console" — user này chỉ dùng qua API.
3. **Set permissions** → **Attach policies directly** → **Create policy** → tab **JSON**:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "AppBucketAccess",
      "Effect": "Allow",
      "Action": [
        "s3:PutObject",
        "s3:GetObject",
        "s3:DeleteObject",
        "s3:AbortMultipartUpload"
      ],
      "Resource": "arn:aws:s3:::<tên-bucket>/*"
    },
    {
      "Sid": "AppBucketLocation",
      "Effect": "Allow",
      "Action": ["s3:GetBucketLocation", "s3:ListBucket"],
      "Resource": "arn:aws:s3:::<tên-bucket>"
    }
  ]
}
```

Quyền vừa đủ, không thừa: `PutObject` để upload, `DeleteObject` để xoá tác phẩm,
`AbortMultipartUpload` để dọn phần upload dở, `GetBucketLocation` cho cơ chế tự dò region.

4. Đặt tên policy `vas-pictures-s3`, tạo xong thì gán cho user.
5. Vào user vừa tạo → tab **Security credentials** → **Create access key** → chọn
   **Application running outside AWS** → **Create**.

⚠️ **Secret access key chỉ hiện đúng một lần.** Copy cả hai giá trị ngay:

```text
AWS_ACCESS_KEY_ID     = AKIA...
AWS_SECRET_ACCESS_KEY = ...
```

Mất thì không xem lại được, phải tạo key mới và xoá key cũ.

> **Nếu VPS chạy trên EC2**: bỏ qua bước tạo access key. Gán IAM role thẳng cho instance và
> để trống hai biến trên trong `.env` — SDK tự lấy thông tin đăng nhập từ metadata của
> instance. An toàn hơn vì không có key nào nằm trên đĩa.

### E4. Kiểm tra quyền thật sự hoạt động

Trên VPS:

```bash
sudo apt install -y awscli
aws configure     # dán access key, secret key, region ap-southeast-1, output để trống

echo "kiem tra" > /tmp/test.txt
aws s3 cp /tmp/test.txt s3://<tên-bucket>/test.txt
curl -s -o /dev/null -w "%{http_code}\n" https://<tên-bucket>.s3.ap-southeast-1.amazonaws.com/test.txt
aws s3 rm s3://<tên-bucket>/test.txt
```

**Đúng thì thấy** lệnh `cp` báo `upload: ...`, lệnh `curl` in ra `200`, lệnh `rm` báo
`delete: ...`.

**Sai thì sửa**:

| Kết quả | Nguyên nhân | Cách sửa |
|---|---|---|
| `cp` báo `AccessDenied` | IAM policy sai bucket name, hoặc chưa gán policy cho user | Xem lại E3 |
| `curl` in `403` | Bucket policy ở E2 chưa đúng, hoặc "Block all public access" vẫn bật | Xem lại E1, E2 |
| `curl` in `301` | Sai region trong URL | Bucket nằm ở region khác — kiểm tra lại và sửa `AWS_REGION` |

⚠️ Sau khi kiểm tra xong, **xoá thông tin đăng nhập khỏi máy**: `rm -rf ~/.aws`. Ứng dụng
đọc key từ `.env`, không cần file này, và để lại chỉ là thêm một chỗ có thể lộ key.

---

## F — Google OAuth

Admin đăng nhập bằng tài khoản Google của trường, không có mật khẩu riêng.

1. Vào [Google Cloud Console](https://console.cloud.google.com/) → tạo project mới, ví dụ
   `vas-pictures`.
2. **APIs & Services** → **OAuth consent screen**:
   - User Type: **Internal** nếu `vaschools.edu.vn` là Google Workspace (khuyến nghị —
     người ngoài tổ chức không đăng nhập được, thêm một tầng bảo vệ). Nếu không thì
     **External**.
   - App name: `VASchools Art Gallery`, support email: email của bạn.
3. **Credentials** → **Create Credentials** → **OAuth client ID**:
   - Application type: **Web application**
   - Authorized JavaScript origins: `https://pictures.vaschools.edu.vn`
   - Authorized redirect URIs: `https://pictures.vaschools.edu.vn/auth/google/callback`

⚠️ **Redirect URI phải trùng từng ký tự** với `GOOGLE_REDIRECT_URL` trong `.env`: đúng
`https` (không phải `http`), đúng tên miền, **không** có dấu `/` ở cuối. Lệch một ký tự thì
Google trả lỗi `redirect_uri_mismatch` và đăng nhập không bao giờ thành công. Đây là lỗi
phổ biến nhất ở bước này.

Copy **Client ID** và **Client secret**.

---

## G — Lấy mã nguồn và cấu hình

### G1. Clone

```bash
sudo mkdir -p /opt/s3-upload-tool
sudo chown $USER:$USER /opt/s3-upload-tool
git clone <repo-url> /opt/s3-upload-tool
cd /opt/s3-upload-tool
```

### G2. Sinh sẵn các chuỗi bí mật

```bash
echo "API_KEY        = $(openssl rand -hex 32)"
echo "SESSION_SECRET = $(openssl rand -hex 32)"
```

Copy hai dòng kết quả ra chỗ tạm — sẽ dán vào `.env` ngay sau đây.

### G3. Viết `.env`

```bash
cp .env.example .env
nano .env
```

Các giá trị **bắt buộc** cho production (thiếu hoặc sai thì server **chủ động từ chối khởi
động** — đây là thiết kế có chủ đích, để một cấu hình không an toàn không thể lặng lẽ chạy):

```env
APP_ENV=production
PORT=8080

# --- Bảo mật API ---
API_REQUIRE_KEY=true
API_KEY=<chuỗi 64 ký tự vừa sinh ở G2>
CORS_ORIGINS=https://pictures.vaschools.edu.vn

# --- S3 ---
AWS_REGION=ap-southeast-1
S3_BUCKET_NAME=<tên-bucket>
AWS_ACCESS_KEY_ID=<từ bước E3>
AWS_SECRET_ACCESS_KEY=<từ bước E3>
S3_USE_PRESIGNED_URL=false

# --- Cơ sở dữ liệu ---
DATABASE_ENABLED=true
DATABASE_URL=vasapp:<mật-khẩu-D2>@tcp(localhost:3306)/va_stu_pic_db_prd
DATABASE_AUTO_MIGRATE=true

# --- Phiên đăng nhập ---
SESSION_SECRET=<chuỗi 64 ký tự vừa sinh ở G2>
CSRF_SECURE_COOKIE=false

# --- Đăng nhập admin ---
GOOGLE_CLIENT_ID=<từ bước F>
GOOGLE_CLIENT_SECRET=<từ bước F>
GOOGLE_REDIRECT_URL=https://pictures.vaschools.edu.vn/auth/google/callback
ADMIN_ALLOWED_EMAIL_DOMAIN=vaschools.edu.vn
ADMIN_ALLOWED_EMAILS=<email1>,<email2>
```

Vài điểm dễ sai:

- **`CORS_ORIGINS` không được để `*`** khi `APP_ENV=production`. Server sẽ từ chối khởi
  động — cố ý, vì `*` cho phép mọi website gọi API này.
- **`CSRF_SECURE_COOKIE=false` lúc này là đúng.** Chưa có HTTPS mà bật `true` thì trình
  duyệt không gửi cookie qua HTTP và bạn không đăng nhập được. Script `setup-https.sh` ở
  giai đoạn H sẽ tự đổi thành `true`.
- **`S3_USE_PRESIGNED_URL=false`**: presigned URL có hạn dùng và sẽ hết hạn; ảnh triển lãm
  phải sống lâu dài.
- **`DATABASE_URL` không cần tham số phía sau.** Ứng dụng tự bổ sung
  `parseTime=true`, `charset=utf8mb4` và `multiStatements=true` nếu thiếu, nên chỉ cần ghi
  đúng `user:pass@tcp(host:port)/dbname`.

### G4. Khoá quyền file `.env`

File này chứa toàn bộ bí mật của hệ thống. Mặc định `cp` tạo file mà mọi người dùng trên
máy đều đọc được.

```bash
sudo chown appuser:appuser .env
sudo chmod 600 .env
ls -l .env
```

**Đúng thì thấy** `-rw------- 1 appuser appuser`.

---

## H — Deploy lần đầu

### H1. Preflight

Script này **chỉ đọc, không sửa gì** — nó kiểm tra mọi điều kiện trước khi bắt tay vào
việc, để lỗi lộ ra sớm thay vì lộ ra giữa chừng.

```bash
cd /opt/s3-upload-tool
sudo bash deploy/preflight.sh
```

**Đúng thì thấy** toàn dấu `✓`. Sửa hết mọi dòng `✗` trước khi đi tiếp — đừng bỏ qua dòng
nào với ý nghĩ "chắc không sao".

### H2. Deploy

```bash
sudo bash deploy/deploy.sh
```

Script làm tuần tự: `git pull` → build giao diện → build binary Go → chạy migration → cài
và khởi động service → cài vhost Nginx → kiểm tra sức khoẻ.

Mất 3–8 phút, lâu nhất là `npm install` lần đầu.

**Đúng thì thấy** dòng cuối báo deploy thành công và health check `200`.

**Sai thì sửa**: script **tự khôi phục binary cũ** nếu bản mới không qua health check, nên
site không chết. Đọc log để biết nguyên nhân:

```bash
sudo journalctl -u s3-upload-tool -n 50 --no-pager
```

| Thông báo trong log | Nguyên nhân | Cách sửa |
|---|---|---|
| `API_KEY is required` | `.env` thiếu `API_KEY` khi `API_REQUIRE_KEY=true` | Xem G3 |
| `CORS_ORIGINS cannot be "*"` | Để `*` ở production | Đặt đúng tên miền |
| `Access denied for user` | Sai thông tin `DATABASE_URL` | Thử lại lệnh ở D3 |
| `dial tcp ... connect: connection refused` | MySQL chưa chạy | `sudo systemctl start mysql` |
| `no such host` khi gọi S3 | Sai `AWS_REGION` hoặc tên bucket | Xem E4 |

### H3. Kiểm tra trước khi bật HTTPS

```bash
curl http://127.0.0.1:8080/api/v1/health
curl -H "Host: pictures.vaschools.edu.vn" http://127.0.0.1/api/v1/health
```

Lệnh đầu kiểm tra **ứng dụng**, lệnh sau kiểm tra **Nginx đã chuyển tiếp đúng**. Cả hai
phải trả JSON có `"status":"ok"`. Nếu lệnh đầu chạy mà lệnh sau không, vấn đề nằm ở Nginx
chứ không phải ứng dụng.

### H4. Bật HTTPS

DNS phải trỏ đúng rồi (bước A2) — kiểm tra lại lần cuối:

```bash
getent hosts pictures.vaschools.edu.vn
```

Phải ra đúng IP VPS. Rồi:

```bash
sudo bash deploy/setup-https.sh
```

Script xin chứng chỉ Let's Encrypt, sửa vhost thêm block 443 và chuyển hướng HTTP→HTTPS, và
đổi `CSRF_SECURE_COOKIE=true` trong `.env` rồi khởi động lại service.

**Sai thì sửa**:

| Lỗi Certbot | Nguyên nhân | Cách sửa |
|---|---|---|
| `DNS problem: NXDOMAIN` | DNS chưa lan truyền | Chờ thêm, kiểm tra bằng `getent hosts` |
| `Timeout during connect` | Cổng 80 bị chặn | `sudo ufw allow 80`; kiểm tra cả firewall của nhà cung cấp |
| `Too many failed authorizations` | Thử quá nhiều lần | Let's Encrypt giới hạn 5 lần/giờ — **chờ 1 tiếng**, sửa DNS cho chắc rồi mới thử lại |

Chứng chỉ tự gia hạn qua timer của Certbot. Kiểm tra:

```bash
sudo systemctl list-timers | grep certbot
sudo certbot renew --dry-run
```

---

## I — Nghiệm thu

Đừng dừng ở "trang chủ mở được". Chạy đủ danh sách này:

```bash
# 1. HTTPS hoạt động và HTTP tự chuyển hướng
curl -sI http://pictures.vaschools.edu.vn | head -1     # mong đợi 301
curl -s https://pictures.vaschools.edu.vn/api/v1/health # mong đợi {"status":"ok"...}

# 2. API public KHÔNG cần API key (đây là chỗ hay hỏng nhất ở production)
curl -s -o /dev/null -w "%{http_code}\n" \
  https://pictures.vaschools.edu.vn/api/v1/public/artworks    # mong đợi 200

# 3. API quản trị VẪN chặn khi thiếu key
curl -s -o /dev/null -w "%{http_code}\n" \
  https://pictures.vaschools.edu.vn/api/v1/upload             # mong đợi 401

# 4. Chứng chỉ hợp lệ và còn hạn
echo | openssl s_client -connect pictures.vaschools.edu.vn:443 2>/dev/null \
  | openssl x509 -noout -dates
```

⚠️ **Mục 2 là bẫy đã biết.** Middleware API key áp lên toàn bộ `/api/*` và hiện chỉ miễn
`/api/v1/health`. Nếu trả `401`, khách ẩn danh không xem được trang public — xem mục **P0.1**
trong [../plan/02-roadmap.md](../plan/02-roadmap.md) để biết cách sửa.

Sau đó kiểm tra bằng trình duyệt, **dùng cửa sổ ẩn danh** để không bị đánh lừa bởi cache và
phiên đăng nhập sẵn có:

- [ ] Trang chủ hiển thị, ảnh tải được
- [ ] Vào phòng triển lãm, lưới tranh hiện đủ
- [ ] Mở một tranh, thả cảm xúc, viết bình luận
- [ ] Dán link chia sẻ một tác phẩm vào Zalo/Messenger — phải hiện ảnh preview
- [ ] Vào `/admin`, đăng nhập bằng Google, thấy bảng điều khiển
- [ ] Upload thử **một ảnh thật** (không phải ảnh test 10KB) qua giao diện admin
- [ ] Ảnh vừa upload hiện đúng ở trang public

Kiểm tra biến thể ảnh đã sinh đúng — nếu chưa, gallery vẫn chạy nhưng nặng gấp nhiều lần:

```bash
mysql -u vasapp -p va_stu_pic_db_prd \
  -e "SELECT id, title, thumbnail_url IS NOT NULL AS co_thumb, \
      JSON_LENGTH(variants) AS so_bien_the FROM artworks ORDER BY id DESC LIMIT 5;"
```

**Đúng thì thấy** `co_thumb = 1` và `so_bien_the = 6` (3 cỡ × 2 định dạng) với ảnh đủ lớn.
Ảnh gốc nhỏ hơn 400px sẽ có ít biến thể hơn — đó là hành vi đúng, không phải lỗi.

---

## J — Việc phải làm ngay sau khi chạy được

Hệ thống đã chạy, nhưng **chưa an toàn để bỏ đấy**. Ba việc dưới đây làm ngay trong hôm nay.

### J1. Xoay vòng log ⚠️ quan trọng nhất

Ứng dụng ghi mỗi ngày một file log và **không bao giờ tự xoá**. Đĩa đầy sẽ làm sập cả MySQL
lẫn ứng dụng — và sập theo kiểu khó chẩn đoán.

```bash
sudo nano /etc/logrotate.d/s3-upload-tool
```

```text
/opt/s3-upload-tool/storage/logs/*.log {
    weekly
    rotate 8
    compress
    delaycompress
    missingok
    notifempty
    create 0640 appuser appuser
}
```

```bash
sudo logrotate -d /etc/logrotate.d/s3-upload-tool    # -d = chạy thử, không sửa gì
```

**Đúng thì thấy** output mô tả các file sẽ xoay, không có dòng `error`.

### J2. Sao lưu cơ sở dữ liệu tự động

Ảnh nằm trên S3 nên tương đối an toàn. Nhưng **toàn bộ metadata** — tên học sinh, giải
thưởng, bình luận — chỉ nằm trong MySQL trên đúng cái VPS này.

```bash
sudo mkdir -p /opt/s3-upload-tool/storage/backups
sudo nano /usr/local/bin/backup-vas-pictures.sh
```

```bash
#!/bin/bash
set -euo pipefail
BACKUP_DIR=/opt/s3-upload-tool/storage/backups
mysqldump -u vasapp -p'<mật-khẩu>' va_stu_pic_db_prd \
  | gzip > "$BACKUP_DIR/db-$(date +%F).sql.gz"
# Giữ 14 ngày gần nhất
find "$BACKUP_DIR" -name 'db-*.sql.gz' -mtime +14 -delete
```

```bash
sudo chmod 700 /usr/local/bin/backup-vas-pictures.sh   # 700 vì file chứa mật khẩu
sudo bash /usr/local/bin/backup-vas-pictures.sh        # chạy thử ngay
ls -lh /opt/s3-upload-tool/storage/backups/
```

Đặt lịch chạy 2 giờ sáng hằng ngày:

```bash
sudo crontab -e
# thêm dòng:
0 2 * * * /usr/local/bin/backup-vas-pictures.sh
```

⚠️ **Bản sao lưu nằm cùng máy với dữ liệu gốc thì không phải là sao lưu.** Mất VPS là mất
cả hai. Đẩy định kỳ lên S3 (bucket **khác**, không public) hoặc tải về máy khác.

⚠️ **Sao lưu chưa từng phục hồi thử thì chưa chắc dùng được.** Thử một lần trên database
tạm để biết quy trình thật sự chạy.

### J3. Theo dõi tối thiểu

```bash
df -h /                                    # đĩa còn trống — theo dõi hằng tuần
free -h                                    # RAM
sudo systemctl status s3-upload-tool       # service còn sống
sudo bash deploy/status.sh                 # tổng quan mọi thứ
```

Đặt lịch nhắc bản thân xem `deploy/status.sh` mỗi tuần, và **đặc biệt là ngày trước khi
công bố kết quả** — đó là lúc lượng truy cập cao nhất.

---

## Tra cứu nhanh

```bash
# Cập nhật code
cd /opt/s3-upload-tool && sudo bash deploy/deploy.sh
sudo bash deploy/deploy.sh --skip-web    # chỉ sửa Go, nhanh hơn nhiều
sudo bash deploy/deploy.sh --no-pull     # deploy code đang có, không git pull

# Vận hành
sudo bash deploy/status.sh
sudo systemctl restart s3-upload-tool
sudo journalctl -u s3-upload-tool -f
sudo journalctl -u s3-upload-tool -p err -n 50

# Sau khi sửa .env — PHẢI restart, ứng dụng chỉ đọc lúc khởi động
sudo systemctl restart s3-upload-tool

# Sau khi sửa cấu hình Nginx
sudo nginx -t && sudo systemctl reload nginx
```

## Đi tiếp

| Cần gì | Đọc |
|---|---|
| Ý nghĩa từng biến trong `.env` | [02-configuration.md](./02-configuration.md) |
| Vận hành, sự cố, quay lui | [03-operations.md](./03-operations.md) |
| Quy trình rút gọn cho lần sau | [01-vps-systemd.md](./01-vps-systemd.md) |
| Hiểu hệ thống được lắp thế nào | [../ARCHITECTURE.md](../ARCHITECTURE.md) |
| Việc còn phải làm | [../plan/02-roadmap.md](../plan/02-roadmap.md) |
