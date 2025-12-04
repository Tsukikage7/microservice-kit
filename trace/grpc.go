package trace

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// metadataCarrier 实现 propagation.TextMapCarrier 接口.
type metadataCarrier metadata.MD

func (mc metadataCarrier) Get(key string) string {
	vals := metadata.MD(mc).Get(key)
	if len(vals) > 0 {
		return vals[0]
	}
	return ""
}

func (mc metadataCarrier) Set(key, value string) {
	metadata.MD(mc).Set(key, value)
}

func (mc metadataCarrier) Keys() []string {
	keys := make([]string, 0, len(mc))
	for k := range mc {
		keys = append(keys, k)
	}
	return keys
}

// UnaryServerInterceptor 返回 gRPC 一元服务端拦截器.
//
// 使用示例:
//
//	server := grpc.NewServer(
//	    grpc.UnaryInterceptor(tracing.UnaryServerInterceptor("my-service")),
//	)
func UnaryServerInterceptor(serviceName string) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// 从 metadata 提取上下文
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			md = metadata.MD{}
		}
		ctx = otel.GetTextMapPropagator().Extract(ctx, metadataCarrier(md))

		tracer := otel.Tracer(serviceName)
		ctx, span := tracer.Start(ctx, info.FullMethod,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				semconv.RPCSystemGRPC,
				semconv.RPCService(serviceName),
				semconv.RPCMethod(info.FullMethod),
			),
		)
		defer span.End()

		// 执行处理器
		resp, err := handler(ctx, req)

		// 记录状态
		if err != nil {
			s, _ := status.FromError(err)
			span.SetAttributes(attribute.String("rpc.grpc.status_code", s.Code().String()))
			span.SetStatus(codes.Error, s.Message())
			span.RecordError(err)
		} else {
			span.SetAttributes(attribute.String("rpc.grpc.status_code", "OK"))
		}

		return resp, err
	}
}

// StreamServerInterceptor 返回 gRPC 流式服务端拦截器.
//
// 使用示例:
//
//	server := grpc.NewServer(
//	    grpc.StreamInterceptor(tracing.StreamServerInterceptor("my-service")),
//	)
func StreamServerInterceptor(serviceName string) grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		ctx := ss.Context()

		// 从 metadata 提取上下文
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			md = metadata.MD{}
		}
		ctx = otel.GetTextMapPropagator().Extract(ctx, metadataCarrier(md))

		tracer := otel.Tracer(serviceName)
		ctx, span := tracer.Start(ctx, info.FullMethod,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				semconv.RPCSystemGRPC,
				semconv.RPCService(serviceName),
				semconv.RPCMethod(info.FullMethod),
				attribute.Bool("rpc.grpc.is_client_stream", info.IsClientStream),
				attribute.Bool("rpc.grpc.is_server_stream", info.IsServerStream),
			),
		)
		defer span.End()

		// 包装 ServerStream 以传递新的 context
		wrappedStream := &serverStreamWrapper{
			ServerStream: ss,
			ctx:          ctx,
		}

		// 执行处理器
		err := handler(srv, wrappedStream)

		// 记录状态
		if err != nil {
			s, _ := status.FromError(err)
			span.SetAttributes(attribute.String("rpc.grpc.status_code", s.Code().String()))
			span.SetStatus(codes.Error, s.Message())
			span.RecordError(err)
		} else {
			span.SetAttributes(attribute.String("rpc.grpc.status_code", "OK"))
		}

		return err
	}
}

// serverStreamWrapper 包装 grpc.ServerStream 以传递追踪上下文.
type serverStreamWrapper struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *serverStreamWrapper) Context() context.Context {
	return w.ctx
}

