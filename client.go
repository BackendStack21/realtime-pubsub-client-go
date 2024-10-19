package realtime_pubsub

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"net"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second    // Time allowed to write a message to the connection
	pongWait       = 15 * time.Second    // Time allowed to read the next Pong message from the server
	pingPeriod     = (pongWait * 9) / 10 // Send pings to the server with this period
	maxMessageSize = 6 * 1024            // Maximum message size allowed from the server
)

// Client encapsulates WebSocket connection, subscription, and message handling.
type Client struct {
	eventEmitter     *EventEmitter
	ws               *websocket.Conn
	config           Config
	logger           *logrus.Logger
	subscribedTopics map[string]struct{}
	isConnecting     bool
	mu               sync.Mutex
	writeChan        chan []byte
	closeChan        chan struct{}
	closeOnce        *sync.Once
	ctx              context.Context
	cancel           context.CancelFunc
	done             chan struct{}
}

// NewClient initializes a new Client instance.
func NewClient(config Config) *Client {
	logger := config.Logger
	if logger == nil {
		logger = logrus.New()
		logger.SetFormatter(&logrus.JSONFormatter{})
		logger.SetOutput(os.Stdout)
		logger.SetLevel(logrus.DebugLevel)
	}

	ctx, cancel := context.WithCancel(context.Background())

	client := &Client{
		eventEmitter:     NewEventEmitter(),
		config:           config,
		logger:           logger,
		subscribedTopics: make(map[string]struct{}),
		writeChan:        make(chan []byte, 256),
		closeChan:        make(chan struct{}),
		closeOnce:        new(sync.Once),
		ctx:              ctx,
		cancel:           cancel,
		done:             make(chan struct{}),
	}

	// Register event listeners
	client.eventEmitter.On("priv/acks.ack", client.onAck)
	client.eventEmitter.On("*.response", client.onResponse)
	client.eventEmitter.On("main.welcome", client.onWelcome)
	return client
}

// closeWebSocket safely closes the WebSocket connection once.
func (c *Client) closeWebSocket() {
	c.mu.Lock()
	if c.ws != nil {
		_ = c.ws.Close()
		c.ws = nil
	}
	c.mu.Unlock()
}

// Connect establishes a connection to the WebSocket server.
func (c *Client) Connect() {
	c.mu.Lock()
	c.isConnecting = false
	c.mu.Unlock()
	go c.connectLoop()
}

func (c *Client) Disconnect() error {
	c.closeOnce.Do(func() {
		c.logger.Infof("Disconnecting client")
		close(c.done) // Signal all goroutines to exit
		close(c.writeChan)
		c.closeWebSocket()
	})
	return nil
}

// Publish publishes a message to a specified topic.
func (c *Client) Publish(topic string, payload interface{}, opts ...PublishOption) (*WaitFor, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ws == nil {
		c.logger.Errorf("Attempted to publish without an active WebSocket connection.")
		return nil, fmt.Errorf("WebSocket connection is not established")
	}

	if opts == nil {
		opts = make([]PublishOption, 0)
	}

	// Set default options
	options := PublishOptions{
		ID:          getRandomID(rand.Reader),
		MessageType: "broadcast",
		Compress:    false,
	}

	// Apply provided options
	for _, opt := range opts {
		opt(&options)
	}

	message := map[string]interface{}{
		"type": "publish",
		"data": map[string]interface{}{
			"topic":       topic,
			"messageType": options.MessageType,
			"compress":    options.Compress,
			"payload":     payload,
			"id":          options.ID,
		},
	}

	messageBytes, err := json.Marshal(message)
	if err != nil {
		c.logger.Errorf("Failed to marshal message: %v", err)
		return nil, err
	}

	c.logger.Debugf("Publishing message to topic %s: %v", topic, payload)
	err = c.sendMessage(messageBytes)
	if err != nil {
		c.logger.Errorf("Failed to send message: %v", err)
		return nil, err
	}

	return &WaitFor{
		client: c,
		id:     options.ID,
	}, nil
}

