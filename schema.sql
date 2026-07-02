-- Nyanpass Database Schema
-- Extracted from SQLite database on VPS (2026-06-30)
-- ORM: GORM (supports SQLite, MySQL, PostgreSQL)

-- ============================================
-- USERS & AUTH
-- ============================================

-- Users table: Tài khoản người dùng
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME,
    updated_at DATETIME,
    username TEXT UNIQUE NOT NULL,
    password TEXT,
    balance TEXT,                    -- Số dư tài khoản (decimal string)
    aff_balance TEXT,                -- Số dư affiliate commission
    inviter INTEGER,                 -- User ID người mời (affiliate)
    invite_config TEXT,              -- Cấu hình affiliate (JSON)
    invite_code TEXT,                -- Mã mời của user này
    plan_id INTEGER,                 -- Plan hiện tại
    group_id INTEGER,                -- User group ID
    max_rules INTEGER,               -- Giới hạn số forward rules
    speed_limit INTEGER,             -- Giới hạn tốc độ (Mbps)
    ip_limit INTEGER,                -- Giới hạn số IP
    connection_limit INTEGER,        -- Giới hạn số connection
    traffic_enable INTEGER NOT NULL DEFAULT 0,  -- Traffic được phân bổ (bytes)
    traffic_used INTEGER NOT NULL DEFAULT 0,     -- Traffic đã dùng (bytes)
    expire INTEGER NOT NULL DEFAULT 0,           -- Unix timestamp hết hạn
    auto_renew NUMERIC,              -- Tự động gia hạn
    banned NUMERIC,                  -- Bị khóa
    admin NUMERIC,                   -- Là admin
    allow_device NUMERIC,            -- Cho phép tạo device group
    telegram_id INTEGER,             -- Telegram user ID đã bind
    note TEXT                        -- Ghi chú của admin
);

-- User logins: Phiên đăng nhập
CREATE TABLE user_logins (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME,
    updated_at DATETIME,
    uid INTEGER,
    token TEXT,
    token_expire INTEGER             -- Unix timestamp hết hạn token
);

-- User groups: Nhóm người dùng
CREATE TABLE user_groups (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME,
    updated_at DATETIME,
    name TEXT,
    show_order INTEGER
);

-- ============================================
-- DEVICE GROUPS & NETWORK
-- ============================================

-- Device groups: Nhóm thiết bị/node
CREATE TABLE device_groups (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME,
    updated_at DATETIME,
    name TEXT,
    type TEXT,                       -- Loại: inbound/outbound
    token TEXT,                      -- Token để node client kết nối
    uid INTEGER,                     -- Owner user ID
    ratio TEXT,                      -- Tỉ lệ (dùng cho load balancing)
    enable_for_gid TEXT,             -- Group IDs được phép dùng (JSON array)
    traffic_used INTEGER NOT NULL DEFAULT 0,
    connect_host TEXT,               -- Host/IP kết nối đến
    port_range TEXT,                 -- Port range (VD: "10000-20000")
    allowed_out TEXT,                -- Các outbound được phép (JSON)
    config TEXT,                     -- Cấu hình bổ sung (JSON)
    down_sec INTEGER,                -- Thời gian down hiện tại (seconds)
    fallback_group INTEGER,         -- Nhóm fallback khi down
    allowed_in TEXT,                 -- Các inbound được phép (JSON)
    note TEXT,
    show_order INTEGER,              -- Thứ tự hiển thị
    hide_status INTEGER              -- Ẩn trạng thái khỏi probe page
);

-- Device group folders: Thư mục phân loại
CREATE TABLE device_group_folders (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME,
    updated_at DATETIME,
    name TEXT NOT NULL
);

-- Device group folder relations: Quan hệ folder↔device group
CREATE TABLE device_group_folder_rels (
    folder_id INTEGER NOT NULL,
    dg_id INTEGER NOT NULL,
    show_order INTEGER,
    PRIMARY KEY (folder_id, dg_id),
    FOREIGN KEY (folder_id) REFERENCES device_group_folders(id) ON DELETE CASCADE,
    FOREIGN KEY (dg_id) REFERENCES device_groups(id) ON DELETE CASCADE
);

-- Chain outbounds: Chuỗi proxy nhiều tầng
CREATE TABLE chain_outbounds (
    group_id INTEGER NOT NULL,
    seq INTEGER NOT NULL,            -- Thứ tự trong chuỗi
    this_hop INTEGER NOT NULL,       -- Device group ID của hop này
    next_hop INTEGER NOT NULL,       -- Device group ID của hop tiếp theo
    mux NUMERIC,                     -- Multiplexing enabled
    PRIMARY KEY (group_id, seq),
    FOREIGN KEY (group_id) REFERENCES device_groups(id) ON DELETE CASCADE
);

-- ============================================
-- FORWARD RULES
-- ============================================

