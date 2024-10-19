package realtime_pubsub

import (
	"github.com/sirupsen/logrus"
)

// Config represents the configuration options for Client.
type Config struct {
	Logger           *logrus.Logger
	WebSocketOptions WebSocketOptions
}

// WebSocketOptions represents the WebSocket configuration options.
type WebSocketOptions struct {
	URLProvider func() (string, error)
}
