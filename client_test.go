package realtime_pubsub

import (
	"encoding/json"
	"github.com/sirupsen/logrus"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// closeConnection safely closes a WebSocket connection.
func closeConnection(conn *websocket.Conn) {
	_ = conn.Close()
}

// disconnectClient safely disconnects the client.
func disconnectClient(c *Client) {
	_ = c.Disconnect()
}

// mockWebSocketServer creates a mock WebSocket server for testing purposes.
// It accepts a testing object and a handler function that manages the WebSocket connection.
func mockWebSocketServer(t *testing.T, handler func(*websocket.Conn)) *httptest.Server {
	upgrader := websocket.Upgrader{}

	// Create an HTTP test server with a handler that upgrades connections to WebSockets.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Upgrade initial HTTP request to a WebSocket connection.
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("Failed to upgrade WebSocket connection: %v", err)
			return
		}

		// Set read limits and handlers for control messages.
		conn.SetReadLimit(maxMessageSize)
		_ = conn.SetReadDeadline(time.Now().Add(pongWait))
		conn.SetPongHandler(func(string) error {
			_ = conn.SetReadDeadline(time.Now().Add(pongWait))
			return nil
		})

		// Start a goroutine to continuously read messages to keep the connection alive.
		go func() {
			defer closeConnection(conn)
			for {
				_, _, err := conn.ReadMessage()
				if err != nil {
					break
				}
			}
		}()

		// Invoke the provided handler function to manage the connection.
		handler(conn)
	}))

	return server
}

// TestClientSend verifies the client's ability to send messages directly to the server.
func TestClientSend(t *testing.T) {
	var (
		receivedMessage []byte
		wg              sync.WaitGroup
	)
	wg.Add(1)

	// Create a mock server that reads the client's message.
	server := mockWebSocketServer(t, func(conn *websocket.Conn) {
		defer closeConnection(conn)
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				break
			}
			// Capture the first message and signal completion.
			if receivedMessage == nil {
				receivedMessage = message
				wg.Done()
				return
			}
		}
	})
	defer server.Close()

	// Prepare the WebSocket URL for the client.
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Configure the client without a custom logger.
	config := Config{
		WebSocketOptions: WebSocketOptions{
			URLProvider: func() (string, error) {
				return wsURL, nil
			},
		},
	}

	// Create and connect the client.
	client := NewClient(config)
	client.Connect()
	defer disconnectClient(client)

	// Allow time for the connection to establish.
	time.Sleep(100 * time.Millisecond)

	// Define the payload to send.
	payload := map[string]interface{}{
		"command": "ping",
	}

	// Send the message to the server.
	_, err := client.Send(payload)
	if err != nil {
		t.Errorf("Failed to send message: %v", err)
	}

	// Wait for the server to receive the message.
	wg.Wait()

	// Verify the received message content.
	var messageData map[string]interface{}
	err = json.Unmarshal(receivedMessage, &messageData)
	if err != nil {
		t.Errorf("Failed to unmarshal received message: %v", err)
	}

	// Validate the message type.
	if messageData["type"] != "message" {
		t.Errorf("Expected message type 'message', got '%v'", messageData["type"])
	}

	// Validate the payload content.
	data, ok := messageData["data"].(map[string]interface{})
	if !ok {
		t.Errorf("Invalid data format in received message")
	}

	payloadReceived, ok := data["payload"].(map[string]interface{})
	if !ok {
		t.Errorf("Invalid payload format in received message")
	}

	if payloadReceived["command"] != "ping" {
		t.Errorf("Expected command 'ping', got '%v'", payloadReceived["command"])
	}
}

