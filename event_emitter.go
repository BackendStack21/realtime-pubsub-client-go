package realtime_pubsub

import (
	"strings"
	"sync"
)

// EventEmitter allows registering and emitting events with wildcard support.
type EventEmitter struct {
	mu         sync.RWMutex
	events     map[string]map[int]ListenerFunc
	listenerID int
}

// NewEventEmitter initializes and returns a new EventEmitter instance.
func NewEventEmitter() *EventEmitter {
	return &EventEmitter{
		events: make(map[string]map[int]ListenerFunc),
	}
}

// On registers a listener for a specific event, supporting wildcards.
// Returns a unique listener ID for future removal.
func (e *EventEmitter) On(event string, listener ListenerFunc) int {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.listenerID++
	if e.events[event] == nil {
		e.events[event] = make(map[int]ListenerFunc)
	}
	e.events[event][e.listenerID] = listener
	return e.listenerID
}

// Off removes a listener for a specific event using the listener ID.
func (e *EventEmitter) Off(event string, id int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if listeners, ok := e.events[event]; ok {
		delete(listeners, id)
		if len(listeners) == 0 {
			delete(e.events, event)
		}
	}
}

// Once registers a listener for a specific event that will be called at most once.
func (e *EventEmitter) Once(event string, listener ListenerFunc) int {
	var id int
	wrapper := func(args ...interface{}) {
		listener(args...)
		e.Off(event, id)
	}
	id = e.On(event, wrapper)
	return id
}

// Emit emits an event with optional arguments, calling all matching listeners.
func (e *EventEmitter) Emit(event string, args ...interface{}) {
	// Collect the listeners to be called.
	e.mu.RLock()
	var listenersToCall []ListenerFunc
	for eventPattern, listeners := range e.events {
		if eventMatches(eventPattern, event) {
			for _, listener := range listeners {
				listenersToCall = append(listenersToCall, listener)
			}
		}
	}
	e.mu.RUnlock() // Release the lock before calling listeners.

	// Call the listeners outside the lock.
	for _, listener := range listenersToCall {
		listener(args...)
	}
}

// eventMatches checks if an event pattern matches an event name, supporting wildcards '*' and '**'.
func eventMatches(pattern, eventName string) bool {
	patternSegments := strings.Split(pattern, ".")
	eventSegments := strings.Split(eventName, ".")
	return matchSegments(patternSegments, eventSegments)
}

// matchSegments matches event segments with pattern segments, handling wildcards.
func matchSegments(patternSegments, eventSegments []string) bool {
	i, j := 0, 0
	for i < len(patternSegments) && j < len(eventSegments) {
		if patternSegments[i] == "**" {
			if i == len(patternSegments)-1 {
				return true
			}
			for k := j; k <= len(eventSegments); k++ {
				if matchSegments(patternSegments[i+1:], eventSegments[k:]) {
					return true
				}
			}
			return false
		} else if patternSegments[i] == "*" {
			i++
			j++
		} else if patternSegments[i] == eventSegments[j] {
			i++
			j++
		} else {
			return false
		}
	}
	for i < len(patternSegments) && patternSegments[i] == "**" {
		i++
	}
	return i == len(patternSegments) && j == len(eventSegments)
}
