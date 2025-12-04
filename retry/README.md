# retry

提供简洁的重试机制，支持链式调用、上下文取消和超时控制。

## 安装

```go
import "github.com/Tsukikage7/microservice-kit/retry"
```

## 快速开始

```go
// 使用默认配置 (3次重试，100ms间隔)
err := retry.Do(ctx, func() error {
    return someOperation()
}).Run()

// 自定义配置
err := retry.Do(ctx, func() error {
    return someOperation()
}).WithMaxAttempts(5).WithDelay(time.Second).Run()
```

## API

### Do

创建重试器实例。

```go
func Do(ctx context.Context, fn func() error) *Retry
```

### 链式方法

| 方法 | 说明 | 默认值 |
|------|------|--------|
| `WithMaxAttempts(n int)` | 设置最大重试次数 | 3 |
| `WithDelay(d time.Duration)` | 设置重试间隔 | 100ms |
| `Run()` | 执行重试 | - |

### 错误

| 错误 | 说明 |
|------|------|
| `ErrMaxAttempts` | 已达到最大重试次数 |

## 使用示例

### 基础用法

```go
ctx := context.Background()

err := retry.Do(ctx, func() error {
    resp, err := http.Get("https://api.example.com/data")
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    return nil
}).Run()

if errors.Is(err, retry.ErrMaxAttempts) {
    log.Println("重试次数已用尽")
}
```

### 自定义重试策略

```go
err := retry.Do(ctx, func() error {
    return connectToDatabase()
}).WithMaxAttempts(10).WithDelay(2 * time.Second).Run()
```

### 配合超时使用

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

err := retry.Do(ctx, func() error {
    return sendRequest()
}).WithMaxAttempts(5).WithDelay(time.Second).Run()

if errors.Is(err, context.DeadlineExceeded) {
    log.Println("操作超时")
}
```

### 可取消的重试

```go
ctx, cancel := context.WithCancel(context.Background())

go func() {
    time.Sleep(5 * time.Second)
    cancel() // 5秒后取消
}()

err := retry.Do(ctx, func() error {
    return longRunningOperation()
}).WithMaxAttempts(100).WithDelay(time.Second).Run()

if errors.Is(err, context.Canceled) {
    log.Println("操作已取消")
}
```

## 特性

- **链式调用**: 流畅的 API 设计
- **上下文支持**: 完整支持 context 取消和超时
- **零依赖**: 仅使用标准库
- **100% 测试覆盖率**
