package main

import (
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	token   = flag.String("t", "", "Device group token")
	panel   = flag.String("u", "https://tfd2.clonod.top", "Panel URL")
	svcName = flag.String("s", "nyanpass", "Service name")
)

// ========== Config V2 Structs (đồng bộ backend) ==========

type ConfigV2Response struct {
	ID            uint64           `json:"id"`
	Name          string           `json:"name"`
	Type          string           `json:"type"`
	Config        string           `json:"config"`
	GroupUUID     string           `json:"group_uuid"`
	FallbackGroup uint64           `json:"fallback_group"`
	Chain         interface{}      `json:"chain"`
	Users         []ConfigV2User   `json:"users"`
	Remotes       []ConfigV2Remote `json:"remotes"`
	Rules         []ConfigV2Rule   `json:"rules"`
}

type ConfigV2User struct {
	UID uint64 `json:"uid"`
}

type ConfigV2Remote struct {
	GroupUUID   string      `json:"group_uuid"`
	DeviceGroup ConfigV2DG  `json:"device_group"`
	Infos       interface{} `json:"infos"`
}

type ConfigV2DG struct {
	ID            uint64 `json:"id"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	Config        string `json:"config"`
	FallbackGroup uint64 `json:"fallback_group,omitempty"`
}

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

// ========== Rule Config Parsed ==========

type RuleConfig struct {
	Dest          []string `json:"dest"`
	LBPolicy      string   `json:"lb_policy,omitempty"`
	ProxyProtocol int      `json:"proxy_protocol,omitempty"` // 0, 1 (v1), 2 (v2)
}

// ========== DG Config Parsed ==========

type DGConfig struct {
	Protocol        string `json:"protocol"`
	UdpSmartBind    bool   `json:"udp_smart_bind"`
	DirectPolicy    int    `json:"direct_policy"`    // 1=optional, 2=force
	BlockedProtocol string `json:"blocked_protocol"`
	BlockedHost     string `json:"blocked_host"`
	BlockedPath     string `json:"blocked_path"`
}

// ========== Global State ==========

var (
	handle     string
	configV2   *ConfigV2Response
	configMu   sync.RWMutex
	handleFile string

	// Health tracking
	health     = &HealthTracker{healthy: make(map[string]bool), lastCheck: make(map[string]time.Time)}

	// Connection tracking for least_conn LB
	connTracker = &ConnTracker{counts: make(map[string]int64)}

	// Round-robin counters per group
	rrCounters   = make(map[uint64]int)
	rrCountersMu sync.Mutex

	// Traffic counting
	trafficCounters   = make(map[uint64]*RuleTraffic)
	trafficCountersMu sync.Mutex

	// Own GID (from config)
	ownGID uint64
)

type HealthTracker struct {
	mu        sync.RWMutex
	healthy   map[string]bool
	lastCheck map[string]time.Time
}

func (h *HealthTracker) IsHealthy(key string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	v, ok := h.healthy[key]
	return ok && v
}

func (h *HealthTracker) SetHealthy(key string, ok bool) {
	h.mu.Lock()
	h.healthy[key] = ok
	h.lastCheck[key] = time.Now()
	h.mu.Unlock()
}

type ConnTracker struct {
	mu     sync.Mutex
	counts map[string]int64
}

func (c *ConnTracker) Inc(key string) {
	c.mu.Lock()
	c.counts[key]++
	c.mu.Unlock()
}

func (c *ConnTracker) Dec(key string) {
	c.mu.Lock()
	c.counts[key]--
	c.mu.Unlock()
}

func (c *ConnTracker) Count(key string) int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.counts[key]
}

type RuleTraffic struct {
	BytesIn  uint64
	BytesOut uint64
}

// ========== Main ==========

func main() {
	flag.Parse()
	if *token == "" {
		log.Fatal("Thiếu token! Dùng: -t <token> -u <panel_url>")
	}

	handle = loadOrGenerateHandle()
	hostname, _ := os.Hostname()
	log.Printf("🚀 Node Client Tunnel v2 — Handle: %s, Hostname: %s, Panel: %s", handle, hostname, *panel)

	// Khởi động config puller
	go configPuller()

	// Đợi config đầu tiên
	log.Printf("⏳ Đang chờ config từ panel...")
	for i := 0; i < 30; i++ {
		configMu.RLock()
		hasConfig := configV2 != nil
		configMu.RUnlock()
		if hasConfig {
			break
		}
		time.Sleep(1 * time.Second)
	}

	configMu.RLock()
	cv2 := configV2
	configMu.RUnlock()

	if cv2 == nil {
		log.Printf("⚠️ Không lấy được config sau 30s, vẫn tiếp tục monitor...")
	} else {
		ownGID = cv2.ID
		log.Printf("✅ Config: Type=%s Name=%s GID=%d Rules=%d Remotes=%d Fallback=%d",
			cv2.Type, cv2.Name, cv2.ID, len(cv2.Rules), len(cv2.Remotes), cv2.FallbackGroup)

		// Khởi động server dựa trên loại
		if cv2.Type == "DeviceGroupType_Inbound" {
			go startInboundServers()
			go healthChecker()
		} else if isOutboundType(cv2.Type) {
			go startOutboundServers()
		}
	}

	// Monitor loop
	for {
		report := collectReport(hostname, handle)
		if err := sendReport(report); err != nil {
			log.Printf("❌ Lỗi gửi báo cáo: %v", err)
		}
		time.Sleep(2 * time.Second)
	}
}

func isOutboundType(t string) bool {
	return strings.Contains(t, "Outbound") || strings.Contains(t, "ChainOutbound")
}

// ========== Config Puller ==========

func configPuller() {
	for {
		newConfig, err := fetchConfigV2()
		if err != nil {
			configMu.RLock()
			hasConfig := configV2 != nil
			configMu.RUnlock()
			if !hasConfig {
				log.Printf("⚠️ Chưa có config, lỗi fetch: %v", err)
			}
			time.Sleep(5 * time.Second)
			continue
		}

		configMu.Lock()
		oldConfig := configV2
		configV2 = newConfig
		configMu.Unlock()

		if oldConfig == nil {
			log.Printf("📋 Config initial: %d rules, %d remotes", len(newConfig.Rules), len(newConfig.Remotes))
		} else if len(oldConfig.Rules) != len(newConfig.Rules) || len(oldConfig.Remotes) != len(newConfig.Remotes) {
			log.Printf("📋 Config updated: %d→%d rules, %d→%d remotes",
				len(oldConfig.Rules), len(newConfig.Rules),
				len(oldConfig.Remotes), len(newConfig.Remotes))
		}

		time.Sleep(30 * time.Second)
	}
}

func fetchConfigV2() (*ConfigV2Response, error) {
	resp, err := http.Get(*panel + "/api/v1/client/config_v2?token=" + *token)
	if err != nil {
		return nil, fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Code int              `json:"code"`
		Data ConfigV2Response `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("json decode: %w", err)
	}
	if result.Code != 0 {
		return nil, fmt.Errorf("api code=%d", result.Code)
	}
	return &result.Data, nil
}

