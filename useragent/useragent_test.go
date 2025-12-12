package useragent

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name       string
		raw        string
		wantBrowser string
		wantOS     string
		wantDevice DeviceType
	}{
		{
			name:       "Chrome on Windows",
			raw:        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			wantBrowser: "Chrome",
			wantOS:     "Windows",
			wantDevice: DeviceTypeDesktop,
		},
		{
			name:       "Safari on macOS",
			raw:        "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15",
			wantBrowser: "Safari",
			wantOS:     "macOS",
			wantDevice: DeviceTypeDesktop,
		},
		{
			name:       "Firefox on Linux",
			raw:        "Mozilla/5.0 (X11; Linux x86_64; rv:120.0) Gecko/20100101 Firefox/120.0",
			wantBrowser: "Firefox",
			wantOS:     "Linux",
			wantDevice: DeviceTypeDesktop,
		},
		{
			name:       "Chrome on Android Mobile",
			raw:        "Mozilla/5.0 (Linux; Android 13; SM-G991B) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36",
			wantBrowser: "Chrome",
			wantOS:     "Android",
			wantDevice: DeviceTypeMobile,
		},
		{
			name:       "Safari on iPhone",
			raw:        "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
			wantBrowser: "Safari",
			wantOS:     "iOS",
			wantDevice: DeviceTypeMobile,
		},
		{
			name:       "Safari on iPad",
			raw:        "Mozilla/5.0 (iPad; CPU OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
			wantBrowser: "Safari",
			wantOS:     "iOS",
			wantDevice: DeviceTypeTablet,
		},
		{
			name:       "Edge on Windows",
			raw:        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0",
			wantBrowser: "Edge",
			wantOS:     "Windows",
			wantDevice: DeviceTypeDesktop,
		},
		{
			name:       "Googlebot",
			raw:        "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)",
			wantBrowser: "",
			wantOS:     "",
			wantDevice: DeviceTypeBot,
		},
		{
			name:       "curl",
			raw:        "curl/7.68.0",
			wantBrowser: "",
			wantOS:     "",
			wantDevice: DeviceTypeBot,
		},
		{
			name:       "Python requests",
			raw:        "python-requests/2.28.0",
			wantBrowser: "",
			wantOS:     "",
			wantDevice: DeviceTypeBot,
		},
		{
			name:       "empty",
			raw:        "",
			wantBrowser: "",
			wantOS:     "",
			wantDevice: DeviceTypeDesktop,
		},
		{
			name:       "Android tablet without mobile",
			raw:        "Mozilla/5.0 (Linux; Android 13; SM-T870) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			wantBrowser: "Chrome",
			wantOS:     "Android",
			wantDevice: DeviceTypeTablet,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ua := Parse(tt.raw)
			if ua.Browser.Name != tt.wantBrowser {
				t.Errorf("Browser.Name = %q, want %q", ua.Browser.Name, tt.wantBrowser)
			}
			if ua.OS.Name != tt.wantOS {
				t.Errorf("OS.Name = %q, want %q", ua.OS.Name, tt.wantOS)
			}
			if ua.Device.Type != tt.wantDevice {
				t.Errorf("Device.Type = %q, want %q", ua.Device.Type, tt.wantDevice)
			}
		})
	}
}

func TestUserAgentMethods(t *testing.T) {
	tests := []struct {
		name       string
		deviceType DeviceType
		isMobile   bool
		isTablet   bool
		isDesktop  bool
		isBot      bool
	}{
		{"Mobile", DeviceTypeMobile, true, false, false, false},
		{"Tablet", DeviceTypeTablet, false, true, false, false},
		{"Desktop", DeviceTypeDesktop, false, false, true, false},
		{"Bot", DeviceTypeBot, false, false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ua := &UserAgent{Device: Device{Type: tt.deviceType}}
			if ua.IsMobile() != tt.isMobile {
				t.Errorf("IsMobile() = %v, want %v", ua.IsMobile(), tt.isMobile)
			}
			if ua.IsTablet() != tt.isTablet {
				t.Errorf("IsTablet() = %v, want %v", ua.IsTablet(), tt.isTablet)
			}
			if ua.IsDesktop() != tt.isDesktop {
				t.Errorf("IsDesktop() = %v, want %v", ua.IsDesktop(), tt.isDesktop)
			}
			if ua.IsBot() != tt.isBot {
				t.Errorf("IsBot() = %v, want %v", ua.IsBot(), tt.isBot)
			}
		})
	}
}

