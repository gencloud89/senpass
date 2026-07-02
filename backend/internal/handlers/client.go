package handlers

import (
	"crypto/md5"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"nyanpass-backend/internal/database"
	"nyanpass-backend/internal/models"
)

// ========== Structs giống web mẫu 100% ==========

type ConfigV2Response struct {
	ID            uint64           `json:"id"`
	Name          string           `json:"name"`
	Type          string           `json:"type"`
	Ratio         string           `json:"ratio"`
	Config        string           `json:"config"`
	GroupUUID     string           `json:"group_uuid"`
	FallbackGroup uint64           `json:"fallback_group,omitempty"` // Chỉ Outbound có (web mẫu: Inbound không có field này)
	Chain         interface{}      `json:"chain"`
	Users         []ConfigV2User   `json:"users"`
	Remotes       []ConfigV2Remote `json:"remotes"`
	Rules         []ConfigV2Rule   `json:"rules"`
}

type ConfigV2User struct {
	UID uint64 `json:"uid"`
}

type ConfigV2Remote struct {
	GroupUUID   string              `json:"group_uuid"`
	DeviceGroup ConfigV2DG          `json:"device_group"`
	Infos       interface{}         `json:"infos"` // []ConfigV2Server HOẶC []ConfigV2ServerLight
}

