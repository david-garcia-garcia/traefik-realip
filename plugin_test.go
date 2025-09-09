package traefik_realip

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const pluginName = "realip"

type noopHandler struct{}

func (n noopHandler) ServeHTTP(rw http.ResponseWriter, _ *http.Request) {
	rw.WriteHeader(http.StatusOK)
}

func TestNew(t *testing.T) {
	t.Run("ValidConfig", func(t *testing.T) {
		cfg := &Config{
			Enabled:    true,
			HeaderName: "X-Real-IP",
			ProcessHeaders: []HeaderConfig{
				{HeaderName: "X-Forwarded-For", Depth: -1},
			},
			ForceOverwrite: true,
			TrustAll:       true,
		}

		plugin, err := New(context.TODO(), &noopHandler{}, cfg, pluginName)
		if err != nil {
			t.Errorf("expected no error, but got: %v", err)
		}
		if plugin == nil {
			t.Error("expected plugin to be created, but got nil")
		}
	})

	t.Run("DisabledPlugin", func(t *testing.T) {
		cfg := &Config{
			Enabled:        false,
			HeaderName:     "",
			ProcessHeaders: nil,
			ForceOverwrite: false,
			TrustAll:       true,
		}

		plugin, err := New(context.TODO(), &noopHandler{}, cfg, pluginName)
		if err != nil {
			t.Errorf("expected no error for disabled plugin, but got: %v", err)
		}
		if plugin == nil {
			t.Error("expected plugin to be created, but got nil")
		}
	})

	t.Run("NoNextHandler", func(t *testing.T) {
		cfg := &Config{
			Enabled:    true,
			HeaderName: "X-Real-IP",
			ProcessHeaders: []HeaderConfig{
				{HeaderName: "X-Forwarded-For", Depth: -1},
			},
			ForceOverwrite: true,
			TrustAll:       true,
		}

		plugin, err := New(context.TODO(), nil, cfg, pluginName)
		if err == nil {
			t.Error("expected error for nil next handler, but got none")
		}
		if plugin != nil {
			t.Error("expected plugin to be nil, but got instance")
		}
	})

	t.Run("NoConfig", func(t *testing.T) {
		plugin, err := New(context.TODO(), &noopHandler{}, nil, pluginName)
		if err == nil {
			t.Error("expected error for nil config, but got none")
		}
		if plugin != nil {
			t.Error("expected plugin to be nil, but got instance")
		}
	})

	t.Run("EmptyHeaderName", func(t *testing.T) {
		cfg := &Config{
			Enabled:    true,
			HeaderName: "",
			ProcessHeaders: []HeaderConfig{
				{HeaderName: "X-Forwarded-For", Depth: -1},
			},
			ForceOverwrite: true,
			TrustAll:       true,
		}

		plugin, err := New(context.TODO(), &noopHandler{}, cfg, pluginName)
		if err == nil {
			t.Error("expected error for empty headerName, but got none")
		}
		if plugin != nil {
			t.Error("expected plugin to be nil, but got instance")
		}
	})

	t.Run("EmptyProcessHeaders", func(t *testing.T) {
		cfg := &Config{
			Enabled:        true,
			HeaderName:     "X-Real-IP",
			ProcessHeaders: []HeaderConfig{},
			ForceOverwrite: true,
			TrustAll:       true,
		}

		plugin, err := New(context.TODO(), &noopHandler{}, cfg, pluginName)
		if err == nil {
			t.Error("expected error for empty processHeaders, but got none")
		}
		if plugin != nil {
			t.Error("expected plugin to be nil, but got instance")
		}
	})

	t.Run("ForceOverwriteEnabled", func(t *testing.T) {
		cfg := &Config{
			Enabled:        true,
			HeaderName:     "X-Real-IP",
			ProcessHeaders: []HeaderConfig{{HeaderName: "X-Forwarded-For", Depth: -1}},
			ForceOverwrite: true,
			TrustAll:       true,
		}

		plugin, err := New(context.TODO(), &noopHandler{}, cfg, pluginName)
		if err != nil {
			t.Errorf("expected no error, but got: %v", err)
		}
		if plugin == nil {
			t.Error("expected plugin to be created, but got nil")
		}
	})

	t.Run("SyntheticClientAddressHeader", func(t *testing.T) {
		cfg := &Config{
			Enabled:        true,
			HeaderName:     "X-Real-IP",
			ProcessHeaders: []HeaderConfig{{HeaderName: "clientAddress", Depth: -1}},
			ForceOverwrite: true,
			TrustAll:       true,
		}

		plugin, err := New(context.TODO(), &noopHandler{}, cfg, pluginName)
		if err != nil {
			t.Errorf("expected no error, but got: %v", err)
		}
		if plugin == nil {
			t.Error("expected plugin to be created, but got nil")
		}
	})
}