// Send sends a message directly to the server.
func (c *Client) Send(payload interface{}, opts ...SendOption) (*WaitFor, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ws == nil {
		c.logger.Errorf("Attempted to send without an active WebSocket connection.")
		return nil, fmt.Errorf("WebSocket connection is not established")
	}

	// Set default options
	options := SendOptions{
		ID:          getRandomID(rand.Reader),
		MessageType: "",    // Can be empty
		Compress:    false, // Default compress is false
	}

	// Apply provided options
	for _, opt := range opts {
		opt(&options)
	}

	data := map[string]interface{}{
		"messageType": options.MessageType,
		"compress":    options.Compress,
		"payload":     payload,
		"id":          options.ID,
	}

	// Remove messageType from data if it's empty
	if options.MessageType == "" {
		delete(data, "messageType")
	}

	message := map[string]interface{}{
		"type": "message",
		"data": data,
	}

	messageBytes, err := json.Marshal(message)
	if err != nil {
		c.logger.Errorf("Failed to marshal message: %v", err)
		return nil, err
	}

	c.logger.Debugf("Outgoing message: %v", string(messageBytes))
	err = c.sendMessage(messageBytes)
	if err != nil {
		c.logger.Errorf("Failed to send message: %v", err)
		return nil, err
	}

	return &WaitFor{
		client: c,
		id:     options.ID,
	}, nil
}

// SubscribeRemoteTopic subscribes to a remote topic to receive messages.
func (c *Client) SubscribeRemoteTopic(topic string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ws == nil {
		c.logger.Errorf("Attempted to subscribe to %s without an active WebSocket connection.", topic)
		return fmt.Errorf("WebSocket connection is not established")
	}
	// Check if already subscribed
	if _, exists := c.subscribedTopics[topic]; exists {
		c.logger.Warnf("Already subscribed to topic: %s", topic)
		return nil
	}

	c.subscribedTopics[topic] = struct{}{}

	message := map[string]interface{}{
		"type": "subscribe",
		"data": map[string]interface{}{
			"topic": topic,
		},
	}
	messageBytes, err := json.Marshal(message)
	if err != nil {
		c.logger.Errorf("Failed to marshal subscribe message: %v", err)
		return err
	}

	c.logger.Infof("Subscribing to topic: %s", topic)
	err = c.sendMessage(messageBytes)
	if err != nil {
		c.logger.Errorf("Failed to send subscribe message: %v", err)
		return err
	}

	return nil
}

// UnsubscribeRemoteTopic unsubscribes from a previously subscribed topic.
func (c *Client) UnsubscribeRemoteTopic(topic string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ws == nil {
		c.logger.Errorf("Attempted to unsubscribe from %s without an active WebSocket connection.", topic)
		return fmt.Errorf("WebSocket connection is not established")
	}

	if _, exists := c.subscribedTopics[topic]; !exists {
		c.logger.Warnf("Not subscribed to topic: %s", topic)
		return nil
	}

	delete(c.subscribedTopics, topic)

	message := map[string]interface{}{
		"type": "unsubscribe",
		"data": map[string]interface{}{
			"topic": topic,
		},
	}
	messageBytes, err := json.Marshal(message)
	if err != nil {
		c.logger.Errorf("Failed to marshal unsubscribe message: %v", err)
		return err
	}

	c.logger.Infof("Unsubscribing from topic: %s", topic)
	err = c.sendMessage(messageBytes)
	if err != nil {
		c.logger.Errorf("Failed to send unsubscribe message: %v", err)
		return err
	}

	return nil
}

// sendMessage queues a message to be sent to the WebSocket connection.
func (c *Client) sendMessage(message []byte) error {
	select {
	case c.writeChan <- message:
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timeout sending message")
	}
}

// onAck handles acknowledgment messages received from the server.
func (c *Client) onAck(args ...interface{}) {
	if len(args) < 1 {
		c.logger.Errorf("onAck called with insufficient arguments")
		return
	}
	message, ok := args[0].(IncomingMessage)
	if !ok {
		c.logger.Errorf("onAck received invalid message format")
		return
	}
	data, ok := message["data"].(map[string]interface{})
	if !ok {
		c.logger.Errorf("onAck message data is invalid")
		return
	}
	ackID, ok := data["data"]
	if !ok {
		c.logger.Errorf("onAck data missing 'data' field")
		return
	}
	c.logger.Debugf("Received ack: %v", data)
	c.eventEmitter.Emit(fmt.Sprintf("ack.%v", ackID), data)
}

