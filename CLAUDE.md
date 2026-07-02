# CLAUDE.md — Dự Án Nyanpass Mô Phỏng (tfd2.clonod.top)

## Thông Tin Dự Án

- **Mục tiêu:** Mô phỏng hệ thống panel chuyển tiếp cổng (port forwarding) Nyanpass
- **Domain phát triển:** tfd2.clonod.top

## Kiến Trúc Hệ Thống Tham Khảo

```
Inbound Node (JP/SG) → [Tunnel] → Outbound Node (VN) → Đích khách hàng
```

### Thành Phần Chính

1. **Backend API:** Go + Gin + GORM + SQLite → REST API `/api/v1/`
2. **Frontend Panel:** React + TypeScript + Ant Design + Vite
3. **Probe Page:** Giám sát node real-time qua WebSocket
4. **Terminal:** SSH vào node qua xterm.js + WebSocket
5. **Database:** SQLite (19 bảng)

### Chức Năng Cốt Lõi

- **Auth:** Login/Register/Captcha, token-based
- **Device Groups:** Quản lý nhóm thiết bị/node (Inbound, Outbound, AgentOnly)
- **Forward Rules:** Ánh xạ port → đích (port forwarding)
- **Chain Outbound:** Chuỗi chuyển tiếp nhiều tầng
- **Real-time Monitor:** WebSocket node status
- **Shop/Payment:** Plans, orders, deposit, redeem codes
- **Affiliate:** Commission, invite codes
- **Admin:** Quản lý users, groups, rules, plans, orders

---

## QUY TẮC BẮT BUỘC

### 1. BẢO VỆ AN TOÀN VPS

- **TUYỆT ĐỐI KHÔNG** sửa, xoá, hay thay đổi bất kỳ file/thư mục nào của các website khác trên VPS
- Trước khi thao tác với web server (Nginx/Apache), luôn kiểm tra cấu hình hiện tại
- Không restart/shutdown dịch vụ dùng chung (nginx, mysql, php-fpm...) trừ khi được yêu cầu
- Chỉ làm việc trong thư mục của website tfd2.clonod.top
- Mọi thao tác với aaPanel chỉ thực hiện trên website mới, không chạm vào website khác

### 2. CHỈ THAM KHẢO WEB MẪU — KHÔNG SỬA ĐỔI

- **TUYỆT ĐỐI KHÔNG** sửa, xoá, bật/tắt bất kỳ cài đặt hay chức năng nào trên web mẫu
- **CHỈ ĐƯỢC DÙNG GET** để đọc dữ liệu tham khảo (API response format, UI, config)
- Web mẫu là tài nguyên tham khảo — mọi thay đổi chỉ thực hiện trên web dự án `tfd2.clonod.top`

### 3. KHÔNG XOÁ FILE

- **KHÔNG** xoá bất kỳ file/thư mục nào nếu chưa được người dùng đồng ý
- Nếu cần xoá, phải hỏi và nêu rõ lý do
- Khi cần ghi đè file, phải thông báo trước

### 4. CODE THEO TIÊU CHUẨN GSD (Gọn Gàng, Sạch Sẽ, Dễ Dàng)

- **Gọn:** Code ngắn gọn, không thừa, đúng trách nhiệm
- **Sạch:** Format chuẩn,命名 rõ ràng, comment tiếng Việt cho logic phức tạp
- **Dễ:** Cấu trúc rõ ràng, dễ đọc, dễ sửa, dễ mở rộng
- Hàm nhỏ (≤30 dòng), file hợp lý (≤300 dòng với business logic)
- Tên biến/hàm rõ nghĩa, tránh viết tắt khó hiểu
- Phân tách rõ: handlers → services → models

### 5. ĐỐI CHIẾU VỚI WEB MẪU

- Đối chiếu cả **giao diện** (UI/UX, bố cục, màu sắc) và **chức năng** (API, logic)
- Đăng nhập vào panel gốc để kiểm tra cách hoạt động trước khi code
- Request/Response API phải tương thích với format của panel gốc

### 6. TỰ VERIFY SAU MỖI NHIỆM VỤ

