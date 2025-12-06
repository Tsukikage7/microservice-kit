package domain

import "time"

// DomainEvent 领域事件接口.
type DomainEvent interface {
	EventName() string
	OccurredTime() time.Time
}

// BaseEvent 领域事件基类.
type BaseEvent struct {
	name         string
	occurredTime time.Time
}

// NewBaseEvent 创建领域事件.
func NewBaseEvent(name string) BaseEvent {
	return BaseEvent{name: name, occurredTime: time.Now()}
}

func (e BaseEvent) EventName() string       { return e.name }
func (e BaseEvent) OccurredTime() time.Time { return e.occurredTime }