-- Forward rules: Luật chuyển tiếp proxy
CREATE TABLE forward_rules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME,
    updated_at DATETIME,
    name TEXT,
    uid INTEGER,                     -- Owner user ID
    paused NUMERIC NOT NULL DEFAULT FALSE,
    listen_port INTEGER,             -- Port lắng nghe inbound
    device_group_in INTEGER,         -- Inbound device group ID
    device_group_out INTEGER,        -- Outbound device group ID
    traffic_used INTEGER NOT NULL DEFAULT 0,
    config TEXT,                     -- Cấu hình chi tiết (JSON)
    status TEXT                      -- Trạng thái rule
);

-- Forward rule folders: Thư mục phân loại
CREATE TABLE forward_rule_folders (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME,
    updated_at DATETIME,
    uid INTEGER,
    name TEXT NOT NULL
);

-- Forward rule folder relations
CREATE TABLE forward_rule_folder_rels (
    folder_id INTEGER NOT NULL,
    rule_id INTEGER NOT NULL,
    PRIMARY KEY (folder_id, rule_id),
    FOREIGN KEY (rule_id) REFERENCES forward_rules(id) ON DELETE CASCADE,
    FOREIGN KEY (folder_id) REFERENCES forward_rule_folders(id) ON DELETE CASCADE
);

-- ============================================
-- SHOP & PAYMENT
-- ============================================

-- Plans: Gói dịch vụ
CREATE TABLE plans (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME,
    updated_at DATETIME,
    type TEXT,                       -- Loại plan (recurring/one-time)
    name TEXT,
    "desc" TEXT,                     -- Mô tả
    price TEXT,                      -- Giá (decimal string)
    multiple INTEGER,                -- Hệ số nhân
    show_order INTEGER,
    hide NUMERIC NOT NULL DEFAULT FALSE,
    group_id INTEGER NOT NULL DEFAULT 1,  -- User group áp dụng
    max_rules INTEGER NOT NULL DEFAULT 0,
    traffic INTEGER NOT NULL DEFAULT 0,  -- Traffic bytes
    speed_limit INTEGER NOT NULL DEFAULT 0,
    ip_limit INTEGER NOT NULL DEFAULT 0,
    connection_limit INTEGER NOT NULL DEFAULT 0
);

-- Orders: Đơn hàng
CREATE TABLE orders (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME,
    updated_at DATETIME,
    type TEXT,                       -- deposit/purchase
    uid INTEGER,
    amount TEXT,                     -- Số tiền (decimal string)
    message TEXT,                    -- Thông tin thêm (JSON)
    status TEXT,                     -- pending/paid/expired/cancelled
    order_no TEXT UNIQUE,            -- Mã đơn hàng
    open_time INTEGER,               -- Thời gian tạo đơn (unix)
    paid_time INTEGER                -- Thời gian thanh toán (unix)
);

-- Redeem codes: Mã đổi thưởng
CREATE TABLE redeem_codes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME,
    updated_at DATETIME,
    code TEXT UNIQUE NOT NULL,
    count INTEGER,                   -- Số lần dùng còn lại
    plan_id INTEGER,                 -- Plan được redeem
    discount_ratio TEXT              -- Tỉ lệ giảm giá (decimal string)
);

-- ============================================
-- AFFILIATE
-- ============================================

-- Affiliate logs: Lịch sử affiliate
CREATE TABLE affiliate_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME,
    updated_at DATETIME,
    type TEXT,                       -- commission/withdraw
    uid INTEGER,
    amount TEXT,
    message TEXT,
    status TEXT,
    telegram_id INTEGER
);

-- ============================================
-- STATISTICS & LOGGING
-- ============================================

-- Statistics: Thống kê time-series
CREATE TABLE statistics (
    type TEXT NOT NULL,              -- Loại thống kê (traffic/user/device)
    key TEXT NOT NULL,               -- Khóa (user_id/device_group_id)
    time INTEGER NOT NULL,           -- Unix timestamp
    number REAL NOT NULL,            -- Giá trị
    PRIMARY KEY (type, key, time)
);

-- Logs: System logs
CREATE TABLE logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME,
    updated_at DATETIME,
    level TEXT,                      -- info/warn/error
    context TEXT,                    -- Context log
    message TEXT,                    -- Nội dung log
    data TEXT                        -- Dữ liệu bổ sung (JSON)
);

-- ============================================
-- KV STORE & NOTIFICATIONS
-- ============================================

-- KV Store: Key-Value cho cấu hình hệ thống
CREATE TABLE kvs (
    key TEXT UNIQUE NOT NULL,
    value TEXT
);

-- User notification settings
CREATE TABLE user_notification_settings (
    uid INTEGER NOT NULL,
    msg_type TEXT NOT NULL,          -- Loại thông báo
    channel INTEGER NOT NULL,        -- Kênh (telegram/email)
    mode INTEGER,                    -- Chế độ
    list TEXT,                       -- Danh sách bổ sung (JSON)
    PRIMARY KEY (uid, msg_type, channel)
);

-- ============================================
-- INDEXES
-- ============================================
CREATE UNIQUE INDEX uidx_group_nexthop ON chain_outbounds(group_id, next_hop);
CREATE UNIQUE INDEX uidx_group_thishop ON chain_outbounds(group_id, this_hop);
CREATE UNIQUE INDEX idx_forward_rules_groupandport ON forward_rules(device_group_in, listen_port);
