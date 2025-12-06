# CQRS

命令查询职责分离。

## 使用

### Command（写操作）

```go
type CreateOrder struct {
    UserID string
    Amount int64
}

bus := cqrs.NewCommandBus(func(ctx context.Context, cmd CreateOrder) error {
    order := NewOrder(cmd.UserID, cmd.Amount)
    return repo.Save(ctx, order)
})

bus.Dispatch(ctx, CreateOrder{UserID: "user-1", Amount: 100})
```

### Query（读操作）

```go
type GetOrder struct {
    ID string
}

type OrderDTO struct {
    ID     string
    Status string
}

bus := cqrs.NewQueryBus(func(ctx context.Context, q GetOrder) (OrderDTO, error) {
    order, err := repo.Find(ctx, q.ID)
    if err != nil {
        return OrderDTO{}, err
    }
    return OrderDTO{ID: order.ID, Status: order.Status}, nil
})

result, _ := bus.Dispatch(ctx, GetOrder{ID: "order-1"})
fmt.Println(result.Status)  // completed
```

## API

| 类型 | 说明 |
|------|------|
| `CommandBus[C]` | 命令总线 |
| `CommandHandler[C]` | 命令处理器 |
| `QueryBus[Q, R]` | 查询总线 |
| `QueryHandler[Q, R]` | 查询处理器 |