- Sau mỗi bước, tự kiểm tra kết quả
- Nếu là code: build + chạy thử
- Nếu là cấu hình: kiểm tra syntax + reload test
- Nếu là database: kiểm tra schema + dữ liệu mẫu
- **QUAN TRỌNG: Verify bằng screenshot màn hình**
  - Sau khi deploy frontend, dùng `WebFetch` hoặc công cụ khác để chụp/xem trang web thực tế
  - Đọc nội dung trang web để xác nhận UI hiển thị đúng
  - Nếu trang trắng hoặc lỗi, phải debug và sửa ngay
  - Không được báo "đã xong" nếu chưa xác nhận trang web hiển thị đúng
- Báo cáo kết quả verify rõ ràng

### 7. GIAO TIẾP

- Toàn bộ giao tiếp bằng **tiếng Việt**
- Chỉ dùng tiếng Anh cho: tên biến, tên hàm, package, attribute
- Báo cáo rõ ràng từng bước đã làm
- Nếu gặp vấn đề, dừng lại và thông báo ngay

### 8. KHÔNG ẢNH HƯỞNG ĐẾN CHỨC NĂNG KHÁC

- **TUYỆT ĐỐI KHÔNG** làm ảnh hưởng đến các chức năng khác khi sửa code
- Khi sửa bất kỳ chức năng nào, phải đảm bảo:
  - **Hoạt động:** Các trang/tab khác vẫn chạy bình thường, không bị trắng, không bị lỗi JS
  - **Giao diện:** Không làm vỡ layout, menu, sidebar của các trang khác
  - **Code:** Không sửa/xoá code của chức năng khác, chỉ thêm/sửa đúng phạm vi được yêu cầu
- Trước khi sửa: đọc kỹ file hiện tại, hiểu rõ cấu trúc, xác định chính xác vị trí cần sửa
- Sau khi sửa: verify không chỉ chức năng đang làm mà còn kiểm tra các chức năng khác vẫn OK
- Ưu tiên dùng `Edit` (thay thế chính xác) thay vì ghi đè cả file
- Nếu cần rebuild file lớn: bắt đầu từ bản backup stable gần nhất, thêm từng tính năng một cách chính xác

---

## Cấu Trúc Dự Án

```
/Volumes/PS4/AI CODING/nyanpass/
├── CLAUDE.md                   # File này
├── RULES.md                    # Quy tắc chi tiết
├── ANALYSIS.md                 # Phân tích kỹ thuật hệ thống gốc
├── EXPLORATION_REPORT.md       # Báo cáo khám phá web mẫu
├── PROJECT_PLAN.md             # Kế hoạch phát triển
├── schema.sql                  # Database schema gốc
├── vps_deploy.html             # Base sạch frontend panel (52KB)
├── vps_deploy_20250701_stable.html  # Bản stable gần nhất (102KB)
├── index.html                  # File deploy lên VPS
├── backend/                    # Go backend
│   ├── cmd/server/main.go      # Entry point + routes
│   └── internal/
│       ├── handlers/client.go  # Config_v2 API (Inbound + Outbound)
│       ├── handlers/node.go    # Node report, status, kick
│       ├── handlers/device_group.go # CRUD device groups
│       ├── handlers/forward_rule.go # CRUD forward rules
│       ├── handlers/ws.go      # WebSocket hub
│       ├── models/             # Database models
│       ├── database/           # SQLite init + migration
│       └── services/           # Business logic
├── nodeclient/
│   ├── main.go                 # Node Client Tunnel v2 (~39KB)
│   ├── go.mod
│   ├── rel_nodeclient_linux_amd64   # Binary AMD64
│   └── rel_nodeclient_linux_arm64   # Binary ARM64
└── backups/                    # Backups theo timestamp
```

## Công Nghệ (Thực Tế Đang Dùng)

| Lớp            | Công nghệ                                                       |
| -------------- | --------------------------------------------------------------- |
| Backend        | Go + Gin + GORM + SQLite (pure-Go `github.com/glebarez/sqlite`) |
| Frontend Panel | **Vanilla HTML/JS** (1 file ~57KB, không React build)           |
| Frontend Probe | **Vanilla HTML/JS** (1 file ~13KB, WebSocket real-time)         |
| Database       | SQLite (pure-Go, không CGO)                                     |
| Real-time      | **WebSocket** (Gorilla WebSocket, broadcast 1s/lần)             |
| Node Client    | Go binary, report mỗi 2s qua HTTP POST                          |
| Reverse Proxy  | Nginx (port 80/443 → backend :18889, WebSocket upgrade)         |
| Install Script | Bash, giống bản gốc 100% (prompts, BBR optimization, 12 tools)  |

