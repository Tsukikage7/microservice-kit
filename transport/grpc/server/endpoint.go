package server

import (
	"context"

	"github.com/Tsukikage7/microservice-kit/transport"
	"google.golang.org/grpc/metadata"
)

// DecodeRequestFunc 从 gRPC 请求解码为业务请求对象.
//
// 示例：
//
//	func decodeGetUserRequest(_ context.Context, req any) (any, error) {
//	    r := req.(*pb.GetUserRequest)
//	    return GetUserRequest{ID: int(r.Id)}, nil
//	}
type DecodeRequestFunc func(ctx context.Context, request any) (any, error)

// EncodeResponseFunc 将业务响应编码为 gRPC 响应.
//
// 示例：
//
//	func encodeGetUserResponse(_ context.Context, resp any) (any, error) {
//	    r := resp.(GetUserResponse)
//	    return &pb.GetUserResponse{
//	        Id:   int64(r.ID),
//	        Name: r.Name,
//	    }, nil
//	}
type EncodeResponseFunc func(ctx context.Context, response any) (any, error)

// RequestFunc 可以从 gRPC metadata 中提取信息并放入 context.
//
// 在 endpoint 调用之前执行.
type RequestFunc func(ctx context.Context, md metadata.MD) context.Context

// ResponseFunc 可以从 context 中提取信息并操作 gRPC metadata.
//
// header 和 trailer 参数允许修改响应的 metadata.
type ResponseFunc func(ctx context.Context, header *metadata.MD, trailer *metadata.MD) context.Context

// Handler gRPC 服务处理器接口.
//
// 实现此接口的类型可以处理 gRPC 请求.
type Handler interface {
	ServeGRPC(ctx context.Context, request any) (context.Context, any, error)
}

// EndpointHandler 将 Endpoint 包装为 gRPC Handler.
//
// 示例：
//
//	getUserHandler := server.NewEndpointHandler(
//	    getUserEndpoint,
//	    decodeGetUserRequest,
//	    encodeGetUserResponse,
//	)
type EndpointHandler struct {
	endpoint transport.Endpoint
	dec      DecodeRequestFunc
	enc      EncodeResponseFunc
	before   []RequestFunc
	after    []ResponseFunc
}

// EndpointOption EndpointHandler 配置选项.
type EndpointOption func(*EndpointHandler)

// NewEndpointHandler 创建 gRPC EndpointHandler.
//
// 示例：
//
//	handler := server.NewEndpointHandler(
//	    getUserEndpoint,
//	    decodeGetUserRequest,
//	    encodeGetUserResponse,
//	    server.WithBefore(extractAuthFromMD),
//	)
func NewEndpointHandler(
	e transport.Endpoint,
	dec DecodeRequestFunc,
	enc EncodeResponseFunc,
	opts ...EndpointOption,
) *EndpointHandler {
	h := &EndpointHandler{
		endpoint: e,
		dec:      dec,
		enc:      enc,
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// ServeGRPC 实现 Handler 接口.
//
// 这个方法应该从你的 gRPC 服务实现中调用.
//
// 示例（在你的 gRPC 服务实现中）：
//
//	func (s *userServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
//	    _, resp, err := s.getUserHandler.ServeGRPC(ctx, req)
//	    if err != nil {
//	        return nil, err
//	    }
//	    return resp.(*pb.GetUserResponse), nil
//	}
func (h *EndpointHandler) ServeGRPC(ctx context.Context, request any) (context.Context, any, error) {
	// 提取 metadata 并执行 before 函数
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		md = metadata.MD{}
	}
	for _, f := range h.before {
		ctx = f(ctx, md)
	}

	// 解码请求
	req, err := h.dec(ctx, request)
	if err != nil {
		return ctx, nil, err
	}

	// 调用 endpoint
	response, err := h.endpoint(ctx, req)
	if err != nil {
		return ctx, nil, err
	}

	// 执行 after 函数
	var header, trailer metadata.MD
	for _, f := range h.after {
		ctx = f(ctx, &header, &trailer)
	}

	// 编码响应
	grpcResp, err := h.enc(ctx, response)
	if err != nil {
		return ctx, nil, err
	}

	return ctx, grpcResp, nil
}

// WithBefore 添加请求前处理函数.
func WithBefore(funcs ...RequestFunc) EndpointOption {
	return func(h *EndpointHandler) {
		h.before = append(h.before, funcs...)
	}
}

// WithAfter 添加响应后处理函数.
func WithAfter(funcs ...ResponseFunc) EndpointOption {
	return func(h *EndpointHandler) {
		h.after = append(h.after, funcs...)
	}
}