// TestClientPublish verifies the client's ability to publish messages to a topic.
func TestClientPublish(t *testing.T) {
	var (
		receivedMessage []byte
		wg              sync.WaitGroup
	)
	wg.Add(1)

	// Create a mock server that reads the client's publish message.
	server := mockWebSocketServer(t, func(conn *websocket.Conn) {
		defer closeConnection(conn)
		// Read the first message from the client.
		_, message, err := conn.ReadMessage()
		if err != nil {
			t.Errorf("Server failed to read message: %v", err)
		}
		receivedMessage = message
		wg.Done()
	})
	defer server.Close()

	// Prepare the WebSocket URL for the client.
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Configure the client with a logger for debugging.
	config := Config{
		WebSocketOptions: WebSocketOptions{
			URLProvider: func() (string, error) {
				return wsURL, nil
			},
		},
		Logger: logrus.New(),
	}

	// Create and connect the client.
	client := NewClient(config)
	client.Connect()
	defer disconnectClient(client)

	// Allow time for the connection to establish.
	time.Sleep(100 * time.Millisecond)

	// Define the payload to publish.
	payload := map[string]interface{}{
		"content": "Hello, World!",
	}

	// Publish the message to the topic.
	_, err := client.Publish("test.topic", payload)
	if err != nil {
		t.Errorf("Failed to publish message: %v", err)
	}

	// Wait for the server to receive the message.
	wg.Wait()

	// Verify the received message content.
	var messageData map[string]interface{}
	err = json.Unmarshal(receivedMessage, &messageData)
	if err != nil {
		t.Errorf("Failed to unmarshal received message: %v", err)
	}

	// Validate the message type.
	if messageData["type"] != "publish" {
		t.Errorf("Expected message type 'publish', got '%v'", messageData["type"])
	}

	// Validate the topic and payload.
	data, ok := messageData["data"].(map[string]interface{})
	if !ok {
		t.Errorf("Invalid data format in received message")
	}

	if data["topic"] != "test.topic" {
		t.Errorf("Expected topic 'test.topic', got '%v'", data["topic"])
	}

	if data["payload"] == nil {
		t.Errorf("Expected payload, but got nil")
	}
}

// TestClientSubscribeRemoteTopic tests the client's ability to subscribe to a remote topic.
func TestClientSubscribeRemoteTopic(t *testing.T) {
	var (
		receivedMessage []byte
		wg              sync.WaitGroup
	)
	wg.Add(1)

	// Create a mock server that reads the client's subscribe message.
	server := mockWebSocketServer(t, func(conn *websocket.Conn) {
		defer closeConnection(conn)
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				break
			}
			// Capture the subscribe message and signal completion.
			if receivedMessage == nil {
				receivedMessage = message
				wg.Done()
				return
			}
		}
	})
	defer server.Close()

	// Prepare the WebSocket URL for the client.
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Configure the client.
	config := Config{
		WebSocketOptions: WebSocketOptions{
			URLProvider: func() (string, error) {
				return wsURL, nil
			},
		},
	}

	// Create and connect the client.
	client := NewClient(config)
	client.Connect()
	defer disconnectClient(client)

	// Allow time for the connection to establish.
	time.Sleep(100 * time.Millisecond)

	// Subscribe to the topic.
	err := client.SubscribeRemoteTopic("test.topic")
	if err != nil {
		t.Errorf("Failed to subscribe to topic: %v", err)
	}

	// Wait for the server to receive the subscribe message.
	wg.Wait()

	// Verify the received message content.
	var messageData map[string]interface{}
	err = json.Unmarshal(receivedMessage, &messageData)
	if err != nil {
		t.Errorf("Failed to unmarshal received message: %v", err)
	}

	// Validate the message type.
	if messageData["type"] != "subscribe" {
		t.Errorf("Expected message type 'subscribe', got '%v'", messageData["type"])
	}

	// Validate the topic.
	data, ok := messageData["data"].(map[string]interface{})
	if !ok {
		t.Errorf("Invalid data format in received message")
	}

	if data["topic"] != "test.topic" {
		t.Errorf("Expected topic 'test.topic', got '%v'", data["topic"])
	}
}

