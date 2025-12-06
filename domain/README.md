# Domain

轻量级 DDD 构建块。

## 使用

### 聚合根

```go
type Order struct {
    domain.AggregateRoot[string]  // 泛型 ID
    UserID string
    Status string
}

func NewOrder(id, userID string) *Order {
    order := &Order{
        AggregateRoot: domain.NewAggregateRoot(id),
        UserID:        userID,
        Status:        "pending",
    }
    order.RaiseEvent(domain.NewBaseEvent("OrderCreated"))
    return order
}

// 支持任意 ID 类型
type Product struct { domain.AggregateRoot[int64] }
type User struct { domain.AggregateRoot[uuid.UUID] }
```

### 事件总线

```go
bus := domain.NewEventBus()

// 订阅
bus.Subscribe("OrderCreated", func(ctx context.Context, e domain.DomainEvent) error {
    log.Println("订单已创建")
    return nil
})

// 发布聚合事件
order := NewOrder("order-1", "user-1")
bus.Dispatch(ctx, order.DomainEvents(), order.ClearDomainEvents)
```

## API

| 类型 | 说明 |
|------|------|
| `AggregateRoot[ID]` | 泛型聚合根 |
| `DomainEvent` | 事件接口 |
| `BaseEvent` | 事件基类 |
| `EventBus` | 事件总线 |
