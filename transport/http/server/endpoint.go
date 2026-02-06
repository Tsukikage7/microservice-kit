package server

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/Tsukikage7/microservice-kit/endpoint"
	"github.com/Tsukikage7/microservice-kit/transport/response"
)

// DecodeRequestFunc 从 HTTP 请求解码为业务请求对象.
//
// 示例：
//
//	func decodeGetUserRequest(_ context.Context, r *http.Request) (any, error) {
//	    vars := mux.Vars(r)
//	    id, err := strconv.Atoi(vars["id"])
//	    if err != nil {
//	        return nil, err
//	    }
//	    return GetUserRequest{ID: id}, nil
//	}
type DecodeRequestFunc func(ctx context.Context, r *http.Request) (request any, err error)

// EncodeResponseFunc 将业务响应编码到 HTTP ResponseWriter.
//
// 示例：
//
//	func encodeJSONResponse(_ context.Context, w http.ResponseWriter, response any) error {
//	    w.Header().Set("Content-Type", "application/json; charset=utf-8")
//	    return json.NewEncoder(w).Encode(response)
//	}
type EncodeResponseFunc func(ctx context.Context, w http.ResponseWriter, response any) error

// EncodeErrorFunc 将错误编码到 HTTP ResponseWriter.
type EncodeErrorFunc func(ctx context.Context, err error, w http.ResponseWriter)

// RequestFunc 可以从 HTTP 请求中提取信息并放入 context.
//
// 在 endpoint 调用之前执行.
type RequestFunc func(ctx context.Context, r *http.Request) context.Context

// ResponseFunc 可以从 context 中提取信息并操作 ResponseWriter.
//
// 在 endpoint 调用之后、写入响应之前执行.
type ResponseFunc func(ctx context.Context, w http.ResponseWriter) context.Context

// EndpointHandler 将 Endpoint 包装为 http.Handler.
//
// 示例：
//
//	getUserHandler := server.NewEndpointHandler(
//	    getUserEndpoint,
//	    decodeGetUserRequest,
//	    encodeJSONResponse,
//	)
//	mux.Handle("/users/{id}", getUserHandler)
type EndpointHandler struct {
	endpoint     endpoint.Endpoint
	dec          DecodeRequestFunc
	enc          EncodeResponseFunc
	before       []RequestFunc
	after        []ResponseFunc
	errorEncoder EncodeErrorFunc
}

// EndpointOption EndpointHandler 配置选项.
type EndpointOption func(*EndpointHandler)

// NewEndpointHandler 创建 EndpointHandler.
//
// 示例：
//
//	handler := server.NewEndpointHandler(
//	    getUserEndpoint,
//	    decodeGetUserRequest,
//	    encodeJSONResponse,
//	    server.WithErrorEncoder(encodeError),
//	    server.WithBefore(extractAuthToken),
//	)
func NewEndpointHandler(
	e endpoint.Endpoint,
	dec DecodeRequestFunc,
	enc EncodeResponseFunc,
	opts ...EndpointOption,
) *EndpointHandler {
	h := &EndpointHandler{
		endpoint:     e,
		dec:          dec,
		enc:          enc,
		errorEncoder: defaultErrorEncoder,
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// ServeHTTP 实现 http.Handler 接口.
func (h *EndpointHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 执行 before 函数
	for _, f := range h.before {
		ctx = f(ctx, r)
	}

	// 解码请求
	request, err := h.dec(ctx, r)
	if err != nil {
		h.errorEncoder(ctx, err, w)
		return
	}

	// 调用 endpoint
	response, err := h.endpoint(ctx, request)
	if err != nil {
		h.errorEncoder(ctx, err, w)
		return
	}

	// 执行 after 函数
	for _, f := range h.after {
		ctx = f(ctx, w)
	}

	// 编码响应
	if err := h.enc(ctx, w, response); err != nil {
		h.errorEncoder(ctx, err, w)
		return
	}
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

// WithErrorEncoder 设置错误编码器.
func WithErrorEncoder(enc EncodeErrorFunc) EndpointOption {
	return func(h *EndpointHandler) {
		h.errorEncoder = enc
	}
}

// WithResponse 启用统一响应格式的错误编码器.
//
// 错误将以 {"code": xxx, "message": "xxx"} 格式返回，
// 并自动映射到正确的 HTTP 状态码.
// 内部错误（5xxxx）的详细信息将被隐藏.
func WithResponse() EndpointOption {
	return func(h *EndpointHandler) {
		h.errorEncoder = responseErrorEncoder
	}
}

// defaultErrorEncoder 默认错误编码器.
func defaultErrorEncoder(_ context.Context, err error, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte(err.Error()))
}

// responseErrorEncoder 统一响应格式的错误编码器.
func responseErrorEncoder(_ context.Context, err error, w http.ResponseWriter) {
	code := response.ExtractCode(err)
	message := response.ExtractMessage(err)

	resp := response.Response[any]{
		Code:    code.Num,
		Message: message,
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code.HTTPStatus)
	_ = json.NewEncoder(w).Encode(resp)
}

// EncodeJSONResponse 通用 JSON 响应编码器.
//
// 便捷函数，用于快速设置 JSON 响应.
func EncodeJSONResponse(_ context.Context, w http.ResponseWriter, resp any) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return json.NewEncoder(w).Encode(resp)
}
