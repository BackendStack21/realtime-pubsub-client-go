package realtime_pubsub

import (
	"context"
	"errors"
	"log"
	"sync"
	"testing"
	"time"
)

// MockEventEmitter is a mock implementation of EventEmitter.
type MockEventEmitter struct {
	mu        sync.RWMutex
	listeners map[string][]ListenerFunc
}

func NewMockEventEmitter() *MockEventEmitter {
	return &MockEventEmitter{
		listeners: make(map[string][]ListenerFunc),
	}
}

func (e *MockEventEmitter) On(event string, listener ListenerFunc) int {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.listeners[event] = append(e.listeners[event], listener)
	// Return a dummy listener ID (not used in the mock)
	return len(e.listeners[event]) - 1
}

func (e *MockEventEmitter) Off(event string, id int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if listeners, ok := e.listeners[event]; ok {
		if id >= 0 && id < len(listeners) {
			// Remove the listener at index id
			e.listeners[event] = append(listeners[:id], listeners[id+1:]...)
		}
	}
}

func (e *MockEventEmitter) Emit(event string, args ...interface{}) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if listeners, ok := e.listeners[event]; ok {
		for _, listener := range listeners {
			// Call the listener in a separate goroutine to mimic asynchronous behavior
			go listener(args...)
		}
	}
}

// MockClient is a mock implementation of Client.
type MockClient struct {
	eventEmitter *MockEventEmitter
	logger       *log.Logger
}

func NewMockClient() *MockClient {
	return &MockClient{
		eventEmitter: NewMockEventEmitter(),
		logger:       log.New(log.Writer(), "mockclient: ", log.LstdFlags),
	}
}

func (c *MockClient) On(event string, listener ListenerFunc) int {
	return c.eventEmitter.On(event, listener)
}

func (c *MockClient) Off(event string, id int) {
	c.eventEmitter.Off(event, id)
}

func (c *MockClient) WaitFor(eventName string, timeout time.Duration) (interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	ch := make(chan interface{}, 1)

	listenerID := c.On(eventName, func(args ...interface{}) {
		select {
		case ch <- args:
		default:
		}
	})
	defer c.Off(eventName, listenerID)

	select {
	case result := <-ch:
		c.logger.Printf("Event '%s' received: %v", eventName, result)
		return result, nil
	case <-ctx.Done():
		c.logger.Printf("Timeout waiting for event '%s'", eventName)
		return nil, ctx.Err()
	}
}

func (c *MockClient) Emit(eventName string, args ...interface{}) {
	c.eventEmitter.Emit(eventName, args...)
}

// Ensure MockClient implements WaitForClient interface
var _ WaitForClient = (*MockClient)(nil)

// TestWaitForAckSuccess tests that WaitForAck successfully returns when the ack event is emitted.
func TestWaitForAckSuccess(t *testing.T) {
	client := NewMockClient()
	waitFor := &WaitFor{
		client: client, // Now acceptable because MockClient implements WaitForClient
		id:     "test-id",
	}

	// Emit the ack event after a short delay
	go func() {
		time.Sleep(100 * time.Millisecond)
		client.Emit("ack.test-id", "ack data")
	}()

	result, err := waitFor.WaitForAck(1 * time.Second)
	if err != nil {
		t.Errorf("Expected WaitForAck to succeed, but got error: %v", err)
	}

	args, ok := result.([]interface{})
	if !ok || len(args) == 0 {
		t.Errorf("Expected result to be non-empty []interface{}, got %v", result)
	}

	if args[0] != "ack data" {
		t.Errorf("Expected ack data to be 'ack data', got '%v'", args[0])
	}
}

// TestWaitForAckTimeout tests that WaitForAck times out if the ack event is not emitted.
func TestWaitForAckTimeout(t *testing.T) {
	client := NewMockClient()
	waitFor := &WaitFor{
		client: client,
		id:     "test-id",
	}

	// Do not emit the ack event
	result, err := waitFor.WaitForAck(200 * time.Millisecond)
	if err == nil {
		t.Errorf("Expected WaitForAck to timeout, but got result: %v", result)
	}

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected error to be context.DeadlineExceeded, got %v", err)
	}
}