func TestUserAgentString(t *testing.T) {
	ua := &UserAgent{
		Browser: Browser{Name: "Chrome"},
		OS:      OS{Name: "Windows"},
		Device:  Device{Type: DeviceTypeDesktop},
	}
	want := "Chrome / Windows / Desktop"
	if got := ua.String(); got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}

	// nil case
	var nilUA *UserAgent
	if got := nilUA.String(); got != "" {
		t.Errorf("nil.String() = %q, want empty", got)
	}
}

func TestContextOperations(t *testing.T) {
	ctx := context.Background()
	ua := &UserAgent{
		Browser: Browser{Name: "Chrome"},
		OS:      OS{Name: "Windows"},
		Device:  Device{Type: DeviceTypeDesktop},
	}

	// Test WithUserAgent and FromContext
	ctx = WithUserAgent(ctx, ua)
	got, ok := FromContext(ctx)
	if !ok {
		t.Error("FromContext() ok = false, want true")
	}
	if got.Browser.Name != ua.Browser.Name {
		t.Errorf("FromContext().Browser.Name = %q, want %q", got.Browser.Name, ua.Browser.Name)
	}

	// Test GetBrowser
	if GetBrowser(ctx) != "Chrome" {
		t.Errorf("GetBrowser() = %q, want %q", GetBrowser(ctx), "Chrome")
	}

	// Test GetOS
	if GetOS(ctx) != "Windows" {
		t.Errorf("GetOS() = %q, want %q", GetOS(ctx), "Windows")
	}

	// Test GetDeviceType
	if GetDeviceType(ctx) != DeviceTypeDesktop {
		t.Errorf("GetDeviceType() = %q, want %q", GetDeviceType(ctx), DeviceTypeDesktop)
	}

	// Test empty context
	emptyCtx := context.Background()
	if GetBrowser(emptyCtx) != "" {
		t.Error("GetBrowser() on empty context should return empty string")
	}
	if GetOS(emptyCtx) != "" {
		t.Error("GetOS() on empty context should return empty string")
	}
	if GetDeviceType(emptyCtx) != DeviceTypeUnknown {
		t.Errorf("GetDeviceType() on empty context = %q, want %q", GetDeviceType(emptyCtx), DeviceTypeUnknown)
	}
}

func TestHTTPMiddleware(t *testing.T) {
	tests := []struct {
		name       string
		userAgent  string
		wantBrowser string
		wantDevice DeviceType
	}{
		{
			name:       "Chrome",
			userAgent:  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			wantBrowser: "Chrome",
			wantDevice: DeviceTypeDesktop,
		},
		{
			name:       "iPhone Safari",
			userAgent:  "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
			wantBrowser: "Safari",
			wantDevice: DeviceTypeMobile,
		},
		{
			name:       "Bot",
			userAgent:  "Googlebot/2.1",
			wantBrowser: "",
			wantDevice: DeviceTypeBot,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotBrowser string
			var gotDevice DeviceType

			handler := HTTPMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotBrowser = GetBrowser(r.Context())
				gotDevice = GetDeviceType(r.Context())
			}))

			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set("User-Agent", tt.userAgent)
			handler.ServeHTTP(httptest.NewRecorder(), req)

			if gotBrowser != tt.wantBrowser {
				t.Errorf("GetBrowser() = %q, want %q", gotBrowser, tt.wantBrowser)
			}
			if gotDevice != tt.wantDevice {
				t.Errorf("GetDeviceType() = %q, want %q", gotDevice, tt.wantDevice)
			}
		})
	}
}

func TestBrowserVersionParsing(t *testing.T) {
	tests := []struct {
		name        string
		raw         string
		wantVersion string
		wantFull    string
	}{
		{
			name:        "Chrome version",
			raw:         "Mozilla/5.0 (Windows NT 10.0) AppleWebKit/537.36 Chrome/120.0.6099.109 Safari/537.36",
			wantVersion: "120",
			wantFull:    "120.0.6099.109",
		},
		{
			name:        "Firefox version",
			raw:         "Mozilla/5.0 (X11; Linux x86_64; rv:120.0) Gecko/20100101 Firefox/120.0",
			wantVersion: "120",
			wantFull:    "120.0",
		},
		{
			name:        "Safari version",
			raw:         "Mozilla/5.0 (Macintosh) AppleWebKit/605.1.15 Version/17.2 Safari/605.1.15",
			wantVersion: "17",
			wantFull:    "17.2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ua := Parse(tt.raw)
			if ua.Browser.Version != tt.wantVersion {
				t.Errorf("Browser.Version = %q, want %q", ua.Browser.Version, tt.wantVersion)
			}
			if ua.Browser.Full != tt.wantFull {
				t.Errorf("Browser.Full = %q, want %q", ua.Browser.Full, tt.wantFull)
			}
		})
	}
}

