# Saga

Saga 分布式事务编排包，通过编排一系列本地事务并在失败时执行补偿操作来保证最终一致性。

## 功能特性

- **编排式 Saga** - 按顺序执行步骤，失败时自动补偿
- **数据传递** - 步骤间共享数据
- **超时控制** - 支持整体超时
- **重试机制** - 步骤失败时自动重试
- **状态持久化** - 可选的状态存储
- **执行钩子** - 步骤开始/结束回调

## 工作原理

```
1. 按顺序执行每个步骤的 Action
2. 如果所有步骤成功 → 事务完成
3. 如果某个步骤失败 → 按逆序执行已完成步骤的 Compensate
4. 补偿完成后返回错误
```

```
步骤执行: Step1 → Step2 → Step3 (失败)
补偿执行: Comp2 → Comp1
```

## 快速开始

### 基本用法

```go
import "github.com/Tsukikage7/microservice-kit/saga"

// 定义步骤函数
reserveInventory := func(ctx context.Context, data *saga.Data) error {
    orderID := data.GetString("order_id")
    // 预留库存
    reservationID, err := inventoryService.Reserve(ctx, orderID, items)
    if err != nil {
        return err
    }
    data.Set("reservation_id", reservationID)
    return nil
}

cancelReservation := func(ctx context.Context, data *saga.Data) error {
    reservationID := data.GetString("reservation_id")
    return inventoryService.Cancel(ctx, reservationID)
}

chargePayment := func(ctx context.Context, data *saga.Data) error {
    orderID := data.GetString("order_id")
    amount := data.GetInt64("amount")
    paymentID, err := paymentService.Charge(ctx, orderID, amount)
    if err != nil {
        return err
    }
    data.Set("payment_id", paymentID)
    return nil
}

refundPayment := func(ctx context.Context, data *saga.Data) error {
    paymentID := data.GetString("payment_id")
    return paymentService.Refund(ctx, paymentID)
}

sendNotification := func(ctx context.Context, data *saga.Data) error {
    orderID := data.GetString("order_id")
    return notificationService.Send(ctx, orderID, "订单创建成功")
}

// 构建 Saga
createOrderSaga := saga.New("create-order").
    Step("reserve-inventory", reserveInventory, cancelReservation).
    Step("charge-payment", chargePayment, refundPayment).
    Step("send-notification", sendNotification, nil). // 通知不需要补偿
    Build()

// 执行
data := saga.NewData()
data.Set("order_id", "ORD-123")
data.Set("amount", int64(9900))

if err := createOrderSaga.ExecuteWithData(ctx, data); err != nil {
    log.Error("创建订单失败", "error", err)
    // 补偿已自动执行
}
```

### 带状态存储

```go
store := saga.NewMemoryStore()
defer store.Close()

createOrderSaga := saga.New("create-order").
    Step("reserve-inventory", reserveInventory, cancelReservation).
    Step("charge-payment", chargePayment, refundPayment).
    Options(saga.WithStore(store)).
    Build()
```

### 带超时和重试

```go
createOrderSaga := saga.New("create-order").
    Step("reserve-inventory", reserveInventory, cancelReservation).
    Step("charge-payment", chargePayment, refundPayment).
    Options(
        saga.WithTimeout(30*time.Second),    // 整体超时
        saga.WithRetry(3, time.Second),      // 每步重试3次，间隔1秒
        saga.WithLogger(log),
    ).
    Build()
```

## 配置选项

| 选项 | 默认值 | 说明 |
|------|--------|------|
| `WithStore(store)` | NopStore | 状态存储 |
| `WithLogger(log)` | - | 日志记录器 |
| `WithTimeout(duration)` | 无超时 | 整体执行超时 |
| `WithRetry(count, delay)` | 不重试 | 步骤重试配置 |
| `WithStepHooks(onStart, onEnd)` | - | 步骤执行钩子 |

## 数据传递

使用 `Data` 在步骤间传递数据：

```go
// 设置数据
data.Set("key", "value")
data.Set("count", 42)
data.Set("amount", int64(9900))
data.Set("enabled", true)

// 获取数据
str := data.GetString("key")      // "value"
num := data.GetInt("count")       // 42
amount := data.GetInt64("amount") // 9900
enabled := data.GetBool("enabled") // true

// 通用获取
val, ok := data.Get("key")

// 删除数据
data.Delete("key")

// 获取所有键
keys := data.Keys()
```

## 状态管理

### Saga 状态

