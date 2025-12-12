package clientip

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseIP(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantAddress string
		wantPort    string
	}{
		{
			name:        "IPv4 only",
			input:       "192.168.1.1",
			wantAddress: "192.168.1.1",
			wantPort:    "",
		},
		{
			name:        "IPv4 with port",
			input:       "192.168.1.1:8080",
			wantAddress: "192.168.1.1",
			wantPort:    "8080",
		},
		{
			name:        "IPv6 only",
			input:       "::1",
			wantAddress: "::1",
			wantPort:    "",
		},
		{
			name:        "IPv6 with brackets",
			input:       "[::1]",
			wantAddress: "::1",
			wantPort:    "",
		},
		{
			name:        "IPv6 with port",
			input:       "[::1]:8080",
			wantAddress: "::1",
			wantPort:    "8080",
		},
		{
			name:        "IPv6 full",
			input:       "[2001:db8::1]:443",
			wantAddress: "2001:db8::1",
			wantPort:    "443",
		},
		{
			name:        "empty",
			input:       "",
			wantAddress: "",
			wantPort:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := ParseIP(tt.input)
			if ip.Address != tt.wantAddress {
				t.Errorf("Address = %q, want %q", ip.Address, tt.wantAddress)
			}
			if ip.Port != tt.wantPort {
				t.Errorf("Port = %q, want %q", ip.Port, tt.wantPort)
			}
		})
	}
}

func TestParseXForwardedFor(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "single IP",
			input: "192.168.1.1",
			want:  "192.168.1.1",
		},
		{
			name:  "multiple IPs",
			input: "192.168.1.1, 10.0.0.1, 172.16.0.1",
			want:  "192.168.1.1",
		},
		{
			name:  "with spaces",
			input: "  192.168.1.1  ,  10.0.0.1  ",
			want:  "192.168.1.1",
		},
		{
			name:  "empty",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseXForwardedFor(tt.input)
			if got != tt.want {
				t.Errorf("ParseXForwardedFor() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseXForwardedForWithTrust(t *testing.T) {
	trustedProxies := map[string]bool{
		"10.0.0.1":   true,
		"172.16.0.1": true,
	}
	isTrusted := func(ip string) bool {
		return trustedProxies[ip]
	}

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "client not trusted",
			input: "192.168.1.1, 10.0.0.1, 172.16.0.1",
			want:  "192.168.1.1",
		},
		{
			name:  "all trusted except first",
			input: "8.8.8.8, 10.0.0.1",
			want:  "8.8.8.8",
		},
		{
			name:  "all trusted",
			input: "10.0.0.1, 172.16.0.1",
			want:  "10.0.0.1",
		},
		{
			name:  "single untrusted",
			input: "203.0.113.1",
			want:  "203.0.113.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseXForwardedForWithTrust(tt.input, isTrusted)
			if got != tt.want {
				t.Errorf("ParseXForwardedForWithTrust() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestContextOperations(t *testing.T) {
	ctx := context.Background()
	ip := &IP{Address: "192.168.1.1", Port: "8080"}

	// Test WithIP and FromContext
	ctx = WithIP(ctx, ip)
	got, ok := FromContext(ctx)
	if !ok {
		t.Error("FromContext() ok = false, want true")
	}
	if got.Address != ip.Address {
		t.Errorf("FromContext().Address = %q, want %q", got.Address, ip.Address)
	}

	// Test GetIP
	if GetIP(ctx) != ip.Address {
		t.Errorf("GetIP() = %q, want %q", GetIP(ctx), ip.Address)
	}

	// Test GetIP with empty context
	if GetIP(context.Background()) != "" {
		t.Error("GetIP() on empty context should return empty string")
	}
}

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		ip   string
		want bool
	}{
		{"192.168.1.1", true},
		{"10.0.0.1", true},
		{"172.16.0.1", true},
		{"127.0.0.1", true},
		{"::1", true},
		{"8.8.8.8", false},
		{"203.0.113.1", false},
		{"invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			if got := IsPrivateIP(tt.ip); got != tt.want {
				t.Errorf("IsPrivateIP(%q) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}

func TestIsValidIP(t *testing.T) {
	tests := []struct {
		ip   string
		want bool
	}{
		{"192.168.1.1", true},
		{"::1", true},
		{"2001:db8::1", true},
		{"invalid", false},
		{"", false},
		{"192.168.1.256", false},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			if got := IsValidIP(tt.ip); got != tt.want {
				t.Errorf("IsValidIP(%q) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}

func TestHTTPMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		remoteAddr     string
		xff            string
		xRealIP        string
		opts           []Option
		wantIP         string
	}{
		{
			name:       "from RemoteAddr",
			remoteAddr: "192.168.1.1:12345",
			wantIP:     "192.168.1.1",
		},
		{
			name:       "from X-Forwarded-For",
			remoteAddr: "10.0.0.1:12345",
			xff:        "203.0.113.1, 10.0.0.1",
			wantIP:     "203.0.113.1",
		},
		{
			name:       "from X-Real-IP",
			remoteAddr: "10.0.0.1:12345",
			xRealIP:    "203.0.113.1",
			wantIP:     "203.0.113.1",
		},
		{
			name:       "X-Forwarded-For priority over X-Real-IP",
			remoteAddr: "10.0.0.1:12345",
			xff:        "8.8.8.8",
			xRealIP:    "1.1.1.1",
			wantIP:     "8.8.8.8",
		},
		{
			name:       "untrusted proxy ignores headers",
			remoteAddr: "192.168.1.1:12345",
			xff:        "fake.ip",
			opts:       []Option{WithTrustedProxies("10.0.0.0/8")},
			wantIP:     "192.168.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotIP string
			handler := HTTPMiddleware(tt.opts...)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotIP = GetIP(r.Context())
			}))

			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}
			if tt.xRealIP != "" {
				req.Header.Set("X-Real-IP", tt.xRealIP)
			}

			handler.ServeHTTP(httptest.NewRecorder(), req)

			if gotIP != tt.wantIP {
				t.Errorf("GetIP() = %q, want %q", gotIP, tt.wantIP)
			}
		})
	}
}

func TestACL(t *testing.T) {
	tests := []struct {
		name    string
		opts    []ACLOption
		ip      string
		allowed bool
	}{
		{
			name:    "default allow all",
			opts:    nil,
			ip:      "192.168.1.1",
			allowed: true,
		},
		{
			name:    "deny list blocks IP",
			opts:    []ACLOption{WithDenyList("192.168.1.0/24")},
			ip:      "192.168.1.1",
			allowed: false,
		},
		{
			name:    "deny list allows other IP",
			opts:    []ACLOption{WithDenyList("192.168.1.0/24")},
			ip:      "10.0.0.1",
			allowed: true,
		},
		{
			name:    "deny all mode blocks by default",
			opts:    []ACLOption{WithACLMode(ACLModeDenyAll)},
			ip:      "192.168.1.1",
			allowed: false,
		},
		{
			name: "deny all mode allows whitelist",
			opts: []ACLOption{
				WithACLMode(ACLModeDenyAll),
				WithAllowList("192.168.1.0/24"),
			},
			ip:      "192.168.1.1",
			allowed: true,
		},
		{
			name: "deny list takes priority over allow list",
			opts: []ACLOption{
				WithAllowList("192.168.0.0/16"),
				WithDenyList("192.168.1.1"),
			},
			ip:      "192.168.1.1",
			allowed: false,
		},
		{
			name:    "single IP in deny list",
			opts:    []ACLOption{WithDenyList("8.8.8.8")},
			ip:      "8.8.8.8",
			allowed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			acl := NewACL(tt.opts...)
			if got := acl.IsAllowed(tt.ip); got != tt.allowed {
				t.Errorf("IsAllowed(%q) = %v, want %v", tt.ip, got, tt.allowed)
			}
		})
	}
}

func TestACLHTTPMiddleware(t *testing.T) {
	acl := NewACL(WithDenyList("192.168.1.100"))

	var called bool
	handler := HTTPMiddleware()(
		ACLHTTPMiddleware(acl)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
		})),
	)

	// Test allowed IP
	called = false
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if !called {
		t.Error("handler should be called for allowed IP")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	// Test denied IP
	called = false
	req = httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if called {
		t.Error("handler should not be called for denied IP")
	}
	if rr.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
}