// ========== Server Info Extraction ==========

// extractServers chuyển infos (interface{}) thành []ConfigV2Server
func extractServers(infos interface{}) []ConfigV2Server {
	if infos == nil {
		return nil
	}
	data, _ := json.Marshal(infos)
	var servers []ConfigV2Server
	json.Unmarshal(data, &servers)
	return servers
}

// ========== Health Checker ==========

func healthChecker() {
	ticker := time.NewTicker(10 * time.Second)
	for range ticker.C {
		configMu.RLock()
		cv2 := configV2
		configMu.RUnlock()
		if cv2 == nil {
			continue
		}

		for _, remote := range cv2.Remotes {
			for _, s := range extractServers(remote.Infos) {
				if s.IP4 == "" {
					continue
				}
				go func(server ConfigV2Server) {
					key := fmt.Sprintf("%s:%d", server.IP4, server.TLSPort)
					conn, err := tls.DialWithDialer(
						&net.Dialer{Timeout: 5 * time.Second},
						"tcp",
						key,
						&tls.Config{InsecureSkipVerify: true},
					)
					if err != nil {
						health.SetHealthy(key, false)
						return
					}
					conn.Close()
					health.SetHealthy(key, true)
				}(s)
			}
		}
	}
}

// ========== Load Balancer ==========

type LBPolicy int

const (
	LBRandom         LBPolicy = iota
	LBRoundRobin
	LBIPHash
	LBLeastConn
	LBFailover
)

func parseLBPolicy(p string) LBPolicy {
	switch strings.ToLower(p) {
	case "round_robin":
		return LBRoundRobin
	case "ip_hash":
		return LBIPHash
	case "least_connections", "least_conn":
		return LBLeastConn
	case "failover":
		return LBFailover
	default:
		return LBRandom
	}
}

// selectServer chọn server từ danh sách theo policy
func selectServer(servers []ConfigV2Server, policy LBPolicy, groupID uint64, clientIP string) *ConfigV2Server {
	if len(servers) == 0 {
		return nil
	}

	// Lọc server healthy
	var healthy []ConfigV2Server
	for _, s := range servers {
		key := fmt.Sprintf("%s:%d", s.IP4, s.TLSPort)
		if health.IsHealthy(key) || len(servers) == 1 {
			healthy = append(healthy, s)
		}
	}
	// Nếu không có server nào healthy, dùng tất cả
	if len(healthy) == 0 {
		healthy = servers
	}

	switch policy {
	case LBRoundRobin:
		rrCountersMu.Lock()
		idx := rrCounters[groupID]
		rrCounters[groupID] = (idx + 1) % len(healthy)
		rrCountersMu.Unlock()
		// Weight-aware: nếu weight > 1, skip weight-1 lần
		return &healthy[idx%len(healthy)]

	case LBIPHash:
		h := sha1.Sum([]byte(fmt.Sprintf("%s-%d", clientIP, groupID)))
		idx := int(h[0])<<8 | int(h[1])
		return &healthy[idx%len(healthy)]

	case LBLeastConn:
		var best *ConfigV2Server
		var bestCount int64 = 1<<63 - 1
		for i := range healthy {
			key := fmt.Sprintf("%s:%d", healthy[i].IP4, healthy[i].TLSPort)
			c := connTracker.Count(key)
			w := int64(healthy[i].Weight)
			if w <= 0 {
				w = 1
			}
			ratio := c / w
			if ratio < bestCount {
				bestCount = ratio
				best = &healthy[i]
			}
		}
		return best

	case LBFailover:
		// Trả về server healthy đầu tiên
		return &healthy[0]

	default: // Random
		// Weighted random
		totalWeight := 0
		for _, s := range healthy {
			w := s.Weight
			if w <= 0 {
				w = 1
			}
			totalWeight += w
		}
		r := int(time.Now().UnixNano()) % totalWeight
		for i := range healthy {
			w := healthy[i].Weight
			if w <= 0 {
				w = 1
			}
			r -= w
			if r < 0 {
				return &healthy[i]
			}
		}
		return &healthy[0]
	}
}

// selectDest chọn destination từ danh sách
func selectDest(dests []string, policy LBPolicy, ruleID uint64) string {
	if len(dests) == 0 {
		return ""
	}
	if len(dests) == 1 {
		return dests[0]
	}

	switch policy {
	case LBRoundRobin:
		rrCountersMu.Lock()
		idx := rrCounters[ruleID]
		rrCounters[ruleID] = (idx + 1) % len(dests)
		rrCountersMu.Unlock()
		return dests[idx%len(dests)]
	case LBIPHash:
		h := sha1.Sum([]byte(fmt.Sprintf("%d", ruleID)))
		return dests[int(h[0])%len(dests)]
	default:
		return dests[int(time.Now().UnixNano())%len(dests)]
	}
}

// ========== Protocol Dialers ==========

// dialOutbound kết nối tới outbound server qua protocol phù hợp
func dialOutbound(server ConfigV2Server) (net.Conn, error) {
	configMu.RLock()
	cv2 := configV2
	configMu.RUnlock()

	// Parse DG config để lấy protocol
	var dgCfg DGConfig
	if cv2 != nil {
		// Tìm remote group để lấy config
		for _, remote := range cv2.Remotes {
			json.Unmarshal([]byte(remote.DeviceGroup.Config), &dgCfg)
			_ = dgCfg
		}
	}

	// Mặc định dùng TLS
	return dialTLS(server)
}

func dialTLS(server ConfigV2Server) (net.Conn, error) {
	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS12,
	}
	return tls.Dial("tcp", fmt.Sprintf("%s:%d", server.IP4, server.TLSPort), tlsConf)
}

