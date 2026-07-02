#!/bin/bash
set -e

# ============================================================
#  Senpass — Bộ Cài Đặt Tự Động (One-Click Install)
#  GitHub: https://github.com/gencloud89/senpass
# ============================================================

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

info()  { echo -e "${GREEN}[✓]${NC} $*"; }
warn()  { echo -e "${YELLOW}[!]${NC} $*"; }
error() { echo -e "${RED}[✗]${NC} $*" && exit 1; }
ask()   { echo -ne "${CYAN}[?]${NC} $* "; }

# ============================================================
# 1. KIỂM TRA HỆ THỐNG
# ============================================================
echo ""
echo "============================================"
echo "  Senpass — Cài Đặt Panel Quản Lý"
echo "  github.com/gencloud89/senpass"
echo "============================================"
echo ""

if [ "$(id -u)" != "0" ]; then error "Vui lòng chạy với quyền root: sudo bash install.sh"; fi

OS=$(cat /etc/os-release | grep "^ID=" | cut -d= -f2 | tr -d '"')
if [ "$OS" != "ubuntu" ] && [ "$OS" != "debian" ]; then
    warn "Khuyên dùng Ubuntu 20.04/22.04 hoặc Debian 11/12. OS hiện tại: $OS"
fi

CPU_CORES=$(nproc)
TOTAL_RAM=$(free -m | awk '/^Mem:/{print $2}')
[ $TOTAL_RAM -lt 512 ] && error "RAM tối thiểu 512MB. Hiện tại: ${TOTAL_RAM}MB"
info "CPU: $CPU_CORES cores | RAM: ${TOTAL_RAM}MB"

# ============================================================
# 2. CÂU HỎI TƯƠNG TÁC
# ============================================================
echo ""
echo "--- Cấu Hình Cơ Bản ---"

while [ -z "$DOMAIN" ]; do
    ask "Ten mien cua ban? (vd: panel.cuaban.com):"
    read DOMAIN
    [ -z "$DOMAIN" ] && warn "Vui long nhap ten mien!"
done

while [ -z "$ADMIN_EMAIL" ]; do
    ask "Email quan tri (dung cho SSL Let's Encrypt):"
    read ADMIN_EMAIL
    [ -z "$ADMIN_EMAIL" ] && warn "Vui long nhap email!"
done

while [ -z "$PANEL_USER" ]; do
    ask "Tai khoan admin cho panel (vd: admin@$DOMAIN):"
    read PANEL_USER
    [ -z "$PANEL_USER" ] && warn "Vui long nhap tai khoan admin!"
done