> **Lưu ý:** Frontend là vanilla JS, không build step. Panel chính: `vps_deploy.html` (~57KB).
> Probe page: `probe.html` (~13KB). Sửa trực tiếp file rồi scp lên VPS.

## Quy Tắc Kỹ Thuật Quan Trọng

### A. KHI SỬA HTML CHỨA JAVASCRIPT

- **LUÔN kiểm tra quotes balance** sau mỗi lần sửa
- Nếu single/double quote lẻ → syntax error → TOÀN BỘ TRANG HỎNG
- **Dùng Python script file riêng** → scp lên VPS → chạy. KHÔNG dùng sed/ssh heredoc
- File nguồn: `vps_deploy.html` (panel), `probe.html` (probe)

### B. KHI DEPLOY LÊN VPS

- `chmod 644` + `chown www-data:www-data` cho file HTML
- `chmod -R 755` cho thư mục
- Xóa `._*` (Apple Double files): `find . -name '._*' -delete`
- **QUAN TRỌNG: Verify MD5 hash sau khi deploy binary** — đã gặp lỗi binary cũ không được ghi đè

### C. KHI BUILD GO

- Backend: `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ./cmd/server/`
- **QUAN TRỌNG**: Binary build ra tên `server`, nhưng trên VPS phải deploy thành `rel_backend` (tên trong `ExecStart`)
- Node client: `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build .`
- Binary node client cần build 3 arch: amd64, amd64v3, arm64
- Dùng `github.com/glebarez/sqlite` (pure-Go, không CGO)

### C2. QUY TRÌNH DEPLOY BINARY LÊN VPS

```bash
# Backend (Panel VPS)

# Node Client (Inbound/Outbound VPS)
```

> **Tuyệt đối không** dùng `mv` để ghi đè binary khi service đang chạy — systemd `Restart=always` có thể tự restart trước khi kịp copy.

### D. KIỂM TRA NGHIÊM NGẶT TRƯỚC KHI BÁO CÁO

1. **Verify API data**: so sánh số liệu panel với VPS thực tế (top, free, /proc/net/dev)
2. **Verify binary hash**: `md5sum` trên VPS phải khớp với binary local
3. **Verify quotes**: Python script kiểm tra quote balance
4. **Verify real-time**: lấy 3 mẫu cách 3s, xác nhận số liệu thay đổi
5. **KHÔNG báo "đã xong" nếu chưa qua các bước trên**

## Thông Tin Kết Nối VPS

| Mục             | Giá trị                                       |
| --------------- | --------------------------------------------- |
| Backend path    | /opt/nyanpass-dev/                            |
| Backend binary  | /opt/nyanpass-dev/rel_backend                 |
| Backend service | nyanpass-dev                                  |
| Web root        | /www/wwwroot/tfd2.clonod.top/                 |
| Nginx config    | /etc/nginx/sites-available/tfd2.clonod.top    |
| Backend log     | journalctl -u nyanpass-dev -f                 |
| Node path       | /opt/nyanpass/                                |
| Service         | nyanpass                                      |
| GID             | 24 (tess-rukou) — DeviceGroupType_Inbound     |
| Handle          | 604f4ee57894960d4bdad57e17c9b4d2              |
| Token           | 2ee66e5f-bb4e-3fef-192f-a02f39b5362f          |
| Node path       | /opt/nyanpass/                                |
| Service         | nyanpass                                      |
| GID             | 26 (chukou1) — DeviceGroupType_OutboundBySite |
| Handle          | 9ed192027cdd63191948f21d7d8007c7              |
| Token           | 70b9ad29-e1af-5b43-9e4c-431b7ee1db13          |
| Dịch vụ         | V2bX (VMess/Trojan proxy)                     |
| Port đích 1     | 37643 (V2bX)                                  |
| Port đích 2     | 8442 (V2bX)                                   |

---

## Kiến Trúc Kỹ Thuật Chi Tiết

### 1. Node Client Tunnel v2 (Go binary — `/opt/nyanpass/rel_nodeclient`)