func dialDirectTunnel(dest string) (net.Conn, error) {
	return net.DialTimeout("tcp", dest, 10*time.Second)
}

// ========== Inbound Server ==========

func startInboundServers() {
	log.Printf("🔌 [INBOUND] Khởi động inbound servers...")

	listeners := make(map[int]net.Listener)
	var lk sync.Mutex

	reload := func() {
		configMu.RLock()
		cv2 := configV2
		configMu.RUnlock()
		if cv2 == nil {
			return
		}

		// Parse inbound DG config (udp_smart_bind, direct_policy)
		var dgCfg DGConfig
		json.Unmarshal([]byte(cv2.Config), &dgCfg)

		// Build set of desired ports
		desiredPorts := make(map[int][]ConfigV2Rule)
		for _, r := range cv2.Rules {
			desiredPorts[r.ListenPort] = append(desiredPorts[r.ListenPort], r)
		}

		lk.Lock()
		defer lk.Unlock()

		// Đóng listener cho port không còn trong rules
		for port, ln := range listeners {
			if _, ok := desiredPorts[port]; !ok {
				log.Printf("🔌 [INBOUND] Đóng port %d (rule đã xoá)", port)
				ln.Close()
				delete(listeners, port)
			}
		}

		// Mở listener cho port mới
		for port, rules := range desiredPorts {
			if _, ok := listeners[port]; ok {
				continue
			}
			ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
			if err != nil {
				log.Printf("❌ [INBOUND] Không thể mở port %d: %v", port, err)
				continue
			}
			listeners[port] = ln
			log.Printf("✅ [INBOUND] Đang lắng nghe port %d (%d rules)", port, len(rules))

			go func(p int, rls []ConfigV2Rule, l net.Listener) {
				for {
					conn, err := l.Accept()
					if err != nil {
						if !strings.Contains(err.Error(), "closed") {
							log.Printf("❌ [INBOUND] Accept port %d: %v", p, err)
						}
						return
					}
					go handleInboundConn(conn, rls)
				}
			}(port, rules, ln)

			// UDP Smart Bind — mở UDP listener nếu inbound config có udp_smart_bind
			if dgCfg.UdpSmartBind {
				go func(p int, rls []ConfigV2Rule) {
					udpAddr, _ := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", p))
					udpConn, err := net.ListenUDP("udp", udpAddr)
					if err != nil {
						log.Printf("⚠️ [INBOUND] Không thể mở UDP port %d: %v", p, err)
						return
					}
					defer udpConn.Close()
					log.Printf("✅ [INBOUND] UDP Smart Bind port %d", p)
					buf := make([]byte, 65535)
					for {
						n, clientAddr, err := udpConn.ReadFromUDP(buf)
						if err != nil {
							return
						}
						go handleInboundUDP(udpConn, clientAddr, buf[:n], rls)
					}
				}(port, rules)
			}
		}
	}

	reload()

	ticker := time.NewTicker(60 * time.Second)
	for range ticker.C {
		reload()
	}
}

func handleInboundConn(clientConn net.Conn, rules []ConfigV2Rule) {
	defer clientConn.Close()

	if len(rules) == 0 {
		return
	}

	// SNI-based routing: tìm rule phù hợp (đơn giản: dùng rule đầu tiên)
	rule := rules[0]
	clientIP := clientConn.RemoteAddr().(*net.TCPAddr).IP.String()

	var rc RuleConfig
	json.Unmarshal([]byte(rule.Config), &rc)
	if len(rc.Dest) == 0 {
		log.Printf("❌ [INBOUND] Rule %d không có dest", rule.ID)
		return
	}

	// Parse inbound DG config
	configMu.RLock()
	cv2 := configV2
	configMu.RUnlock()

	var dgCfg DGConfig
	if cv2 != nil {
		json.Unmarshal([]byte(cv2.Config), &dgCfg)
	}

	// Direct Policy — bỏ qua outbound, kết nối thẳng đến đích
	if dgCfg.DirectPolicy == 2 {
		dest := selectDest(rc.Dest, parseLBPolicy(rc.LBPolicy), rule.ID)
		log.Printf("🔗 [INBOUND] Port %d → DIRECT → %s", rule.ListenPort, dest)
		destConn, err := dialDirectTunnel(dest)
		if err != nil {
			log.Printf("❌ [INBOUND] Direct %s: %v", dest, err)
			return
		}
		defer destConn.Close()
		bidirectionalCopy(clientConn, destConn, rule.ID)
		log.Printf("✅ [INBOUND] Kết thúc phiên DIRECT port %d", rule.ListenPort)
		return
	}

	// Tìm outbound server cho rule này
	outboundServer := findOutboundServer(rule, clientIP)
	if outboundServer == nil {
		// Thử fallback
		outboundServer = findFallbackServer(rule, clientIP)
		if outboundServer == nil {
			log.Printf("❌ [INBOUND] Không tìm thấy outbound server cho rule %d", rule.ID)
			return
		}
	}

	// Direct Policy optional (1): thử direct trước, nếu fail thì dùng outbound
	if dgCfg.DirectPolicy == 1 {
		dest := selectDest(rc.Dest, parseLBPolicy(rc.LBPolicy), rule.ID)
		destConn, err := dialDirectTunnel(dest)
		if err == nil {
			log.Printf("🔗 [INBOUND] Port %d → DIRECT (optional) → %s", rule.ListenPort, dest)
			defer destConn.Close()
			bidirectionalCopy(clientConn, destConn, rule.ID)
			return
		}
		log.Printf("⚠️ [INBOUND] Direct failed, chuyển sang tunnel: %v", err)
	}

	dest := selectDest(rc.Dest, parseLBPolicy(rc.LBPolicy), rule.ID)
	log.Printf("🔗 [INBOUND] Port %d → Tunnel TLS tới %s:%d → %s",
		rule.ListenPort, outboundServer.IP4, outboundServer.TLSPort, dest)

	// Tạo TLS tunnel tới outbound
	outboundConn, err := dialOutbound(*outboundServer)
	if err != nil {
		log.Printf("❌ [INBOUND] Không thể kết nối outbound %s:%d: %v",
			outboundServer.IP4, outboundServer.TLSPort, err)
		// Thử fallback
		outboundServer2 := findFallbackServer(rule, clientIP)
		if outboundServer2 != nil {
			outboundConn, err = dialOutbound(*outboundServer2)
			if err != nil {
				log.Printf("❌ [INBOUND] Fallback cũng thất bại: %v", err)
				return
			}
			log.Printf("🔄 [INBOUND] Đã chuyển sang fallback server %s:%d", outboundServer2.IP4, outboundServer2.TLSPort)
		} else {
			return
		}
	}
	defer outboundConn.Close()

	// Track connection
	serverKey := fmt.Sprintf("%s:%d", outboundServer.IP4, outboundServer.TLSPort)
	connTracker.Inc(serverKey)
	defer connTracker.Dec(serverKey)

	// Gửi header: 2 byte dest length + 2 byte inbound GID + dest
	destBytes := []byte(dest)
	header := make([]byte, 4+len(destBytes))
	header[0] = byte(len(destBytes) >> 8)
	header[1] = byte(len(destBytes) & 0xFF)
	header[2] = byte(ownGID >> 8)
	header[3] = byte(ownGID & 0xFF)
	copy(header[4:], destBytes)

	if _, err := outboundConn.Write(header); err != nil {
		log.Printf("❌ [INBOUND] Lỗi gửi header: %v", err)
		return
	}

	bidirectionalCopy(clientConn, outboundConn, rule.ID)
	log.Printf("✅ [INBOUND] Kết thúc phiên port %d", rule.ListenPort)
}

