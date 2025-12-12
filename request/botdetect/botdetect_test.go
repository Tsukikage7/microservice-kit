package botdetect

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDetector(t *testing.T) {
	detector := New()

	tests := []struct {
		name         string
		userAgent    string
		wantIsBot    bool
		wantCategory Category
		wantName     string
	}{
		{
			name:         "Chrome browser",
			userAgent:    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			wantIsBot:    false,
			wantCategory: CategoryHuman,
		},
		{
			name:         "Googlebot",
			userAgent:    "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)",
			wantIsBot:    true,
			wantCategory: CategorySearch,
			wantName:     "Googlebot",
		},
		{
			name:         "Bingbot",
			userAgent:    "Mozilla/5.0 (compatible; bingbot/2.0; +http://www.bing.com/bingbot.htm)",
			wantIsBot:    true,
			wantCategory: CategorySearch,
			wantName:     "Bingbot",
		},
		{
			name:         "Baiduspider",
			userAgent:    "Mozilla/5.0 (compatible; Baiduspider/2.0; +http://www.baidu.com/search/spider.html)",
			wantIsBot:    true,
			wantCategory: CategorySearch,
			wantName:     "Baiduspider",
		},
		{
			name:         "Facebook",
			userAgent:    "facebookexternalhit/1.1 (+http://www.facebook.com/externalhit_uatext.php)",
			wantIsBot:    true,
			wantCategory: CategorySocial,
			wantName:     "Facebot",
		},
		{
			name:         "Twitter",
			userAgent:    "Twitterbot/1.0",
			wantIsBot:    true,
			wantCategory: CategorySocial,
			wantName:     "Twitterbot",
		},
		{
			name:         "curl",
			userAgent:    "curl/7.68.0",
			wantIsBot:    true,
			wantCategory: CategoryTool,
			wantName:     "curl",
		},
		{
			name:         "wget",
			userAgent:    "Wget/1.21",
			wantIsBot:    true,
			wantCategory: CategoryTool,
			wantName:     "wget",
		},
		{
			name:         "Python requests",
			userAgent:    "python-requests/2.28.0",
			wantIsBot:    true,
			wantCategory: CategoryScraper,
		},
		{
			name:         "Go HTTP client",
			userAgent:    "Go-http-client/1.1",
			wantIsBot:    true,
			wantCategory: CategoryScraper,
		},
		{
			name:         "Empty",
			userAgent:    "",
			wantIsBot:    true,
			wantCategory: CategoryUnknown,
		},
		{
			name:         "UptimeRobot",
			userAgent:    "UptimeRobot/2.0",
			wantIsBot:    true,
			wantCategory: CategoryMonitor,
			wantName:     "UptimeRobot",
		},
		{
			name:         "Slackbot",
			userAgent:    "Slackbot-LinkExpanding 1.0 (+https://api.slack.com/robots)",
			wantIsBot:    true,
			wantCategory: CategorySocial,
			wantName:     "Slackbot",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.Detect(tt.userAgent)
			if result.IsBot != tt.wantIsBot {
				t.Errorf("IsBot = %v, want %v", result.IsBot, tt.wantIsBot)
			}
			if result.Category != tt.wantCategory {
				t.Errorf("Category = %q, want %q", result.Category, tt.wantCategory)
			}
			if tt.wantName != "" && result.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", result.Name, tt.wantName)
			}
		})
	}
}

func TestResultMethods(t *testing.T) {
	// Good bot
	goodBot := &Result{IsBot: true, Intent: IntentGood}
	if !goodBot.IsGoodBot() {
		t.Error("IsGoodBot() should be true")
	}
	if goodBot.IsBadBot() {
		t.Error("IsBadBot() should be false")
	}

	// Bad bot
	badBot := &Result{IsBot: true, Intent: IntentBad}
	if badBot.IsGoodBot() {
		t.Error("IsGoodBot() should be false")
	}
	if !badBot.IsBadBot() {
		t.Error("IsBadBot() should be true")
	}

	// Human
	human := &Result{IsBot: false, Intent: IntentNeutral}
	if human.IsGoodBot() {
		t.Error("IsGoodBot() should be false for human")
	}
	if human.IsBadBot() {
		t.Error("IsBadBot() should be false for human")
	}
}

