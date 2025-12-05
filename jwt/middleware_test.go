package jwt

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Tsukikage7/microservice-kit/logger"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

// mockLogger 测试用日志记录器（完整实现 Logger 接口）.
type mockLogger struct{}

func (m *mockLogger) Debug(args ...any)                          {}
func (m *mockLogger) Debugf(format string, args ...any)          {}
func (m *mockLogger) Info(args ...any)                           {}
func (m *mockLogger) Infof(format string, args ...any)           {}
func (m *mockLogger) Warn(args ...any)                           {}
func (m *mockLogger) Warnf(format string, args ...any)           {}
func (m *mockLogger) Error(args ...any)                          {}
func (m *mockLogger) Errorf(format string, args ...any)          {}
func (m *mockLogger) Fatal(args ...any)                          {}
func (m *mockLogger) Fatalf(format string, args ...any)          {}
func (m *mockLogger) Panic(args ...any)                          {}
func (m *mockLogger) Panicf(format string, args ...any)          {}
func (m *mockLogger) With(fields ...logger.Field) logger.Logger  { return m }
func (m *mockLogger) WithContext(ctx context.Context) logger.Logger { return m }
func (m *mockLogger) Sync() error                                { return nil }
func (m *mockLogger) Close() error                               { return nil }

// testClaims 测试用 Claims.
type testClaims struct {
	jwt.RegisteredClaims
	UserID   string `json:"user_id"`
	Username string `json:"username"`
}

// newTestJWT 创建测试用 JWT 服务.
func newTestJWT() *JWT {
	return New(
		WithSecretKey("test-secret-key-for-testing"),
		WithLogger(&mockLogger{}),
		WithIssuer("test-issuer"),
	)
}

// generateTestToken 生成测试令牌.
func generateTestToken(j *JWT, subject string) string {
	claims := &StandardClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subject,
			Issuer:    "test-issuer",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token, _ := j.Generate(claims)
	return token
}

func TestNewSigner(t *testing.T) {
	j := newTestJWT()

	t.Run("成功签名", func(t *testing.T) {
		claims := &StandardClaims{
			RegisteredClaims: jwt.RegisteredClaims{
				Subject:   "user-123",
				Issuer:    "test-issuer",
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
			},
		}

		var capturedToken string
		endpoint := func(ctx context.Context, req any) (any, error) {
			// 验证 context 中有 Token
			token, ok := TokenFromContext(ctx)
			assert.True(t, ok)
			assert.NotEmpty(t, token)
			capturedToken = token
			return "success", nil
		}

		middleware := NewSigner(j)
		wrapped := middleware(endpoint)

		// 创建带 Claims 的 context
		ctx := ContextWithClaims(context.Background(), claims)

		resp, err := wrapped(ctx, nil)

		assert.NoError(t, err)
		assert.Equal(t, "success", resp)
		assert.NotEmpty(t, capturedToken)

		// 验证生成的令牌可以被解析
		validatedClaims, err := j.Validate(capturedToken)
		assert.NoError(t, err)
		subject, _ := validatedClaims.GetSubject()
		assert.Equal(t, "user-123", subject)
	})

	t.Run("无 Claims 时跳过签名", func(t *testing.T) {
		endpoint := func(ctx context.Context, req any) (any, error) {
			// 验证 context 中没有 Token
			_, ok := TokenFromContext(ctx)
			assert.False(t, ok)
			return "success", nil
		}

		middleware := NewSigner(j)
		wrapped := middleware(endpoint)

		resp, err := wrapped(context.Background(), nil)

		assert.NoError(t, err)
		assert.Equal(t, "success", resp)
	})
}

func TestNewParser(t *testing.T) {
	j := newTestJWT()

	t.Run("成功验证", func(t *testing.T) {
		token := generateTestToken(j, "user-123")

		endpoint := func(ctx context.Context, req any) (any, error) {
			// 验证 context 中有 Claims
			claims, ok := ClaimsFromContext(ctx)
			assert.True(t, ok)
			assert.NotNil(t, claims)

			// 验证 context 中有 Token
			ctxToken, ok := TokenFromContext(ctx)
			assert.True(t, ok)
			assert.NotEmpty(t, ctxToken)

			return "success", nil
		}

		middleware := NewParser(j)
		wrapped := middleware(endpoint)

		// 创建带 token 的 context
		ctx := metadata.NewIncomingContext(context.Background(),
			metadata.Pairs("authorization", token))

		resp, err := wrapped(ctx, nil)

		assert.NoError(t, err)
		assert.Equal(t, "success", resp)
	})

	t.Run("缺少令牌", func(t *testing.T) {
		endpoint := func(ctx context.Context, req any) (any, error) {
			return "success", nil
		}

		middleware := NewParser(j)
		wrapped := middleware(endpoint)

		resp, err := wrapped(context.Background(), nil)

		assert.Error(t, err)
		assert.Equal(t, ErrTokenNotFound, err)
		assert.Nil(t, resp)
	})

	t.Run("无效令牌", func(t *testing.T) {
		endpoint := func(ctx context.Context, req any) (any, error) {
			return "success", nil
		}

		middleware := NewParser(j)
		wrapped := middleware(endpoint)

		ctx := metadata.NewIncomingContext(context.Background(),
			metadata.Pairs("authorization", "Bearer invalid-token"))

		resp, err := wrapped(ctx, nil)

		assert.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("过期令牌", func(t *testing.T) {
		// 创建过期令牌
		claims := &StandardClaims{
			RegisteredClaims: jwt.RegisteredClaims{
				Subject:   "user-123",
				Issuer:    "test-issuer",
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)), // 已过期
				IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			},
		}
		token, _ := j.Generate(claims)

		endpoint := func(ctx context.Context, req any) (any, error) {
			return "success", nil
		}

		middleware := NewParser(j)
		wrapped := middleware(endpoint)

		ctx := metadata.NewIncomingContext(context.Background(),
			metadata.Pairs("authorization", token))

		resp, err := wrapped(ctx, nil)

		assert.Error(t, err)
		assert.Nil(t, resp)
	})
}