| 状态 | 说明 |
|------|------|
| `pending` | 待执行 |
| `running` | 执行中 |
| `completed` | 执行完成 |
| `failed` | 执行失败 |
| `compensating` | 补偿中 |
| `compensated` | 已补偿 |
| `compensate_failed` | 补偿失败 |

### 步骤状态

| 状态 | 说明 |
|------|------|
| `pending` | 待执行 |
| `running` | 执行中 |
| `completed` | 执行完成 |
| `failed` | 执行失败 |
| `compensating` | 补偿中 |
| `compensated` | 已补偿 |
| `compensate_failed` | 补偿失败 |

### 查询状态

```go
store := saga.NewMemoryStore()

// 查询失败的 Saga
failedSagas, _ := store.List(ctx, saga.SagaStatusFailed, 100)

// 查询补偿失败的 Saga（需要人工干预）
compFailedSagas, _ := store.List(ctx, saga.SagaStatusCompensateFailed, 100)
```

## 最佳实践

### 1. 补偿操作必须幂等

```go
// ✅ 好的补偿：检查状态后再执行
cancelReservation := func(ctx context.Context, data *saga.Data) error {
    reservationID := data.GetString("reservation_id")
    if reservationID == "" {
        return nil // 没有预留，无需取消
    }

    reservation, err := inventoryService.Get(ctx, reservationID)
    if err != nil || reservation.Status == "cancelled" {
        return nil // 已取消，幂等返回
    }

    return inventoryService.Cancel(ctx, reservationID)
}
```

### 2. 合理安排步骤顺序

```go
// ✅ 好的顺序：先执行可补偿的操作，最后执行不可补偿的
saga.New("create-order").
    Step("reserve-inventory", reserve, cancelReserve).  // 可补偿
    Step("charge-payment", charge, refund).             // 可补偿
    Step("send-notification", notify, nil).             // 不可补偿，放最后
    Build()
```

### 3. 适当保存中间状态

```go
step := func(ctx context.Context, data *saga.Data) error {
    result, err := externalService.Call(ctx, req)
    if err != nil {
        return err
    }

    // 保存外部服务返回的 ID，补偿时需要使用
    data.Set("external_id", result.ID)
    return nil
}
```

### 4. 处理补偿失败

```go
// 定期检查补偿失败的 Saga
func checkCompensationFailures(store saga.Store) {
    ctx := context.Background()
    sagas, _ := store.List(ctx, saga.SagaStatusCompensateFailed, 100)

    for _, s := range sagas {
        // 发送告警
        alert.Send("Saga 补偿失败需要人工干预", s.ID, s.Name)

        // 或者重试补偿
        // retryCompensation(ctx, s)
    }
}
```

### 5. 使用日志和钩子监控

```go
createOrderSaga := saga.New("create-order").
    Step("reserve", reserve, cancel).
    Step("charge", charge, refund).
    Options(
        saga.WithLogger(log),
        saga.WithStepHooks(
            func(name string) {
                metrics.SagaStepStarted.WithLabelValues(name).Inc()
            },
            func(name string, err error) {
                if err != nil {
                    metrics.SagaStepFailed.WithLabelValues(name).Inc()
                } else {
                    metrics.SagaStepCompleted.WithLabelValues(name).Inc()
                }
            },
        ),
    ).
    Build()
```

## 典型场景

### 订单创建

```go
saga.New("create-order").
    Step("validate", validateOrder, nil).
    Step("reserve-inventory", reserveInventory, cancelReservation).
    Step("charge-payment", chargePayment, refundPayment).
    Step("create-order-record", createOrderRecord, deleteOrderRecord).
    Step("send-notification", sendNotification, nil).
    Build()
```

### 用户注册

```go
saga.New("user-registration").
    Step("create-user", createUser, deleteUser).
    Step("create-profile", createProfile, deleteProfile).
    Step("setup-permissions", setupPermissions, revokePermissions).
    Step("send-welcome-email", sendWelcomeEmail, nil).
    Build()
```

### 转账

```go
saga.New("transfer-money").
    Step("debit-source", debitSourceAccount, creditSourceAccount).
    Step("credit-target", creditTargetAccount, debitTargetAccount).
    Step("record-transaction", recordTransaction, deleteTransaction).
    Step("notify-parties", notifyParties, nil).
    Build()
```

## 注意事项

1. **补偿可能失败** - 设计时考虑补偿失败的情况，可能需要人工干预
2. **不保证隔离性** - Saga 在执行过程中，中间状态对其他事务可见
3. **最终一致性** - Saga 保证的是最终一致性，不是强一致性
4. **补偿要幂等** - 补偿操作可能被多次调用，必须是幂等的
5. **内存存储不持久化** - 生产环境建议实现持久化存储
