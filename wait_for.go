package realtime_pubsub

import (
	"context"
	"fmt"
	"time"
)

// WaitFor provides methods to wait for acknowledgments or replies.
type WaitFor struct {
	client WaitForClient
	id     string
}

// WaitForClient defines the interface that the client must implement.
type WaitForClient interface {
	WaitFor(eventName string, timeout time.Duration) (interface{}, error)
}

// WaitForAck waits for an acknowledgment event with a timeout.
func (w *WaitFor) WaitForAck(timeout time.Duration) (interface{}, error) {
	eventName := fmt.Sprintf("ack.%v", w.id)
	return w.client.WaitFor(eventName, timeout)
}

// WaitForReply waits for a reply event with a timeout.
func (w *WaitFor) WaitForReply(timeout time.Duration) (ResponseMessage, error) {
	eventName := fmt.Sprintf("response.%v", w.id)

	// Wait for the reply event
	result, err := w.client.WaitFor(eventName, timeout)
	if err != nil {
		return nil, err
	}

	// Extract the reply message directly
	args, ok := result.([]interface{})
	if !ok || len(args) == 0 {
		return nil, fmt.Errorf("invalid reply format")
	}

	msg, ok := args[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid reply message format")
	}

	return msg, nil
}

// WaitFor waits for a specific event to occur within a timeout period.
func (c *Client) WaitFor(eventName string, timeout time.Duration) (interface{}, error) {
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
		return result, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