while [ -z "$PANEL_PASS" ]; do
    ask "Mat khau admin (toi thieu 6 ky tu):"
    read -s PANEL_PASS
    echo ""
    [ ${#PANEL_PASS} -lt 6 ] && PANEL_PASS="" && warn "Mat khau phai it nhat 6 ky tu!"
done

INSTALL_DIR="/opt/senpass"
SERVICE_NAME="senpass"

echo ""
info "Cau hinh cua ban:"
echo "   Ten mien:      $DOMAIN"
echo "   Email:         $ADMIN_EMAIL"
echo "   Admin user:    $PANEL_USER"
echo "   Thu muc cai:   $INSTALL_DIR"

ask "Tiep tuc cai dat? (y/n):"
read CONFIRM
[ "${CONFIRM,,}" != "y" ] && error "Da huy."

# ============================================================
# 3. CÀI ĐẶT DEPENDENCIES
# ============================================================
echo ""
echo "--- Cai Dat Dependencies ---"

info "Dang cap nhat package list..."
apt-get update -qq

info "Cai dat cac goi can thiet (nginx, git, curl, certbot)..."
apt-get install -y -qq nginx git curl certbot python3-certbot-nginx ufw

# Cài Go nếu chưa có
if ! command -v go &>/dev/null; then
    GO_VERSION="1.22.5"
    info "Cai dat Go $GO_VERSION..."
    curl -fLSs -o /tmp/go.tar.gz "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz"
    tar -C /usr/local -xzf /tmp/go.tar.gz
    echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
    export PATH=$PATH:/usr/local/go/bin
    rm -f /tmp/go.tar.gz
    info "Go $(go version | awk '{print $3}') da duoc cai dat"
else
    info "Go $(go version | awk '{print $3}') da co san"
fi

# ============================================================
# 4. TẠO CẤU TRÚC THƯ MỤC
# ============================================================
echo ""
info "Tao thu muc cai dat..."

mkdir -p "$INSTALL_DIR"/{backend,frontend,download,db}
cd "$INSTALL_DIR"

# ============================================================
# 5. TẢI MÃ NGUỒN TỪ GITHUB
# ============================================================
info "Tai ma nguon Senpass tu GitHub..."

if [ -d "$INSTALL_DIR/.git" ]; then
    cd "$INSTALL_DIR"
    git pull origin main 2>/dev/null || true
else
    git clone https://github.com/gencloud89/senpass.git /tmp/senpass_repo 2>/dev/null || {
        warn "Khong the clone tu GitHub. Thu cai bang tay..."
        error "Hay chay: git clone https://github.com/gencloud89/senpass.git $INSTALL_DIR"
    }
    cp -r /tmp/senpass_repo/* "$INSTALL_DIR/" 2>/dev/null || true
    cp -r /tmp/senpass_repo/.git "$INSTALL_DIR/" 2>/dev/null || true
    rm -rf /tmp/senpass_repo
fi

# ============================================================
# 6. BUILD BACKEND
# ============================================================
echo ""
info "Build backend..."

cd "$INSTALL_DIR/backend"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o "$INSTALL_DIR/senpass_server" ./cmd/server/

# Set env cho admin
cat > "$INSTALL_DIR/env.sh" << ENVEOF
export ADMIN_USER="$PANEL_USER"
export ADMIN_PASS="$PANEL_PASS"
export DB_PATH="$INSTALL_DIR/db/senpass.db"
ENVEOF
chmod 644 "$INSTALL_DIR/env.sh"

# ============================================================
# 7. CẤU HÌNH NGINX
# ============================================================
echo ""
info "Cau hinh Nginx..."

cat > /etc/nginx/sites-available/senpass << NGXEOF
server {
    listen 80;
    server_name $DOMAIN;

    root $INSTALL_DIR/frontend;
    index index.html;

    location /api/ {
        proxy_pass http://127.0.0.1:18889;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }

    location /api/v1/system/node/status_ws {
        proxy_pass http://127.0.0.1:18889;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
    }

    location /download/ {
        alias $INSTALL_DIR/download/;
    }

    location / {
        try_files \$uri \$uri/ /index.html;
    }
}
NGXEOF

ln -sf /etc/nginx/sites-available/senpass /etc/nginx/sites-enabled/
rm -f /etc/nginx/sites-enabled/default

nginx -t || error "Cau hinh Nginx loi!"
systemctl reload nginx

# ============================================================
# 8. CÀI ĐẶT SSL (Let's Encrypt)
# ============================================================
echo ""
info "Cai dat SSL cho $DOMAIN..."

certbot --nginx -d "$DOMAIN" --non-interactive --agree-tos --email "$ADMIN_EMAIL" --redirect || {
    warn "Khong the cai SSL tu dong. Co the domain chua tro ve VPS nay."
    warn "Thu cai thu cong sau: certbot --nginx -d $DOMAIN"
}
systemctl reload nginx

# ============================================================
# 9. CẤU HÌNH FIREWALL
# ============================================================
info "Mo port 80, 443..."
ufw allow 80/tcp
ufw allow 443/tcp
ufw --force enable

# ============================================================
# 10. TẠO SYSTEMD SERVICE
# ============================================================
info "Tao systemd service..."

cat > /etc/systemd/system/${SERVICE_NAME}.service << SVCEOF
[Unit]
Description=Senpass Panel Backend
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
WorkingDirectory=$INSTALL_DIR
EnvironmentFile=$INSTALL_DIR/env.sh
ExecStart=$INSTALL_DIR/senpass_server
Restart=always
RestartSec=3
LimitNOFILE=999999

[Install]
WantedBy=multi-user.target
SVCEOF

systemctl daemon-reload
systemctl enable "$SERVICE_NAME"
systemctl restart "$SERVICE_NAME"

# ============================================================
# 11. TRIỂN KHAI FRONTEND
# ============================================================
info "Trien khai frontend..."

cp "$INSTALL_DIR/frontend/index.html" /tmp/senpass_frontend_backup.html 2>/dev/null || true

chown -R www-data:www-data "$INSTALL_DIR/frontend" 2>/dev/null || true
chmod 755 "$INSTALL_DIR/frontend" 2>/dev/null || true
chmod 644 "$INSTALL_DIR/frontend/index.html" 2>/dev/null || true

# ============================================================
# 12. KIỂM TRA CUỐI CÙNG
# ============================================================
sleep 3

if systemctl is-active --quiet "$SERVICE_NAME"; then
    echo ""
    echo "============================================"
    echo -e "  ${GREEN}CAI DAT THANH CONG!${NC}"
    echo "============================================"
    echo ""
    echo "  🌐 Panel URL:  https://$DOMAIN"
    echo "  👤 Admin:      $PANEL_USER"
    echo "  🔑 Mat khau:   (nhu ban da nhap)"
    echo ""
    echo "  📂 Thu muc:    $INSTALL_DIR"
    echo "  📋 Service:    systemctl status $SERVICE_NAME"
    echo "  📝 Log:        journalctl -u $SERVICE_NAME -f"
    echo ""
    echo "  📥 Lenh cai node client:"
    echo "     bash <(curl -fLSs https://$DOMAIN/download/install.sh) rel_nodeclient -t TOKEN -u https://$DOMAIN"
    echo ""
    echo "============================================"
else
    error "Backend khong khoi dong duoc! Kiem tra log: journalctl -u $SERVICE_NAME -n 30"
fi