// TestClientUnsubscribeRemoteTopic tests the client's ability to unsubscribe from a remote topic.
func TestClientUnsubscribeRemoteTopic(t *testing.T) {
	var (
		receivedMessages [][]byte
		wg               sync.WaitGroup
	)
	// We expect to receive both subscribe and unsubscribe messages.
	wg.Add(2)

	// Create a mock server that reads the client's messages.
	server := mockWebSocketServer(t, func(conn *websocket.Conn) {
		defer closeConnection(conn)
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				break
			}
			receivedMessages = append(receivedMessages, message)
			wg.Done()
			if len(receivedMessages) >= 2 {
				// Received both messages; exit the loop.
				break
			}
		}
	})
	defer server.Close()

	// Prepare the WebSocket URL for the client.
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Configure the client.
	config := Config{
		WebSocketOptions: WebSocketOptions{
			URLProvider: func() (string, error) {
				return wsURL, nil
			},
		},
	}

	// Create and connect the client.
	client := NewClient(config)
	client.Connect()
	defer disconnectClient(client)

	// Allow time for the connection to establish.
	time.Sleep(100 * time.Millisecond)

	// Subscribe to the topic.
	err := client.SubscribeRemoteTopic("test.topic")
	if err != nil {
		t.Errorf("Failed to subscribe to topic: %v", err)
	}

	// Unsubscribe from the topic.
	err = client.UnsubscribeRemoteTopic("test.topic")
	if err != nil {
		t.Errorf("Failed to unsubscribe from topic: %v", err)
	}

	// Wait for the server to receive both messages.
	wg.Wait()

	// Verify the unsubscribe message content.
	var unsubscribeMessage map[string]interface{}
	err = json.Unmarshal(receivedMessages[1], &unsubscribeMessage)
	if err != nil {
		t.Errorf("Failed to unmarshal unsubscribe message: %v", err)
	}

	// Validate the message type.
	if unsubscribeMessage["type"] != "unsubscribe" {
		t.Errorf("Expected message type 'unsubscribe', got '%v'", unsubscribeMessage["type"])
	}

	// Validate the topic.
	data, ok := unsubscribeMessage["data"].(map[string]interface{})
	if !ok {
		t.Errorf("Invalid data format in unsubscribe message")
	}

	if data["topic"] != "test.topic" {
		t.Errorf("Expected topic 'test.topic', got '%v'", data["topic"])
	}
}

// TestClientOnMessage tests the client's ability to handle incoming messages from the server.
func TestClientOnMessage(t *testing.T) {
	var (
		messageReceived bool
		wg              sync.WaitGroup
	)
	wg.Add(1)

	// Mock message to be sent from the server to the client.
	mockMessage := map[string]interface{}{
		"topic":       "test.topic",
		"messageType": "update",
		"data": map[string]interface{}{
			"content": "Hello, Client!",
		},
	}
	mockMessageBytes, _ := json.Marshal(mockMessage)

	// Create a mock server that sends a message to the client.
	server := mockWebSocketServer(t, func(conn *websocket.Conn) {
		defer closeConnection(conn)
		// Read messages in a goroutine to handle Ping/Pong frames.
		go func() {
			for {
				_, _, err := conn.ReadMessage()
				if err != nil {
					break
				}
			}
		}()

		// Send the mock message to the client.
		time.Sleep(100 * time.Millisecond) // Ensure client is ready.
		err := conn.WriteMessage(websocket.TextMessage, mockMessageBytes)
		if err != nil {
			t.Errorf("Server write error: %v", err)
		}
	})
	defer server.Close()

	// Prepare the WebSocket URL for the client.
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Configure the client.
	config := Config{
		WebSocketOptions: WebSocketOptions{
			URLProvider: func() (string, error) {
				return wsURL, nil
			},
		},
	}

	// Create and connect the client.
	client := NewClient(config)
	client.Connect()
	defer disconnectClient(client)

	// Register a listener for the specific message type.
	client.On("test.topic.update", func(args ...interface{}) {
		defer wg.Done()
		messageReceived = true
		if len(args) < 1 {
			t.Errorf("Expected at least 1 argument, got %d", len(args))
			return
		}
		message, ok := args[0].(IncomingMessage)
		if !ok {
			t.Errorf("Invalid message format")
			return
		}
		data, ok := message.Data().(map[string]interface{})
		if !ok {
			t.Errorf("Invalid data format in message")
			return
		}
		if data["content"] != "Hello, Client!" {
			t.Errorf("Expected content 'Hello, Client!', got '%v'", data["content"])
		}
	})

	// Wait for the message to be received.
	wg.Wait()

	if !messageReceived {
		t.Errorf("Expected to receive a message, but did not")
	}
}