func TestNewParserWithClaims(t *testing.T) {
	j := newTestJWT()

	t.Run("自定义 Claims 类型", func(t *testing.T) {
		// 生成带自定义 Claims 的令牌
		customClaims := &testClaims{
			RegisteredClaims: jwt.RegisteredClaims{
				Subject:   "user-456",
				Issuer:    "test-issuer",
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
			},
			UserID:   "456",
			Username: "testuser",
		}
		token, err := j.Generate(customClaims)
		require.NoError(t, err)

		endpoint := func(ctx context.Context, req any) (any, error) {
			claims, ok := ClaimsFromContext(ctx)
			assert.True(t, ok)
			assert.NotNil(t, claims)
			return "success", nil
		}

		claimsFactory := func() Claims {
			return &testClaims{}
		}

		middleware := NewParserWithClaims(j, claimsFactory)
		wrapped := middleware(endpoint)

		ctx := metadata.NewIncomingContext(context.Background(),
			metadata.Pairs("authorization", token))

		resp, err := wrapped(ctx, nil)

		assert.NoError(t, err)
		assert.Equal(t, "success", resp)
	})
}

func TestNewParser_Whitelist(t *testing.T) {
	whitelist := NewWhitelist()
	whitelist.AddHTTPPaths("/health", "/metrics")

	j := New(
		WithSecretKey("test-secret-key"),
		WithLogger(&mockLogger{}),
		WithWhitelist(whitelist),
	)

	t.Run("白名单路径跳过验证", func(t *testing.T) {
		endpoint := func(ctx context.Context, req any) (any, error) {
			// 白名单请求不应有 Claims
			_, ok := ClaimsFromContext(ctx)
			assert.False(t, ok)
			return "success", nil
		}

		middleware := NewParser(j)
		wrapped := middleware(endpoint)

		// 模拟白名单请求（通过 HTTP 请求）
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		resp, err := wrapped(context.Background(), req)

		assert.NoError(t, err)
		assert.Equal(t, "success", resp)
	})
}

func TestNewParser_ContextPropagation(t *testing.T) {
	j := newTestJWT()
	token := generateTestToken(j, "user-789")

	t.Run("Claims 正确传播到下游", func(t *testing.T) {
		var capturedClaims Claims

		endpoint := func(ctx context.Context, req any) (any, error) {
			claims, ok := ClaimsFromContext(ctx)
			if ok {
				capturedClaims = claims
			}
			return "success", nil
		}

		middleware := NewParser(j)
		wrapped := middleware(endpoint)

		ctx := metadata.NewIncomingContext(context.Background(),
			metadata.Pairs("authorization", token))

		resp, err := wrapped(ctx, nil)

		assert.NoError(t, err)
		assert.Equal(t, "success", resp)
		assert.NotNil(t, capturedClaims)

		subject, err := capturedClaims.GetSubject()
		assert.NoError(t, err)
		assert.Equal(t, "user-789", subject)
	})

	t.Run("Token 正确传播到下游", func(t *testing.T) {
		var capturedToken string

		endpoint := func(ctx context.Context, req any) (any, error) {
			if t, ok := TokenFromContext(ctx); ok {
				capturedToken = t
			}
			return "success", nil
		}

		middleware := NewParser(j)
		wrapped := middleware(endpoint)

		ctx := metadata.NewIncomingContext(context.Background(),
			metadata.Pairs("authorization", token))

		resp, err := wrapped(ctx, nil)

		assert.NoError(t, err)
		assert.Equal(t, "success", resp)
		assert.NotEmpty(t, capturedToken)
	})
}