// UnaryClientInterceptor 返回 gRPC 一元客户端拦截器.
//
// 使用示例:
//
//	conn, err := grpc.Dial(address,
//	    grpc.WithUnaryInterceptor(tracing.UnaryClientInterceptor("my-service")),
//	)
func UnaryClientInterceptor(serviceName string) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		tracer := otel.Tracer(serviceName)
		ctx, span := tracer.Start(ctx, method,
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithAttributes(
				semconv.RPCSystemGRPC,
				semconv.RPCService(serviceName),
				semconv.RPCMethod(method),
			),
		)
		defer span.End()

		// 注入追踪信息到 metadata
		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			md = metadata.MD{}
		} else {
			md = md.Copy()
		}
		otel.GetTextMapPropagator().Inject(ctx, metadataCarrier(md))
		ctx = metadata.NewOutgoingContext(ctx, md)

		// 执行调用
		err := invoker(ctx, method, req, reply, cc, opts...)

		// 记录状态
		if err != nil {
			s, _ := status.FromError(err)
			span.SetAttributes(attribute.String("rpc.grpc.status_code", s.Code().String()))
			span.SetStatus(codes.Error, s.Message())
			span.RecordError(err)
		} else {
			span.SetAttributes(attribute.String("rpc.grpc.status_code", "OK"))
		}

		return err
	}
}

// StreamClientInterceptor 返回 gRPC 流式客户端拦截器.
//
// 使用示例:
//
//	conn, err := grpc.Dial(address,
//	    grpc.WithStreamInterceptor(tracing.StreamClientInterceptor("my-service")),
//	)
func StreamClientInterceptor(serviceName string) grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		tracer := otel.Tracer(serviceName)
		ctx, span := tracer.Start(ctx, method,
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithAttributes(
				semconv.RPCSystemGRPC,
				semconv.RPCService(serviceName),
				semconv.RPCMethod(method),
				attribute.Bool("rpc.grpc.is_client_stream", desc.ClientStreams),
				attribute.Bool("rpc.grpc.is_server_stream", desc.ServerStreams),
			),
		)

		// 注入追踪信息到 metadata
		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			md = metadata.MD{}
		} else {
			md = md.Copy()
		}
		otel.GetTextMapPropagator().Inject(ctx, metadataCarrier(md))
		ctx = metadata.NewOutgoingContext(ctx, md)

		// 执行调用
		clientStream, err := streamer(ctx, desc, cc, method, opts...)

		// 记录错误（如果有）
		if err != nil {
			s, _ := status.FromError(err)
			span.SetAttributes(attribute.String("rpc.grpc.status_code", s.Code().String()))
			span.SetStatus(codes.Error, s.Message())
			span.RecordError(err)
			span.End()
			return nil, err
		}

		// 包装 ClientStream 以在流结束时关闭 span
		return &clientStreamWrapper{
			ClientStream: clientStream,
			span:         span,
		}, nil
	}
}

// clientStreamWrapper 包装 grpc.ClientStream 以在流结束时关闭 span.
type clientStreamWrapper struct {
	grpc.ClientStream
	span trace.Span
}

func (w *clientStreamWrapper) RecvMsg(m interface{}) error {
	err := w.ClientStream.RecvMsg(m)
	if err != nil {
		// 流结束或错误时关闭 span
		if err.Error() == "EOF" {
			w.span.SetAttributes(attribute.String("rpc.grpc.status_code", "OK"))
		} else {
			s, _ := status.FromError(err)
			w.span.SetAttributes(attribute.String("rpc.grpc.status_code", s.Code().String()))
			w.span.SetStatus(codes.Error, s.Message())
			w.span.RecordError(err)
		}
		w.span.End()
	}
	return err
}

// InjectGRPCMetadata 将追踪信息注入到 gRPC metadata.
//
// 用于手动传播追踪上下文.
//
// 使用示例:
//
//	ctx = tracing.InjectGRPCMetadata(ctx)
//	client.SomeMethod(ctx, req)
func InjectGRPCMetadata(ctx context.Context) context.Context {
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		md = metadata.MD{}
	} else {
		md = md.Copy()
	}
	otel.GetTextMapPropagator().Inject(ctx, metadataCarrier(md))
	return metadata.NewOutgoingContext(ctx, md)
}

// ExtractGRPCMetadata 从 gRPC metadata 提取追踪信息.
//
// 用于手动提取追踪上下文.
//
// 使用示例:
//
//	ctx = tracing.ExtractGRPCMetadata(ctx)
//	span := tracing.SpanFromContext(ctx)
func ExtractGRPCMetadata(ctx context.Context) context.Context {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx
	}
	return otel.GetTextMapPropagator().Extract(ctx, metadataCarrier(md))
}