// TestClientHandleError verifies that the client handles errors correctly.
func TestClientHandleError(t *testing.T) {
	var (
		errorReceived error
		wg            sync.WaitGroup
	)
	wg.Add(1)

	// Create a mock server that closes the connection immediately to simulate an error.
	server := mockWebSocketServer(t, func(conn *websocket.Conn) {
		closeConnection(conn)
	})
	defer server.Close()

	// Prepare the WebSocket URL for the client.
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Configure the client.
	config := Config{
		WebSocketOptions: WebSocketOptions{
			URLProvider: func() (string, error) {
				return wsURL, nil
			},
		},
	}

	// Create the client.
	client := NewClient(config)

	// Register an error listener.
	client.On("error", func(args ...interface{}) {
		defer wg.Done()
		if len(args) < 1 {
			t.Errorf("Expected at least 1 argument, got %d", len(args))
			return
		}
		err, ok := args[0].(error)
		if !ok {
			t.Errorf("Expected an error, got %v", args[0])
			return
		}
		errorReceived = err
	})

	// Connect the client.
	client.Connect()
	defer disconnectClient(client)

	// Wait for the error to be received.
	waitCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitCh)
	}()

	select {
	case <-waitCh:
		// Error received.
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for error event")
	}

	if errorReceived == nil {
		t.Errorf("Expected an error, but got nil")
	}
}

// TestClientWaitFor tests the client's ability to wait for acknowledgments.
func TestClientWaitFor(t *testing.T) {
	// Create a mock server that sends an acknowledgment back to the client.
	server := mockWebSocketServer(t, func(conn *websocket.Conn) {
		defer closeConnection(conn)
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				break
			}

			// Unmarshal the message to get the ID.
			var msgData map[string]interface{}
			err = json.Unmarshal(message, &msgData)
			if err != nil {
				t.Errorf("Failed to unmarshal message: %v", err)
				continue
			}

			data, _ := msgData["data"].(map[string]interface{})
			id := data["id"]

			// Send an acknowledgment message back to the client.
			ackMessage := map[string]interface{}{
				"type":        "event",
				"topic":       "priv/acks",
				"messageType": "ack",
				"data": map[string]interface{}{
					"data": id,
				},
			}

			ackBytes, _ := json.Marshal(ackMessage)
			err = conn.WriteMessage(websocket.TextMessage, ackBytes)
			if err != nil {
				t.Errorf("Server write error: %v", err)
			}
			break // Exit after sending acknowledgment.
		}
	})
	defer server.Close()

	// Prepare the WebSocket URL for the client.
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Configure the client.
	config := Config{
		WebSocketOptions: WebSocketOptions{
			URLProvider: func() (string, error) {
				return wsURL, nil
			},
		},
	}

	// Create and connect the client.
	client := NewClient(config)
	client.Connect()
	defer disconnectClient(client)

	// Allow time for the connection to establish.
	time.Sleep(100 * time.Millisecond)

	// Define the payload to publish.
	payload := map[string]interface{}{
		"content": "Hello, World!",
	}

	// Publish the message and wait for an acknowledgment.
	waitFor, err := client.Publish("test.topic", payload)
	if err != nil {
		t.Errorf("Failed to publish message: %v", err)
		return
	}

	// Wait for the acknowledgment with a timeout.
	result, err := waitFor.WaitForAck(1 * time.Second)
	if err != nil {
		t.Errorf("Failed to receive acknowledgment: %v", err)
		return
	}

	// Verify the acknowledgment ID matches.
	args, ok := result.([]interface{})
	if !ok || len(args) == 0 {
		t.Errorf("Expected result to be non-empty []interface{}, got %v", result)
		return
	}

	ackData, ok := args[0].(map[string]interface{})
	if !ok {
		t.Errorf("Expected acknowledgment data to be map[string]interface{}, got %T", args[0])
		return
	}

	ackID := ackData["data"]
	if ackID != waitFor.id {
		t.Errorf("Expected acknowledgment ID '%s', got '%v'", waitFor.id, ackID)
	}
}