func TestNewParser_Concurrent(t *testing.T) {
	j := newTestJWT()

	endpoint := func(ctx context.Context, req any) (any, error) {
		claims, ok := ClaimsFromContext(ctx)
		assert.True(t, ok)
		assert.NotNil(t, claims)
		return "ok", nil
	}

	middleware := NewParser(j)
	wrapped := middleware(endpoint)

	// 并发调用
	done := make(chan bool, 100)
	for i := 0; i < 100; i++ {
		go func(id int) {
			token := generateTestToken(j, "user-"+string(rune('a'+id%26)))
			ctx := metadata.NewIncomingContext(context.Background(),
				metadata.Pairs("authorization", token))

			resp, err := wrapped(ctx, nil)
			assert.NoError(t, err)
			assert.Equal(t, "ok", resp)
			done <- true
		}(i)
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 100; i++ {
		<-done
	}
}

func TestHTTPMiddleware(t *testing.T) {
	j := newTestJWT()

	t.Run("成功验证", func(t *testing.T) {
		token := generateTestToken(j, "user-123")

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 验证 context 中有 Claims
			claims, ok := ClaimsFromContext(r.Context())
			assert.True(t, ok)
			assert.NotNil(t, claims)

			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})

		wrapped := HTTPMiddleware(j)(handler)

		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		req.Header.Set("Authorization", token)
		rec := httptest.NewRecorder()

		wrapped.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "OK", rec.Body.String())
	})

	t.Run("缺少令牌", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		wrapped := HTTPMiddleware(j)(handler)

		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		rec := httptest.NewRecorder()

		wrapped.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("无效令牌", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		wrapped := HTTPMiddleware(j)(handler)

		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		rec := httptest.NewRecorder()

		wrapped.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}

func TestHTTPMiddleware_Whitelist(t *testing.T) {
	whitelist := NewWhitelist()
	whitelist.AddHTTPPaths("/health", "/public/")

	j := New(
		WithSecretKey("test-secret-key"),
		WithLogger(&mockLogger{}),
		WithWhitelist(whitelist),
	)

	t.Run("精确匹配白名单", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})

		wrapped := HTTPMiddleware(j)(handler)

		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()

		wrapped.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("前缀匹配白名单", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		wrapped := HTTPMiddleware(j)(handler)

		req := httptest.NewRequest(http.MethodGet, "/public/images/logo.png", nil)
		rec := httptest.NewRecorder()

		wrapped.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestExtractToken(t *testing.T) {
	t.Run("从 gRPC metadata 提取", func(t *testing.T) {
		ctx := metadata.NewIncomingContext(context.Background(),
			metadata.Pairs("authorization", "Bearer test-token"))

		token, err := ExtractToken(ctx, nil)

		assert.NoError(t, err)
		assert.Equal(t, "test-token", token)
	})

	t.Run("从 HTTP 请求提取", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer http-token")

		token, err := ExtractToken(context.Background(), req)

		assert.NoError(t, err)
		assert.Equal(t, "http-token", token)
	})

	t.Run("从上下文提取", func(t *testing.T) {
		ctx := ContextWithToken(context.Background(), "context-token")

		token, err := ExtractToken(ctx, nil)

		assert.NoError(t, err)
		assert.Equal(t, "context-token", token)
	})

	t.Run("未找到令牌", func(t *testing.T) {
		token, err := ExtractToken(context.Background(), nil)

		assert.Error(t, err)
		assert.Equal(t, ErrTokenNotFound, err)
		assert.Empty(t, token)
	})
}

func TestExtractTokenFromHeader(t *testing.T) {
	testCases := []struct {
		name     string
		header   string
		expected string
	}{
		{"带 Bearer 前缀", "Bearer token123", "token123"},
		{"带小写 bearer 前缀", "bearer token123", "token123"},
		{"无前缀", "token123", "token123"},
		{"带空格", "Bearer   token123  ", "token123"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := extractTokenFromHeader(tc.header)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestContextFunctions(t *testing.T) {
	t.Run("Claims 上下文操作", func(t *testing.T) {
		claims := &StandardClaims{
			RegisteredClaims: jwt.RegisteredClaims{
				Subject: "user-123",
			},
		}

		ctx := ContextWithClaims(context.Background(), claims)

		retrieved, ok := ClaimsFromContext(ctx)
		assert.True(t, ok)
		assert.NotNil(t, retrieved)

		subject, err := retrieved.GetSubject()
		assert.NoError(t, err)
		assert.Equal(t, "user-123", subject)
	})

	t.Run("Token 上下文操作", func(t *testing.T) {
		ctx := ContextWithToken(context.Background(), "test-token")

		token, ok := TokenFromContext(ctx)
		assert.True(t, ok)
		assert.Equal(t, "test-token", token)
	})

	t.Run("获取 Subject", func(t *testing.T) {
		claims := &StandardClaims{
			RegisteredClaims: jwt.RegisteredClaims{
				Subject: "user-456",
			},
		}
		ctx := ContextWithClaims(context.Background(), claims)

		subject, ok := GetSubjectFromContext(ctx)
		assert.True(t, ok)
		assert.Equal(t, "user-456", subject)
	})

	t.Run("无 Claims 获取 Subject", func(t *testing.T) {
		subject, ok := GetSubjectFromContext(context.Background())
		assert.False(t, ok)
		assert.Empty(t, subject)
	})
}
