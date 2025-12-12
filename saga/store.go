package saga

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Tsukikage7/microservice-kit/storage/cache"
)

// Store Saga 状态存储接口.
type Store interface {
	// Save 保存 Saga 状态.
	Save(ctx context.Context, state *State) error

	// Get 获取 Saga 状态.
	Get(ctx context.Context, id string) (*State, error)

	// Delete 删除 Saga 状态.
	Delete(ctx context.Context, id string) error

	// List 列出指定状态的 Saga.
	List(ctx context.Context, status SagaStatus, limit int) ([]*State, error)
}

// KV Saga 状态存储所需的键值存储接口.
//
// 这是 saga 包的最小依赖接口.
// 可以用 cache.Cache、Redis 客户端或其他存储实现.
type KV interface {
	// Get 获取键的值.
	Get(ctx context.Context, key string) (string, error)

	// Set 设置键值对.
	Set(ctx context.Context, key string, value string, ttl time.Duration) error

	// Del 删除键.
	Del(ctx context.Context, keys ...string) error
}

// stateDTO 用于序列化的状态结构.
type stateDTO struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Status      SagaStatus      `json:"status"`
	CurrentStep int             `json:"current_step"`
	StepResults []stepResultDTO `json:"step_results"`
	Error       string          `json:"error,omitempty"`
	StartedAt   time.Time       `json:"started_at"`
	CompletedAt *time.Time      `json:"completed_at,omitempty"`
	Data        map[string]any  `json:"data,omitempty"`
}

// stepResultDTO 用于序列化的步骤结果结构.
type stepResultDTO struct {
	StepName string     `json:"step_name"`
	Status   StepStatus `json:"status"`
	Error    string     `json:"error,omitempty"`
	Duration int64      `json:"duration"`
}

// toDTO 将 State 转换为 DTO.
func toDTO(state *State) *stateDTO {
	dto := &stateDTO{
		ID:          state.ID,
		Name:        state.Name,
		Status:      state.Status,
		CurrentStep: state.CurrentStep,
		StepResults: make([]stepResultDTO, len(state.StepResults)),
		Error:       state.Error,
		StartedAt:   state.StartedAt,
		CompletedAt: state.CompletedAt,
		Data:        state.Data,
	}

	for i, r := range state.StepResults {
		dto.StepResults[i] = stepResultDTO{
			StepName: r.StepName,
			Status:   r.Status,
			Duration: r.Duration,
		}
		if r.Error != nil {
			dto.StepResults[i].Error = r.Error.Error()
		}
	}

	return dto
}

// fromDTO 将 DTO 转换为 State.
func fromDTO(dto *stateDTO) *State {
	state := &State{
		ID:          dto.ID,
		Name:        dto.Name,
		Status:      dto.Status,
		CurrentStep: dto.CurrentStep,
		StepResults: make([]StepResult, len(dto.StepResults)),
		Error:       dto.Error,
		StartedAt:   dto.StartedAt,
		CompletedAt: dto.CompletedAt,
		Data:        dto.Data,
	}

	for i, r := range dto.StepResults {
		state.StepResults[i] = StepResult{
			StepName: r.StepName,
			Status:   r.Status,
			Duration: r.Duration,
		}
		// 注意：序列化后 error 信息会丢失类型，只保留消息
	}

	return state
}

// KVStore 基于 KV 接口的 Saga 状态存储.
//
// 适用于分布式部署场景.
type KVStore struct {
	kv         KV
	keyPrefix  string
	defaultTTL time.Duration
}

// KVStoreOption KV 存储配置选项.
type KVStoreOption func(*KVStore)

// WithStoreKeyPrefix 设置键前缀.
func WithStoreKeyPrefix(prefix string) KVStoreOption {
	return func(s *KVStore) {
		s.keyPrefix = prefix
	}
}

// WithStoreTTL 设置默认 TTL.
func WithStoreTTL(ttl time.Duration) KVStoreOption {
	return func(s *KVStore) {
		s.defaultTTL = ttl
	}
}

// NewKVStore 创建 KV 存储.
//
// kv: KV 存储实现（可用 CacheKV 适配 cache.Cache）
func NewKVStore(kv KV, opts ...KVStoreOption) *KVStore {
	s := &KVStore{
		kv:         kv,
		keyPrefix:  "saga:",
		defaultTTL: 24 * time.Hour, // 默认24小时
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Save 保存 Saga 状态.
func (s *KVStore) Save(ctx context.Context, state *State) error {
	dto := toDTO(state)
	data, err := json.Marshal(dto)
	if err != nil {
		return err
	}

	key := s.keyPrefix + state.ID
	return s.kv.Set(ctx, key, string(data), s.defaultTTL)
}

// Get 获取 Saga 状态.
func (s *KVStore) Get(ctx context.Context, id string) (*State, error) {
	key := s.keyPrefix + id
	data, err := s.kv.Get(ctx, key)
	if err != nil {
		return nil, ErrSagaNotFound
	}
	if data == "" {
		return nil, ErrSagaNotFound
	}

	var dto stateDTO
	if err := json.Unmarshal([]byte(data), &dto); err != nil {
		return nil, err
	}

	return fromDTO(&dto), nil
}

// Delete 删除 Saga 状态.
func (s *KVStore) Delete(ctx context.Context, id string) error {
	key := s.keyPrefix + id
	return s.kv.Del(ctx, key)
}

// List 列出指定状态的 Saga.
//
// 注意: KV 实现不支持高效的条件查询，返回空列表.
// 建议在生产环境使用专门的索引或数据库来支持列表查询.
func (s *KVStore) List(ctx context.Context, status SagaStatus, limit int) ([]*State, error) {
	// KV 不支持高效的条件查询，返回空列表
	// 如果需要此功能，建议使用数据库存储或维护额外的索引
	return nil, nil
}

// cacheKV 是 cache.Cache 到 KV 的适配器.
type cacheKV struct {
	cache cache.Cache
}

// CacheKV 将 cache.Cache 适配为 KV 接口.
//
// 示例:
//
//	redisCache, _ := cache.New(&cache.Config{Type: "redis", ...})
//	kv := saga.CacheKV(redisCache)
//	store := saga.NewKVStore(kv)
func CacheKV(c cache.Cache) KV {
	return &cacheKV{cache: c}
}

func (c *cacheKV) Get(ctx context.Context, key string) (string, error) {
	return c.cache.Get(ctx, key)
}

func (c *cacheKV) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	return c.cache.Set(ctx, key, value, ttl)
}

func (c *cacheKV) Del(ctx context.Context, keys ...string) error {
	return c.cache.Del(ctx, keys...)
}

// nopStore 空存储，不保存任何状态.
//
// 适用于不需要持久化状态的场景.
type nopStore struct{}

// newNopStore 创建空存储（内部使用）.
func newNopStore() *nopStore {
	return &nopStore{}
}

// Save 保存状态（空实现）.
func (s *nopStore) Save(ctx context.Context, state *State) error {
	return nil
}

// Get 获取状态（始终返回未找到）.
func (s *nopStore) Get(ctx context.Context, id string) (*State, error) {
	return nil, ErrSagaNotFound
}

// Delete 删除状态（空实现）.
func (s *nopStore) Delete(ctx context.Context, id string) error {
	return nil
}

// List 列出状态（返回空列表）.
func (s *nopStore) List(ctx context.Context, status SagaStatus, limit int) ([]*State, error) {
	return nil, nil
}
