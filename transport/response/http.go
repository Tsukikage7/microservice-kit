package response

import (
	"encoding/json"
	"net/http"
)

// WriteJSON 写入 JSON 响应.
func WriteJSON(w http.ResponseWriter, statusCode int, data any) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	return json.NewEncoder(w).Encode(data)
}

// WriteSuccess 写入成功响应.
func WriteSuccess[T any](w http.ResponseWriter, data T) error {
	return WriteJSON(w, http.StatusOK, OK(data))
}

// WriteFail 写入失败响应.
func WriteFail(w http.ResponseWriter, code Code) error {
	return WriteJSON(w, code.HTTPStatus, Fail[any](code))
}

// WriteFailWithMessage 写入带自定义消息的失败响应.
func WriteFailWithMessage(w http.ResponseWriter, code Code, message string) error {
	return WriteJSON(w, code.HTTPStatus, FailWithMessage[any](code, message))
}

// WriteError 写入错误响应.
//
// 自动从 error 提取错误码和消息.
func WriteError(w http.ResponseWriter, err error) error {
	code := ExtractCode(err)
	return WriteJSON(w, code.HTTPStatus, FailWithError[any](err))
}

// WritePaged 写入分页响应.
func WritePaged[T any](w http.ResponseWriter, resp PagedResponse[T]) error {
	return WriteJSON(w, http.StatusOK, resp)
}
