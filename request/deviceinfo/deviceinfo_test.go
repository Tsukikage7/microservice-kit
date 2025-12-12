package deviceinfo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseClientHints(t *testing.T) {
	parser := New()

	tests := []struct {
		name           string
		headers        Headers
		wantIsMobile   bool
		wantPlatform   string
		wantBrowser    string
		wantSource     DataSource
	}{
		{
			name: "Chrome on Windows Desktop",
			headers: Headers{
				SecCHUA:         `"Chromium";v="120", "Google Chrome";v="120", "Not-A.Brand";v="24"`,
				SecCHUAMobile:   "?0",
				SecCHUAPlatform: `"Windows"`,
			},
			wantIsMobile: false,
			wantPlatform: "Windows",
			wantBrowser:  "Google Chrome",
			wantSource:   SourceClientHints,
		},
		{
			name: "Chrome on Android Mobile",
			headers: Headers{
				SecCHUA:         `"Chromium";v="120", "Google Chrome";v="120", "Not-A.Brand";v="24"`,
				SecCHUAMobile:   "?1",
				SecCHUAPlatform: `"Android"`,
			},
			wantIsMobile: true,
			wantPlatform: "Android",
			wantBrowser:  "Google Chrome",
			wantSource:   SourceClientHints,
		},
		{
			name: "Edge on Windows",
			headers: Headers{
				SecCHUA:         `"Chromium";v="120", "Microsoft Edge";v="120", "Not-A.Brand";v="24"`,
				SecCHUAMobile:   "?0",
				SecCHUAPlatform: `"Windows"`,
			},
			wantIsMobile: false,
			wantPlatform: "Windows",
			wantBrowser:  "Microsoft Edge",
			wantSource:   SourceClientHints,
		},
		{
			name: "Safari on macOS (no Client Hints, UA fallback)",
			headers: Headers{
				UserAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15",
			},
			wantIsMobile: false,
			wantPlatform: "macOS",
			wantBrowser:  "Safari",
			wantSource:   SourceUserAgent,
		},
		{
			name: "Chrome on iPhone (UA fallback)",
			headers: Headers{
				UserAgent: "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/120.0.0.0 Mobile/15E148 Safari/604.1",
			},
			wantIsMobile: true,
			wantPlatform: "iOS",
			wantBrowser:  "Google Chrome",
			wantSource:   SourceUserAgent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := parser.Parse(tt.headers)
			if info.IsMobile != tt.wantIsMobile {
				t.Errorf("IsMobile = %v, want %v", info.IsMobile, tt.wantIsMobile)
			}
			if info.Platform != tt.wantPlatform {
				t.Errorf("Platform = %q, want %q", info.Platform, tt.wantPlatform)
			}
			if info.Browser != tt.wantBrowser {
				t.Errorf("Browser = %q, want %q", info.Browser, tt.wantBrowser)
			}
			if info.Source != tt.wantSource {
				t.Errorf("Source = %q, want %q", info.Source, tt.wantSource)
			}
		})
	}
}

func TestParseDeviceHints(t *testing.T) {
	parser := New()

	headers := Headers{
		SecCHUA:       `"Google Chrome";v="120"`,
		SecCHUAMobile: "?0",
		DeviceMemory:  "8",
		ViewportWidth: "1920",
		DPR:           "2",
	}

	info := parser.Parse(headers)

	if info.DeviceMemory != 8 {
		t.Errorf("DeviceMemory = %f, want 8", info.DeviceMemory)
	}
	if info.ViewportWidth != 1920 {
		t.Errorf("ViewportWidth = %d, want 1920", info.ViewportWidth)
	}
	if info.DPR != 2 {
		t.Errorf("DPR = %f, want 2", info.DPR)
	}
}

func TestInfoMethods(t *testing.T) {
	// Desktop
	desktop := &Info{IsMobile: false}
	if !desktop.IsDesktop() {
		t.Error("IsDesktop() should be true")
	}

	// Mobile
	mobile := &Info{IsMobile: true}
	if mobile.IsDesktop() {
		t.Error("IsDesktop() should be false for mobile")
	}

	// High DPI
	highDPI := &Info{DPR: 2.0}
	if !highDPI.IsHighDPI() {
		t.Error("IsHighDPI() should be true for DPR > 1")
	}

	lowDPI := &Info{DPR: 1.0}
	if lowDPI.IsHighDPI() {
		t.Error("IsHighDPI() should be false for DPR = 1")
	}

	// Low memory
	lowMem := &Info{DeviceMemory: 2}
	if !lowMem.IsLowMemory() {
		t.Error("IsLowMemory() should be true for < 4GB")
	}

	highMem := &Info{DeviceMemory: 8}
	if highMem.IsLowMemory() {
		t.Error("IsLowMemory() should be false for >= 4GB")
	}
}

