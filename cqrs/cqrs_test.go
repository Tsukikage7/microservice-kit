package cqrs

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommandBus(t *testing.T) {
	ctx := context.Background()

	type CreateOrder struct {
		OrderID string
		UserID  string
	}

	var handled CreateOrder

	bus := NewCommandBus(func(ctx context.Context, cmd CreateOrder) error {
		handled = cmd
		return nil
	})

	err := bus.Dispatch(ctx, CreateOrder{OrderID: "order-1", UserID: "user-1"})

	assert.NoError(t, err)
	assert.Equal(t, "order-1", handled.OrderID)
	assert.Equal(t, "user-1", handled.UserID)
}

func TestQueryBus(t *testing.T) {
	ctx := context.Background()

	type GetOrder struct {
		OrderID string
	}

	type OrderDTO struct {
		ID     string
		Status string
	}

	bus := NewQueryBus(func(ctx context.Context, q GetOrder) (OrderDTO, error) {
		return OrderDTO{ID: q.OrderID, Status: "completed"}, nil
	})

	result, err := bus.Dispatch(ctx, GetOrder{OrderID: "order-1"})

	assert.NoError(t, err)
	assert.Equal(t, "order-1", result.ID)
	assert.Equal(t, "completed", result.Status)
}