func findOutboundServer(rule ConfigV2Rule, clientIP string) *ConfigV2Server {
	configMu.RLock()
	cv2 := configV2
	configMu.RUnlock()

	for _, remote := range cv2.Remotes {
		if remote.DeviceGroup.ID == rule.DeviceGroupOut {
			servers := extractServers(remote.Infos)
			if len(servers) > 0 {
				return selectServer(servers, parseLBPolicy(""), rule.DeviceGroupOut, clientIP)
			}
		}
	}
	return nil
}

func findFallbackServer(rule ConfigV2Rule, clientIP string) *ConfigV2Server {
	configMu.RLock()
	cv2 := configV2
	configMu.RUnlock()

	for _, remote := range cv2.Remotes {
		if remote.DeviceGroup.ID == rule.DeviceGroupOut {
			if remote.DeviceGroup.FallbackGroup > 0 {
				// Tìm fallback group
				for _, fbRemote := range cv2.Remotes {
					if fbRemote.DeviceGroup.ID == remote.DeviceGroup.FallbackGroup {
						servers := extractServers(fbRemote.Infos)
						if len(servers) > 0 {
							log.Printf("🔄 [FALLBACK] Dùng fallback group %d cho rule %d", remote.DeviceGroup.FallbackGroup, rule.ID)
							return selectServer(servers, parseLBPolicy(""), remote.DeviceGroup.FallbackGroup, clientIP)
						}
					}
				}
			}
		}
	}
	return nil
}

// ========== Bidirectional Copy với Traffic Counting ==========

type countingReader struct {
	r       io.Reader
	counter *uint64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	atomic.AddUint64(c.counter, uint64(n))
	return n, err
}

func bidirectionalCopy(a, b net.Conn, ruleID uint64) {
	trafficCountersMu.Lock()
	if _, ok := trafficCounters[ruleID]; !ok {
		trafficCounters[ruleID] = &RuleTraffic{}
	}
	tc := trafficCounters[ruleID]
	trafficCountersMu.Unlock()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		cr := &countingReader{r: a, counter: &tc.BytesIn}
		io.Copy(b, cr)
		b.Close()
	}()

	go func() {
		defer wg.Done()
		cr := &countingReader{r: b, counter: &tc.BytesOut}
		io.Copy(a, cr)
		a.Close()
	}()

	wg.Wait()
}

// ========== Outbound Server ==========

func startOutboundServers() {
	log.Printf("🔌 [OUTBOUND] Khởi động outbound servers...")

	// Tính toán ports từ handle
	ports := computeServerPorts(handle)
	log.Printf("📋 [OUTBOUND] Ports: direct=%d ws=%d tls=%d udp=%d", ports.direct, ports.ws, ports.tls, ports.udp)

	// Tạo TLS certificate self-signed
	cert, err := generateSelfSignedCert()
	if err != nil {
		log.Fatalf("❌ [OUTBOUND] Không thể tạo TLS cert: %v", err)
	}

	tlsConf := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	// Lắng nghe TLS port
	tlsLn, err := tls.Listen("tcp", fmt.Sprintf(":%d", ports.tls), tlsConf)
	if err != nil {
		log.Fatalf("❌ [OUTBOUND] Không thể mở TLS port %d: %v", ports.tls, err)
	}
	log.Printf("✅ [OUTBOUND] TLS tunnel port %d", ports.tls)

	// WebSocket listener (plain TCP, upgrade to WS)
	wsLn, err := net.Listen("tcp", fmt.Sprintf(":%d", ports.ws))
	if err != nil {
		log.Printf("⚠️ [OUTBOUND] Không thể mở WS port %d: %v", ports.ws, err)
	} else {
		log.Printf("✅ [OUTBOUND] WS tunnel port %d", ports.ws)
		go func() {
			for {
				conn, err := wsLn.Accept()
				if err != nil {
					continue
				}
				go handleWSOutboundConn(conn)
			}
		}()
	}

	// Direct port (plain TCP tunnel)
	directLn, err := net.Listen("tcp", fmt.Sprintf(":%d", ports.direct))
	if err != nil {
		log.Printf("⚠️ [OUTBOUND] Không thể mở Direct port %d: %v", ports.direct, err)
	} else {
		log.Printf("✅ [OUTBOUND] Direct tunnel port %d", ports.direct)
		go func() {
			for {
				conn, err := directLn.Accept()
				if err != nil {
					continue
				}
				go handleOutboundConn(conn, false)
			}
		}()
	}

	// UDP Smart Bind relay — outbound lắng nghe UDP port để nhận packet từ inbound
	go startUDPRelay(ports.udp)
	log.Printf("✅ [OUTBOUND] UDP relay port %d", ports.udp)

	// Config reload cho outbound (allowed_in, fallback)
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		for range ticker.C {
			configMu.RLock()
			cv2 := configV2
			configMu.RUnlock()
			if cv2 != nil {
				ownGID = cv2.ID
			}
		}
	}()

	// Accept TLS connections
	for {
		conn, err := tlsLn.Accept()
		if err != nil {
			log.Printf("❌ [OUTBOUND] Accept TLS: %v", err)
			continue
		}
		go handleOutboundConn(conn, true)
	}
}

