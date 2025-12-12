package locale

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
		wantLang     string
		wantRegion   string
		wantString   string
		wantTagCount int
	}{
		{
			name:         "simple language",
			raw:          "en",
			wantLang:     "en",
			wantRegion:   "",
			wantString:   "en",
			wantTagCount: 1,
		},
		{
			name:         "language with region",
			raw:          "zh-CN",
			wantLang:     "zh",
			wantRegion:   "CN",
			wantString:   "zh-CN",
			wantTagCount: 1,
		},
		{
			name:         "multiple languages with quality",
			raw:          "zh-CN,zh;q=0.9,en;q=0.8",
			wantLang:     "zh",
			wantRegion:   "CN",
			wantString:   "zh-CN",
			wantTagCount: 3,
		},
		{
			name:         "quality order",
			raw:          "en;q=0.5,zh-CN;q=0.9,ja;q=0.7",
			wantLang:     "zh",
			wantRegion:   "CN",
			wantString:   "zh-CN",
			wantTagCount: 3,
		},
		{
			name:         "with script",
			raw:          "zh-Hans-CN",
			wantLang:     "zh",
			wantRegion:   "CN",
			wantString:   "zh-Hans-CN",
			wantTagCount: 1,
		},
		{
			name:         "empty",
			raw:          "",
			wantLang:     "",
			wantRegion:   "",
			wantString:   "",
			wantTagCount: 0,
		},
		{
			name:         "wildcard",
			raw:          "*",
			wantLang:     "*",
			wantRegion:   "",
			wantString:   "*",
			wantTagCount: 1,
		},
		{
			name:         "complex",
			raw:          "en-US,en;q=0.9,zh-CN;q=0.8,zh;q=0.7,ja;q=0.6",
			wantLang:     "en",
			wantRegion:   "US",
			wantString:   "en-US",
			wantTagCount: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loc := Parse(tt.raw)
			if loc.Language() != tt.wantLang {
				t.Errorf("Language() = %q, want %q", loc.Language(), tt.wantLang)
			}
			if loc.Region() != tt.wantRegion {
				t.Errorf("Region() = %q, want %q", loc.Region(), tt.wantRegion)
			}
			if loc.String() != tt.wantString {
				t.Errorf("String() = %q, want %q", loc.String(), tt.wantString)
			}
			if len(loc.Preferred) != tt.wantTagCount {
				t.Errorf("len(Preferred) = %d, want %d", len(loc.Preferred), tt.wantTagCount)
			}
		})
	}
}

