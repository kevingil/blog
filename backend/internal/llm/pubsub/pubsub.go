package pubsub

import (
	"sync"
)

type EventType string

const (
	CreatedEvent EventType = "created"
	UpdatedEvent EventType = "updated"
	DeletedEvent EventType = "deleted"
)

type Subscriber[T any] interface {
	Subscribe(eventType EventType, handler func(T))
	Unsubscribe(eventType EventType, handler func(T))
}

type Suscriber[T any] interface {
	Subscribe(eventType EventType, handler func(T))
	Unsubscribe(eventType EventType, handler func(T))
}

type Publisher[T any] interface {
	Publish(eventType EventType, data T)
}

type Broker[T any] struct {
	mu          sync.RWMutex
	subscribers map[EventType][]func(T)
}

func NewBroker[T any]() *Broker[T] {
	return &Broker[T]{
		subscribers: make(map[EventType][]func(T)),
	}
}

func (b *Broker[T]) Subscribe(eventType EventType, handler func(T)) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, exists := b.subscribers[eventType]; !exists {
		b.subscribers[eventType] = make([]func(T), 0)
	}
	b.subscribers[eventType] = append(b.subscribers[eventType], handler)
}

func (b *Broker[T]) Unsubscribe(eventType EventType, handler func(T)) {
	b.mu.Lock()
	defer b.mu.Unlock()

	handlers, exists := b.subscribers[eventType]
	if !exists {
		return
	}

	// Note: This is a simple implementation that removes the first matching handler
	// In practice, you might want a more sophisticated way to match handlers
	for i, h := range handlers {
		if &h == &handler {
			b.subscribers[eventType] = append(handlers[:i], handlers[i+1:]...)
			break
		}
	}
}

func (b *Broker[T]) Publish(eventType EventType, data T) {
	b.mu.RLock()
	handlers, exists := b.subscribers[eventType]
	if !exists {
		b.mu.RUnlock()
		return
	}

	// Make a copy to avoid holding the lock during handler execution
	handlersCopy := make([]func(T), len(handlers))
	copy(handlersCopy, handlers)
	b.mu.RUnlock()

	// Execute handlers
	for _, handler := range handlersCopy {
		go handler(data)
	}
}