func handleOutboundConn(inboundConn net.Conn, isTLS bool) {
	defer inboundConn.Close()

	// Đọc header: 2 byte dest length + 2 byte inbound GID + dest
	lenBuf := make([]byte, 2)
	if _, err := io.ReadFull(inboundConn, lenBuf); err != nil {
		log.Printf("❌ [OUTBOUND] Lỗi đọc header: %v", err)
		return
	}

	destLen := int(lenBuf[0])<<8 | int(lenBuf[1])
	if destLen > 256 || destLen < 1 {
		log.Printf("❌ [OUTBOUND] Độ dài destination không hợp lệ: %d", destLen)
		return
	}

	// Đọc inbound GID (2 bytes)
	gidBuf := make([]byte, 2)
	if _, err := io.ReadFull(inboundConn, gidBuf); err != nil {
		log.Printf("❌ [OUTBOUND] Lỗi đọc inbound GID: %v", err)
		return
	}
	inboundGID := uint64(gidBuf[0])<<8 | uint64(gidBuf[1])

	// Access Control: kiểm tra allowed_in
	if !checkAccessAllowed(inboundGID) {
		log.Printf("🚫 [OUTBOUND] Từ chối inbound GID %d (không trong allowed_in)", inboundGID)
		return
	}

	destBytes := make([]byte, destLen)
	if _, err := io.ReadFull(inboundConn, destBytes); err != nil {
		log.Printf("❌ [OUTBOUND] Lỗi đọc destination: %v", err)
		return
	}
	dest := string(destBytes)

	log.Printf("🔗 [OUTBOUND] Tunnel từ GID %d → %s", inboundGID, dest)

	// Kết nối tới destination
	destConn, err := net.DialTimeout("tcp", dest, 10*time.Second)
	if err != nil {
		log.Printf("❌ [OUTBOUND] Không thể kết nối %s: %v", dest, err)
		return
	}
	defer destConn.Close()

	// Proxy Protocol nếu cần
	sendProxyProtocol(destConn, inboundConn.RemoteAddr(), inboundGID)

	// Bidirectional copy với traffic counting
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		io.Copy(destConn, inboundConn)
		destConn.Close()
	}()

	go func() {
		defer wg.Done()
		io.Copy(inboundConn, destConn)
		inboundConn.Close()
	}()

	wg.Wait()
	log.Printf("✅ [OUTBOUND] Kết thúc phiên tới %s", dest)
}

// checkAccessAllowed kiểm tra inbound GID có được phép kết nối không
func checkAccessAllowed(inboundGID uint64) bool {
	configMu.RLock()
	cv2 := configV2
	configMu.RUnlock()
	if cv2 == nil {
		return true // Chưa có config, cho phép tạm
	}

	// Web mẫu: outbound remotes chứa danh sách inbound được phép
	for _, remote := range cv2.Remotes {
		if remote.DeviceGroup.ID == inboundGID {
			return true
		}
	}

	// Nếu remotes rỗng (outbound cũ), cho phép tất cả
	if len(cv2.Remotes) == 0 {
		return true
	}

	return false
}

// sendProxyProtocol gửi PROXY protocol header nếu rule yêu cầu
func sendProxyProtocol(destConn net.Conn, srcAddr net.Addr, inboundGID uint64) {
	configMu.RLock()
	cv2 := configV2
	configMu.RUnlock()
	if cv2 == nil {
		return
	}

	// Tìm rule phù hợp cho inbound GID này
	for _, rule := range cv2.Rules {
		if rule.DeviceGroupIn != inboundGID {
			continue
		}
		var rc RuleConfig
		json.Unmarshal([]byte(rule.Config), &rc)
		if rc.ProxyProtocol <= 0 {
			continue
		}

		srcIP, srcPort := parseNetAddr(srcAddr)
		dstIP, dstPort := parseNetAddr(destConn.RemoteAddr())

		if rc.ProxyProtocol == 1 {
			// v1 text
			header := fmt.Sprintf("PROXY TCP4 %s %s %s %s\r\n", srcIP, dstIP, srcPort, dstPort)
			destConn.Write([]byte(header))
		} else if rc.ProxyProtocol == 2 {
			// v2 binary
			sendProxyProtoV2(destConn, srcIP, dstIP, srcPort, dstPort)
		}
		return
	}
}

func sendProxyProtoV2(conn net.Conn, srcIP, dstIP string, srcPort, dstPort string) {
	// Magic 12 bytes
	magic := []byte{0x0D, 0x0A, 0x0D, 0x0A, 0x00, 0x0D, 0x0A, 0x51, 0x55, 0x49, 0x54, 0x0A}
	// Version 2 + Command PROXY
	verCmd := byte(0x21)
	// Protocol TCP over IPv4
	family := byte(0x11)

	// Parse IPs
	src := net.ParseIP(srcIP).To4()
	dst := net.ParseIP(dstIP).To4()
	if src == nil || dst == nil {
		return
	}

	sp, _ := strconv.Atoi(srcPort)
	dp, _ := strconv.Atoi(dstPort)

	// Address length: 12 bytes for IPv4 (4+4+2+2)
	addrLen := make([]byte, 2)
	binary.BigEndian.PutUint16(addrLen, 12)

	// Build header
	header := append(magic, verCmd, family)
	header = append(header, addrLen...)
	header = append(header, src...)
	header = append(header, dst...)
	header = append(header, byte(sp>>8), byte(sp&0xFF))
	header = append(header, byte(dp>>8), byte(dp&0xFF))

	conn.Write(header)
}

func parseNetAddr(addr net.Addr) (ip, port string) {
	switch a := addr.(type) {
	case *net.TCPAddr:
		return a.IP.String(), strconv.Itoa(a.Port)
	default:
		parts := strings.Split(addr.String(), ":")
		if len(parts) >= 2 {
			return parts[0], parts[len(parts)-1]
		}
		return "0.0.0.0", "0"
	}
}

// ========== WebSocket Handler (Outbound) ==========