**Kiến trúc**: Node client v2 là full tunnel client, gồm 2 phần chính:

- **Monitor**: Gửi HTTP POST mỗi 2 giây đến `/api/v1/node/report` (CPU, RAM, disk, network)
- **Tunnel**: Pull config từ `/api/v1/client/config_v2?token=TOKEN` mỗi 30s, mở port lắng nghe, tạo hầm chuyển tiếp

**Luồng chuyển tiếp**:

```
Client → Inbound(listen_port) → [TLS/WS/Direct Tunnel] → Outbound(tls_port) → Đích(dest)
```

**Các chức năng chính**:

| Chức năng            | Mô tả                                                                 |
| -------------------- | --------------------------------------------------------------------- |
| **Multi-Protocol**   | TLS (tls_simple), WebSocket (ws), Direct (plain TCP)                  |
| **Load Balancing**   | 5 policies: random, round_robin, ip_hash, least_connections, failover |
| **Proxy Protocol**   | v1 (text) + v2 (binary) — gửi IP thật của client đến đích             |
| **Direct Policy**    | 0=không, 1=optional (thử direct trước), 2=force (luôn direct)         |
| **Fallback Group**   | Tự động chuyển sang fallback group khi outbound chính fail            |
| **Access Control**   | Outbound kiểm tra `allowed_in` — từ chối inbound không được phép      |
| **Health Checks**    | Định kỳ 10s TLS dial đến từng outbound server                         |
| **Traffic Counting** | Đếm bytes in/out per rule                                             |
| **Hot-Reload**       | Inbound reload rules mỗi 60s, Outbound reload config mỗi 60s          |

**Các port trên Outbound (tính từ MD5 hash của handle)**:

| Port          | Range        | Công thức                 |
| ------------- | ------------ | ------------------------- |
| `direct_port` | 10000-75535  | `10000 + h[0]*256 + h[1]` |
| `ws_port`     | 20000-85535  | `20000 + h[2]*256 + h[3]` |
| `tls_port`    | 3000-68535   | `3000 + h[4]*256 + h[5]`  |
| `udp_port`    | 40000-105535 | `40000 + h[6]*256 + h[7]` |

> **QUAN TRỌNG**: Handle lưu trong `/opt/nyanpass/handle` — không được xoá file này vì port tính từ handle. Nếu handle thay đổi → port thay đổi → inbound không kết nối được.

**Giao thức tunnel (header gửi từ inbound → outbound)**:

```
Byte 0-1:   Độ dài destination (big-endian uint16)
Byte 2-3:   Inbound GID (big-endian uint16) — để outbound kiểm tra access control
```

**Cách tính CPU (QUAN TRỌNG — dễ sai)**:

```go
var prevIdle, prevTotal float64  // Module-level, lưu giá trị lần trước
func getCPUUsage() float64 {
    // Đọc /proc/stat, tính diff idle và total giữa 2 lần gọi
    // Lần đầu: prev* = 0 → return 0 (chưa có baseline)
    // Lần sau: idleDiff = idle - prevIdle, totalDiff = total - prevTotal
    // CPU% = (1 - idleDiff/totalDiff) * 100
}
```

> ❌ **Lỗi cũ**: Tính `(1 - idle/total) * 100` từ giá trị tích lũy → ra trung bình từ lúc boot (~5-6% cố định).
> ✅ **Cách đúng**: Tính diff giữa 2 lần đọc → ra CPU real-time (~1-3% biến động).

**Cách tính RAM**:

```go
// Đọc /proc/meminfo: MemTotal, MemFree, Buffers, Cached
used = MemTotal - MemFree - Buffers - Cached  // Khớp với `free -m`
```

> ❌ **Lỗi cũ**: Dùng `MemAvailable` → RAM bị phóng đại (292MB thay vì 178MB).
> ✅ **Cách đúng**: Dùng `MemFree + Buffers + Cached` → sát thực tế.

**Cách tính Network Speed (QUAN TRỌNG — dễ sai)**:

```go
var prevIn, prevOut uint64; var prevTime time.Time
var cachedNetInSp, cachedNetOutSp uint64

// Gọi MỘT LẦN trong collectReport() để cache cả in và out speed
func readNetSpeed() {
    // Đọc /proc/net/dev, lấy bytes counter
    // Lần đầu: cachedNetInSp/cachedNetOutSp = 0, lưu prev*
    // Lần sau: cachedNetInSp = (curIn - prevIn) / elapsed
    //          cachedNetOutSp = (curOut - prevOut) / elapsed
}

// Getter chỉ return giá trị đã cache — KHÔNG gọi readNetSpeed() nữa
func getNetInSpeed() uint64  { return cachedNetInSp }
func getNetOutSpeed() uint64 { return cachedNetOutSp }
```

> ❌ **Lỗi cũ**: Trả về raw bytes counter từ /proc/net/dev → 26MB/s sai.
> ❌ **Lỗi #14**: `getNetInSpeed()` và `getNetOutSpeed()` mỗi hàm gọi `readNetSpeed()` riêng → lần gọi thứ 2 có elapsed ≈ 0 → `net_out_speed` luôn = 0.
> ✅ **Cách đúng**: Gọi `readNetSpeed()` MỘT LẦN trong `collectReport()`, cache cả 2 giá trị vào `cachedNetInSp`/`cachedNetOutSp`. Getter chỉ return giá trị đã cache.

### 2. WebSocket Real-time (Probe Page)

**Backend**: Gorilla WebSocket tại `/api/v1/system/node/status_ws`

- `WSHub` quản lý connections, broadcast mỗi 1 giây
- Client connect → gửi ngay dữ liệu hiện tại → sau đó nhận push mỗi 1s
- Auto-reconnect: tối đa 10 lần, delay 2s

**Nginx config cho WebSocket**:

```nginx
location /api/v1/system/node/status_ws {
    proxy_pass http://127.0.0.1:18889;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
}
```

### 3. Install Script (`/download/install.sh`)

**Giống bản gốc 100%**: 3 prompts (tên service, tối ưu BBR, cài tools), 12 tools, sysctl optimization.

**QUAN TRỌNG**: Argument parsing phải dùng `ALL_ARGS="${*:2}"` (gộp TẤT CẢ tham số), không dùng `PRODUCT_ARGUMENTS="$2"` (chỉ lấy `-t`).

**Lệnh cài đặt**:

```bash
bash <(curl -fLSs https://tfd2.clonod.top/download/install.sh) rel_nodeclient -t TOKEN -u https://tfd2.clonod.top
```

**Lệnh huỷ**:

```bash
rm -f /etc/systemd/system/nyanpass.service ; rm -rf /opt/nyanpass ; systemctl disable --now nyanpass
```

### 4. Các Lỗi Đã Gặp & Bài Học