// onResponse handles response messages received from other subscribers or backend services.
func (c *Client) onResponse(args ...interface{}) {
	if len(args) < 1 {
		c.logger.Errorf("onResponse called with insufficient arguments")
		return
	}
	message, ok := args[0].(IncomingMessage)
	if !ok {
		c.logger.Errorf("onResponse received invalid message format")
		return
	}
	topic, _ := message["topic"].(string)
	if strings.HasPrefix(topic, "priv/") {
		c.logger.Debugf("Received response for topic %s: %v", topic, message["data"])
		data, ok := message["data"].(map[string]interface{})
		if !ok {
			c.logger.Errorf("onResponse message data is invalid")
			return
		}
		payload, ok := data["payload"].(map[string]interface{})
		if !ok {
			c.logger.Errorf("onResponse payload is invalid")
			return
		}
		id := payload["id"]
		c.eventEmitter.Emit(fmt.Sprintf("response.%v", id), payload)
	}
}

// onWelcome handles 'welcome' messages to indicate that the session has started.
func (c *Client) onWelcome(args ...interface{}) {
	if len(args) < 1 {
		c.logger.Errorf("onWelcome called with insufficient arguments")
		return
	}
	message, _ := args[0].(IncomingMessage)
	data, _ := message["data"].(map[string]interface{})
	connection, _ := data["connection"].(map[string]interface{})

	c.logger.Infof("Session started, connection details: %v", connection)
	c.eventEmitter.Emit("session.started", ConnectionInfo(connection))
}

// onMessage handles incoming WebSocket messages.
func (c *Client) onMessage(message []byte) {
	var messageData map[string]interface{}
	err := json.Unmarshal(message, &messageData)
	if err != nil {
		c.handleError(fmt.Errorf("failed to unmarshal message: %v", err))
		return
	}
	topic, ok := messageData["topic"].(string)
	if !ok {
		c.handleError(fmt.Errorf("message missing 'topic' field: %v", messageData))
		return
	}
	messageType, ok := messageData["messageType"].(string)
	if !ok {
		c.handleError(fmt.Errorf("message missing 'messageType' field: %v", messageData))
		return
	}
	data := messageData["data"]
	messageEvent := IncomingMessage{
		"topic":       topic,
		"messageType": messageType,
		"data":        data,
		"compression": false,
	}
	c.logger.Debugf("Incoming message: %v", string(message))
	if messageType != "" {
		c.eventEmitter.Emit(fmt.Sprintf("%s.%s", topic, messageType), messageEvent, c.reply(messageEvent))
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.closeWebSocket()
	}()
	for {
		select {
		case message, ok := <-c.writeChan:
			if !ok {
				// The write channel was closed.
				return
			}
			c.mu.Lock()
			ws := c.ws
			c.mu.Unlock()
			if ws == nil {
				return
			}
			_ = ws.SetWriteDeadline(time.Now().Add(writeWait))
			err := ws.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				c.handleError(err)
				return
			}
		case <-ticker.C:
			c.mu.Lock()
			ws := c.ws
			c.mu.Unlock()
			if ws == nil {
				return
			}
			_ = ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err := ws.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.handleError(err)
				return
			}
		case <-c.done:
			ticker.Stop()
			return
		}
	}
}