type ConfigV2DG struct {
	ID            uint64 `json:"id"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	Ratio         string `json:"ratio"`
	Config        string `json:"config"`
	FallbackGroup uint64 `json:"fallback_group,omitempty"`
}

// ConfigV2Server — đầy đủ IP + ports (dùng cho inbound remotes)
type ConfigV2Server struct {
	UUID       string `json:"u"`
	IP4        string `json:"ip4"`
	Version    int    `json:"v"`
	Weight     int    `json:"w"`
	DirectPort int    `json:"direct_port"`
	WSPort     int    `json:"ws_port"`
	TLSPort    int    `json:"tls_port"`
	UDPPort    int    `json:"udp_port"`
}

// ConfigV2ServerLight — chỉ UUID + version + weight (dùng cho outbound remotes)
type ConfigV2ServerLight struct {
	UUID    string `json:"u"`
	Version int    `json:"v"`
	Weight  int    `json:"w"`
}

type ConfigV2Rule struct {
	ID             uint64 `json:"id"`
	UID            uint64 `json:"uid"`
	ListenPort     int    `json:"listen_port"`
	DeviceGroupIn  uint64 `json:"device_group_in"`
	DeviceGroupOut uint64 `json:"device_group_out"`
	Config         string `json:"config"`
}

// genGroupUUID tạo UUID cố định dựa trên GID (giống web mẫu dùng UUID v5)
func genGroupUUID(gid uint64) string {
	ns := uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
	data := []byte(fmt.Sprintf("nyanpass-device-group-%d", gid))
	u := uuid.NewHash(md5.New(), ns, data, 5)
	_ = u
	return u.String()
}

// genServerUUID tạo UUID cố định dựa trên handle
func genServerUUID(handle string) string {
	ns := uuid.MustParse("6ba7b811-9dad-11d1-80b4-00c04fd430c8")
	return uuid.NewSHA1(ns, []byte("nyanpass-server-"+handle)).String()
}

// isOutboundType kiểm tra group có phải outbound không
func isOutboundType(t models.DeviceGroupType) bool {
	return t == models.DeviceGroupTypeOutboundBySite ||
		t == models.DeviceGroupTypeOutboundByUser ||
		t == models.DeviceGroupTypeChainOutbound
}

// GetClientConfigV2 — GET /api/v1/client/config_v2?token=TOKEN
func GetClientConfigV2(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "无效访问"})
		return
	}

	// 1. Tìm node client hoặc device group theo token
	var dg models.DeviceGroup
	if err := database.DB.Where("token = ?", token).First(&dg).Error; err != nil {
		// Thử tìm trong node_clients
		var nc models.NodeClient
		if err2 := database.DB.Where("token = ?", token).First(&nc).Error; err2 != nil {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "无效访问"})
			return
		}
		// Tìm device group của node client
		if err := database.DB.Where("id = ?", nc.GroupID).First(&dg).Error; err != nil {
			c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "无效访问"})
			return
		}
	}

	// 2. Base response — name để rỗng giống web mẫu (bảo mật token)
	resp := ConfigV2Response{
		ID:        dg.ID,
		Name:      "", // Web mẫu luôn để rỗng
		Type:      string(dg.Type),
		Ratio:     dg.Ratio,
		Config:    dg.Config,
		GroupUUID: genGroupUUID(dg.ID),
		Chain:     map[string]interface{}{},
		Users:     []ConfigV2User{},
		Remotes:   []ConfigV2Remote{},
		Rules:     []ConfigV2Rule{},
	}

	switch dg.Type {
	case models.DeviceGroupTypeInbound:
		buildInboundConfig(&resp, dg)
	case models.DeviceGroupTypeOutboundBySite,
		models.DeviceGroupTypeOutboundByUser,
		models.DeviceGroupTypeChainOutbound:
		buildOutboundConfig(&resp, dg)
	case models.DeviceGroupTypeAgentOnly:
		// Chỉ trả về thông tin cơ bản (đã có trong resp)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": resp})
}

// buildInboundConfig — Inbound group: rules + remotes (outbound groups) + users
func buildInboundConfig(resp *ConfigV2Response, dg models.DeviceGroup) {
	// 3. Tìm tất cả forward rules cho group này
	var rules []models.ForwardRule
	database.DB.Where("device_group_in = ?", dg.ID).Order("id ASC").Find(&rules)

	uidSet := make(map[uint64]bool)
	for _, r := range rules {
		resp.Rules = append(resp.Rules, ConfigV2Rule{
			ID:             r.ID,
			UID:            r.UID,
			ListenPort:     r.ListenPort,
			DeviceGroupIn:  r.DeviceGroupIn,
			DeviceGroupOut: r.DeviceGroupOut,
			Config:         r.Config,
		})
		uidSet[r.UID] = true
	}
	for uid := range uidSet {
		resp.Users = append(resp.Users, ConfigV2User{UID: uid})
	}

	// 4. Tìm các outbound group duy nhất được dùng bởi các rules
	outGroupIDs := make(map[uint64]bool)
	for _, r := range rules {
		outGroupIDs[r.DeviceGroupOut] = true
	}

	for ogID := range outGroupIDs {
		var og models.DeviceGroup
		if err := database.DB.Where("id = ?", ogID).First(&og).Error; err != nil {
			continue
		}

		remote := ConfigV2Remote{
			GroupUUID: genGroupUUID(og.ID),
			DeviceGroup: ConfigV2DG{
				ID:            og.ID,
				Name:          "", // Web mẫu luôn để rỗng
				Type:          string(og.Type),
				Ratio:         og.Ratio,
				Config:        og.Config,
				FallbackGroup: og.FallbackGroup,
			},
			Infos: nil, // null khi không có server — giống web mẫu
		}

		// 5. Lấy danh sách server trong outbound group (đầy đủ IP + ports)
		var servers []models.NodeClient
		database.DB.Where("group_id = ?", og.ID).Find(&servers)

		var fullInfos []ConfigV2Server
		for _, s := range servers {
			if s.IP4 == "" || s.Handle == "" {
				continue
			}
			// Port mapping từ handle hash (giống node client)
			h := md5.Sum([]byte(s.Handle))
			fullInfos = append(fullInfos, ConfigV2Server{
				UUID:       genServerUUID(s.Handle),
				IP4:        s.IP4,
				Version:    4,
				Weight:     s.Weight,
				DirectPort: 10000 + int(h[0])*256 + int(h[1]),
				WSPort:     20000 + (int(h[2])*256 + int(h[3])) % (65535 - 20000),
				TLSPort:    3000 + int(h[4])*256 + int(h[5]),
				UDPPort:    40000 + (int(h[6])*256 + int(h[7])) % (65535 - 40000),
			})
		}

		if len(fullInfos) > 0 {
			remote.Infos = fullInfos
		}
		// Luôn thêm remote dù không có server (giống web mẫu)
		resp.Remotes = append(resp.Remotes, remote)
	}

	// 6. Chain outbounds
	var chains []models.ChainOutbound
	database.DB.Where("group_id = ?", dg.ID).Order("seq ASC").Find(&chains)
	if len(chains) > 0 {
		resp.Chain = chains
	}
}

// buildOutboundConfig — Outbound group: rules + remotes (inbound groups) + users + fallback
func buildOutboundConfig(resp *ConfigV2Response, dg models.DeviceGroup) {
	// 1. Fallback group ở top level
	resp.FallbackGroup = dg.FallbackGroup

	// 2. Tìm TẤT CẢ forward rules trỏ đến outbound này
	var rules []models.ForwardRule
	database.DB.Where("device_group_out = ?", dg.ID).Order("id ASC").Find(&rules)

	uidSet := make(map[uint64]bool)
	inGroupIDs := make(map[uint64]bool)

	for _, r := range rules {
		resp.Rules = append(resp.Rules, ConfigV2Rule{
			ID:             r.ID,
			UID:            r.UID,
			ListenPort:     r.ListenPort,
			DeviceGroupIn:  r.DeviceGroupIn,
			DeviceGroupOut: r.DeviceGroupOut,
			Config:         r.Config,
		})
		uidSet[r.UID] = true
		inGroupIDs[r.DeviceGroupIn] = true
	}

	for uid := range uidSet {
		resp.Users = append(resp.Users, ConfigV2User{UID: uid})
	}

	// 3. Tìm các inbound group có rules trỏ đến outbound này
	for igID := range inGroupIDs {
		var ig models.DeviceGroup
		if err := database.DB.Where("id = ?", igID).First(&ig).Error; err != nil {
			continue
		}

		remote := ConfigV2Remote{
			GroupUUID: genGroupUUID(ig.ID),
			DeviceGroup: ConfigV2DG{
				ID:     ig.ID,
				Name:   "", // Web mẫu luôn để rỗng
				Type:   string(ig.Type),
				Ratio:  ig.Ratio,
				Config: ig.Config,
			},
			Infos: []ConfigV2ServerLight{}, // Mặc định mảng rỗng
		}

		// 4. Lấy server trong inbound group (CHỈ u, v, w — không IP, không ports)
		var servers []models.NodeClient
		database.DB.Where("group_id = ?", ig.ID).Find(&servers)

		var lightInfos []ConfigV2ServerLight
		for _, s := range servers {
			if s.Handle == "" {
				continue
			}
			lightInfos = append(lightInfos, ConfigV2ServerLight{
				UUID:    genServerUUID(s.Handle),
				Version: 4,
				Weight:  s.Weight,
			})
		}

		if len(lightInfos) > 0 {
			remote.Infos = lightInfos
		} else {
			remote.Infos = nil // null trong JSON nếu không có server
		}

		resp.Remotes = append(resp.Remotes, remote)
	}

	// 5. Chain outbounds
	var chains []models.ChainOutbound
	database.DB.Where("group_id = ?", dg.ID).Order("seq ASC").Find(&chains)
	if len(chains) > 0 {
		resp.Chain = chains
	}

	// 6. Thêm allowed_in / allowed_out vào config nếu có
	// Web mẫu không gửi allowed_in trong config_v2, nên ta giữ nguyên
}

// contains giúp kiểm tra chuỗi trong danh sách
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
