package cqrs

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// GetOrderQuery 测试用查询.
type GetOrderQuery struct {
	OrderID string
}

// OrderDTO 测试用查询结果.
type OrderDTO struct {
	ID     string
	Status string
	Total  float64
}

// getOrderHandler 测试用查询处理器.
type getOrderHandler struct {
	shouldFail bool
}

func (h *getOrderHandler) Handle(ctx context.Context, query GetOrderQuery) (OrderDTO, error) {
	if h.shouldFail {
		return OrderDTO{}, errors.New("order not found")
	}
	return OrderDTO{
		ID:     query.OrderID,
		Status: "completed",
		Total:  99.99,
	}, nil
}

func TestApplyQueryHandler_Success(t *testing.T) {
	ctx := context.Background()
	handler := &getOrderHandler{}

	query := GetOrderQuery{OrderID: "ORD-123"}

	result, err := ApplyQueryHandler(ctx, query, handler)

	require.NoError(t, err)
	assert.Equal(t, "ORD-123", result.ID)
	assert.Equal(t, "completed", result.Status)
	assert.Equal(t, 99.99, result.Total)
}

func TestApplyQueryHandler_Error(t *testing.T) {
	ctx := context.Background()
	handler := &getOrderHandler{shouldFail: true}

	query := GetOrderQuery{OrderID: "ORD-123"}

	result, err := ApplyQueryHandler(ctx, query, handler)

	require.Error(t, err)
	assert.Equal(t, "order not found", err.Error())
	assert.Empty(t, result.ID)
}

func TestApplyQueryHandler_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	handler := &contextAwareQueryHandler{}

	query := GetOrderQuery{OrderID: "ORD-123"}

	_, err := ApplyQueryHandler(ctx, query, handler)

	require.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

// contextAwareQueryHandler 检查 context 的处理器.
type contextAwareQueryHandler struct{}

func (h *contextAwareQueryHandler) Handle(ctx context.Context, query GetOrderQuery) (OrderDTO, error) {
	if err := ctx.Err(); err != nil {
		return OrderDTO{}, err
	}
	return OrderDTO{ID: query.OrderID, Status: "completed"}, nil
}

// ListOrdersQuery 测试用列表查询.
type ListOrdersQuery struct {
	UserID string
	Limit  int
	Offset int
}

// listOrdersHandler 返回切片的处理器.
type listOrdersHandler struct{}

func (h *listOrdersHandler) Handle(ctx context.Context, query ListOrdersQuery) ([]OrderDTO, error) {
	return []OrderDTO{
		{ID: "ORD-1", Status: "completed", Total: 10.0},
		{ID: "ORD-2", Status: "pending", Total: 20.0},
	}, nil
}

func TestApplyQueryHandler_SliceResult(t *testing.T) {
	ctx := context.Background()
	handler := &listOrdersHandler{}

	query := ListOrdersQuery{UserID: "user-1", Limit: 10, Offset: 0}

	result, err := ApplyQueryHandler(ctx, query, handler)

	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "ORD-1", result[0].ID)
	assert.Equal(t, "ORD-2", result[1].ID)
}
