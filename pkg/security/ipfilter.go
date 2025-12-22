package security

import (
	"net"
	"strings"
	"sync"

	"github.com/lk2023060901/xdooria/pkg/config"
)

// IPFilterConfig IP 过滤配置
type IPFilterConfig struct {
	// 模式：whitelist（白名单）或 blacklist（黑名单）
	Mode string `mapstructure:"mode" json:"mode"`

	// IP 列表（支持单个 IP 和 CIDR）
	// 如：["192.168.1.100", "10.0.0.0/8", "172.16.0.0/12"]
	IPs []string `mapstructure:"ips" json:"ips"`

	// 是否信任代理头
	TrustProxy bool `mapstructure:"trust_proxy" json:"trust_proxy"`

	// 代理头列表（按优先级顺序尝试）
	// 如：["X-Forwarded-For", "X-Real-IP", "CF-Connecting-IP"]
	ProxyHeaders []string `mapstructure:"proxy_headers" json:"proxy_headers"`

	// 跳过检查的路径
	SkipPaths []string `mapstructure:"skip_paths" json:"skip_paths"`
}

// IP 过滤模式
const (
	IPFilterModeWhitelist = "whitelist"
	IPFilterModeBlacklist = "blacklist"
)

// DefaultIPFilterConfig 返回默认 IP 过滤配置（最小可用配置）
func DefaultIPFilterConfig() *IPFilterConfig {
	return &IPFilterConfig{
		Mode:       IPFilterModeWhitelist,
		TrustProxy: false,
	}
}

// IPFilter IP 过滤器
type IPFilter struct {
	config  *IPFilterConfig
	ipNets  []*net.IPNet
	ipAddrs []net.IP
	mu      sync.RWMutex
}

// NewIPFilter 创建 IP 过滤器
func NewIPFilter(cfg *IPFilterConfig) (*IPFilter, error) {
	newCfg, err := config.MergeConfig(DefaultIPFilterConfig(), cfg)
	if err != nil {
		return nil, err
	}

	// 验证模式
	if newCfg.Mode != IPFilterModeWhitelist && newCfg.Mode != IPFilterModeBlacklist {
		return nil, ErrModeInvalid
	}

	// 验证 IP 列表
	if len(newCfg.IPs) == 0 {
		return nil, ErrIPListEmpty
	}

	f := &IPFilter{
		config:  newCfg,
		ipNets:  make([]*net.IPNet, 0),
		ipAddrs: make([]net.IP, 0),
	}

	// 解析 IP 列表
	if err := f.parseIPs(newCfg.IPs); err != nil {
		return nil, err
	}

	return f, nil
}

// parseIPs 解析 IP 列表
func (f *IPFilter) parseIPs(ips []string) error {
	for _, ipStr := range ips {
		ipStr = strings.TrimSpace(ipStr)
		if ipStr == "" {
			continue
		}

		// 尝试解析为 CIDR
		if strings.Contains(ipStr, "/") {
			_, ipNet, err := net.ParseCIDR(ipStr)
			if err != nil {
				return ErrCIDRInvalid
			}
			f.ipNets = append(f.ipNets, ipNet)
		} else {
			// 解析为单个 IP
			ip := net.ParseIP(ipStr)
			if ip == nil {
				return ErrIPInvalid
			}
			f.ipAddrs = append(f.ipAddrs, ip)
		}
	}

	return nil
}

// Allow 检查 IP 是否允许访问
func (f *IPFilter) Allow(ip string) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	matched := f.matchIP(parsedIP)

	// 白名单模式：匹配则允许
	// 黑名单模式：匹配则拒绝
	if f.config.Mode == IPFilterModeWhitelist {
		return matched
	}
	return !matched
}

// matchIP 检查 IP 是否匹配规则
func (f *IPFilter) matchIP(ip net.IP) bool {
	// 检查单个 IP
	for _, addr := range f.ipAddrs {
		if addr.Equal(ip) {
			return true
		}
	}

	// 检查 CIDR
	for _, ipNet := range f.ipNets {
		if ipNet.Contains(ip) {
			return true
		}
	}

	return false
}

