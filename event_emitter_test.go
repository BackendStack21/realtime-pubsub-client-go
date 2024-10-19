package realtime_pubsub

import (
	"sync"
	"testing"
	"time"
)

// TestEventEmitterOnAndEmit tests the basic functionality of registering and emitting events.
func TestEventEmitterOnAndEmit(t *testing.T) {
	emitter := NewEventEmitter()
	var wg sync.WaitGroup
	wg.Add(1)

	eventName := "test.event"
	expectedData := "Hello, World!"

	emitter.On(eventName, func(args ...interface{}) {
		defer wg.Done()
		if len(args) != 1 {
			t.Errorf("Expected 1 argument, got %d", len(args))
			return
		}
		if args[0] != expectedData {
			t.Errorf("Expected data '%s', got '%v'", expectedData, args[0])
		}
	})

	emitter.Emit(eventName, expectedData)

	// Wait for the listener to be called
	wg.Wait()
}

// TestEventEmitterOff tests that listeners can be removed and no longer receive events.
func TestEventEmitterOff(t *testing.T) {
	emitter := NewEventEmitter()
	eventName := "test.event"
	var called bool

	listenerID := emitter.On(eventName, func(args ...interface{}) {
		called = true
	})

	// Remove the listener
	emitter.Off(eventName, listenerID)

	emitter.Emit(eventName, "data")

	// Ensure the listener was not called
	if called {
		t.Errorf("Listener was called after being removed")
	}
}

// TestEventEmitterOnce tests that a listener registered with Once is only called once.
func TestEventEmitterOnce(t *testing.T) {
	emitter := NewEventEmitter()
	var callCount int
	eventName := "test.event"

	emitter.Once(eventName, func(args ...interface{}) {
		callCount++
	})

	// Emit the event twice
	emitter.Emit(eventName, "data1")
	emitter.Emit(eventName, "data2")

	time.Sleep(100 * time.Millisecond)
	// Listener should have been called only once
	if callCount != 1 {
		t.Errorf("Expected listener to be called once, but was called %d times", callCount)
	}
}

// TestEventEmitterWildcard tests that wildcard patterns in event names are supported.
func TestEventEmitterWildcard(t *testing.T) {
	emitter := NewEventEmitter()
	var wg sync.WaitGroup
	wg.Add(4)

	eventPatterns := []string{"user.*", "order.**", "*.created"}

	expectedCalls := map[string]int{
		"user.*":    1,
		"order.**":  1,
		"*.created": 2,
	}

	callCounts := make(map[string]int)
	mu := sync.Mutex{}

	for _, pattern := range eventPatterns {
		pattern := pattern
		emitter.On(pattern, func(args ...interface{}) {
			defer wg.Done()
			mu.Lock()
			callCounts[pattern]++
			mu.Unlock()
		})
	}

	// Emit events
	emitter.Emit("user.login", "User logged in")
	emitter.Emit("order.created", "Order created")
	emitter.Emit("product.created", "Product created")

	// Wait for all listeners to be called
	wg.Wait()

	// Verify call counts
	for pattern, expected := range expectedCalls {
		if callCounts[pattern] != expected {
			t.Errorf("Expected pattern '%s' to be called %d times, but was called %d times", pattern, expected, callCounts[pattern])
		}
	}
}

// TestEventEmitterNoMatch tests that listeners are not called if the event does not match.
func TestEventEmitterNoMatch(t *testing.T) {
	emitter := NewEventEmitter()
	var called bool

	emitter.On("test.event", func(args ...interface{}) {
		called = true
	})

	// Emit a different event
	emitter.Emit("other.event", "data")

	if called {
		t.Errorf("Listener was called for a non-matching event")
	}
}

// TestEventEmitterMultipleListeners tests that multiple listeners for the same event are all called.
func TestEventEmitterMultipleListeners(t *testing.T) {
	emitter := NewEventEmitter()
	var wg sync.WaitGroup
	wg.Add(2)

	eventName := "test.event"

	emitter.On(eventName, func(args ...interface{}) {
		defer wg.Done()
	})

	emitter.On(eventName, func(args ...interface{}) {
		defer wg.Done()
	})

	emitter.Emit(eventName, "data")

	// Wait for both listeners to be called
	wg.Wait()
}

// TestEventEmitterListenerDataIsolation tests that listeners receive their own data and are not affected by others.
func TestEventEmitterListenerDataIsolation(t *testing.T) {
	emitter := NewEventEmitter()
	var wg sync.WaitGroup
	wg.Add(2)

	eventName1 := "test.event1"
	eventName2 := "test.event2"

	emitter.On(eventName1, func(args ...interface{}) {
		defer wg.Done()
		if args[0] != "data1" {
			t.Errorf("Listener for %s expected 'data1', got '%v'", eventName1, args[0])
		}
	})

	emitter.On(eventName2, func(args ...interface{}) {
		defer wg.Done()
		if args[0] != "data2" {
			t.Errorf("Listener for %s expected 'data2', got '%v'", eventName2, args[0])
		}
	})

	// Emit events separately
	emitter.Emit(eventName1, "data1")
	emitter.Emit(eventName2, "data2")

	// Wait for both listeners to be called
	wg.Wait()
}