// TestClientDisconnect verifies the client's ability to disconnect gracefully.
func TestClientDisconnect(t *testing.T) {
	var (
		wg               sync.WaitGroup
		connectionClosed bool
	)
	wg.Add(1)

	// Create a mock server that detects when the connection is closed.
	server := mockWebSocketServer(t, func(conn *websocket.Conn) {
		defer closeConnection(conn)
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				connectionClosed = true
				wg.Done()
				break
			}
		}
	})
	defer server.Close()

	// Prepare the WebSocket URL for the client.
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Configure the client.
	config := Config{
		WebSocketOptions: WebSocketOptions{
			URLProvider: func() (string, error) {
				return wsURL, nil
			},
		},
	}

	// Create and connect the client.
	client := NewClient(config)
	client.Connect()

	// Allow time for the connection to establish.
	time.Sleep(100 * time.Millisecond)

	// Disconnect the client.
	err := client.Disconnect()
	if err != nil {
		t.Errorf("Failed to disconnect client: %v", err)
	}

	// Wait for the server to detect the disconnection.
	waitCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitCh)
	}()

	select {
	case <-waitCh:
		// Disconnection detected.
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for server to detect disconnection")
	}

	if !connectionClosed {
		t.Errorf("Expected server to detect connection closure, but it did not")
	}

	// Wait to ensure the client does not attempt to reconnect.
	time.Sleep(1 * time.Second)

	// Verify that the client's WebSocket connection is nil after disconnection.
	client.mu.Lock()
	if client.ws != nil {
		t.Errorf("Expected client.ws to be nil after disconnect, but it is not")
	}
	client.mu.Unlock()
}

// TestClientReconnect tests the client's ability to automatically reconnect after a disconnection.
func TestClientReconnect(t *testing.T) {
	// Set a timeout for the test to prevent it from hanging indefinitely.
	testTimeout := time.After(10 * time.Second)
	done := make(chan struct{})

	go func() {
		defer close(done)

		// WaitGroups to synchronize the test steps.
		var (
			initialConnectionEstablished sync.WaitGroup
			reconnectionEstablished      sync.WaitGroup
		)
		initialConnectionEstablished.Add(1)
		reconnectionEstablished.Add(1)

		upgrader := websocket.Upgrader{}
		var (
			connCount   int
			connCountMu sync.Mutex
		)

		// Start the mock WebSocket server.
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Upgrade the HTTP connection to a WebSocket.
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				t.Errorf("Failed to upgrade: %v", err)
				return
			}

			// Increment the connection count.
			connCountMu.Lock()
			connCount++
			currentConnCount := connCount
			connCountMu.Unlock()

			if currentConnCount == 1 {
				// Initial connection established.
				initialConnectionEstablished.Done()

				// Simulate server closing the connection after 500ms.
				go func() {
					time.Sleep(500 * time.Millisecond)
					closeConnection(conn)
				}()
			} else if currentConnCount == 2 {
				// Reconnection established.
				reconnectionEstablished.Done()
			}

			// Read messages to keep the connection open.
			for {
				_, _, err := conn.ReadMessage()
				if err != nil {
					return
				}
			}
		}))
		defer server.Close()

		// Prepare the WebSocket URL for the client.
		wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

		// Configure the client with a logger.
		config := Config{
			WebSocketOptions: WebSocketOptions{
				URLProvider: func() (string, error) {
					return wsURL, nil
				},
			},
			Logger: logrus.New(),
		}

		// Create and connect the client.
		client := NewClient(config)
		client.Connect()
		defer disconnectClient(client)

		// Wait for the client to establish the initial connection.
		initialConnectionEstablished.Wait()

		// Wait for the client to reconnect after disconnection.
		reconnectionEstablished.Wait()

		// Check that the client reconnected.
		connCountMu.Lock()
		if connCount < 2 {
			t.Errorf("Expected client to reconnect, but it did not")
		}
		connCountMu.Unlock()
	}()

	// Enforce the test timeout.
	select {
	case <-done:
		// Test completed successfully.
	case <-testTimeout:
		t.Fatal("Test timed out")
	}
}