func handleWSOutboundConn(conn net.Conn) {
	defer conn.Close()

	// Đọc HTTP upgrade request
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		return
	}
	request := string(buf[:n])

	// Kiểm tra WebSocket upgrade
	if !strings.Contains(request, "Upgrade: websocket") {
		// Fallback: xử lý như direct tunnel
		handleDirectTunnel(conn, buf[:n])
		return
	}

	// Tìm Sec-WebSocket-Key
	key := ""
	for _, line := range strings.Split(request, "\r\n") {
		if strings.HasPrefix(line, "Sec-WebSocket-Key:") {
			key = strings.TrimSpace(strings.TrimPrefix(line, "Sec-WebSocket-Key:"))
			break
		}
	}

	if key == "" {
		return
	}

	// Tính Accept key
	h := sha1.Sum([]byte(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	acceptKey := base64.StdEncoding.EncodeToString(h[:])

	// Gửi response
	response := "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + acceptKey + "\r\n\r\n"
	conn.Write([]byte(response))

	// Đọc frame WebSocket và chuyển tiếp
	handleWSTunnel(conn)
}

func handleDirectTunnel(conn net.Conn, firstData []byte) {
	// Đọc header từ firstData
	if len(firstData) < 2 {
		return
	}
	destLen := int(firstData[0])<<8 | int(firstData[1])
	if destLen > 256 || destLen < 1 || len(firstData) < 4+destLen {
		return
	}

	inboundGID := uint64(firstData[2])<<8 | uint64(firstData[3])
	dest := string(firstData[4 : 4+destLen])

	if !checkAccessAllowed(inboundGID) {
		log.Printf("🚫 [OUTBOUND-DIRECT] Từ chối inbound GID %d", inboundGID)
		return
	}

	log.Printf("🔗 [OUTBOUND-DIRECT] Tunnel → %s", dest)

	destConn, err := net.DialTimeout("tcp", dest, 10*time.Second)
	if err != nil {
		return
	}
	defer destConn.Close()

	// Gửi dữ liệu còn lại sau header
	remaining := firstData[4+destLen:]
	if len(remaining) > 0 {
		destConn.Write(remaining)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); io.Copy(destConn, conn); destConn.Close() }()
	go func() { defer wg.Done(); io.Copy(conn, destConn); conn.Close() }()
	wg.Wait()
}

func handleWSTunnel(conn net.Conn) {
	// Đọc frame WebSocket đơn giản
	buf := make([]byte, 65536)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			break
		}
		if n < 2 {
			continue
		}

		// Parse WebSocket frame header
		opcode := buf[0] & 0x0F
		masked := (buf[1] & 0x80) != 0
		payloadLen := int(buf[1] & 0x7F)
		pos := 2

		if payloadLen == 126 {
			if n < 4 {
				continue
			}
			payloadLen = int(buf[2])<<8 | int(buf[3])
			pos = 4
		} else if payloadLen == 127 {
			if n < 10 {
				continue
			}
			payloadLen = int(binary.BigEndian.Uint64(buf[2:10]))
			pos = 10
		}

		var maskKey [4]byte
		if masked {
			copy(maskKey[:], buf[pos:pos+4])
			pos += 4
		}

		if n < pos+payloadLen {
			continue
		}

		payload := buf[pos : pos+payloadLen]
		if masked {
			for i := range payload {
				payload[i] ^= maskKey[i%4]
			}
		}

		// Opcode 0x8 = close, 0x9 = ping
		switch opcode {
		case 0x8: // Close
			conn.Write([]byte{0x88, 0x00})
			return
		case 0x9: // Ping → Pong
			pong := []byte{0x8A, byte(len(payload))}
			pong = append(pong, payload...)
			conn.Write(pong)
			continue
		case 0x1, 0x2: // Text/Binary
			// Đây là data tunnel — xử lý header và forward
			if len(payload) >= 4 {
				destLen := int(payload[0])<<8 | int(payload[1])
				if destLen > 0 && destLen <= 256 && len(payload) >= 4+destLen {
					gid := uint64(payload[2])<<8 | uint64(payload[3])
					dest := string(payload[4 : 4+destLen])
					_ = gid
					data := payload[4+destLen:]

					destConn, err := net.DialTimeout("tcp", dest, 10*time.Second)
					if err == nil {
						if len(data) > 0 {
							destConn.Write(data)
						}
						// Relay phần còn lại
						go func() {
							io.Copy(destConn, conn)
							destConn.Close()
						}()
						io.Copy(conn, destConn)
						conn.Close()
					}
					return
				}
			}
		}
	}
}

// ========== Port Calculation (giống backend) ==========

type serverPorts struct {
	direct, ws, tls, udp int
}

func computeServerPorts(h string) serverPorts {
	sum := md5.Sum([]byte(h))
	return serverPorts{
		direct: 10000 + int(sum[0])*256 + int(sum[1]),
		ws:     20000 + (int(sum[2])*256 + int(sum[3])) % (65535 - 20000),
		tls:    3000 + int(sum[4])*256 + int(sum[5]),
		udp:    40000 + (int(sum[6])*256 + int(sum[7])) % (65535 - 40000),
	}
}

// ========== TLS Self-Signed Cert ==========

func generateSelfSignedCert() (tls.Certificate, error) {
	certPath := "/tmp/nyanpass_outbound_cert.pem"
	keyPath := "/tmp/nyanpass_outbound_key.pem"

	if _, err := os.Stat(certPath); err == nil {
		if _, err := os.Stat(keyPath); err == nil {
			cert, err := tls.LoadX509KeyPair(certPath, keyPath)
			if err == nil {
				return cert, nil
			}
		}
	}

	cmd := exec.Command("openssl", "req", "-x509", "-newkey", "rsa:2048",
		"-keyout", keyPath, "-out", certPath,
		"-days", "3650", "-nodes",
		"-subj", "/C=US/ST=CA/L=SF/O=Nyanpass/CN=nyanpass-tunnel",
		"-addext", "subjectAltName=IP:127.0.0.1")
	if out, err := cmd.CombinedOutput(); err != nil {
		return tls.Certificate{}, fmt.Errorf("openssl: %w: %s", err, string(out))
	}

	return tls.LoadX509KeyPair(certPath, keyPath)
}

// ========== UDP Smart Bind ==========

// startUDPRelay lắng nghe UDP port và relay packet từ inbound đến đích
func startUDPRelay(port int) {
	udpAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Printf("⚠️ [OUTBOUND] Không thể mở UDP relay port %d: %v", port, err)
		return
	}
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Printf("⚠️ [OUTBOUND] Không thể mở UDP relay port %d: %v", port, err)
		return
	}
	defer conn.Close()

	buf := make([]byte, 65535)
	for {
		n, clientAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			continue
		}
		go relayUDPPacket(conn, clientAddr, buf[:n])
	}
}