func TestTagString(t *testing.T) {
	tests := []struct {
		name string
		tag  Tag
		want string
	}{
		{
			name: "language only",
			tag:  Tag{Language: "en"},
			want: "en",
		},
		{
			name: "language and region",
			tag:  Tag{Language: "zh", Region: "CN"},
			want: "zh-CN",
		},
		{
			name: "with script",
			tag:  Tag{Language: "zh", Script: "Hans", Region: "CN"},
			want: "zh-Hans-CN",
		},
		{
			name: "raw preserved",
			tag:  Tag{Language: "zh", Region: "CN", Raw: "zh-Hans-CN"},
			want: "zh-Hans-CN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.tag.String(); got != tt.want {
				t.Errorf("Tag.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLocaleMatch(t *testing.T) {
	tests := []struct {
		name       string
		raw        string
		languages  []string
		wantMatch  bool
	}{
		{
			name:       "match first choice",
			raw:        "zh-CN,en;q=0.9",
			languages:  []string{"zh"},
			wantMatch:  true,
		},
		{
			name:       "match second choice",
			raw:        "zh-CN,en;q=0.9",
			languages:  []string{"en"},
			wantMatch:  true,
		},
		{
			name:       "no match",
			raw:        "zh-CN,en;q=0.9",
			languages:  []string{"ja"},
			wantMatch:  false,
		},
		{
			name:       "empty locale",
			raw:        "",
			languages:  []string{"en"},
			wantMatch:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loc := Parse(tt.raw)
			if got := loc.Match(tt.languages...); got != tt.wantMatch {
				t.Errorf("Match() = %v, want %v", got, tt.wantMatch)
			}
		})
	}
}

func TestLocaleBest(t *testing.T) {
	tests := []struct {
		name       string
		raw        string
		candidates []string
		wantBest   string
	}{
		{
			name:       "match first choice",
			raw:        "zh-CN,en;q=0.9",
			candidates: []string{"en-US", "zh-CN", "ja-JP"},
			wantBest:   "zh-CN",
		},
		{
			name:       "match second choice",
			raw:        "fr,en;q=0.9",
			candidates: []string{"en-US", "zh-CN", "ja-JP"},
			wantBest:   "en-US",
		},
		{
			name:       "no match returns first",
			raw:        "ko-KR",
			candidates: []string{"en-US", "zh-CN", "ja-JP"},
			wantBest:   "en-US",
		},
		{
			name:       "empty locale returns first",
			raw:        "",
			candidates: []string{"en-US", "zh-CN"},
			wantBest:   "en-US",
		},
		{
			name:       "empty candidates",
			raw:        "zh-CN",
			candidates: []string{},
			wantBest:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loc := Parse(tt.raw)
			if got := loc.Best(tt.candidates...); got != tt.wantBest {
				t.Errorf("Best() = %q, want %q", got, tt.wantBest)
			}
		})
	}
}

func TestContextOperations(t *testing.T) {
	ctx := context.Background()
	loc := Parse("zh-CN,en;q=0.9")

	// Test WithLocale and FromContext
	ctx = WithLocale(ctx, loc)
	got, ok := FromContext(ctx)
	if !ok {
		t.Error("FromContext() ok = false, want true")
	}
	if got.Language() != loc.Language() {
		t.Errorf("FromContext().Language() = %q, want %q", got.Language(), loc.Language())
	}

	// Test GetLanguage
	if GetLanguage(ctx) != "zh" {
		t.Errorf("GetLanguage() = %q, want %q", GetLanguage(ctx), "zh")
	}

	// Test GetRegion
	if GetRegion(ctx) != "CN" {
		t.Errorf("GetRegion() = %q, want %q", GetRegion(ctx), "CN")
	}

	// Test GetLocale
	if GetLocale(ctx) != "zh-CN" {
		t.Errorf("GetLocale() = %q, want %q", GetLocale(ctx), "zh-CN")
	}

	// Test empty context
	emptyCtx := context.Background()
	if GetLanguage(emptyCtx) != "" {
		t.Error("GetLanguage() on empty context should return empty string")
	}
	if GetRegion(emptyCtx) != "" {
		t.Error("GetRegion() on empty context should return empty string")
	}
	if GetLocale(emptyCtx) != "" {
		t.Error("GetLocale() on empty context should return empty string")
	}
}

func TestHTTPMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		acceptLanguage string
		wantLang       string
		wantRegion     string
	}{
		{
			name:           "Chinese",
			acceptLanguage: "zh-CN,zh;q=0.9,en;q=0.8",
			wantLang:       "zh",
			wantRegion:     "CN",
		},
		{
			name:           "English",
			acceptLanguage: "en-US,en;q=0.9",
			wantLang:       "en",
			wantRegion:     "US",
		},
		{
			name:           "empty",
			acceptLanguage: "",
			wantLang:       "",
			wantRegion:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotLang, gotRegion string

			handler := HTTPMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotLang = GetLanguage(r.Context())
				gotRegion = GetRegion(r.Context())
			}))

			req := httptest.NewRequest("GET", "/", nil)
			if tt.acceptLanguage != "" {
				req.Header.Set("Accept-Language", tt.acceptLanguage)
			}
			handler.ServeHTTP(httptest.NewRecorder(), req)

			if gotLang != tt.wantLang {
				t.Errorf("GetLanguage() = %q, want %q", gotLang, tt.wantLang)
			}
			if gotRegion != tt.wantRegion {
				t.Errorf("GetRegion() = %q, want %q", gotRegion, tt.wantRegion)
			}
		})
	}
}

func TestParseTag(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantLang   string
		wantScript string
		wantRegion string
		wantQ      float64
	}{
		{
			name:     "simple",
			input:    "en",
			wantLang: "en",
			wantQ:    1.0,
		},
		{
			name:       "with region",
			input:      "zh-CN",
			wantLang:   "zh",
			wantRegion: "CN",
			wantQ:      1.0,
		},
		{
			name:       "with script",
			input:      "zh-Hans",
			wantLang:   "zh",
			wantScript: "Hans",
			wantQ:      1.0,
		},
		{
			name:       "full",
			input:      "zh-Hans-CN",
			wantLang:   "zh",
			wantScript: "Hans",
			wantRegion: "CN",
			wantQ:      1.0,
		},
		{
			name:     "with quality",
			input:    "en;q=0.8",
			wantLang: "en",
			wantQ:    0.8,
		},
		{
			name:       "full with quality",
			input:      "zh-Hans-CN;q=0.9",
			wantLang:   "zh",
			wantScript: "Hans",
			wantRegion: "CN",
			wantQ:      0.9,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tag := parseTag(tt.input)
			if tag.Language != tt.wantLang {
				t.Errorf("Language = %q, want %q", tag.Language, tt.wantLang)
			}
			if tag.Script != tt.wantScript {
				t.Errorf("Script = %q, want %q", tag.Script, tt.wantScript)
			}
			if tag.Region != tt.wantRegion {
				t.Errorf("Region = %q, want %q", tag.Region, tt.wantRegion)
			}
			if tag.Quality != tt.wantQ {
				t.Errorf("Quality = %f, want %f", tag.Quality, tt.wantQ)
			}
		})
	}
}