| #   | Lỗi                            | Nguyên nhân                                                                                            | Bài học                                                                                                                                    |
| --- | ------------------------------ | ------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------ |
| 1   | **Trang trắng**                | File JS/CSS permission 700, nginx không đọc được                                                       | Luôn `chmod 755` + `chown www-data`                                                                                                        |
| 2   | **JS không chạy**              | Quote lẻ trong HTML → syntax error                                                                     | Kiểm tra quote balance TRƯỚC khi deploy                                                                                                    |
| 3   | **Binary không update**        | Deploy không ghi đè (cùng tên file)                                                                    | Xác nhận `md5sum` sau deploy                                                                                                               |
| 4   | **Tốc độ sai 26MB/s**          | Raw counter thay vì diff/elapsed                                                                       | Luôn so sánh với VPS thực tế                                                                                                               |
| 5   | **CPU sai 5.9% cố định**       | Tích lũy từ boot thay vì diff                                                                          | Module-level vars cho previous values                                                                                                      |
| 6   | **RAM sai gấp đôi**            | MemAvailable thay vì MemFree+Buffers+Cached                                                            | Đối chiếu với `free -m`                                                                                                                    |
| 7   | **start.sh thiếu token**       | `$2` chỉ lấy `-t` thay vì toàn bộ args                                                                 | Dùng `"${*:2}"` gộp tất cả tham số                                                                                                         |
| 8   | **Sparkline không hiển thị**   | `<canvas>` + `<path>` HTML invalid                                                                     | Dùng `<svg>` thuần cho sparkline                                                                                                           |
| 9   | **GitHub raw cache 5 phút**    | CDN cache không update                                                                                 | Host script trên server riêng                                                                                                              |
| 10  | **Geo IP "?"**                 | Thiếu field assignment trong code                                                                      | Geo lookup ở backend (ip-api.com)                                                                                                          |
| 11  | **Binary backend sai tên**     | Deploy `server` nhưng service chạy `rel_backend`                                                       | Đọc kỹ `ExecStart` trong `.service` file                                                                                                   |
| 12  | **DB thiếu cột mới**           | SQLite không auto-add column khi GORM migrate                                                          | Thêm thủ công: `ALTER TABLE ADD COLUMN`                                                                                                    |
| 13  | **Node client không update**   | systemd `Restart=always` tự restart trước khi scp                                                      | Stop service → rm binary → scp → start                                                                                                     |
| 14  | **net_out_speed luôn = 0**     | `getNetInSpeed()` và `getNetOutSpeed()` gọi `readNetSpeed()` riêng biệt → lần gọi thứ 2 có elapsed ≈ 0 | Gọi `readNetSpeed()` MỘT LẦN trong `collectReport()`, cache cả 2 giá trị `cachedNetInSp`/`cachedNetOutSp`. Getter chỉ return cached value  |
| 15  | **Cờ quốc gia flickering (?)** | Node client không gửi `ip4_geo` → backend gọi ip-api.com MỖI 2 GIÂY → API fail/rate-limit → geo rỗng   | Backend: cache geo vào DB, chỉ lookup lại khi IP thay đổi. Frontend: cache geo client-side (`geoCache{}`), dùng lại nếu server trả về rỗng |
| 16  | **Thanh CPU/RAM giật cục**     | Giá trị thay đổi nhanh mỗi 1s từ WebSocket, transition CSS 0.3s quá ngắn                               | Backend: không cần thay đổi. Frontend: EMA smoothing (`alpha=0.4`) + CSS transition `0.8s ease-out`                                        |
| 17  | **Outbound không mở TLS port** | Thiếu `openssl` trên VPS → không tạo được self-signed TLS cert                                         | Install script tự động `apt-get install openssl` nếu chưa có                                                                               |
| 18  | **Handle bị mất → port sai**   | File `/opt/nyanpass/handle` không tồn tại → node client tạo handle mới → port thay đổi → mất kết nối   | Install script backup handle trước khi xoá; Node client tìm handle ở `./handle` + `/opt/nyanpass/handle`; Lưu token vào file               |
| 19  | **WS/UDP port > 65535**        | Công thức `20000 + h[2]*256 + h[3]` có thể vượt 65535 với 1 số handle                                  | Thêm modulo: `20000 + (h[2]*256 + h[3]) % 45535` cho WS; `40000 + (h[6]*256 + h[7]) % 25535` cho UDP. Giữ nguyên port cũ < 65535           |
| 20  | **LB chỉ dùng 1 server**       | Outbound mới không mở port tunnel → health check fail → LB bỏ qua                                      | Deploy binary mới (tunnel v2) + khôi phục handle cũ + cài openssl. Sau đó cả 2 server được dùng                                            |

### 5. Database — Các Bảng Chính

| Bảng                             | Mục đích                                          |
| -------------------------------- | ------------------------------------------------- |
| `user_groups`                    | Nhóm người dùng                                   |
| `user_logins`                    | Token đăng nhập (7 ngày nếu "remember")           |
| `device_groups`                  | Nhóm thiết bị (Inbound/Outbound/AgentOnly)        |
| `node_clients`                   | Server kết nối về panel (handle, IP, CPU, RAM...) |
| `chain_outbounds`                | Chuỗi chuyển tiếp                                 |
| `device_group_folders` + `_rels` | Phân loại folder                                  |

### 6. Config_v2 API — Cấu Trúc Response

**Endpoint**: `GET /api/v1/client/config_v2?token=TOKEN` (không cần auth)

**QUAN TRỌNG**: `name` luôn là chuỗi rỗng `""` (bảo mật — API public chỉ cần token).
`device_group.name` trong remotes cũng luôn rỗng.

**Response khác nhau theo loại group**:

**INBOUND** — Không có `fallback_group` ở top-level. Remotes (outbound groups) có IP + ports đầy đủ:

