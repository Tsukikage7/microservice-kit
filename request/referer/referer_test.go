package referer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name         string
		raw          string
		wantType     SourceType
		wantSource   string
		wantDomain   string
	}{
		{
			name:       "empty",
			raw:        "",
			wantType:   SourceTypeDirect,
			wantSource: "",
			wantDomain: "",
		},
		{
			name:         "google search",
			raw:          "https://www.google.com/search?q=golang",
			wantType:     SourceTypeSearch,
			wantSource:   "google",
			wantDomain:   "www.google.com",
		},
		{
			name:         "baidu search",
			raw:          "https://www.baidu.com/s?wd=golang",
			wantType:     SourceTypeSearch,
			wantSource:   "baidu",
			wantDomain:   "www.baidu.com",
		},
		{
			name:         "bing search",
			raw:          "https://www.bing.com/search?q=golang",
			wantType:     SourceTypeSearch,
			wantSource:   "bing",
			wantDomain:   "www.bing.com",
		},
		{
			name:         "facebook",
			raw:          "https://www.facebook.com/",
			wantType:     SourceTypeSocial,
			wantSource:   "facebook",
			wantDomain:   "www.facebook.com",
		},
		{
			name:         "twitter",
			raw:          "https://t.co/abc123",
			wantType:     SourceTypeSocial,
			wantSource:   "twitter",
			wantDomain:   "t.co",
		},
		{
			name:         "weibo",
			raw:          "https://weibo.com/path",
			wantType:     SourceTypeSocial,
			wantSource:   "weibo",
			wantDomain:   "weibo.com",
		},
		{
			name:         "external site",
			raw:          "https://example.com/page",
			wantType:     SourceTypeReferral,
			wantSource:   "example",
			wantDomain:   "example.com",
		},
		{
			name:         "opaque url treated as referral",
			raw:          "not-a-valid-url:::",
			wantType:     SourceTypeReferral,
			wantSource:   "",
			wantDomain:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref := Parse(tt.raw)
			if ref.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", ref.Type, tt.wantType)
			}
			if ref.Source != tt.wantSource {
				t.Errorf("Source = %q, want %q", ref.Source, tt.wantSource)
			}
			if ref.Domain != tt.wantDomain {
				t.Errorf("Domain = %q, want %q", ref.Domain, tt.wantDomain)
			}
		})
	}
}

func TestParseWithHost(t *testing.T) {
	tests := []struct {
		name        string
		raw         string
		currentHost string
		wantType    SourceType
	}{
		{
			name:        "same domain",
			raw:         "https://www.example.com/page1",
			currentHost: "www.example.com",
			wantType:    SourceTypeInternal,
		},
		{
			name:        "same domain different www",
			raw:         "https://example.com/page1",
			currentHost: "www.example.com",
			wantType:    SourceTypeInternal,
		},
		{
			name:        "different domain",
			raw:         "https://other.com/page",
			currentHost: "example.com",
			wantType:    SourceTypeReferral,
		},
		{
			name:        "search engine",
			raw:         "https://www.google.com/search?q=test",
			currentHost: "example.com",
			wantType:    SourceTypeSearch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref := ParseWithHost(tt.raw, tt.currentHost)
			if ref.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", ref.Type, tt.wantType)
			}
		})
	}
}

