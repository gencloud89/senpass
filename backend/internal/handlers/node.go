package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"nyanpass-backend/internal/database"
	"nyanpass-backend/internal/models"
)

type NodeHandler struct{}

func NewNodeHandler() *NodeHandler { return &NodeHandler{} }

// NodeStatusResponse là response cho node status
type NodeStatusResponse struct {
	Name       string       `json:"name"`
	GID        uint64       `json:"gid"`
	GType      string       `json:"gType"`
	HideStatus int          `json:"hide_status,omitempty"`
	Servers    []ServerInfo `json:"servers"`
}

type ServerInfo struct {
	Name        string                 `json:"name"`
	Handle      string                 `json:"handle"`
	Online      bool                   `json:"online"`
	IP4Geo      string                 `json:"ip4_geo,omitempty"`
	IP4         string                 `json:"ip4,omitempty"`
	IP6Geo      string                 `json:"ip6_geo,omitempty"`
	IP6         string                 `json:"ip6,omitempty"`
	SystemState map[string]interface{} `json:"system_state,omitempty"`
	SystemInfo  map[string]interface{} `json:"system_info,omitempty"`
	LastSeen    int64                  `json:"last_seen"`
	LastPull    int64                  `json:"last_pull,omitempty"`
	Weight      int                    `json:"weight,omitempty"`
}

// GetNodeStatus — GET /api/v1/system/node/status
func (h *NodeHandler) GetNodeStatus(c *gin.Context) {
	var groups []models.DeviceGroup
	database.DB.Order("id ASC").Find(&groups)

	var result []NodeStatusResponse
	for _, g := range groups {
		// Lấy danh sách node client đang kết nối với group này
		var nodes []models.NodeClient
		database.DB.Where("group_id = ?", g.ID).Find(&nodes)

		var servers []ServerInfo
		for _, n := range nodes {
			online := n.LastSeen > 0 && (n.LastSeen+60) > currentUnix()
			servers = append(servers, ServerInfo{
				Name:   n.Name,
				Handle: n.Handle,
				Online: online,
				IP4Geo: n.IP4Geo,
				IP4:    n.IP4,
				IP6Geo: n.IP6Geo,
				IP6:    n.IP6,
				SystemState: map[string]interface{}{
					"cpu":             n.CPU,
					"mem_used":        n.MemUsed,
					"disk_used":       n.DiskUsed,
					"net_in_transfer": n.NetInTransfer,
					"net_out_transfer": n.NetOutTransfer,
					"net_in_speed":    n.NetInSpeed,
					"net_out_speed":   n.NetOutSpeed,
					"uptime":          n.Uptime,
					"load1":           n.Load1,
					"load5":           n.Load5,
					"load15":          n.Load15,
					"tcp_conn_count":  n.TCPConnCount,
					"udp_conn_count":  n.UDPConnCount,
					"process_count":   n.ProcessCount,
				},
				SystemInfo: map[string]interface{}{
					"platform":         n.Platform,
					"platform_version":  n.PlatformVersion,
					"cpu":              []string{n.CPUModel},
					"mem_total":        n.MemTotal,
					"disk_total":       n.DiskTotal,
					"arch":             n.Arch,
					"boot_time":        n.BootTime,
					"hostname":         n.Hostname,
					"version":          "nc20260701",
				},
				LastSeen: n.LastSeen,
				LastPull: n.LastPull,
				Weight:   n.Weight,
			})
		}
		if servers == nil {
			servers = []ServerInfo{}
		}

		result = append(result, NodeStatusResponse{
			Name:       g.Name,
			GID:        g.ID,
			GType:      string(g.Type),
			HideStatus: g.HideStatus,
			Servers:    servers,
		})
	}
	if result == nil {
		result = []NodeStatusResponse{}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result, "msg": ""})
}

// NodeReportData là dữ liệu server gửi từ node client
type NodeReportData struct {
	Name           string  `json:"name"`
	Handle         string  `json:"handle"`
	IP4            string  `json:"ip4"`
	IP4Geo         string  `json:"ip4_geo,omitempty"`
	IP6            string  `json:"ip6,omitempty"`
	IP6Geo         string  `json:"ip6_geo,omitempty"`
	CPU            float64 `json:"cpu"`
	CPUModel       string  `json:"cpu_model"`
	MemUsed        uint64  `json:"mem_used"`
	MemTotal       uint64  `json:"mem_total"`
	DiskUsed       uint64  `json:"disk_used"`
	DiskTotal      uint64  `json:"disk_total"`
	NetInSpeed     uint64  `json:"net_in_speed"`
	NetOutSpeed    uint64  `json:"net_out_speed"`
	NetInTransfer  uint64  `json:"net_in_transfer"`
	NetOutTransfer uint64  `json:"net_out_transfer"`
	Uptime         uint64  `json:"uptime"`
	Load1          float64 `json:"load1"`
	Load5          float64 `json:"load5"`
	Load15         float64 `json:"load15"`
	TCPConnCount   int     `json:"tcp_conn_count"`
	UDPConnCount   int     `json:"udp_conn_count"`
	ProcessCount   int     `json:"process_count"`
	Platform       string  `json:"platform"`
	PlatformVersion string `json:"platform_version"`
	Arch           string  `json:"arch"`
	BootTime       int64   `json:"boot_time"`
	Hostname       string  `json:"hostname"`
}

