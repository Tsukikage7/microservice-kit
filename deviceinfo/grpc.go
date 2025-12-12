package deviceinfo

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// gRPC metadata 键名.
const (
	MetadataKeyUserAgent              = "user-agent"
	MetadataKeySecCHUA                = "sec-ch-ua"
	MetadataKeySecCHUAMobile          = "sec-ch-ua-mobile"
	MetadataKeySecCHUAPlatform        = "sec-ch-ua-platform"
	MetadataKeySecCHUAPlatformVersion = "sec-ch-ua-platform-version"
	MetadataKeySecCHUAArch            = "sec-ch-ua-arch"
	MetadataKeySecCHUAModel           = "sec-ch-ua-model"
	MetadataKeyDeviceMemory           = "device-memory"
	MetadataKeyDPR                    = "dpr"
)

// UnaryServerInterceptor 返回一元 gRPC 拦截器，解析设备信息并存入 context.
func UnaryServerInterceptor(opts ...Option) grpc.UnaryServerInterceptor {
	parser := New(opts...)

	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		ctx = parseAndStore(ctx, parser)
		return handler(ctx, req)
	}
}

// StreamServerInterceptor 返回流 gRPC 拦截器，解析设备信息并存入 context.
func StreamServerInterceptor(opts ...Option) grpc.StreamServerInterceptor {
	parser := New(opts...)

	return func(
		srv any,
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		ctx := parseAndStore(ss.Context(), parser)
		wrapped := &wrappedServerStream{
			ServerStream: ss,
			ctx:          ctx,
		}
		return handler(srv, wrapped)
	}
}

// parseAndStore 从 gRPC metadata 解析设备信息并存入 context.
func parseAndStore(ctx context.Context, parser *Parser) context.Context {
	headers := Headers{}

	if md, ok := metadata.FromIncomingContext(ctx); ok {
		headers.UserAgent = getMetadataValue(md, MetadataKeyUserAgent)
		headers.SecCHUA = getMetadataValue(md, MetadataKeySecCHUA)
		headers.SecCHUAMobile = getMetadataValue(md, MetadataKeySecCHUAMobile)
		headers.SecCHUAPlatform = getMetadataValue(md, MetadataKeySecCHUAPlatform)
		headers.SecCHUAPlatformVersion = getMetadataValue(md, MetadataKeySecCHUAPlatformVersion)
		headers.SecCHUAArch = getMetadataValue(md, MetadataKeySecCHUAArch)
		headers.SecCHUAModel = getMetadataValue(md, MetadataKeySecCHUAModel)
		headers.DeviceMemory = getMetadataValue(md, MetadataKeyDeviceMemory)
		headers.DPR = getMetadataValue(md, MetadataKeyDPR)
	}

	deviceInfo := parser.Parse(headers)
	return WithInfo(ctx, deviceInfo)
}

// getMetadataValue 从 metadata 获取单个值.
func getMetadataValue(md metadata.MD, key string) string {
	if values := md.Get(key); len(values) > 0 {
		return values[0]
	}
	return ""
}

// wrappedServerStream 包装 grpc.ServerStream 以提供自定义 context.
type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

// Context 返回包装后的 context.
func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}