func TestUTMParams(t *testing.T) {
	tests := []struct {
		name       string
		raw        string
		wantType   SourceType
		wantSource string
		wantUTM    UTMParams
	}{
		{
			name:       "full UTM params",
			raw:        "https://example.com/?utm_source=google&utm_medium=cpc&utm_campaign=spring_sale&utm_term=shoes&utm_content=banner1",
			wantType:   SourceTypePaid,
			wantSource: "google",
			wantUTM: UTMParams{
				Source:   "google",
				Medium:   "cpc",
				Campaign: "spring_sale",
				Term:     "shoes",
				Content:  "banner1",
			},
		},
		{
			name:       "email UTM",
			raw:        "https://example.com/?utm_source=newsletter&utm_medium=email&utm_campaign=weekly",
			wantType:   SourceTypeEmail,
			wantSource: "newsletter",
			wantUTM: UTMParams{
				Source:   "newsletter",
				Medium:   "email",
				Campaign: "weekly",
			},
		},
		{
			name:       "social UTM",
			raw:        "https://example.com/?utm_source=facebook&utm_medium=social&utm_campaign=launch",
			wantType:   SourceTypeSocial,
			wantSource: "facebook",
			wantUTM: UTMParams{
				Source:   "facebook",
				Medium:   "social",
				Campaign: "launch",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref := Parse(tt.raw)
			if ref.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", ref.Type, tt.wantType)
			}
			if ref.Source != tt.wantSource {
				t.Errorf("Source = %q, want %q", ref.Source, tt.wantSource)
			}
			if ref.UTM.Source != tt.wantUTM.Source {
				t.Errorf("UTM.Source = %q, want %q", ref.UTM.Source, tt.wantUTM.Source)
			}
			if ref.UTM.Medium != tt.wantUTM.Medium {
				t.Errorf("UTM.Medium = %q, want %q", ref.UTM.Medium, tt.wantUTM.Medium)
			}
			if ref.UTM.Campaign != tt.wantUTM.Campaign {
				t.Errorf("UTM.Campaign = %q, want %q", ref.UTM.Campaign, tt.wantUTM.Campaign)
			}
		})
	}
}

func TestSearchQuery(t *testing.T) {
	tests := []struct {
		name      string
		raw       string
		wantQuery string
	}{
		{
			name:      "google",
			raw:       "https://www.google.com/search?q=golang+tutorial",
			wantQuery: "golang tutorial",
		},
		{
			name:      "baidu",
			raw:       "https://www.baidu.com/s?wd=golang教程",
			wantQuery: "golang教程",
		},
		{
			name:      "bing",
			raw:       "https://www.bing.com/search?q=go+programming",
			wantQuery: "go programming",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref := Parse(tt.raw)
			if ref.SearchQuery != tt.wantQuery {
				t.Errorf("SearchQuery = %q, want %q", ref.SearchQuery, tt.wantQuery)
			}
		})
	}
}

func TestRefererMethods(t *testing.T) {
	// Test IsEmpty
	emptyRef := Parse("")
	if !emptyRef.IsEmpty() {
		t.Error("IsEmpty() should be true for empty referer")
	}

	// Test IsDirect
	if !emptyRef.IsDirect() {
		t.Error("IsDirect() should be true for empty referer")
	}

	// Test IsSearch
	searchRef := Parse("https://www.google.com/search?q=test")
	if !searchRef.IsSearch() {
		t.Error("IsSearch() should be true for google search")
	}

	// Test IsSocial
	socialRef := Parse("https://www.facebook.com/")
	if !socialRef.IsSocial() {
		t.Error("IsSocial() should be true for facebook")
	}
}

func TestUTMParamsHasUTM(t *testing.T) {
	tests := []struct {
		name    string
		utm     UTMParams
		wantHas bool
	}{
		{
			name:    "empty",
			utm:     UTMParams{},
			wantHas: false,
		},
		{
			name:    "with source",
			utm:     UTMParams{Source: "google"},
			wantHas: true,
		},
		{
			name:    "with medium",
			utm:     UTMParams{Medium: "cpc"},
			wantHas: true,
		},
		{
			name:    "with campaign",
			utm:     UTMParams{Campaign: "sale"},
			wantHas: true,
		},
		{
			name:    "only term",
			utm:     UTMParams{Term: "keyword"},
			wantHas: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.utm.HasUTM(); got != tt.wantHas {
				t.Errorf("HasUTM() = %v, want %v", got, tt.wantHas)
			}
		})
	}
}