// NodeReport — POST /api/v1/node/report (nhận báo cáo từ node client)
func (h *NodeHandler) NodeReport(c *gin.Context) {
	var req struct {
		Token  string         `json:"token"`
		Server NodeReportData `json:"server"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	// Tìm device group theo token
	var dg models.DeviceGroup
	if err := database.DB.Where("token = ?", req.Token).First(&dg).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "Token không hợp lệ"})
		return
	}

	// Upsert node client (tìm record hiện có trước để check geo cache)
	var nc models.NodeClient
	database.DB.Where("handle = ?", req.Server.Handle).First(&nc)

	// Geo IP lookup — dùng cache từ DB nếu IP không thay đổi, tránh gọi HTTP mỗi 2s
	geo := req.Server.IP4Geo
	if geo == "" {
		if nc.IP4 == req.Server.IP4 && nc.IP4Geo != "" {
			geo = nc.IP4Geo // Dùng lại giá trị đã cache trong DB
		} else {
			geo = lookupGeoIP(req.Server.IP4) // Chỉ lookup khi IP mới hoặc chưa có cache
		}
	}
	nc.GroupID = dg.ID
	nc.Token = req.Token
	nc.Name = req.Server.Name
	nc.Handle = req.Server.Handle
	nc.IP4 = req.Server.IP4
	nc.IP4Geo = geo
	nc.IP6 = req.Server.IP6
	if req.Server.IP6Geo != "" {
		nc.IP6Geo = req.Server.IP6Geo
	}
	nc.CPU = req.Server.CPU
	nc.CPUModel = req.Server.CPUModel
	nc.MemUsed = req.Server.MemUsed
	nc.MemTotal = req.Server.MemTotal
	nc.DiskUsed = req.Server.DiskUsed
	nc.DiskTotal = req.Server.DiskTotal
	nc.NetInSpeed = req.Server.NetInSpeed
	nc.NetOutSpeed = req.Server.NetOutSpeed
	nc.NetInTransfer = req.Server.NetInTransfer
	nc.NetOutTransfer = req.Server.NetOutTransfer
	nc.Uptime = req.Server.Uptime
	nc.Load1 = req.Server.Load1
	nc.Load5 = req.Server.Load5
	nc.Load15 = req.Server.Load15
	nc.TCPConnCount = req.Server.TCPConnCount
	nc.UDPConnCount = req.Server.UDPConnCount
	nc.ProcessCount = req.Server.ProcessCount
	nc.Platform = req.Server.Platform
	nc.PlatformVersion = req.Server.PlatformVersion
	nc.Arch = req.Server.Arch
	nc.BootTime = req.Server.BootTime
	nc.Hostname = req.Server.Hostname
	nc.LastSeen = currentUnix()
	nc.LastPull = currentUnix()

	if err := database.DB.Save(&nc).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "Báo cáo thành công"})
}

// SetWeight — PUT /api/v1/system/node/weight/:gid/:handle
func (h *NodeHandler) SetWeight(c *gin.Context) {
	handle := c.Param("handle")
	var req struct {
		Value string `json:"value"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}
	weight := 0
	if req.Value != "" {
		if _, err := fmt.Sscanf(req.Value, "%d", &weight); err != nil {
			weight = 0
		}
	}
	if err := database.DB.Model(&models.NodeClient{}).Where("handle = ?", handle).Update("weight", weight).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "Cập nhật thành công"})
}

// CreateTerminal — POST /api/v1/system/node/terminal/:handle
func (h *NodeHandler) CreateTerminal(c *gin.Context) {
	handle := c.Param("handle")
	sessionID := fmt.Sprintf("term_%s_%d", handle[:8], currentUnix())
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": sessionID})
}

// KickServer — POST /api/v1/system/node/kick/:handle
func (h *NodeHandler) KickServer(c *gin.Context) {
	handle := c.Param("handle")
	if handle == "offline" {
		database.DB.Where("last_seen < ?", currentUnix()-120).Delete(&models.NodeClient{})
	} else {
		database.DB.Where("handle = ?", handle).Delete(&models.NodeClient{})
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "Đã xóa"})
}

// lookupGeoIP tra cứu mã quốc gia từ IP qua ip-api.com (miễn phí)
func lookupGeoIP(ip string) string {
	if ip == "" || ip == "127.0.0.1" || strings.HasPrefix(ip, "10.") || strings.HasPrefix(ip, "192.168.") {
		return ""
	}
	resp, err := http.Get("https://ip-api.com/json/" + ip + "?fields=countryCode")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	var result struct {
		CountryCode string `json:"countryCode"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return ""
	}
	return result.CountryCode
}

func currentUnix() int64 {
	return database.DB.NowFunc().Unix()
}