func TestServeHTTP(t *testing.T) {
	t.Run("DisabledPlugin", func(t *testing.T) {
		cfg := &Config{
			Enabled:        false,
			HeaderName:     "X-Real-IP",
			ProcessHeaders: []HeaderConfig{{HeaderName: "X-Forwarded-For", Depth: -1}},
			ForceOverwrite: false,
			TrustAll:       true,
		}

		plugin, err := New(context.TODO(), &noopHandler{}, cfg, pluginName)
		if err != nil {
			t.Fatalf("failed to create plugin: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Forwarded-For", "203.0.113.1")

		rr := httptest.NewRecorder()
		plugin.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status code %d, but got: %d", http.StatusOK, rr.Code)
		}

		// Verify that no header was added when plugin is disabled
		if req.Header.Get("X-Real-IP") != "" {
			t.Error("expected no X-Real-IP header when plugin is disabled, but got one")
		}
	})

	t.Run("SingleIPFromXForwardedFor", func(t *testing.T) {
		cfg := &Config{
			Enabled:        true,
			HeaderName:     "X-Real-IP",
			ProcessHeaders: []HeaderConfig{{HeaderName: "X-Forwarded-For", Depth: -1}},
			TrustAll:       true,
		}

		plugin, err := New(context.TODO(), &noopHandler{}, cfg, pluginName)
		if err != nil {
			t.Fatalf("failed to create plugin: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Forwarded-For", "203.0.113.1")

		rr := httptest.NewRecorder()
		plugin.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status code %d, but got: %d", http.StatusOK, rr.Code)
		}

		realIP := req.Header.Get("X-Real-IP")
		if realIP != "203.0.113.1" {
			t.Errorf("expected X-Real-IP to be '203.0.113.1', but got: '%s'", realIP)
		}
	})

	t.Run("MultipleIPsFromXForwardedFor", func(t *testing.T) {
		cfg := &Config{
			Enabled:        true,
			HeaderName:     "X-Real-IP",
			ProcessHeaders: []HeaderConfig{{HeaderName: "X-Forwarded-For", Depth: -1}},
			TrustAll:       true,
		}

		plugin, err := New(context.TODO(), &noopHandler{}, cfg, pluginName)
		if err != nil {
			t.Fatalf("failed to create plugin: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Forwarded-For", "203.0.113.1, 198.51.100.1, 192.168.1.1")

		rr := httptest.NewRecorder()
		plugin.ServeHTTP(rr, req)

		realIP := req.Header.Get("X-Real-IP")
		if realIP != "203.0.113.1" {
			t.Errorf("expected X-Real-IP to be '203.0.113.1' (first IP), but got: '%s'", realIP)
		}
	})

	t.Run("IPWithPort", func(t *testing.T) {
		cfg := &Config{
			Enabled:        true,
			HeaderName:     "X-Real-IP",
			ProcessHeaders: []HeaderConfig{{HeaderName: "X-Forwarded-For", Depth: -1}},
			TrustAll:       true,
		}

		plugin, err := New(context.TODO(), &noopHandler{}, cfg, pluginName)
		if err != nil {
			t.Fatalf("failed to create plugin: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Forwarded-For", "203.0.113.1:8080")

		rr := httptest.NewRecorder()
		plugin.ServeHTTP(rr, req)

		realIP := req.Header.Get("X-Real-IP")
		if realIP != "203.0.113.1" {
			t.Errorf("expected X-Real-IP to be '203.0.113.1' (without port), but got: '%s'", realIP)
		}
	})

	t.Run("IPv6Address", func(t *testing.T) {
		cfg := &Config{
			Enabled:        true,
			HeaderName:     "X-Real-IP",
			ProcessHeaders: []HeaderConfig{{HeaderName: "X-Forwarded-For", Depth: -1}},
			TrustAll:       true,
		}

		plugin, err := New(context.TODO(), &noopHandler{}, cfg, pluginName)
		if err != nil {
			t.Fatalf("failed to create plugin: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Forwarded-For", "2001:db8::1")

		rr := httptest.NewRecorder()
		plugin.ServeHTTP(rr, req)

		realIP := req.Header.Get("X-Real-IP")
		if realIP != "2001:db8::1" {
			t.Errorf("expected X-Real-IP to be '2001:db8::1', but got: '%s'", realIP)
		}
	})

	t.Run("IPv6AddressWithPort", func(t *testing.T) {
		cfg := &Config{
			Enabled:        true,
			HeaderName:     "X-Real-IP",
			ProcessHeaders: []HeaderConfig{{HeaderName: "X-Forwarded-For", Depth: -1}},
			TrustAll:       true,
		}

		plugin, err := New(context.TODO(), &noopHandler{}, cfg, pluginName)
		if err != nil {
			t.Fatalf("failed to create plugin: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Forwarded-For", "[2001:db8::1]:8080")

		rr := httptest.NewRecorder()
		plugin.ServeHTTP(rr, req)

		realIP := req.Header.Get("X-Real-IP")
		if realIP != "2001:db8::1" {
			t.Errorf("expected X-Real-IP to be '2001:db8::1' (without port), but got: '%s'", realIP)
		}
	})

	t.Run("MultipleHeadersInOrder", func(t *testing.T) {
		cfg := &Config{
			Enabled:        true,
			HeaderName:     "X-Real-IP",
			ProcessHeaders: []HeaderConfig{{HeaderName: "X-Forwarded-For", Depth: -1}, {HeaderName: "X-Real-IP", Depth: -1}, {HeaderName: "CF-Connecting-IP", Depth: -1}},
			TrustAll:       true,
		}

		plugin, err := New(context.TODO(), &noopHandler{}, cfg, pluginName)
		if err != nil {
			t.Fatalf("failed to create plugin: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Real-IP", "198.51.100.1")
		req.Header.Set("CF-Connecting-IP", "203.0.113.1")

		rr := httptest.NewRecorder()
		plugin.ServeHTTP(rr, req)

		realIP := req.Header.Get("X-Real-IP")
		if realIP != "198.51.100.1" {
			t.Errorf("expected X-Real-IP to be '198.51.100.1' (from X-Real-IP header), but got: '%s'", realIP)
		}
	})

	t.Run("FirstHeaderPriority", func(t *testing.T) {
		cfg := &Config{
			Enabled:        true,
			HeaderName:     "X-Real-IP",
			ProcessHeaders: []HeaderConfig{{HeaderName: "X-Forwarded-For", Depth: -1}, {HeaderName: "CF-Connecting-IP", Depth: -1}},
			TrustAll:       true,
		}

		plugin, err := New(context.TODO(), &noopHandler{}, cfg, pluginName)
		if err != nil {
			t.Fatalf("failed to create plugin: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Forwarded-For", "203.0.113.1")
		req.Header.Set("CF-Connecting-IP", "198.51.100.1")

		rr := httptest.NewRecorder()
		plugin.ServeHTTP(rr, req)

		realIP := req.Header.Get("X-Real-IP")
		if realIP != "203.0.113.1" {
			t.Errorf("expected X-Real-IP to be '203.0.113.1' (from X-Forwarded-For), but got: '%s'", realIP)
		}
	})

	t.Run("NoValidHeaders", func(t *testing.T) {
		cfg := &Config{
			Enabled:        true,
			HeaderName:     "X-Real-IP",
			ProcessHeaders: []HeaderConfig{{HeaderName: "X-Forwarded-For", Depth: -1}, {HeaderName: "CF-Connecting-IP", Depth: -1}},
			TrustAll:       true,
		}

		plugin, err := New(context.TODO(), &noopHandler{}, cfg, pluginName)
		if err != nil {
			t.Fatalf("failed to create plugin: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		// No headers set

		rr := httptest.NewRecorder()
		plugin.ServeHTTP(rr, req)

		realIP := req.Header.Get("X-Real-IP")
		if realIP != "" {
			t.Errorf("expected X-Real-IP to be empty when no headers are present, but got: '%s'", realIP)
		}
	})

	t.Run("InvalidIPAddresses", func(t *testing.T) {
		cfg := &Config{
			Enabled:        true,
			HeaderName:     "X-Real-IP",
			ProcessHeaders: []HeaderConfig{{HeaderName: "X-Forwarded-For", Depth: -1}},
			TrustAll:       true,
		}

		plugin, err := New(context.TODO(), &noopHandler{}, cfg, pluginName)
		if err != nil {
			t.Fatalf("failed to create plugin: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Forwarded-For", "invalid-ip, not-an-ip, 203.0.113.1")

		rr := httptest.NewRecorder()
		plugin.ServeHTTP(rr, req)

		realIP := req.Header.Get("X-Real-IP")
		if realIP != "invalid-ip" {
			t.Errorf("expected X-Real-IP to be 'invalid-ip' (first value after cleaning), but got: '%s'", realIP)
		}
	})

	t.Run("WhitespaceHandling", func(t *testing.T) {
		cfg := &Config{
			Enabled:        true,
			HeaderName:     "X-Real-IP",
			ProcessHeaders: []HeaderConfig{{HeaderName: "X-Forwarded-For", Depth: -1}},
			TrustAll:       true,
		}

		plugin, err := New(context.TODO(), &noopHandler{}, cfg, pluginName)
		if err != nil {
			t.Fatalf("failed to create plugin: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Forwarded-For", "  203.0.113.1  ,  198.51.100.1  ")

		rr := httptest.NewRecorder()
		plugin.ServeHTTP(rr, req)

		realIP := req.Header.Get("X-Real-IP")
		if realIP != "203.0.113.1" {
			t.Errorf("expected X-Real-IP to be '203.0.113.1' (trimmed), but got: '%s'", realIP)
		}
	})

	t.Run("CustomHeaderName", func(t *testing.T) {
		cfg := &Config{
			Enabled:        true,
			HeaderName:     "X-Client-IP",
			ProcessHeaders: []HeaderConfig{{HeaderName: "X-Forwarded-For", Depth: -1}},
			TrustAll:       true,
		}

		plugin, err := New(context.TODO(), &noopHandler{}, cfg, pluginName)
		if err != nil {
			t.Fatalf("failed to create plugin: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Forwarded-For", "203.0.113.1")

		rr := httptest.NewRecorder()
		plugin.ServeHTTP(rr, req)

		clientIP := req.Header.Get("X-Client-IP")
		if clientIP != "203.0.113.1" {
			t.Errorf("expected X-Client-IP to be '203.0.113.1', but got: '%s'", clientIP)
		}

		// Ensure the default header name wasn't set
		realIP := req.Header.Get("X-Real-IP")
		if realIP != "" {
			t.Errorf("expected X-Real-IP to be empty, but got: '%s'", realIP)
		}
	})

	t.Run("UpdateExistingHeader", func(t *testing.T) {
		cfg := &Config{
			Enabled:        true,
			HeaderName:     "X-Real-IP",
			ProcessHeaders: []HeaderConfig{{HeaderName: "X-Forwarded-For", Depth: -1}},
			TrustAll:       true,
		}

		plugin, err := New(context.TODO(), &noopHandler{}, cfg, pluginName)
		if err != nil {
			t.Fatalf("failed to create plugin: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Real-IP", "old-value")
		req.Header.Set("X-Forwarded-For", "203.0.113.1")

		rr := httptest.NewRecorder()
		plugin.ServeHTTP(rr, req)

		realIP := req.Header.Get("X-Real-IP")
		if realIP != "203.0.113.1" {
			t.Errorf("expected X-Real-IP to be updated to '203.0.113.1', but got: '%s'", realIP)
		}
	})

	t.Run("ForceOverwriteEnabled", func(t *testing.T) {
		cfg := &Config{
			Enabled:        true,
			HeaderName:     "X-Real-IP",
			ProcessHeaders: []HeaderConfig{{HeaderName: "X-Forwarded-For", Depth: -1}},
			ForceOverwrite: true,
			TrustAll:       true,
		}

		plugin, err := New(context.TODO(), &noopHandler{}, cfg, pluginName)
		if err != nil {
			t.Fatalf("failed to create plugin: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		// No headers set, should set header to empty value due to forceOverwrite

		rr := httptest.NewRecorder()
		plugin.ServeHTTP(rr, req)

		// Header should be set to empty string when forceOverwrite is true but no IP found
		realIP := req.Header.Get("X-Real-IP")
		if realIP != "" {
			t.Errorf("expected X-Real-IP to be empty when forceOverwrite is enabled but no IP found, but got: '%s'", realIP)
		}

		// Verify the header was actually set (Go's Set method with empty string does set the header)
		values := req.Header.Values("X-Real-IP")
		if len(values) == 0 {
			t.Error("expected X-Real-IP header to be set when forceOverwrite is enabled, but header is missing")
		}
	})

	t.Run("ForceOverwriteDisabled", func(t *testing.T) {
		cfg := &Config{
			Enabled:        true,
			HeaderName:     "X-Real-IP",
			ProcessHeaders: []HeaderConfig{{HeaderName: "X-Forwarded-For", Depth: -1}},
			ForceOverwrite: false,
			TrustAll:       true,
		}

		plugin, err := New(context.TODO(), &noopHandler{}, cfg, pluginName)
		if err != nil {
			t.Fatalf("failed to create plugin: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		// No headers set, should not set header when forceOverwrite is disabled

		rr := httptest.NewRecorder()
		plugin.ServeHTTP(rr, req)

		// Header should not be set at all
		_, exists := req.Header[http.CanonicalHeaderKey("X-Real-IP")]
		if exists {
			t.Error("expected X-Real-IP header not to be set when forceOverwrite is disabled and no IP found")
		}
	})

	t.Run("SyntheticClientAddressHeader", func(t *testing.T) {
		cfg := &Config{
			Enabled:        true,
			HeaderName:     "X-Real-IP",
			ProcessHeaders: []HeaderConfig{{HeaderName: "clientAddress", Depth: -1}},
			ForceOverwrite: true,
			TrustAll:       true,
		}

		plugin, err := New(context.TODO(), &noopHandler{}, cfg, pluginName)
		if err != nil {
			t.Fatalf("failed to create plugin: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		// Should use RemoteAddr via synthetic clientAddress header

		rr := httptest.NewRecorder()
		plugin.ServeHTTP(rr, req)

		realIP := req.Header.Get("X-Real-IP")
		if realIP == "" {
			t.Error("expected X-Real-IP to be set from RemoteAddr via clientAddress, but got empty")
		}
		// The default test request has RemoteAddr of "192.0.2.1:1234"
		if realIP != "192.0.2.1" {
			t.Errorf("expected X-Real-IP to be '192.0.2.1' (from RemoteAddr), but got: '%s'", realIP)
		}
	})

	t.Run("SyntheticClientAddressWithHeaderPriority", func(t *testing.T) {
		cfg := &Config{
			Enabled:        true,
			HeaderName:     "X-Real-IP",
			ProcessHeaders: []HeaderConfig{{HeaderName: "X-Forwarded-For", Depth: -1}, {HeaderName: "clientAddress", Depth: -1}},
			TrustAll:       true,
			ForceOverwrite: true,
		}

		plugin, err := New(context.TODO(), &noopHandler{}, cfg, pluginName)
		if err != nil {
			t.Fatalf("failed to create plugin: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Forwarded-For", "203.0.113.1")
		// Should use header IP, not synthetic clientAddress

		rr := httptest.NewRecorder()
		plugin.ServeHTTP(rr, req)

		realIP := req.Header.Get("X-Real-IP")
		if realIP != "203.0.113.1" {
			t.Errorf("expected X-Real-IP to be '203.0.113.1' (from header, not clientAddress), but got: '%s'", realIP)
		}
	})

	t.Run("ForceOverwritePreventsHeaderSpoofing", func(t *testing.T) {
		cfg := &Config{
			Enabled:        true,
			HeaderName:     "X-Real-IP",
			ProcessHeaders: []HeaderConfig{{HeaderName: "X-Forwarded-For", Depth: -1}},
			ForceOverwrite: true,
			TrustAll:       true,
		}

		plugin, err := New(context.TODO(), &noopHandler{}, cfg, pluginName)
		if err != nil {
			t.Fatalf("failed to create plugin: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Real-IP", "spoofed-value")
		// No X-Forwarded-For header, so should overwrite spoofed value with empty

		rr := httptest.NewRecorder()
		plugin.ServeHTTP(rr, req)

		realIP := req.Header.Get("X-Real-IP")
		if realIP != "" {
			t.Errorf("expected X-Real-IP to be empty (overwriting spoofed value), but got: '%s'", realIP)
		}
	})

	t.Run("DepthLeftmost", func(t *testing.T) {
		cfg := &Config{
			Enabled:        true,
			HeaderName:     "X-Real-IP",
			ProcessHeaders: []HeaderConfig{{HeaderName: "X-Forwarded-For", Depth: -1}},
			ForceOverwrite: true,
			TrustAll:       true,
		}

		plugin, err := New(context.TODO(), &noopHandler{}, cfg, pluginName)
		if err != nil {
			t.Fatalf("failed to create plugin: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Forwarded-For", "203.0.113.1, 198.51.100.1, 192.168.1.1")

		rr := httptest.NewRecorder()
		plugin.ServeHTTP(rr, req)

		realIP := req.Header.Get("X-Real-IP")
		if realIP != "203.0.113.1" {
			t.Errorf("expected X-Real-IP to be '203.0.113.1' (leftmost), but got: '%s'", realIP)
		}
	})

	t.Run("DepthRightmost", func(t *testing.T) {
		cfg := &Config{
			Enabled:        true,
			HeaderName:     "X-Real-IP",
			ProcessHeaders: []HeaderConfig{{HeaderName: "X-Forwarded-For", Depth: 0}},
			ForceOverwrite: true,
			TrustAll:       true,
		}

		plugin, err := New(context.TODO(), &noopHandler{}, cfg, pluginName)
		if err != nil {
			t.Fatalf("failed to create plugin: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Forwarded-For", "203.0.113.1, 198.51.100.1, 192.168.1.1")

		rr := httptest.NewRecorder()
		plugin.ServeHTTP(rr, req)

		realIP := req.Header.Get("X-Real-IP")
		if realIP != "192.168.1.1" {
			t.Errorf("expected X-Real-IP to be '192.168.1.1' (rightmost), but got: '%s'", realIP)
		}
	})

	t.Run("DepthSecondFromRight", func(t *testing.T) {
		cfg := &Config{
			Enabled:        true,
			HeaderName:     "X-Real-IP",
			ProcessHeaders: []HeaderConfig{{HeaderName: "X-Forwarded-For", Depth: 1}},
			ForceOverwrite: true,
			TrustAll:       true,
		}

		plugin, err := New(context.TODO(), &noopHandler{}, cfg, pluginName)
		if err != nil {
			t.Fatalf("failed to create plugin: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Forwarded-For", "203.0.113.1, 198.51.100.1, 192.168.1.1")

		rr := httptest.NewRecorder()
		plugin.ServeHTTP(rr, req)

		realIP := req.Header.Get("X-Real-IP")
		if realIP != "198.51.100.1" {
			t.Errorf("expected X-Real-IP to be '198.51.100.1' (second from right), but got: '%s'", realIP)
		}
	})

	t.Run("DepthOutOfBounds", func(t *testing.T) {
		cfg := &Config{
			Enabled:        true,
			HeaderName:     "X-Real-IP",
			ProcessHeaders: []HeaderConfig{{HeaderName: "X-Forwarded-For", Depth: 5}},
			ForceOverwrite: true,
			TrustAll:       true,
		}

		plugin, err := New(context.TODO(), &noopHandler{}, cfg, pluginName)
		if err != nil {
			t.Fatalf("failed to create plugin: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Forwarded-For", "203.0.113.1, 198.51.100.1")

		rr := httptest.NewRecorder()
		plugin.ServeHTTP(rr, req)

		// Should be empty because depth 5 is out of bounds for only 2 IPs
		realIP := req.Header.Get("X-Real-IP")
		if realIP != "" {
			t.Errorf("expected X-Real-IP to be empty (depth out of bounds), but got: '%s'", realIP)
		}
	})

	// Edge case tests for crash prevention

	t.Run("EmptyHeaderName", func(t *testing.T) {
		cfg := &Config{
			Enabled:        true,
			HeaderName:     "X-Real-IP",
			ProcessHeaders: []HeaderConfig{{HeaderName: "", Depth: -1}},
			ForceOverwrite: true,
			TrustAll:       true,
		}

		plugin, err := New(context.TODO(), &noopHandler{}, cfg, pluginName)
		if err != nil {
			t.Fatalf("failed to create plugin: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rr := httptest.NewRecorder()

		// This should not panic with empty header name
		plugin.ServeHTTP(rr, req)
	})

	t.Run("VeryLongString", func(t *testing.T) {
		cfg := &Config{
			Enabled:        true,
			HeaderName:     "X-Real-IP",
			ProcessHeaders: []HeaderConfig{{HeaderName: "X-Forwarded-For", Depth: -1}},
			ForceOverwrite: true,
			TrustAll:       true,
		}

		plugin, err := New(context.TODO(), &noopHandler{}, cfg, pluginName)
		if err != nil {
			t.Fatalf("failed to create plugin: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		// Very long string
		longString := strings.Repeat("a", 1000)
		req.Header.Set("X-Forwarded-For", longString)

		rr := httptest.NewRecorder()

		// This should not panic with very long strings
		plugin.ServeHTTP(rr, req)

		// Should pass through the long string (no validation)
		realIP := req.Header.Get("X-Real-IP")
		if realIP != longString {
			t.Errorf("expected X-Real-IP to be the long string, but got: '%s'", realIP[:50]+"...")
		}
	})

	t.Run("NegativeDepthExtreme", func(t *testing.T) {
		cfg := &Config{
			Enabled:        true,
			HeaderName:     "X-Real-IP",
			ProcessHeaders: []HeaderConfig{{HeaderName: "X-Forwarded-For", Depth: -1000000}},
			ForceOverwrite: true,
			TrustAll:       true,
		}

		plugin, err := New(context.TODO(), &noopHandler{}, cfg, pluginName)
		if err != nil {
			t.Fatalf("failed to create plugin: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Forwarded-For", "203.0.113.1, 198.51.100.1")

		rr := httptest.NewRecorder()

		// This should not panic with extreme negative depth
		plugin.ServeHTTP(rr, req)

		// Should still get leftmost IP since any negative depth means leftmost
		realIP := req.Header.Get("X-Real-IP")
		if realIP != "203.0.113.1" {
			t.Errorf("expected X-Real-IP to be '203.0.113.1' (leftmost for negative depth), but got: '%s'", realIP)
		}
	})

	t.Run("VeryLargeDepth", func(t *testing.T) {
		cfg := &Config{
			Enabled:        true,
			HeaderName:     "X-Real-IP",
			ProcessHeaders: []HeaderConfig{{HeaderName: "X-Forwarded-For", Depth: 1000000}},
			ForceOverwrite: true,
			TrustAll:       true,
		}

		plugin, err := New(context.TODO(), &noopHandler{}, cfg, pluginName)
		if err != nil {
			t.Fatalf("failed to create plugin: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Forwarded-For", "203.0.113.1, 198.51.100.1")

		rr := httptest.NewRecorder()

		// This should not panic with very large depth
		plugin.ServeHTTP(rr, req)

		// Should result in empty header due to depth out of bounds
		realIP := req.Header.Get("X-Real-IP")
		if realIP != "" {
			t.Errorf("expected X-Real-IP to be empty for out of bounds depth, but got: '%s'", realIP)
		}
	})

	// Trust functionality tests
	t.Run("TrustedIPAllowsHeaders", func(t *testing.T) {
		cfg := &Config{
			Enabled:        true,
			HeaderName:     "X-Real-IP",
			ProcessHeaders: []HeaderConfig{{HeaderName: "X-Forwarded-For", Depth: -1}},
			ForceOverwrite: true,
			TrustAll:       false,                    // Don't trust all
			TrustedIPs:     []string{"192.0.2.0/24"}, // Trust test network
			TrustedHeader:  "X-Is-Trusted",
		}

		plugin, err := New(context.TODO(), &noopHandler{}, cfg, pluginName)
		if err != nil {
			t.Fatalf("failed to create plugin: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.0.2.1:1234" // Trusted IP
		req.Header.Set("X-Forwarded-For", "203.0.113.1")

		rr := httptest.NewRecorder()
		plugin.ServeHTTP(rr, req)

		// Should process headers from trusted source
		realIP := req.Header.Get("X-Real-IP")
		if realIP != "203.0.113.1" {
			t.Errorf("expected X-Real-IP to be '203.0.113.1' from trusted source, but got: '%s'", realIP)
		}

		// Should set trust header to "yes"
		trusted := req.Header.Get("X-Is-Trusted")
		if trusted != "yes" {
			t.Errorf("expected X-Is-Trusted to be 'yes' for trusted source, but got: '%s'", trusted)
		}
	})

	t.Run("UntrustedIPIgnoresHeaders", func(t *testing.T) {
		cfg := &Config{
			Enabled:        true,
			HeaderName:     "X-Real-IP",
			ProcessHeaders: []HeaderConfig{{HeaderName: "X-Forwarded-For", Depth: -1}, {HeaderName: "clientAddress", Depth: -1}},
			TrustAll:       false, // Don't trust all
			ForceOverwrite: true,
			TrustedIPs:     []string{"192.0.2.0/24"}, // Trust only test network
			TrustedHeader:  "X-Is-Trusted",
		}

		plugin, err := New(context.TODO(), &noopHandler{}, cfg, pluginName)
		if err != nil {
			t.Fatalf("failed to create plugin: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "203.0.113.1:1234" // Untrusted IP (not in 192.0.2.0/24)
		req.Header.Set("X-Forwarded-For", "spoofed-ip")

		rr := httptest.NewRecorder()
		plugin.ServeHTTP(rr, req)

		// Should ignore X-Forwarded-For and use clientAddress instead
		realIP := req.Header.Get("X-Real-IP")
		if realIP != "203.0.113.1" {
			t.Errorf("expected X-Real-IP to be '203.0.113.1' (from RemoteAddr), but got: '%s'", realIP)
		}

		// Should set trust header to "no"
		trusted := req.Header.Get("X-Is-Trusted")
		if trusted != "no" {
			t.Errorf("expected X-Is-Trusted to be 'no' for untrusted source, but got: '%s'", trusted)
		}
	})

	t.Run("NoTrustedIPsConfigured", func(t *testing.T) {
		cfg := &Config{
			Enabled:        true,
			HeaderName:     "X-Real-IP",
			ProcessHeaders: []HeaderConfig{{HeaderName: "X-Forwarded-For", Depth: -1}},
			ForceOverwrite: true,
			TrustAll:       true,       // Trust all when no specific IPs configured
			TrustedIPs:     []string{}, // No trusted IPs configured
			TrustedHeader:  "",         // No trust header
		}

		plugin, err := New(context.TODO(), &noopHandler{}, cfg, pluginName)
		if err != nil {
			t.Fatalf("failed to create plugin: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "203.0.113.1:1234" // Any IP
		req.Header.Set("X-Forwarded-For", "198.51.100.1")

		rr := httptest.NewRecorder()
		plugin.ServeHTTP(rr, req)

		// Should process headers normally (trust all when no trustedIPs configured)
		realIP := req.Header.Get("X-Real-IP")
		if realIP != "198.51.100.1" {
			t.Errorf("expected X-Real-IP to be '198.51.100.1' when no trustedIPs configured, but got: '%s'", realIP)
		}

		// Should not set trust header when not configured
		trusted := req.Header.Get("X-Is-Trusted")
		if trusted != "" {
			t.Errorf("expected X-Is-Trusted to be empty when not configured, but got: '%s'", trusted)
		}
	})

	t.Run("InvalidTrustedIPConfig", func(t *testing.T) {
		cfg := &Config{
			Enabled:        true,
			HeaderName:     "X-Real-IP",
			ProcessHeaders: []HeaderConfig{{HeaderName: "X-Forwarded-For", Depth: -1}},
			ForceOverwrite: true,
			TrustAll:       false,
			TrustedIPs:     []string{"invalid-cidr"}, // Invalid CIDR
			TrustedHeader:  "X-Is-Trusted",
		}

		plugin, err := New(context.TODO(), &noopHandler{}, cfg, pluginName)
		if err == nil {
			t.Error("expected error for invalid trusted IP CIDR, but got none")
		}
		if plugin != nil {
			t.Error("expected plugin to be nil for invalid config, but got instance")
		}
	})

	t.Run("LocalIPRangesTrusted", func(t *testing.T) {
		localIPRanges := []string{
			"127.0.0.0/8",    // IPv4 loopback
			"10.0.0.0/8",     // RFC1918
			"172.16.0.0/12",  // RFC1918
			"192.168.0.0/16", // RFC1918
			"::1/128",        // IPv6 loopback
			"fc00::/7",       // IPv6 unique local addr
			"fe80::/10",      // IPv6 link-local addr
		}

		cfg := &Config{
			Enabled:        true,
			HeaderName:     "X-Real-IP",
			ProcessHeaders: []HeaderConfig{{HeaderName: "X-Forwarded-For", Depth: -1}, {HeaderName: "clientAddress", Depth: -1}},
			ForceOverwrite: true,
			TrustAll:       false,
			TrustedIPs:     localIPRanges,
			TrustedHeader:  "X-Is-Trusted",
		}

		plugin, err := New(context.TODO(), &noopHandler{}, cfg, pluginName)
		if err != nil {
			t.Fatalf("failed to create plugin: %v", err)
		}

		testCases := []struct {
			name          string
			remoteAddr    string
			expectedTrust string
			shouldProcess bool
		}{
			{"IPv4 loopback", "127.0.0.1:8080", "yes", true},
			{"IPv4 RFC1918 10.x", "10.1.2.3:8080", "yes", true},
			{"IPv4 RFC1918 172.16.x", "172.16.1.1:8080", "yes", true},
			{"IPv4 RFC1918 192.168.x", "192.168.1.100:8080", "yes", true},
			{"IPv6 loopback", "[::1]:8080", "yes", true},
			{"IPv6 unique local", "[fc00::1]:8080", "yes", true},
			{"IPv6 link-local", "[fe80::1]:8080", "yes", true},
			{"Public IPv4", "8.8.8.8:8080", "no", false},
			{"Public IPv6", "[2001:db8::1]:8080", "no", false},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				req.RemoteAddr = tc.remoteAddr
				req.Header.Set("X-Forwarded-For", "spoofed-ip")

				rr := httptest.NewRecorder()
				plugin.ServeHTTP(rr, req)

				// Check trust header
				trusted := req.Header.Get("X-Is-Trusted")
				if trusted != tc.expectedTrust {
					t.Errorf("expected X-Is-Trusted to be '%s' for %s, but got: '%s'", tc.expectedTrust, tc.remoteAddr, trusted)
				}

				// Check if headers were processed correctly
				realIP := req.Header.Get("X-Real-IP")
				if tc.shouldProcess {
					// Trusted source should process X-Forwarded-For
					if realIP != "spoofed-ip" {
						t.Errorf("trusted source should process headers, expected 'spoofed-ip', got: '%s'", realIP)
					}
				} else {
					// Untrusted source should use RemoteAddr (cleaned)
					expectedIP := plugin.(*Plugin).cleanIPAddress(tc.remoteAddr)
					if realIP != expectedIP {
						t.Errorf("untrusted source should use RemoteAddr, expected '%s', got: '%s'", expectedIP, realIP)
					}
				}
			})
		}
	})

	t.Run("TrustAllEnabled", func(t *testing.T) {
		cfg := &Config{
			Enabled:        true,
			HeaderName:     "X-Real-IP",
			ProcessHeaders: []HeaderConfig{{HeaderName: "X-Forwarded-For", Depth: -1}},
			ForceOverwrite: true,
			TrustAll:       true,       // Trust all sources
			TrustedIPs:     []string{}, // Can be empty when trustAll is true
			TrustedHeader:  "X-Is-Trusted",
		}

		plugin, err := New(context.TODO(), &noopHandler{}, cfg, pluginName)
		if err != nil {
			t.Fatalf("failed to create plugin: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "203.0.113.1:1234" // Any IP should be trusted
		req.Header.Set("X-Forwarded-For", "198.51.100.1")

		rr := httptest.NewRecorder()
		plugin.ServeHTTP(rr, req)

		// Should process headers from any source when trustAll is true
		realIP := req.Header.Get("X-Real-IP")
		if realIP != "198.51.100.1" {
			t.Errorf("expected X-Real-IP to be '198.51.100.1' when trustAll is true, but got: '%s'", realIP)
		}

		// Should set trust header to "yes"
		trusted := req.Header.Get("X-Is-Trusted")
		if trusted != "yes" {
			t.Errorf("expected X-Is-Trusted to be 'yes' when trustAll is true, but got: '%s'", trusted)
		}
	})

	t.Run("TrustAllDisabledNoTrustedIPs", func(t *testing.T) {
		cfg := &Config{
			Enabled:        true,
			HeaderName:     "X-Real-IP",
			ProcessHeaders: []HeaderConfig{{HeaderName: "X-Forwarded-For", Depth: -1}},
			ForceOverwrite: true,
			TrustAll:       false,      // Don't trust all
			TrustedIPs:     []string{}, // Empty trusted IPs
			TrustedHeader:  "X-Is-Trusted",
		}

		plugin, err := New(context.TODO(), &noopHandler{}, cfg, pluginName)
		if err == nil {
			t.Error("expected error when trustAll is false and trustedIPs is empty, but got none")
		}
		if plugin != nil {
			t.Error("expected plugin to be nil for invalid config, but got instance")
		}
	})
}

func TestExtractRealIP(t *testing.T) {
	cfg := &Config{
		Enabled:        true,
		HeaderName:     "X-Real-IP",
		ProcessHeaders: []HeaderConfig{{HeaderName: "X-Forwarded-For", Depth: -1}, {HeaderName: "X-Real-IP", Depth: -1}, {HeaderName: "CF-Connecting-IP", Depth: -1}},
		TrustAll:       true,
	}

	plugin, err := New(context.TODO(), &noopHandler{}, cfg, pluginName)
	if err != nil {
		t.Fatalf("failed to create plugin: %v", err)
	}

	p := plugin.(*Plugin)

	tests := []struct {
		name     string
		headers  map[string]string
		expected string
	}{
		{
			name: "SingleIPFromFirstHeader",
			headers: map[string]string{
				"X-Forwarded-For": "203.0.113.1",
			},
			expected: "203.0.113.1",
		},
		{
			name: "MultipleIPsFromFirstHeader",
			headers: map[string]string{
				"X-Forwarded-For": "203.0.113.1, 198.51.100.1",
			},
			expected: "203.0.113.1",
		},
		{
			name: "SecondHeaderWhenFirstEmpty",
			headers: map[string]string{
				"X-Real-IP": "198.51.100.1",
			},
			expected: "198.51.100.1",
		},
		{
			name: "FirstHeaderPriorityOverSecond",
			headers: map[string]string{
				"X-Forwarded-For": "203.0.113.1",
				"X-Real-IP":       "198.51.100.1",
			},
			expected: "203.0.113.1",
		},
		{
			name:     "NoHeaders",
			headers:  map[string]string{},
			expected: "",
		},
		{
			name: "InvalidIPsNotSkipped",
			headers: map[string]string{
				"X-Forwarded-For": "invalid-ip, 203.0.113.1",
			},
			expected: "invalid-ip",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			for name, value := range tt.headers {
				req.Header.Set(name, value)
			}

			result := p.extractRealIP(req, true)
			if result != tt.expected {
				t.Errorf("expected '%s', but got '%s'", tt.expected, result)
			}
		})
	}
}

func TestCleanIPAddress(t *testing.T) {
	cfg := &Config{
		Enabled:        true,
		HeaderName:     "X-Real-IP",
		ProcessHeaders: []HeaderConfig{{HeaderName: "X-Forwarded-For", Depth: -1}},
		TrustAll:       true,
	}

	plugin, err := New(context.TODO(), &noopHandler{}, cfg, pluginName)
	if err != nil {
		t.Fatalf("failed to create plugin: %v", err)
	}

	p := plugin.(*Plugin)

	tests := []struct {
		input    string
		expected string
	}{
		{"203.0.113.1", "203.0.113.1"},
		{"203.0.113.1:8080", "203.0.113.1"},
		{"  203.0.113.1  ", "203.0.113.1"},
		{"  203.0.113.1:8080  ", "203.0.113.1"},
		{"2001:db8::1", "2001:db8::1"},
		{"[2001:db8::1]:8080", "2001:db8::1"},
		{"", ""},
		{"   ", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := p.cleanIPAddress(tt.input)
			if result != tt.expected {
				t.Errorf("cleanIPAddress(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCreateConfig(t *testing.T) {
	config := CreateConfig()

	if !config.Enabled {
		t.Error("expected default config to be enabled")
	}

	if config.HeaderName != "X-Real-IP" {
		t.Errorf("expected default HeaderName to be 'X-Real-IP', but got: '%s'", config.HeaderName)
	}

	if !config.ForceOverwrite {
		t.Error("expected default ForceOverwrite to be true")
	}

	if !config.TrustAll {
		t.Error("expected default TrustAll to be true")
	}

	expectedHeaders := []HeaderConfig{
		{HeaderName: "X-Forwarded-For", Depth: -1},
		{HeaderName: "X-Real-IP", Depth: -1},
		{HeaderName: "CF-Connecting-IP", Depth: -1},
		{HeaderName: "clientAddress", Depth: -1},
	}
	if len(config.ProcessHeaders) != len(expectedHeaders) {
		t.Errorf("expected %d process headers, but got %d", len(expectedHeaders), len(config.ProcessHeaders))
	}

	for i, expected := range expectedHeaders {
		if i >= len(config.ProcessHeaders) || config.ProcessHeaders[i].HeaderName != expected.HeaderName || config.ProcessHeaders[i].Depth != expected.Depth {
			t.Errorf("expected ProcessHeaders[%d] to be %+v, but got %+v", i, expected, config.ProcessHeaders[i])
		}
	}
}