// AllowFromRequest 从请求中提取 IP 并检查
// remoteAddr 是原始远程地址（如 "192.168.1.100:12345"）
// headers 是请求头（用于获取代理头）
func (f *IPFilter) AllowFromRequest(remoteAddr string, headers map[string]string) bool {
	ip := f.ExtractIP(remoteAddr, headers)
	return f.Allow(ip)
}

// ExtractIP 从请求中提取客户端 IP
func (f *IPFilter) ExtractIP(remoteAddr string, headers map[string]string) string {
	// 如果信任代理头，先从代理头获取
	if f.config.TrustProxy {
		if proxyIP := f.getIPFromProxyHeader(headers); proxyIP != "" {
			return proxyIP
		}
	}

	// 从远程地址提取 IP
	return extractIPFromAddr(remoteAddr)
}

// getIPFromProxyHeader 从代理头获取 IP
func (f *IPFilter) getIPFromProxyHeader(headers map[string]string) string {
	for _, header := range f.config.ProxyHeaders {
		if val, ok := headers[header]; ok && val != "" {
			return extractFirstIP(val)
		}
	}
	return ""
}

// extractIPFromAddr 从地址字符串提取 IP（去除端口）
func extractIPFromAddr(addr string) string {
	// 处理 IPv6
	if strings.HasPrefix(addr, "[") {
		if idx := strings.LastIndex(addr, "]"); idx != -1 {
			return addr[1:idx]
		}
	}

	// 处理 IPv4
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		return addr[:idx]
	}

	return addr
}

// extractFirstIP 从逗号分隔的 IP 列表中提取第一个 IP
func extractFirstIP(ipList string) string {
	ips := strings.Split(ipList, ",")
	if len(ips) > 0 {
		return strings.TrimSpace(ips[0])
	}
	return ""
}

// ShouldSkip 检查路径是否需要跳过检查
func (f *IPFilter) ShouldSkip(path string) bool {
	for _, skipPath := range f.config.SkipPaths {
		if matchPath(skipPath, path) {
			return true
		}
	}
	return false
}

// AddIP 动态添加 IP（线程安全）
func (f *IPFilter) AddIP(ipStr string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	ipStr = strings.TrimSpace(ipStr)

	if strings.Contains(ipStr, "/") {
		_, ipNet, err := net.ParseCIDR(ipStr)
		if err != nil {
			return ErrCIDRInvalid
		}
		f.ipNets = append(f.ipNets, ipNet)
	} else {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			return ErrIPInvalid
		}
		f.ipAddrs = append(f.ipAddrs, ip)
	}

	return nil
}

// RemoveIP 动态移除 IP（线程安全）
func (f *IPFilter) RemoveIP(ipStr string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	ipStr = strings.TrimSpace(ipStr)

	if strings.Contains(ipStr, "/") {
		// 移除 CIDR
		_, targetNet, err := net.ParseCIDR(ipStr)
		if err != nil {
			return
		}
		for i, ipNet := range f.ipNets {
			if ipNet.String() == targetNet.String() {
				f.ipNets = append(f.ipNets[:i], f.ipNets[i+1:]...)
				return
			}
		}
	} else {
		// 移除单个 IP
		targetIP := net.ParseIP(ipStr)
		if targetIP == nil {
			return
		}
		for i, ip := range f.ipAddrs {
			if ip.Equal(targetIP) {
				f.ipAddrs = append(f.ipAddrs[:i], f.ipAddrs[i+1:]...)
				return
			}
		}
	}
}

// GetConfig 获取配置
func (f *IPFilter) GetConfig() *IPFilterConfig {
	return f.config
}

// GetIPs 获取当前 IP 列表
func (f *IPFilter) GetIPs() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	result := make([]string, 0, len(f.ipAddrs)+len(f.ipNets))

	for _, ip := range f.ipAddrs {
		result = append(result, ip.String())
	}

	for _, ipNet := range f.ipNets {
		result = append(result, ipNet.String())
	}

	return result
}

// MatchCIDRs 检查 IP 是否匹配给定的 CIDR 列表
func MatchCIDRs(ipStr string, cidrs []string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	for _, cidr := range cidrs {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if ipNet.Contains(ip) {
			return true
		}
	}

	return false
}