func TestOSVersionParsing(t *testing.T) {
	tests := []struct {
		name        string
		raw         string
		wantOS      string
		wantVersion string
	}{
		{
			name:        "Windows 10",
			raw:         "Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
			wantOS:      "Windows",
			wantVersion: "10",
		},
		{
			name:        "Windows 7",
			raw:         "Mozilla/5.0 (Windows NT 6.1; Win64; x64)",
			wantOS:      "Windows",
			wantVersion: "7",
		},
		{
			name:        "macOS",
			raw:         "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
			wantOS:      "macOS",
			wantVersion: "10.15",
		},
		{
			name:        "iOS",
			raw:         "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X)",
			wantOS:      "iOS",
			wantVersion: "17.0",
		},
		{
			name:        "Android",
			raw:         "Mozilla/5.0 (Linux; Android 13; SM-G991B)",
			wantOS:      "Android",
			wantVersion: "13",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ua := Parse(tt.raw)
			if ua.OS.Name != tt.wantOS {
				t.Errorf("OS.Name = %q, want %q", ua.OS.Name, tt.wantOS)
			}
			if ua.OS.Version != tt.wantVersion {
				t.Errorf("OS.Version = %q, want %q", ua.OS.Version, tt.wantVersion)
			}
		})
	}
}

func TestDeviceBrandParsing(t *testing.T) {
	tests := []struct {
		name      string
		raw       string
		wantBrand string
		wantModel string
	}{
		{
			name:      "iPhone",
			raw:       "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X)",
			wantBrand: "Apple",
			wantModel: "iPhone",
		},
		{
			name:      "iPad",
			raw:       "Mozilla/5.0 (iPad; CPU OS 17_0 like Mac OS X)",
			wantBrand: "Apple",
			wantModel: "iPad",
		},
		{
			name:      "Mac",
			raw:       "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
			wantBrand: "Apple",
			wantModel: "Mac",
		},
		{
			name:      "Samsung Galaxy",
			raw:       "Mozilla/5.0 (Linux; Android 13; SM-G991B)",
			wantBrand: "Samsung",
			wantModel: "Galaxy G991B",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ua := Parse(tt.raw)
			if ua.Device.Brand != tt.wantBrand {
				t.Errorf("Device.Brand = %q, want %q", ua.Device.Brand, tt.wantBrand)
			}
			if ua.Device.Model != tt.wantModel {
				t.Errorf("Device.Model = %q, want %q", ua.Device.Model, tt.wantModel)
			}
		})
	}
}

func TestEngineParsing(t *testing.T) {
	tests := []struct {
		name       string
		raw        string
		wantEngine string
	}{
		{
			name:       "Blink (Chrome)",
			raw:        "Mozilla/5.0 (Windows NT 10.0) AppleWebKit/537.36 Chrome/120.0.0.0 Safari/537.36",
			wantEngine: "Blink",
		},
		{
			name:       "WebKit (Safari)",
			raw:        "Mozilla/5.0 (Macintosh) AppleWebKit/605.1.15 Version/17.0 Safari/605.1.15",
			wantEngine: "WebKit",
		},
		{
			name:       "Gecko (Firefox)",
			raw:        "Mozilla/5.0 (X11; Linux x86_64; rv:120.0) Gecko/20100101 Firefox/120.0",
			wantEngine: "Gecko",
		},
		{
			name:       "Trident (IE)",
			raw:        "Mozilla/5.0 (Windows NT 10.0; Trident/7.0; rv:11.0) like Gecko",
			wantEngine: "Trident",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ua := Parse(tt.raw)
			if ua.Engine.Name != tt.wantEngine {
				t.Errorf("Engine.Name = %q, want %q", ua.Engine.Name, tt.wantEngine)
			}
		})
	}
}