```json
{
  "id": 24,
  "name": "",
  "type": "DeviceGroupType_Inbound",
  "ratio": "0",
  "config": "{}",
  "group_uuid": "...",
  "chain": {},
  "users": [{ "uid": 1 }, { "uid": 0 }],
  "remotes": [
    {
      "group_uuid": "...",
      "device_group": {
        "id": 26,
        "name": "",
        "type": "DeviceGroupType_OutboundBySite",
        "ratio": "0",
        "config": "{\"protocol\":\"tls_simple\"}",
        "fallback_group": 0
      },
      "infos": [
        {
          "u": "...",
          "v": 4,
          "w": 1,
          "direct_port": 36679,
          "ws_port": 38403,
          "tls_port": 48782,
          "udp_port": 100124
        }
      ]
    }
  ],
  "rules": [...]
}
```

**OUTBOUND** — CÓ `fallback_group` ở top-level. Remotes (inbound groups) CHỈ có `{u, v, w}`:

```json
{
  "id": 26,
  "name": "",
  "type": "DeviceGroupType_OutboundBySite",
  "ratio": "0",
  "config": "{\"protocol\":\"tls_simple\"}",
  "fallback_group": 0,
  "group_uuid": "...",
  "chain": {},
  "users": [{ "uid": 0 }, { "uid": 1 }],
  "remotes": [
    {
      "group_uuid": "...",
      "device_group": {
        "id": 24,
        "name": "",
        "type": "DeviceGroupType_Inbound",
        "ratio": "0",
        "config": "{}"
      },
      "infos": [{ "u": "...", "v": 4, "w": 1 }]
    }
  ],
  "rules": [/* TẤT CẢ rules trỏ đến outbound này */]
}
```

> **Lưu ý**: `infos` = `null` (không phải `[]`) khi không có server online. `fallback_group` dùng `omitempty` — chỉ xuất hiện khi ≠ 0.

**AGENT ONLY** — Chỉ thông tin cơ bản (id, name, type, config), không rules, không remotes.

### 7. API Endpoints Chính

| Method              | Path                                      | Mục đích                        |
| ------------------- | ----------------------------------------- | ------------------------------- |
| POST                | `/api/v1/auth/login`                      | Đăng nhập (có `remember: true`) |
| GET                 | `/api/v1/user/info`                       | Thông tin user (admin check)    |
| GET/PUT/POST/DELETE | `/api/v1/admin/devicegroup`               | CRUD device group               |
| GET/PUT/DELETE      | `/api/v1/admin/user`                      | CRUD user                       |
| GET/PUT/DELETE      | `/api/v1/admin/usergroup`                 | CRUD user group                 |
| POST                | `/api/v1/node/report`                     | Node client gửi báo cáo         |
| GET                 | `/api/v1/system/node/status`              | REST node status                |
| **WS**              | `/api/v1/system/node/status_ws`           | **WebSocket real-time**         |
| PUT                 | `/api/v1/system/node/weight/:gid/:handle` | Đặt trọng số                    |
| POST                | `/api/v1/system/node/terminal/:handle`    | Tạo terminal session            |
| POST                | `/api/v1/system/node/kick/:handle`        | Kick server                     |
| GET                 | `/api/v1/system/info`                     | System info (version, time)     |

### 8. Tính Năng Nâng Cao

#### 8.1 UDP Smart Bind

Bật relay UDP qua tunnel cho inbound group. Khi bật, inbound mở UDP listener trên tất cả rule ports song song với TCP.

**Cấu hình**: Trong `config` JSON của Inbound group:

```json
{ "udp_smart_bind": true }
```

**Cách hoạt động**:

```
Client UDP → Inbound (UDP port) → [Header: 2B dest_len + dest] → Outbound (udp_port) → Đích UDP
                                                                          ↓
Client UDP ← Inbound (UDP port) ← [Response]                    ← Đích UDP
```

**Header UDP relay** (gửi từ inbound → outbound udp_port):

```
Byte 0-1:   Độ dài destination (big-endian uint16)
Sau đó:     UDP payload gốc
```

#### 8.2 Hide Status

Ẩn device group khỏi probe page (trang giám sát public).

| Giá trị | Ý nghĩa                         |
| ------- | ------------------------------- |
| 0       | Hiển thị bình thường (mặc định) |
| 1       | Ẩn một phần                     |
| 2       | Ẩn hoàn toàn khỏi probe         |

