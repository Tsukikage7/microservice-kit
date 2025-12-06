package domain

// AggregateRoot 聚合根基类.
type AggregateRoot[ID any] struct {
	id     ID
	events []DomainEvent
}

// NewAggregateRoot 创建聚合根.
func NewAggregateRoot[ID any](id ID) AggregateRoot[ID] {
	return AggregateRoot[ID]{id: id}
}

// ID 返回聚合根 ID.
func (a *AggregateRoot[ID]) ID() ID { return a.id }

// RaiseEvent 发起领域事件.
func (a *AggregateRoot[ID]) RaiseEvent(event DomainEvent) {
	a.events = append(a.events, event)
}

// DomainEvents 返回待发布的领域事件.
func (a *AggregateRoot[ID]) DomainEvents() []DomainEvent { return a.events }

// ClearDomainEvents 清除领域事件.
func (a *AggregateRoot[ID]) ClearDomainEvents() { a.events = nil }