func (c *Client) readPump() {
	defer func() {
		c.closeWebSocket()
		c.handleClose()
	}()
	c.ws.SetReadLimit(maxMessageSize)
	_ = c.ws.SetReadDeadline(time.Now().Add(pongWait))
	c.ws.SetPongHandler(func(string) error {
		_ = c.ws.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		select {
		case <-c.done:
			// Client is shutting down.
			return
		default:
			// Continue reading messages
		}

		_, message, err := c.ws.ReadMessage()
		if err != nil {
			select {
			case <-c.done:
				return
			default:
				c.handleError(err)
				return
			}
		}
		if message != nil {
			c.onMessage(message)
		} else {
			continue
		}

	}
}

func (c *Client) connectLoop() {
	c.mu.Lock()
	if c.isConnecting {
		c.mu.Unlock()
		return
	}
	c.isConnecting = true
	c.mu.Unlock()

	backoff := 1 * time.Second
	maxBackoff := 60 * time.Second

	for {
		c.logger.Debugf("connectLoop tick...")
		select {
		case <-c.done:
			c.logger.Debugf("Connect loop exiting due to client disconnection")
			return
		default:
			// Continue with connection attempts
		}

		wsURL, err := c.config.WebSocketOptions.URLProvider()
		if err != nil {
			c.logger.Warnf("Error obtaining WebSocket URL: %v", err)
			time.Sleep(backoff)
			continue
		}

		u, err := url.Parse(wsURL)
		if err != nil {
			c.logger.Warnf("Invalid WebSocket URL: %v", err)
			time.Sleep(backoff)
			continue
		}

		// Create a net.Dialer with timeouts
		netDialer := &net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 5 * time.Second,
		}

		// Create a websocket.Dialer that uses net.Dialer
		dialer := websocket.Dialer{
			NetDialContext: netDialer.DialContext,
		}

		// Attempt to connect with a context that can be canceled
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			select {
			case <-c.done:
				cancel()
			case <-ctx.Done():
				// Do nothing
			}
		}()

		conn, _, err := dialer.DialContext(ctx, u.String(), nil)
		cancel() // Ensure the context is canceled to free resources

		if err != nil {
			select {
			case <-c.done:
				c.logger.Debugf("Dial canceled due to client disconnection")
				return
			default:
				// Continue
			}
			c.handleError(err)
			c.logger.Warnf("Retrying connection in %v after failure: %v", backoff, err)
			time.Sleep(backoff)
			if backoff < maxBackoff {
				backoff *= 2
			} else {
				backoff = maxBackoff
			}
			continue
		}

		c.mu.Lock()
		c.ws = conn
		c.closeOnce = new(sync.Once) // Reset closeOnce for the new connection
		c.logger.Infof("Connected to WebSocket URL: %.70s", wsURL)
		c.isConnecting = false
		c.mu.Unlock()

		// Start writePump and readPump
		go c.writePump()
		go c.readPump()
		return
	}
}

// handleError handles WebSocket errors by logging and emitting an 'error' event.
func (c *Client) handleError(err error) {
	c.logger.Errorf("WebSocket error: %v", err)
	c.eventEmitter.Emit("error", err)
}

func (c *Client) handleClose() {
	c.mu.Lock()
	c.ws = nil
	c.mu.Unlock()

	select {
	case <-c.done:
		return
	default:
		// Continue
	}

	c.logger.Warnf("WebSocket closed unexpectedly, attempting to reconnect.")
	c.eventEmitter.Emit("close")
	go c.connectLoop()
}

// reply creates a reply function for the given client and message.
func (c *Client) reply(message map[string]interface{}) ReplyFunc {
	return func(data interface{}, status string, opts ...ReplyOption) error {
		if status == "" {
			status = "ok"
		}

		dataMap, ok := message["data"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("data field is missing or not a map in the message")
		}

		clientMap, ok := dataMap["client"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("client field is missing or not a map in the message data")
		}

		connectionID, ok := clientMap["connectionId"]
		if !ok {
			return fmt.Errorf("connectionId is missing in the client data")
		}

		originalID := dataMap["id"]

		// Set default options
		options := ReplyOptions{
			Compress: false,
		}

		// Apply provided options
		for _, opt := range opts {
			opt(&options)
		}

		payload := map[string]interface{}{
			"data":   data,
			"status": status,
			"id":     originalID,
		}

		_, err := c.Publish(fmt.Sprintf("priv/%v", connectionID), payload,
			WithPublishMessageType("response"),
			WithPublishCompress(options.Compress),
		)
		return err
	}
}

// On registers a listener for a specific event.
func (c *Client) On(event string, listener ListenerFunc) int {
	return c.eventEmitter.On(event, listener)
}

// Off removes a listener for a specific event using the listener ID.
func (c *Client) Off(event string, id int) {
	c.eventEmitter.Off(event, id)
}

// Once registers a listener for a specific event that will be called at most once.
func (c *Client) Once(event string, listener ListenerFunc) int {
	return c.eventEmitter.Once(event, listener)
}
