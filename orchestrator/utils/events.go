package utils

import (
	"sync"
)

// Event represents a general event structure with a name and payload. The payload is generic.
type Event[T any] struct {
	Name    string
	Payload T
}

// EventBus stores the information about subscribers and is generic over the event payload.
type EventBus[T any] struct {
	subscribers map[string][]chan Event[T]
	lock        sync.RWMutex
}

// NewEventBus creates a new EventBus instance.
func NewEventBus[T any]() *EventBus[T] {
	return &EventBus[T]{
		subscribers: make(map[string][]chan Event[T]),
	}
}

// Subscribe adds a new subscriber to one or more events.
func (eb *EventBus[T]) Subscribe(eventNames ...string) <-chan Event[T] {
	eb.lock.Lock()
	defer eb.lock.Unlock()

	ch := make(chan Event[T], 10) // Create a new channel with a buffer.
	for _, eventName := range eventNames {
		if _, ok := eb.subscribers[eventName]; !ok {
			eb.subscribers[eventName] = []chan Event[T]{}
		}
		eb.subscribers[eventName] = append(eb.subscribers[eventName], ch)
	}
	return ch // Return the channel to the subscriber.
}

// Publish publishes an event to all its subscribers.
func (eb *EventBus[T]) PublishEvent(event Event[T]) {
	eb.lock.RLock()
	defer eb.lock.RUnlock()

	if channels, ok := eb.subscribers[event.Name]; ok {
		for _, ch := range channels {
			go func(c chan Event[T]) {
				c <- event // Send event to subscriber channel.
			}(ch)
		}
	}
}

func (eb *EventBus[T]) Publish(name string, payload T) {
	event := Event[T]{Name: name, Payload: payload}
	eb.lock.RLock()
	defer eb.lock.RUnlock()

	if channels, ok := eb.subscribers[event.Name]; ok {
		for _, ch := range channels {
			go func(c chan Event[T]) {
				c <- event // Send event to subscriber channel.
			}(ch)
		}
	}
}

func (eb *EventBus[T]) AreEventsInBus() bool {
	eb.lock.RLock()
	defer eb.lock.RUnlock()

	for _, channels := range eb.subscribers {
		for _, ch := range channels {
			if len(ch) > 0 {
				return true
			}
		}
	}
	return false
}
