# Senpass — Port Forwarding Management Panel

Hệ thống quản lý chuyển tiếp cổng (port forwarding) — Inbound → Outbound → Destination.

Tự xây dựng mạng lưới tunnel riêng, hỗ trợ load balancing, health check, multi-protocol, real-time monitoring.

---

## Kiến Trúc

```
Client → Inbound Node (nghe port) → [TLS/WS/Direct Tunnel] → Outbound Node → Đích khách hàng
```

| Thành phần      | Vai trò                                                            |
| --------------- | ------------------------------------------------------------------ |
| **Panel**       | Web admin quản lý users, device groups, forward rules              |
| **Inbound**     | Nhận kết nối từ khách, mở port lắng nghe, tạo tunnel đến outbound  |
| **Outbound**    | Nhận tunnel từ inbound, chuyển tiếp đến đích                       |
| **Node Client** | Go binary chạy trên VPS inbound/outbound, tự động kết nối về panel |

---

## Cài Đặt Panel

### Yêu cầu

- **VPS:** Ubuntu 20.04/22.04 hoặc Debian 11/12 (1 CPU, 512MB RAM)
- **Domain:** 1 domain trỏ về VPS
- **Port:** 80, 443 mở

### Cài đặt 1 lệnh

```bash
bash <(curl -fLSs https://raw.githubusercontent.com/gencloud89/senpass/main/install.sh)
```

Script sẽ hỏi:

1. **Tên miền** — domain cho panel (vd: panel.cuaban.com)
2. **Email** — dùng cho SSL Let's Encrypt
3. **Tài khoản admin** — username + password cho panel

Sau khi chạy xong:

- Panel: `https://DOMAIN_CUA_BAN`
- Backend service: `systemctl status senpass`
- Log: `journalctl -u senpass -f`

---

## Cài Đặt Node Client (Inbound / Outbound)

### 1. Tạo Device Group trong Panel

Vào panel → **设备组管理** → thêm group loại `入口` (Inbound) hoặc `出口` (Outbound). Mỗi group tạo ra sẽ có 1 **Token**.

### 2. Cài Node Client lên VPS

```bash
bash <(curl -fLSs https://DOMAIN_CUA_BAN/download/install.sh) rel_nodeclient -t TOKEN -u https://DOMAIN_CUA_BAN
```

Script sẽ:

- Tự detect architecture (amd64, amd64v3, arm64)
- Cài openssl (cần cho TLS tunnel)
- Tạo systemd service
- Tối ưu BBR + sysctl

### 3. Huỷ cài đặt

```bash
rm -f /etc/systemd/system/nyanpass.service
rm -rf /opt/nyanpass
systemctl disable --now nyanpass
```

---

## Tính Năng Chính

### Panel Admin

- Quản lý Device Groups (Inbound, Outbound, AgentOnly)
- CRUD Forward Rules (port → destination)
- User/User Group management
- Real-time node status qua WebSocket
- Terminal SSH vào node client

### Node Client Tunnel v2

- **Multi-Protocol:** TLS, WebSocket, Direct TCP
- **Load Balancing:** 5 policies (random, round_robin, ip_hash, least_connections, failover)
- **Health Check:** TLS dial định kỳ 10s, tự động bỏ qua server lỗi
- **Fallback Group:** Tự động chuyển sang group dự phòng
- **Direct Policy:** Hỗ trợ入口直出 (0=không, 1=optional, 2=force)
- **UDP Smart Bind:** Relay UDP qua tunnel
- **Proxy Protocol:** v1 + v2 (HAProxy)
- **Traffic Counting:** Đếm bytes in/out per rule
- **Config Hot-Reload:** Inbound 60s, Outbound 60s

---

## Công Nghệ

| Lớp            | Công nghệ                         |
| -------------- | --------------------------------- |
| Backend API    | Go + Gin + GORM + SQLite          |
| Frontend Panel | Vanilla HTML/JS (single file)     |
| Database       | SQLite (pure-Go, không CGO)       |
| Real-time      | WebSocket (Gorilla, broadcast 1s) |
| Node Client    | Go binary, report 2s/lần          |

---

## Cấu Trúc Dự Án

```
senpass/
├── install.sh                # Cài đặt panel 1 lệnh
├── backend/                  # Go API server
│   ├── cmd/server/main.go    # Entry point + routes
│   └── internal/
│       ├── handlers/         # API handlers
│       ├── models/           # Database models
│       ├── database/         # SQLite init + migration
│       └── services/         # Business logic
├── frontend/
│   └── index.html            # Panel admin (vanilla JS)
├── nodeclient/               # Node client tunnel
│   └── main.go
├── download/
│   └── install.sh            # Script cài node client
└── schema.sql                # Database schema
```

---

## API Endpoints Chính

| Method              | Path                            | Mục đích                |
| ------------------- | ------------------------------- | ----------------------- |
| POST                | `/api/v1/auth/login`            | Đăng nhập               |
| GET                 | `/api/v1/user/info`             | Thông tin user          |
| GET/PUT/POST/DELETE | `/api/v1/admin/devicegroup`     | CRUD device group       |
| GET/PUT/POST/DELETE | `/api/v1/admin/forward`         | CRUD forward rule       |
| GET/PUT/POST/DELETE | `/api/v1/admin/user`            | CRUD user               |
| POST                | `/api/v1/node/report`           | Node client gửi báo cáo |
| GET                 | `/api/v1/system/node/status`    | REST node status        |
| WS                  | `/api/v1/system/node/status_ws` | WebSocket real-time     |
| GET                 | `/api/v1/client/config_v2`      | Node client pull config |

---

## DNS Load Balancing — Mở Rộng Băng Thông

Muốn dùng 4 VPS Inbound (mỗi cái 200Mbps) với 1 domain duy nhất:

1. Tạo 1 Inbound Group trong panel → lấy Token
2. Cài node client lên cả 4 VPS, **dùng cùng Token**
3. Cấu hình DNS 4 A records cùng domain:

```
jp.cuaban.com → A → IP VPS 1
jp.cuaban.com → A → IP VPS 2
jp.cuaban.com → A → IP VPS 3
jp.cuaban.com → A → IP VPS 4
```

Kết quả: 800Mbps tổng, khách chỉ thấy 1 domain, DNS tự phân bổ.

---

## Build Từ Source

```bash
# Backend
cd backend
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server ./cmd/server/

# Node Client
cd nodeclient
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o rel_nodeclient_linux_amd64 .
```

---

## License

MIT License

---

## Tác Giả

**GenCloud** — [github.com/gencloud89](https://github.com/gencloud89)