func TestContextOperations(t *testing.T) {
	ctx := context.Background()
	ref := Parse("https://www.google.com/search?q=test")

	// Test WithReferer and FromContext
	ctx = WithReferer(ctx, ref)
	got, ok := FromContext(ctx)
	if !ok {
		t.Error("FromContext() ok = false, want true")
	}
	if got.Source != ref.Source {
		t.Errorf("FromContext().Source = %q, want %q", got.Source, ref.Source)
	}

	// Test GetSource
	if GetSource(ctx) != "google" {
		t.Errorf("GetSource() = %q, want %q", GetSource(ctx), "google")
	}

	// Test GetSourceType
	if GetSourceType(ctx) != SourceTypeSearch {
		t.Errorf("GetSourceType() = %q, want %q", GetSourceType(ctx), SourceTypeSearch)
	}

	// Test GetDomain
	if GetDomain(ctx) != "www.google.com" {
		t.Errorf("GetDomain() = %q, want %q", GetDomain(ctx), "www.google.com")
	}

	// Test empty context
	emptyCtx := context.Background()
	if GetSource(emptyCtx) != "" {
		t.Error("GetSource() on empty context should return empty string")
	}
	if GetSourceType(emptyCtx) != SourceTypeUnknown {
		t.Errorf("GetSourceType() on empty context = %q, want %q", GetSourceType(emptyCtx), SourceTypeUnknown)
	}
}

func TestHTTPMiddleware(t *testing.T) {
	tests := []struct {
		name       string
		referer    string
		host       string
		wantType   SourceType
		wantSource string
	}{
		{
			name:       "google search",
			referer:    "https://www.google.com/search?q=test",
			host:       "example.com",
			wantType:   SourceTypeSearch,
			wantSource: "google",
		},
		{
			name:       "internal",
			referer:    "https://example.com/page1",
			host:       "example.com",
			wantType:   SourceTypeInternal,
			wantSource: "example",
		},
		{
			name:       "empty",
			referer:    "",
			host:       "example.com",
			wantType:   SourceTypeDirect,
			wantSource: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotType SourceType
			var gotSource string

			handler := HTTPMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotType = GetSourceType(r.Context())
				gotSource = GetSource(r.Context())
			}))

			req := httptest.NewRequest("GET", "/", nil)
			req.Host = tt.host
			if tt.referer != "" {
				req.Header.Set("Referer", tt.referer)
			}
			handler.ServeHTTP(httptest.NewRecorder(), req)

			if gotType != tt.wantType {
				t.Errorf("GetSourceType() = %q, want %q", gotType, tt.wantType)
			}
			if gotSource != tt.wantSource {
				t.Errorf("GetSource() = %q, want %q", gotSource, tt.wantSource)
			}
		})
	}
}

func TestSocialNetworks(t *testing.T) {
	tests := []struct {
		name       string
		raw        string
		wantSource string
	}{
		{"facebook", "https://www.facebook.com/share", "facebook"},
		{"twitter t.co", "https://t.co/xyz", "twitter"},
		{"instagram", "https://www.instagram.com/p/123", "instagram"},
		{"linkedin", "https://www.linkedin.com/feed", "linkedin"},
		{"weibo", "https://weibo.com/u/123", "weibo"},
		{"youtube", "https://www.youtube.com/watch?v=123", "youtube"},
		{"tiktok", "https://www.tiktok.com/@user", "tiktok"},
		{"reddit", "https://www.reddit.com/r/golang", "reddit"},
		{"zhihu", "https://www.zhihu.com/question/123", "zhihu"},
		{"bilibili", "https://www.bilibili.com/video/BV123", "bilibili"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref := Parse(tt.raw)
			if ref.Type != SourceTypeSocial {
				t.Errorf("Type = %q, want %q", ref.Type, SourceTypeSocial)
			}
			if ref.Source != tt.wantSource {
				t.Errorf("Source = %q, want %q", ref.Source, tt.wantSource)
			}
		})
	}
}