func TestTrustedProxies(t *testing.T) {
	o := defaultOptions()

	// Default: trust all
	if !o.isTrustedProxy("any.ip") {
		t.Error("default should trust all proxies")
	}

	// With specific proxies
	WithTrustedProxies("10.0.0.0/8", "192.168.1.1")(o)
	if o.isTrustedProxy("8.8.8.8") {
		t.Error("should not trust 8.8.8.8")
	}
	if !o.isTrustedProxy("10.0.0.1") {
		t.Error("should trust 10.0.0.1")
	}
	if !o.isTrustedProxy("192.168.1.1") {
		t.Error("should trust 192.168.1.1")
	}
}

func TestGeoInfoContext(t *testing.T) {
	ctx := context.Background()
	geo := &GeoInfo{
		Country:     "CN",
		CountryName: "中国",
		City:        "北京",
	}

	ctx = WithGeoInfo(ctx, geo)

	got, ok := GeoInfoFromContext(ctx)
	if !ok {
		t.Error("GeoInfoFromContext() ok = false, want true")
	}
	if got.Country != geo.Country {
		t.Errorf("Country = %q, want %q", got.Country, geo.Country)
	}

	if GetCountry(ctx) != "CN" {
		t.Errorf("GetCountry() = %q, want %q", GetCountry(ctx), "CN")
	}

	if GetCity(ctx) != "北京" {
		t.Errorf("GetCity() = %q, want %q", GetCity(ctx), "北京")
	}

	// Empty context
	if GetCountry(context.Background()) != "" {
		t.Error("GetCountry() on empty context should return empty string")
	}
}

func TestCountryACL(t *testing.T) {
	tests := []struct {
		name    string
		opts    []CountryACLOption
		country string
		allowed bool
	}{
		{
			name:    "default allow all",
			opts:    nil,
			country: "US",
			allowed: true,
		},
		{
			name:    "deny specific country",
			opts:    []CountryACLOption{WithDenyCountries("XX")},
			country: "XX",
			allowed: false,
		},
		{
			name: "deny all except whitelist",
			opts: []CountryACLOption{
				WithCountryACLMode(ACLModeDenyAll),
				WithAllowCountries("CN", "US"),
			},
			country: "CN",
			allowed: true,
		},
		{
			name: "deny all blocks unlisted",
			opts: []CountryACLOption{
				WithCountryACLMode(ACLModeDenyAll),
				WithAllowCountries("CN"),
			},
			country: "US",
			allowed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			acl := NewCountryACL(tt.opts...)
			if got := acl.IsAllowed(tt.country); got != tt.allowed {
				t.Errorf("IsAllowed(%q) = %v, want %v", tt.country, got, tt.allowed)
			}
		})
	}
}