func TestContextOperations(t *testing.T) {
	ctx := context.Background()
	result := &Result{
		IsBot:    true,
		Name:     "Googlebot",
		Category: CategorySearch,
	}

	// Test WithResult and FromContext
	ctx = WithResult(ctx, result)
	got, ok := FromContext(ctx)
	if !ok {
		t.Error("FromContext() ok = false, want true")
	}
	if got.Name != result.Name {
		t.Errorf("FromContext().Name = %q, want %q", got.Name, result.Name)
	}

	// Test IsBot
	if !IsBot(ctx) {
		t.Error("IsBot() = false, want true")
	}

	// Test GetBotName
	if GetBotName(ctx) != "Googlebot" {
		t.Errorf("GetBotName() = %q, want %q", GetBotName(ctx), "Googlebot")
	}

	// Test GetCategory
	if GetCategory(ctx) != CategorySearch {
		t.Errorf("GetCategory() = %q, want %q", GetCategory(ctx), CategorySearch)
	}

	// Test empty context
	emptyCtx := context.Background()
	if IsBot(emptyCtx) {
		t.Error("IsBot() on empty context should be false")
	}
	if GetBotName(emptyCtx) != "" {
		t.Error("GetBotName() on empty context should return empty string")
	}
	if GetCategory(emptyCtx) != CategoryUnknown {
		t.Errorf("GetCategory() on empty context = %q, want %q", GetCategory(emptyCtx), CategoryUnknown)
	}
}

func TestHTTPMiddleware(t *testing.T) {
	tests := []struct {
		name      string
		userAgent string
		wantIsBot bool
	}{
		{
			name:      "Chrome",
			userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			wantIsBot: false,
		},
		{
			name:      "Googlebot",
			userAgent: "Googlebot/2.1",
			wantIsBot: true,
		},
		{
			name:      "curl",
			userAgent: "curl/7.68.0",
			wantIsBot: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotIsBot bool

			handler := HTTPMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotIsBot = IsBot(r.Context())
			}))

			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set("User-Agent", tt.userAgent)
			handler.ServeHTTP(httptest.NewRecorder(), req)

			if gotIsBot != tt.wantIsBot {
				t.Errorf("IsBot() = %v, want %v", gotIsBot, tt.wantIsBot)
			}
		})
	}
}

func TestBlockBotsMiddleware(t *testing.T) {
	tests := []struct {
		name       string
		userAgent  string
		wantStatus int
	}{
		{
			name:       "Chrome allowed",
			userAgent:  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/120.0.0.0 Safari/537.36",
			wantStatus: http.StatusOK,
		},
		{
			name:       "Googlebot allowed (good bot)",
			userAgent:  "Googlebot/2.1",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := BlockBotsMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set("User-Agent", tt.userAgent)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rr.Code, tt.wantStatus)
			}
		})
	}
}

func TestWithThreshold(t *testing.T) {
	// Low threshold - more likely to detect bots
	lowThreshold := New(WithThreshold(0.2))
	result := lowThreshold.Detect("python-requests/2.28.0")
	if !result.IsBot {
		t.Error("python-requests should be detected with low threshold")
	}

	// High threshold - less likely to detect unknown patterns
	highThreshold := New(WithThreshold(0.9))
	result = highThreshold.Detect("python-requests/2.28.0")
	// With very high threshold, unknown patterns may not be detected
	// This tests that threshold configuration works
	t.Logf("python-requests with 0.9 threshold: IsBot=%v, Confidence=%f", result.IsBot, result.Confidence)

	// Known bots should always be detected regardless of threshold
	result = highThreshold.Detect("Googlebot/2.1")
	if !result.IsBot {
		t.Error("Known bot Googlebot should be detected regardless of threshold")
	}
}

func TestSearchEngineIntent(t *testing.T) {
	detector := New()

	searchEngines := []string{
		"Googlebot/2.1",
		"bingbot/2.0",
		"Baiduspider/2.0",
	}

	for _, ua := range searchEngines {
		result := detector.Detect(ua)
		if result.Intent != IntentGood {
			t.Errorf("Search engine %q should have IntentGood, got %q", ua, result.Intent)
		}
	}
}

func TestSocialBotIntent(t *testing.T) {
	detector := New()

	socialBots := []string{
		"facebookexternalhit/1.1",
		"Twitterbot/1.0",
		"LinkedInBot/1.0",
		"Slackbot-LinkExpanding 1.0",
	}

	for _, ua := range socialBots {
		result := detector.Detect(ua)
		if result.Intent != IntentGood {
			t.Errorf("Social bot %q should have IntentGood, got %q", ua, result.Intent)
		}
	}
}
