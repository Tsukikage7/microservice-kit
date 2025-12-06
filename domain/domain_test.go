package domain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAggregateRoot(t *testing.T) {
	agg := NewAggregateRoot("order-123")

	assert.Equal(t, "order-123", agg.ID())
	assert.Empty(t, agg.DomainEvents())

	// 发起事件
	agg.RaiseEvent(NewBaseEvent("OrderCreated"))
	assert.Len(t, agg.DomainEvents(), 1)

	// 清除事件
	agg.ClearDomainEvents()
	assert.Empty(t, agg.DomainEvents())
}

func TestAggregateRoot_Int64ID(t *testing.T) {
	agg := NewAggregateRoot(int64(12345))
	assert.Equal(t, int64(12345), agg.ID())
}

func TestBaseEvent(t *testing.T) {
	event := NewBaseEvent("OrderCreated")
	assert.Equal(t, "OrderCreated", event.EventName())
	assert.False(t, event.OccurredTime().IsZero())
}

func TestEventBus_Subscribe(t *testing.T) {
	bus := NewEventBus()
	ctx := context.Background()

	var handled []string

	bus.Subscribe("OrderCreated", func(ctx context.Context, event DomainEvent) error {
		handled = append(handled, event.EventName())
		return nil
	})

	err := bus.Publish(ctx, NewBaseEvent("OrderCreated"))
	assert.NoError(t, err)
	assert.Contains(t, handled, "OrderCreated")

	// 未订阅的事件不会触发
	handled = nil
	err = bus.Publish(ctx, NewBaseEvent("UserCreated"))
	assert.NoError(t, err)
	assert.Empty(t, handled)
}

func TestEventBus_SubscribeAll(t *testing.T) {
	bus := NewEventBus()
	ctx := context.Background()

	var handled []string

	bus.SubscribeAll(func(ctx context.Context, event DomainEvent) error {
		handled = append(handled, event.EventName())
		return nil
	})

	bus.Publish(ctx, NewBaseEvent("OrderCreated"))
	bus.Publish(ctx, NewBaseEvent("UserCreated"))

	assert.Contains(t, handled, "OrderCreated")
	assert.Contains(t, handled, "UserCreated")
}

func TestEventBus_Dispatch(t *testing.T) {
	bus := NewEventBus()
	ctx := context.Background()

	var handled []string
	bus.SubscribeAll(func(ctx context.Context, event DomainEvent) error {
		handled = append(handled, event.EventName())
		return nil
	})

	// 创建聚合并发起事件
	agg := NewAggregateRoot("order-1")
	agg.RaiseEvent(NewBaseEvent("OrderCreated"))
	agg.RaiseEvent(NewBaseEvent("ItemAdded"))

	// 分发事件
	err := bus.Dispatch(ctx, agg.DomainEvents(), agg.ClearDomainEvents)
	assert.NoError(t, err)
	assert.Contains(t, handled, "OrderCreated")
	assert.Contains(t, handled, "ItemAdded")
	assert.Empty(t, agg.DomainEvents())
}
