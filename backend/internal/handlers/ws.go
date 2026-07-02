package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"nyanpass-backend/internal/database"
	"nyanpass-backend/internal/models"
)

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

// WSHub quản lý tất cả WebSocket connections
type WSHub struct {
	clients map[*websocket.Conn]bool
	mu      sync.Mutex
}

var hub = &WSHub{clients: make(map[*websocket.Conn]bool)}

// StartWSBroadcast khởi động broadcast node status mỗi 1 giây
func StartWSBroadcast() {
	hub.Start()
}

// Start broadcasting
func (h *WSHub) Start() {
	ticker := time.NewTicker(1 * time.Second)
	go func() {
		for range ticker.C {
			h.broadcast()
		}
	}()
}

func (h *WSHub) broadcast() {
	data := getNodeStatusData()
	if data == nil {
		return
	}
	msg, _ := json.Marshal(data)

	h.mu.Lock()
	defer h.mu.Unlock()
	for conn := range h.clients {
		if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			conn.Close()
			delete(h.clients, conn)
		}
	}
}

// NodeStatusWS — GET /api/v1/system/node/status_ws
func NodeStatusWS(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	hub.mu.Lock()
	hub.clients[conn] = true
	hub.mu.Unlock()

	log.Printf("[WS] Client connected, total: %d", len(hub.clients))

	// Gửi ngay dữ liệu hiện tại
	data := getNodeStatusData()
	if data != nil {
		msg, _ := json.Marshal(data)
		conn.WriteMessage(websocket.TextMessage, msg)
	}

	// Giữ connection mở, đọc để detect disconnect
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			hub.mu.Lock()
			delete(hub.clients, conn)
			hub.mu.Unlock()
			conn.Close()
			break
		}
	}
}

// getNodeStatusData trả về dữ liệu node status (dùng chung cho WS và REST)
func getNodeStatusData() []NodeStatusResponse {
	var groups []models.DeviceGroup
	database.DB.Order("id ASC").Find(&groups)

	var result []NodeStatusResponse
	for _, g := range groups {
		var nodes []models.NodeClient
		database.DB.Where("group_id = ?", g.ID).Find(&nodes)

		var servers []ServerInfo
		for _, n := range nodes {
			online := n.LastSeen > 0 && (n.LastSeen+60) > time.Now().Unix()
			servers = append(servers, ServerInfo{
				Name:   n.Name,
				Handle: n.Handle,
				Online: online,
				IP4Geo: n.IP4Geo,
				IP4:    n.IP4,
				IP6Geo: n.IP6Geo,
				IP6:    n.IP6,
				SystemState: map[string]interface{}{
					"cpu": n.CPU, "mem_used": n.MemUsed, "disk_used": n.DiskUsed,
					"net_in_transfer": n.NetInTransfer, "net_out_transfer": n.NetOutTransfer,
					"net_in_speed": n.NetInSpeed, "net_out_speed": n.NetOutSpeed,
					"uptime": n.Uptime, "load1": n.Load1,
					"tcp_conn_count": n.TCPConnCount, "udp_conn_count": n.UDPConnCount,
					"process_count": n.ProcessCount,
				},
				SystemInfo: map[string]interface{}{
					"platform": n.Platform, "platform_version": n.PlatformVersion,
					"cpu": []string{n.CPUModel}, "mem_total": n.MemTotal,
					"disk_total": n.DiskTotal, "arch": n.Arch,
					"boot_time": n.BootTime, "hostname": n.Hostname, "version": "nc20260701",
				},
				LastSeen: n.LastSeen, LastPull: n.LastPull, Weight: n.Weight,
			})
		}
		if servers == nil {
			servers = []ServerInfo{}
		}
		result = append(result, NodeStatusResponse{
			Name: g.Name, GID: g.ID, GType: string(g.Type), HideStatus: g.HideStatus, Servers: servers,
		})
	}
	if result == nil {
		result = []NodeStatusResponse{}
	}
	return result
}