// TestWaitForReplySuccess tests that WaitForReply successfully returns when the response event is emitted.
func TestWaitForReplySuccess(t *testing.T) {
	client := NewMockClient()
	waitFor := &WaitFor{
		client: client,
		id:     "reply-id",
	}

	// Emit the response event after a short delay
	go func() {
		time.Sleep(100 * time.Millisecond)
		client.Emit("response.reply-id", map[string]interface{}{
			"data":   "reply data",
			"id":     "reply-id",
			"status": "ok",
		})
	}()

	result, err := waitFor.WaitForReply(1 * time.Second)
	if err != nil {
		t.Errorf("Expected WaitForReply to succeed, but got error: %v", err)
	}

	if result.Data() != "reply data" {
		t.Errorf("Expected reply data to be 'reply data', got '%v'", result.Data())
	}
	if result.Id() != "reply-id" {
		t.Errorf("Expected reply id to be 'reply-id', got '%v'", result.Id())
	}
	if result.Status() != "ok" {
		t.Errorf("Expected reply status to be 'ok', got '%v'", result.Status())
	}

}

// TestWaitForReplyTimeout tests that WaitForReply times out if the response event is not emitted.
func TestWaitForReplyTimeout(t *testing.T) {
	client := NewMockClient()
	waitFor := &WaitFor{
		client: client,
		id:     "reply-id",
	}

	// Do not emit the response event
	result, err := waitFor.WaitForReply(200 * time.Millisecond)
	if err == nil {
		t.Errorf("Expected WaitForReply to timeout, but got result: %v", result)
	}

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected error to be context.DeadlineExceeded, got %v", err)
	}
}

// TestClientWaitForSuccess tests that Client.WaitFor successfully returns when the event is emitted.
func TestClientWaitForSuccess(t *testing.T) {
	client := NewMockClient()

	// Emit the event after a short delay
	go func() {
		time.Sleep(100 * time.Millisecond)
		client.Emit("test.event", "event data")
	}()

	result, err := client.WaitFor("test.event", 1*time.Second)
	if err != nil {
		t.Errorf("Expected WaitFor to succeed, but got error: %v", err)
	}

	args, ok := result.([]interface{})
	if !ok || len(args) == 0 {
		t.Errorf("Expected result to be non-empty []interface{}, got %v", result)
	}

	if args[0] != "event data" {
		t.Errorf("Expected event data to be 'event data', got '%v'", args[0])
	}
}

// TestClientWaitForTimeout tests that Client.WaitFor times out if the event is not emitted.
func TestClientWaitForTimeout(t *testing.T) {
	client := NewMockClient()

	// Do not emit the event
	result, err := client.WaitFor("test.event", 200*time.Millisecond)
	if err == nil {
		t.Errorf("Expected WaitFor to timeout, but got result: %v", result)
	}

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected error to be context.DeadlineExceeded, got %v", err)
	}
}

// TestWaitForMultipleListeners tests that multiple listeners can wait for the same event.
func TestWaitForMultipleListeners(t *testing.T) {
	client := NewMockClient()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		result, err := client.WaitFor("test.event", 1*time.Second)
		if err != nil {
			t.Errorf("Listener 1: Expected WaitFor to succeed, but got error: %v", err)
			return
		}
		args, _ := result.([]interface{})
		if args[0] != "event data" {
			t.Errorf("Listener 1: Expected event data to be 'event data', got '%v'", args[0])
		}
	}()

	go func() {
		defer wg.Done()
		result, err := client.WaitFor("test.event", 1*time.Second)
		if err != nil {
			t.Errorf("Listener 2: Expected WaitFor to succeed, but got error: %v", err)
			return
		}
		args, _ := result.([]interface{})
		if args[0] != "event data" {
			t.Errorf("Listener 2: Expected event data to be 'event data', got '%v'", args[0])
		}
	}()

	// Emit the event after a short delay
	time.Sleep(100 * time.Millisecond)
	client.Emit("test.event", "event data")

	wg.Wait()
}

// TestWaitForEventReceivedAfterTimeout tests that events received after timeout are ignored.
func TestWaitForEventReceivedAfterTimeout(t *testing.T) {
	client := NewMockClient()

	resultCh := make(chan interface{})
	errCh := make(chan error)

	go func() {
		result, err := client.WaitFor("test.event", 100*time.Millisecond)
		if err != nil {
			errCh <- err
			return
		}
		resultCh <- result
	}()

	// Emit the event after the timeout
	time.Sleep(200 * time.Millisecond)
	client.Emit("test.event", "event data")

	select {
	case <-resultCh:
		t.Errorf("Expected WaitFor to timeout, but it succeeded")
	case err := <-errCh:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("Expected error to be context.DeadlineExceeded, got %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Errorf("Test timed out waiting for WaitFor to return")
	}
}