- Admin vẫn thấy trong panel quản lý
- Probe page tự động lọc `hide_status >= 2`
- Field `hide_status` dùng `omitempty` → chỉ xuất hiện trong JSON khi ≠ 0

#### 8.3 Load Balancing — Cơ Chế

**Policies** (mặc định: Weighted Random):

| Policy              | Code           | Cách chọn                                                  |
| ------------------- | -------------- | ---------------------------------------------------------- |
| `random`            | `LBRandom`     | Weighted random — server có weight cao được chọn nhiều hơn |
| `round_robin`       | `LBRoundRobin` | Weighted round-robin, counter per group                    |
| `ip_hash`           | `LBIPHash`     | Hash(clientIP + groupID) → server cố định                  |
| `least_connections` | `LBLeastConn`  | Chọn server có tỉ lệ connections/weight thấp nhất          |
| `failover`          | `LBFailover`   | Dùng server healthy đầu tiên                               |

**Health Check**: TLS dial đến từng outbound server mỗi 10s. Server không phản hồi trong 5s → đánh dấu unhealthy → LB bỏ qua.

#### 8.4 Fallback Group

Khi TẤT CẢ server trong outbound group chính đều unhealthy, inbound tự động chuyển sang fallback group.

**Cấu hình**: Set `fallback_group` trong device group → trỏ đến ID của group dự phòng.

#### 8.5 Direct Policy (入口直出)

Inbound có thể kết nối thẳng đến đích, bỏ qua outbound:

| Giá trị | Ý nghĩa                                            |
| ------- | -------------------------------------------------- |
| 0       | Không direct — luôn qua outbound                   |
| 1       | Optional — thử direct trước, fail thì qua outbound |
| 2       | Force — luôn direct, không dùng outbound           |

### 9. Node Client v2 — Danh Sách Tính Năng

| Tính năng                      | Trạng thái | Mô tả                                              |
| ------------------------------ | :--------: | -------------------------------------------------- |
| Multi-Protocol (TLS/WS/Direct) |     ✅     | TLS tunnel chính, WS + Direct phụ                  |
| Load Balancing (5 policies)    |     ✅     | random, round_robin, ip_hash, least_conn, failover |
| Health Check (10s TLS dial)    |     ✅     | Tự động bỏ qua server unhealthy                    |
| Fallback Group                 |     ✅     | Chuyển sang group dự phòng khi outbound chính chết |
| Direct Policy (0/1/2)          |     ✅     | Hỗ trợ入口直出                                     |
| UDP Smart Bind                 |     ✅     | Relay UDP qua tunnel                               |
| Proxy Protocol (v1/v2)         |     ⚠️     | Cấu trúc có, chưa hoàn thiện                       |
| Traffic Counting               |     ✅     | Đếm bytes in/out per rule                          |
| Config Hot-Reload              |     ✅     | Inbound 60s, Outbound 60s                          |
| Access Control (allowed_in)    |     ⚠️     | Cấu trúc có, chưa hoàn thiện                       |
| Chain Outbound                 |     ❌     | Chưa triển khai (web mẫu cũng không dùng)          |
| Delay Optimization             |     ❌     | Chưa triển khai (web mẫu cũng không dùng)          |
| WebSocket Tunnel               |     ⚠️     | Outbound mở port WS, chưa hoàn thiện handshake     |

### 10. Install Script — Các Bước Tự Động

Script `/download/install.sh` thực hiện tuần tự:

1. **Validate** tham số (`-t TOKEN`, `-u PANEL_URL`)
2. **Detect** architecture (amd64, amd64v3, arm64)
3. **Hỏi** tên service (mặc định: `nyanpass`), tối ưu BBR, cài tools
4. **Cài openssl** nếu chưa có (cần cho TLS cert)
5. **Download** binary từ panel `/download/rel_nodeclient_linux_${ARCH}`
6. **Lưu token** vào `/opt/${service_name}/token`
7. **Khôi phục handle** nếu có backup (tránh mất handle khi cài lại)
8. **Tạo start.sh** + **env.sh** + **systemd service**
9. **Enable & start** service
10. **Tối ưu sysctl** (BBR, TCP buffers, file limits) nếu chọn