// relayUDPPacket chuyển tiếp UDP packet từ inbound đến đích và ngược lại
func relayUDPPacket(conn *net.UDPConn, clientAddr *net.UDPAddr, data []byte) {
	if len(data) < 3 {
		return
	}
	// Header: [2 bytes dest_len BE] + [dest] + [payload]
	destLen := int(binary.BigEndian.Uint16(data[0:2]))
	if 2+destLen > len(data) {
		return
	}
	dest := string(data[2 : 2+destLen])
	payload := data[2+destLen:]

	destConn, err := net.DialTimeout("udp", dest, 10*time.Second)
	if err != nil {
		return
	}
	defer destConn.Close()

	destConn.Write(payload)

	// Đọc response
	respBuf := make([]byte, 65535)
	destConn.SetReadDeadline(time.Now().Add(5 * time.Second))
	rn, _ := destConn.Read(respBuf)
	if rn > 0 {
		conn.WriteToUDP(respBuf[:rn], clientAddr)
	}
}

// handleInboundUDP xử lý UDP packet từ client, chuyển tiếp qua outbound
func handleInboundUDP(conn *net.UDPConn, clientAddr *net.UDPAddr, data []byte, rules []ConfigV2Rule) {
	if len(rules) == 0 {
		return
	}
	rule := rules[0]
	var rc RuleConfig
	json.Unmarshal([]byte(rule.Config), &rc)
	if len(rc.Dest) == 0 {
		return
	}

	// Lấy outbound server
	outboundServer := findOutboundServer(rule, clientAddr.IP.String())
	if outboundServer == nil {
		outboundServer = findFallbackServer(rule, clientAddr.IP.String())
		if outboundServer == nil {
			return
		}
	}

	dest := selectDest(rc.Dest, parseLBPolicy(rc.LBPolicy), rule.ID)

	// Tạo header: [2 bytes dest_len] + [dest]
	header := make([]byte, 2+len(dest))
	binary.BigEndian.PutUint16(header[0:2], uint16(len(dest)))
	copy(header[2:], dest)

	// Gửi đến outbound UDP port
	outboundAddr, _ := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", outboundServer.IP4, outboundServer.UDPPort))
	conn.WriteToUDP(append(header, data...), outboundAddr)
}

// ========== Handle Management ==========

func loadOrGenerateHandle() string {
	// Thử load handle từ working directory trước, sau đó fallback về đường dẫn mặc định
	paths := []string{"handle", "/opt/nyanpass/handle"}
	for _, p := range paths {
		if data, err := os.ReadFile(p); err == nil {
			h := strings.TrimSpace(string(data))
			if len(h) == 32 {
				handleFile = p
				log.Printf("📌 Load handle: %s", h)
				return h
			}
		}
	}

	// Tạo handle mới
	b := make([]byte, 16)
	rand.Read(b)
	h := hex.EncodeToString(b)

	// Lưu vào working directory (ưu tiên) và cả /opt/nyanpass/ (tương thích ngược)
	os.MkdirAll("/opt/nyanpass", 0755)
	os.WriteFile("handle", []byte(h), 0644)
	os.WriteFile("/opt/nyanpass/handle", []byte(h), 0644)
	handleFile = "handle"
	log.Printf("📌 Handle mới: %s", h)
	return h
}

// ========== Monitor / Report ==========

type Report struct {
	Token   string     `json:"token"`
	GroupID uint64     `json:"group_id,omitempty"`
	Server  ServerData `json:"server"`
}

type ServerData struct {
	Name            string  `json:"name"`
	Handle          string  `json:"handle"`
	IP4             string  `json:"ip4"`
	CPU             float64 `json:"cpu"`
	CPUModel        string  `json:"cpu_model"`
	MemUsed         uint64  `json:"mem_used"`
	MemTotal        uint64  `json:"mem_total"`
	DiskUsed        uint64  `json:"disk_used"`
	DiskTotal       uint64  `json:"disk_total"`
	NetInSpeed      uint64  `json:"net_in_speed"`
	NetOutSpeed     uint64  `json:"net_out_speed"`
	NetInTransfer   uint64  `json:"net_in_transfer"`
	NetOutTransfer  uint64  `json:"net_out_transfer"`
	Uptime          uint64  `json:"uptime"`
	Load1           float64 `json:"load1"`
	Load5           float64 `json:"load5"`
	Load15          float64 `json:"load15"`
	TCPConnCount    int     `json:"tcp_conn_count"`
	UDPConnCount    int     `json:"udp_conn_count"`
	ProcessCount    int     `json:"process_count"`
	Platform        string  `json:"platform"`
	PlatformVersion string  `json:"platform_version"`
	Arch            string  `json:"arch"`
	BootTime        int64   `json:"boot_time"`
	Hostname        string  `json:"hostname"`
}

func collectReport(hostname, handle string) Report {
	load1, load5, load15 := getLoadAvg()
	readNetSpeed() // Gọi một lần để cache cả in và out speed
	return Report{
		Token: *token,
		Server: ServerData{
			Name:            "0",
			Handle:          handle,
			IP4:             getOutboundIP(),
			CPU:             getCPUUsage(),
			CPUModel:        getCPUModel(),
			MemUsed:         getMemUsed(),
			MemTotal:        getMemTotal(),
			DiskUsed:        getDiskUsed(),
			DiskTotal:       getDiskTotal(),
			NetInSpeed:      getNetInSpeed(),
			NetOutSpeed:     getNetOutSpeed(),
			NetInTransfer:   prevIn,
			NetOutTransfer:  prevOut,
			Uptime:          getUptime(),
			Load1:           load1,
			Load5:           load5,
			Load15:          load15,
			TCPConnCount:    getTCPCount(),
			UDPConnCount:    getUDPCount(),
			ProcessCount:    getProcessCount(),
			Platform:        runtime.GOOS,
			PlatformVersion: getPlatformVersion(),
			Arch:            runtime.GOARCH,
			BootTime:        getBootTime(),
			Hostname:        hostname,
		},
	}
}