func TestContextOperations(t *testing.T) {
	ctx := context.Background()
	info := &Info{
		IsMobile: true,
		Platform: "Android",
		Browser:  "Google Chrome",
	}

	// Test WithInfo and FromContext
	ctx = WithInfo(ctx, info)
	got, ok := FromContext(ctx)
	if !ok {
		t.Error("FromContext() ok = false, want true")
	}
	if got.Platform != info.Platform {
		t.Errorf("FromContext().Platform = %q, want %q", got.Platform, info.Platform)
	}

	// Test IsMobile
	if !IsMobile(ctx) {
		t.Error("IsMobile() = false, want true")
	}

	// Test GetPlatform
	if GetPlatform(ctx) != "Android" {
		t.Errorf("GetPlatform() = %q, want %q", GetPlatform(ctx), "Android")
	}

	// Test GetBrowser
	if GetBrowser(ctx) != "Google Chrome" {
		t.Errorf("GetBrowser() = %q, want %q", GetBrowser(ctx), "Google Chrome")
	}

	// Test empty context
	emptyCtx := context.Background()
	if IsMobile(emptyCtx) {
		t.Error("IsMobile() on empty context should be false")
	}
	if GetPlatform(emptyCtx) != "" {
		t.Error("GetPlatform() on empty context should return empty string")
	}
}

func TestHTTPMiddleware(t *testing.T) {
	tests := []struct {
		name         string
		headers      map[string]string
		wantIsMobile bool
		wantPlatform string
	}{
		{
			name: "Client Hints Mobile",
			headers: map[string]string{
				"Sec-CH-UA-Mobile":   "?1",
				"Sec-CH-UA-Platform": `"Android"`,
				"Sec-CH-UA":          `"Google Chrome";v="120"`,
			},
			wantIsMobile: true,
			wantPlatform: "Android",
		},
		{
			name: "Client Hints Desktop",
			headers: map[string]string{
				"Sec-CH-UA-Mobile":   "?0",
				"Sec-CH-UA-Platform": `"Windows"`,
				"Sec-CH-UA":          `"Google Chrome";v="120"`,
			},
			wantIsMobile: false,
			wantPlatform: "Windows",
		},
		{
			name: "UA Fallback Mobile",
			headers: map[string]string{
				"User-Agent": "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X)",
			},
			wantIsMobile: true,
			wantPlatform: "iOS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotIsMobile bool
			var gotPlatform string

			handler := HTTPMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotIsMobile = IsMobile(r.Context())
				gotPlatform = GetPlatform(r.Context())
			}))

			req := httptest.NewRequest("GET", "/", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			handler.ServeHTTP(httptest.NewRecorder(), req)

			if gotIsMobile != tt.wantIsMobile {
				t.Errorf("IsMobile() = %v, want %v", gotIsMobile, tt.wantIsMobile)
			}
			if gotPlatform != tt.wantPlatform {
				t.Errorf("GetPlatform() = %q, want %q", gotPlatform, tt.wantPlatform)
			}
		})
	}
}

func TestWithAcceptCH(t *testing.T) {
	handler := HTTPMiddleware(WithAcceptCH(true))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	acceptCH := rr.Header().Get("Accept-CH")
	if acceptCH == "" {
		t.Error("Accept-CH header should be set")
	}
	if !contains(acceptCH, "Sec-CH-UA-Mobile") {
		t.Error("Accept-CH should contain Sec-CH-UA-Mobile")
	}
}

func TestWithUAFallbackDisabled(t *testing.T) {
	parser := New(WithUAFallback(false))

	headers := Headers{
		UserAgent: "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X)",
	}

	info := parser.Parse(headers)

	// Without UA fallback, info should be empty
	if info.Source != SourceUnknown {
		t.Errorf("Source = %q, want %q", info.Source, SourceUnknown)
	}
	if info.Platform != "" {
		t.Errorf("Platform should be empty without UA fallback, got %q", info.Platform)
	}
}

func TestParseBrandList(t *testing.T) {
	tests := []struct {
		name        string
		list        string
		wantBrowser string
		wantVersion string
	}{
		{
			name:        "Chrome with Not brand",
			list:        `"Chromium";v="120", "Google Chrome";v="120", "Not-A.Brand";v="24"`,
			wantBrowser: "Google Chrome",
			wantVersion: "120",
		},
		{
			name:        "Edge",
			list:        `"Chromium";v="120", "Microsoft Edge";v="120", "Not-A.Brand";v="24"`,
			wantBrowser: "Microsoft Edge",
			wantVersion: "120",
		},
		{
			name:        "Only Chromium",
			list:        `"Chromium";v="120", "Not-A.Brand";v="24"`,
			wantBrowser: "Chromium",
			wantVersion: "120",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			browser, version := parseBrandList(tt.list)
			if browser != tt.wantBrowser {
				t.Errorf("browser = %q, want %q", browser, tt.wantBrowser)
			}
			if version != tt.wantVersion {
				t.Errorf("version = %q, want %q", version, tt.wantVersion)
			}
		})
	}
}

func TestTrimQuotes(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`"Windows"`, "Windows"},
		{`"Android"`, "Android"},
		{`Windows`, "Windows"},
		{`""`, ""},
		{``, ""},
	}

	for _, tt := range tests {
		if got := trimQuotes(tt.input); got != tt.want {
			t.Errorf("trimQuotes(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestAcceptCHHeader(t *testing.T) {
	header := AcceptCHHeader()
	expectedParts := []string{
		"Sec-CH-UA",
		"Sec-CH-UA-Mobile",
		"Sec-CH-UA-Platform",
		"Device-Memory",
		"DPR",
	}

	for _, part := range expectedParts {
		if !contains(header, part) {
			t.Errorf("AcceptCHHeader() should contain %q", part)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
