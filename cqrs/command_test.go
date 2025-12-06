package cqrs

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// CreateOrderCommand 测试用命令.
type CreateOrderCommand struct {
	UserID    string
	ProductID string
	Quantity  int
}

// CreateOrderResult 测试用命令结果.
type CreateOrderResult struct {
	OrderID string
}

// createOrderHandler 测试用命令处理器.
type createOrderHandler struct {
	shouldFail bool
}

func (h *createOrderHandler) Handle(ctx context.Context, cmd CreateOrderCommand) (CreateOrderCommand, CreateOrderResult, error) {
	if h.shouldFail {
		return cmd, CreateOrderResult{}, errors.New("create order failed")
	}
	return cmd, CreateOrderResult{OrderID: "ORD-" + cmd.UserID + "-" + cmd.ProductID}, nil
}

func TestApplyCommand_Success(t *testing.T) {
	ctx := context.Background()
	handler := &createOrderHandler{}

	cmd := CreateOrderCommand{
		UserID:    "user-1",
		ProductID: "prod-1",
		Quantity:  5,
	}

	returnedCmd, result, err := ApplyCommand(ctx, cmd, handler)

	require.NoError(t, err)
	assert.Equal(t, cmd, returnedCmd)
	assert.Equal(t, "ORD-user-1-prod-1", result.OrderID)
}

func TestApplyCommand_Error(t *testing.T) {
	ctx := context.Background()
	handler := &createOrderHandler{shouldFail: true}

	cmd := CreateOrderCommand{
		UserID:    "user-1",
		ProductID: "prod-1",
		Quantity:  5,
	}

	returnedCmd, result, err := ApplyCommand(ctx, cmd, handler)

	require.Error(t, err)
	assert.Equal(t, "create order failed", err.Error())
	assert.Equal(t, cmd, returnedCmd)
	assert.Empty(t, result.OrderID)
}

func TestApplyCommand_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	handler := &contextAwareCommandHandler{}

	cmd := CreateOrderCommand{UserID: "user-1"}

	_, _, err := ApplyCommand(ctx, cmd, handler)

	require.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

// contextAwareCommandHandler 检查 context 的处理器.
type contextAwareCommandHandler struct{}

func (h *contextAwareCommandHandler) Handle(ctx context.Context, cmd CreateOrderCommand) (CreateOrderCommand, CreateOrderResult, error) {
	if err := ctx.Err(); err != nil {
		return cmd, CreateOrderResult{}, err
	}
	return cmd, CreateOrderResult{OrderID: "ORD-123"}, nil
}