// TestEventEmitterConcurrentAccess tests that the EventEmitter is safe for concurrent use.
func TestEventEmitterConcurrentAccess(t *testing.T) {
	emitter := NewEventEmitter()
	eventName := "test.event"
	var wg sync.WaitGroup
	listenerCount := 100
	wg.Add(listenerCount)

	for i := 0; i < listenerCount; i++ {
		emitter.On(eventName, func(args ...interface{}) {
			defer wg.Done()
		})
	}

	// Emit the event
	emitter.Emit(eventName, "data")

	// Wait for all listeners to be called
	wg.Wait()
}

// TestEventEmitterRecursiveEmit tests that emitting events within listeners does not cause deadlocks.
func TestEventEmitterRecursiveEmit(t *testing.T) {
	emitter := NewEventEmitter()
	eventName1 := "event1"
	eventName2 := "event2"
	var wg sync.WaitGroup
	wg.Add(2)

	emitter.On(eventName1, func(args ...interface{}) {
		defer wg.Done()
		// Emit in a new goroutine to prevent potential deadlock
		go emitter.Emit(eventName2, "data")
	})

	emitter.On(eventName2, func(args ...interface{}) {
		defer wg.Done()
	})

	// Start the chain by emitting event1
	emitter.Emit(eventName1, "data")

	// Wait for both listeners to be called
	wg.Wait()
}

// TestEventEmitterWildcardMultipleLevels tests '**' wildcard matching across multiple event levels.
func TestEventEmitterWildcardMultipleLevels(t *testing.T) {
	emitter := NewEventEmitter()
	var wg sync.WaitGroup
	wg.Add(3)

	eventPattern := "app.**"
	emittedEvents := []string{
		"app.start",
		"app.module.init",
		"app.module.component.load",
	}

	emitter.On(eventPattern, func(args ...interface{}) {
		defer wg.Done()
	})

	// Emit events
	for _, event := range emittedEvents {
		emitter.Emit(event, "data")
	}

	// Wait for all listeners to be called
	wg.Wait()
}

// TestEventEmitterOffNonExistentListener tests that calling Off on a non-existent listener does not cause a panic.
func TestEventEmitterOffNonExistentListener(t *testing.T) {
	emitter := NewEventEmitter()
	// Attempt to remove a listener that doesn't exist
	emitter.Off("nonexistent.event", 999)
	// If no panic occurs, the test passes
}

// TestEventEmitterEmitNoListeners tests that emitting an event with no listeners does not cause any issues.
func TestEventEmitterEmitNoListeners(t *testing.T) {
	emitter := NewEventEmitter()
	// Emit an event that has no listeners
	emitter.Emit("no.listeners", "data")
	// If no panic occurs, the test passes
}

// TestEventEmitterListenerIDUniqueness tests that listener IDs are unique.
func TestEventEmitterListenerIDUniqueness(t *testing.T) {
	emitter := NewEventEmitter()
	eventName := "test.event"

	listenerIDs := make(map[int]struct{})
	for i := 0; i < 100; i++ {
		id := emitter.On(eventName, func(args ...interface{}) {})
		if _, exists := listenerIDs[id]; exists {
			t.Errorf("Duplicate listener ID detected: %d", id)
		}
		listenerIDs[id] = struct{}{}
	}
}

// TestEventEmitterEventPatternMatching tests various event pattern matching scenarios.
func TestEventEmitterEventPatternMatching(t *testing.T) {
	testCases := []struct {
		pattern   string
		eventName string
		expected  bool
	}{
		{"user.*", "user.login", true},
		{"user.*", "user.profile.update", false},
		{"user.**", "user.profile.update", true},
		{"**", "any.event.name", true},
		{"order.**", "order.created", true},
		{"order.**", "order.processed.shipped", true},
		{"*.created", "user.created", true},
		{"*.created", "order.created", true},
		{"*.created", "user.profile.created", false},
		{"app.*.start", "app.server.start", true},
		{"app.*.start", "app.client.start", true},
		{"app.*.start", "app.server.stop", false},
	}

	for _, tc := range testCases {
		match := eventMatches(tc.pattern, tc.eventName)
		if match != tc.expected {
			t.Errorf("Pattern '%s' matching event '%s': expected %v, got %v", tc.pattern, tc.eventName, tc.expected, match)
		}
	}
}