func sendReport(r Report) error {
	data, _ := json.Marshal(r)
	resp, err := http.Post(*panel+"/api/v1/node/report", "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// ========== System Info Collectors ==========

func getCPUModel() string {
	data, _ := os.ReadFile("/proc/cpuinfo")
	for _, l := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(l, "model name") {
			parts := strings.SplitN(l, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return "Unknown CPU"
}

var prevIdle, prevTotal float64

func getCPUUsage() float64 {
	data, _ := os.ReadFile("/proc/stat")
	for _, l := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(l, "cpu ") {
			fields := strings.Fields(l)
			if len(fields) >= 5 {
				var total, idle float64
				for i, f := range fields[1:] {
					v, _ := strconv.ParseFloat(f, 64)
					total += v
					if i == 3 {
						idle = v
					}
				}
				if prevTotal == 0 {
					prevIdle, prevTotal = idle, total
					return 0
				}
				idleDiff := idle - prevIdle
				totalDiff := total - prevTotal
				prevIdle, prevTotal = idle, total
				if totalDiff > 0 {
					return (1 - idleDiff/totalDiff) * 100
				}
				return 0
			}
		}
	}
	return 0
}

var cachedMemUsed, cachedMemTotal uint64

func getMemUsed() uint64 {
	used, _ := readMemory()
	return used
}
func getMemTotal() uint64 {
	_, total := readMemory()
	return total
}

func readMemory() (used, total uint64) {
	data, _ := os.ReadFile("/proc/meminfo")
	var memTotal, memFree, buffers, cached uint64
	for _, l := range strings.Split(string(data), "\n") {
		f := strings.Fields(l)
		if len(f) < 2 {
			continue
		}
		v, _ := strconv.ParseUint(f[1], 10, 64)
		switch {
		case strings.HasPrefix(l, "MemTotal:"):
			memTotal = v * 1024
		case strings.HasPrefix(l, "MemFree:"):
			memFree = v * 1024
		case strings.HasPrefix(l, "Buffers:"):
			buffers = v * 1024
		case strings.HasPrefix(l, "Cached:"):
			cached = v * 1024
		}
	}
	total = memTotal
	if memFree > 0 {
		used = memTotal - memFree - buffers - cached
	} else {
		used = memTotal / 2
	}
	return
}

var cachedDiskUsed, cachedDiskTotal uint64

func getDiskUsed() uint64 {
	u, _ := readDisk()
	return u
}
func getDiskTotal() uint64 {
	_, t := readDisk()
	return t
}

func readDisk() (used, total uint64) {
	out, _ := exec.Command("df", "-k", "/").Output()
	lines := strings.Split(string(out), "\n")
	if len(lines) >= 2 {
		fields := strings.Fields(lines[1])
		if len(fields) >= 4 {
			used, _ = strconv.ParseUint(fields[2], 10, 64)
			total, _ = strconv.ParseUint(fields[1], 10, 64)
			used *= 1024
			total *= 1024
		}
	}
	return
}

func getUptime() uint64 {
	data, _ := os.ReadFile("/proc/uptime")
	fields := strings.Fields(string(data))
	if len(fields) > 0 {
		v, _ := strconv.ParseFloat(fields[0], 64)
		return uint64(v)
	}
	return 0
}

func getLoadAvg() (float64, float64, float64) {
	data, _ := os.ReadFile("/proc/loadavg")
	fields := strings.Fields(string(data))
	if len(fields) >= 3 {
		l1, _ := strconv.ParseFloat(fields[0], 64)
		l5, _ := strconv.ParseFloat(fields[1], 64)
		l15, _ := strconv.ParseFloat(fields[2], 64)
		return l1, l5, l15
	}
	return 0, 0, 0
}

func getTCPCount() int {
	out, _ := exec.Command("ss", "-tlnp").Output()
	n := strings.Count(string(out), "\n") - 1
	if n < 0 {
		n = 0
	}
	return n
}

func getUDPCount() int {
	out, _ := exec.Command("ss", "-ulnp").Output()
	n := strings.Count(string(out), "\n") - 1
	if n < 0 {
		n = 0
	}
	return n
}

func getProcessCount() int {
	out, _ := exec.Command("sh", "-c", "ls -d /proc/[0-9]* 2>/dev/null | wc -l").Output()
	v, _ := strconv.Atoi(strings.TrimSpace(string(out)))
	return v
}

var prevIn, prevOut uint64
var prevTime time.Time
var cachedNetInSp, cachedNetOutSp uint64

func getNetInSpeed() uint64 {
	return cachedNetInSp
}
func getNetOutSpeed() uint64 {
	return cachedNetOutSp
}

func readNetSpeed() {
	data, _ := os.ReadFile("/proc/net/dev")
	for _, l := range strings.Split(string(data), "\n") {
		if strings.Contains(l, "eth0:") || strings.Contains(l, "ens3:") || strings.Contains(l, "enp") {
			fields := strings.Fields(l)
			if len(fields) >= 10 {
				curIn, _ := strconv.ParseUint(fields[1], 10, 64)
				curOut, _ := strconv.ParseUint(fields[9], 10, 64)
				now := time.Now()
				if prevTime.IsZero() {
					prevIn, prevOut, prevTime = curIn, curOut, now
					cachedNetInSp, cachedNetOutSp = 0, 0
					return
				}
				elapsed := now.Sub(prevTime).Seconds()
				if elapsed > 0 {
					cachedNetInSp = uint64(float64(curIn-prevIn) / elapsed)
					cachedNetOutSp = uint64(float64(curOut-prevOut) / elapsed)
				}
				prevIn, prevOut, prevTime = curIn, curOut, now
				return
			}
		}
	}
	cachedNetInSp, cachedNetOutSp = 0, 0
}

func getPlatformVersion() string {
	data, _ := os.ReadFile("/etc/os-release")
	for _, l := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(l, "VERSION_ID=") {
			return strings.Trim(strings.TrimPrefix(l, "VERSION_ID="), "\"")
		}
	}
	return "unknown"
}

func getBootTime() int64 {
	out, _ := exec.Command("cat", "/proc/stat").Output()
	for _, l := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(l, "btime ") {
			f := strings.Fields(l)
			if len(f) >= 2 {
				v, _ := strconv.ParseInt(f[1], 10, 64)
				return v
			}
		}
	}
	return 0
}

func getOutboundIP() string {
	resp, err := http.Get("https://ifconfig.me/ip")
	if err != nil {
		return "127.0.0.1"
	}
	defer resp.Body.Close()
	ip, _ := io.ReadAll(resp.Body)
	return strings.TrimSpace(string(ip))
}
