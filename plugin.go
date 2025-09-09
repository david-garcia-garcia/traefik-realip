package traefik_realip

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
)

// HeaderConfig defines a header to process with optional depth specification.
type HeaderConfig struct {
	HeaderName string `json:"headerName"` // Name of the header to check
	Depth      int    `json:"depth"`      // Depth for IP extraction: -1 = leftmost, 0 = rightmost, 1 = second from right, etc.
}

// Config defines the plugin configuration.
type Config struct {
	// Core settings
	Enabled bool `json:"enabled,omitempty"` // Enable/disable the plugin

	// Header configuration
	HeaderName     string         `json:"headerName,omitempty"`     // Header name where IP will be populated
	ProcessHeaders []HeaderConfig `json:"processHeaders,omitempty"` // List of headers to process with depth configuration
	ForceOverwrite bool           `json:"forceOverwrite,omitempty"` // Always set the header, even if empty (to prevent header spoofing)
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		Enabled:    true,
		HeaderName: "X-Real-IP",
		ProcessHeaders: []HeaderConfig{
			{HeaderName: "X-Forwarded-For", Depth: -1},
			{HeaderName: "X-Real-IP", Depth: -1},
			{HeaderName: "CF-Connecting-IP", Depth: -1},
			{HeaderName: "clientAddress", Depth: -1},
		},
		ForceOverwrite: true,
	}
}

// Plugin holds the plugin instance data.
type Plugin struct {
	next           http.Handler
	name           string
	enabled        bool
	headerName     string
	processHeaders []HeaderConfig
	forceOverwrite bool
}

// New creates a new plugin instance.
func New(ctx context.Context, next http.Handler, cfg *Config, name string) (http.Handler, error) {
	if next == nil {
		return nil, fmt.Errorf("%s: no next handler provided", name)
	}

	if cfg == nil {
		return nil, fmt.Errorf("%s: no config provided", name)
	}

	// Validate configuration
	if cfg.Enabled && cfg.HeaderName == "" {
		return nil, fmt.Errorf("%s: headerName cannot be empty when plugin is enabled", name)
	}

	if cfg.Enabled && len(cfg.ProcessHeaders) == 0 {
		return nil, fmt.Errorf("%s: processHeaders cannot be empty when plugin is enabled", name)
	}

	plugin := &Plugin{
		next:           next,
		name:           name,
		enabled:        cfg.Enabled,
		headerName:     cfg.HeaderName,
		processHeaders: cfg.ProcessHeaders,
		forceOverwrite: cfg.ForceOverwrite,
	}

	return plugin, nil
}

// ServeHTTP implements the http.Handler interface.
func (p *Plugin) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if !p.enabled {
		p.next.ServeHTTP(rw, req)
		return
	}

	// Extract the first valid IP address from the configured headers
	realIP := p.extractRealIP(req)

	// Always set the header if forceOverwrite is true, even if empty
	// This prevents clients from spoofing the header
	if (p.forceOverwrite || realIP != "") && p.headerName != "" {
		req.Header.Set(p.headerName, realIP)
	}

	p.next.ServeHTTP(rw, req)
}

// extractRealIP processes the configured headers in order and returns the first valid IP address found.
// Special synthetic header "clientAddress" maps to req.RemoteAddr for direct access to the connection's remote address.
func (p *Plugin) extractRealIP(req *http.Request) string {
	for _, headerConfig := range p.processHeaders {
		var headerValue string

		// Handle synthetic "clientAddress" header
		if headerConfig.HeaderName == "clientAddress" {
			headerValue = req.RemoteAddr
		} else {
			headerValue = req.Header.Get(headerConfig.HeaderName)
		}

		if headerValue == "" {
			continue
		}

		// Process comma-separated IPs in the header with depth logic
		ips := strings.Split(headerValue, ",")

		// Clean all IPs first
		var cleanIPs []string
		for _, ip := range ips {
			cleanIP := p.cleanIPAddress(ip)
			if cleanIP != "" {
				cleanIPs = append(cleanIPs, cleanIP)
			}
		}

		if len(cleanIPs) == 0 {
			continue
		}

		// Apply depth logic
		var selectedIP string
		if headerConfig.Depth < 0 {
			// Any negative depth means leftmost (first) IP
			selectedIP = cleanIPs[0]
		} else {
			// Depth from rightmost: 0 = rightmost, 1 = second from right, etc.
			rightIndex := len(cleanIPs) - 1 - headerConfig.Depth
			if rightIndex >= 0 && rightIndex < len(cleanIPs) {
				selectedIP = cleanIPs[rightIndex]
			} else {
				// Depth out of bounds, skip this header
				continue
			}
		}

		if selectedIP != "" {
			return selectedIP
		}
	}

	return ""
}

// cleanIPAddress removes whitespace and port numbers from IP addresses.
func (p *Plugin) cleanIPAddress(ip string) string {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return ""
	}

	// Remove port if present (e.g., "192.168.1.1:8080" -> "192.168.1.1")
	host, _, err := net.SplitHostPort(ip)
	if err == nil {
		return host
	}

	// If SplitHostPort fails, it means there's no port, return the original IP
	return ip
}
