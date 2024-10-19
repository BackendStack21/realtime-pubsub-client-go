# Realtime Pub/Sub Client for Go

The `realtime-pubsub-client-go` is a Go client library for interacting with [Realtime Pub/Sub](https://realtime.21no.de)
applications. It enables developers to manage real-time WebSocket connections, handle subscriptions, and process
messages efficiently. The library provides a simple and flexible API to interact with realtime applications, supporting
features like publishing/sending messages, subscribing to topics, handling acknowledgements, and waiting for replies
with timeout support.

## Features

- **WebSocket Connection Management**: Seamlessly connect and disconnect from the Realtime Pub/Sub service with
  automatic reconnection support.
- **Topic Subscription**: Subscribe and unsubscribe to topics for receiving messages.
- **Topic Publishing**: Publish messages to specific topics with optional message types and compression.
- **Message Sending**: Send messages to backend applications with optional message types and compression.
- **Event Handling**: Handle incoming messages with custom event listeners.
- **Acknowledgements and Replies**: Wait for gateway acknowledgements or replies to messages with timeout support.
- **Error Handling**: Robust error handling and logging capabilities.
- **Concurrency Support**: Safe for use in concurrent environments with thread-safe operations.

## Installation

Install the `realtime-pubsub-client-go` library using `go get`:

```bash
go get github.com/backendstack21/realtime-pubsub-client-go
```

Or, if you're using Go modules, simply import the package in your project, and Go will handle the installation.

## Getting Started

This guide will help you set up and use the `realtime-pubsub-client-go` library in your Go project.

### Prerequisites

- Go 1.16 or later
- A Realtime Pub/Sub account and an application set up at [Realtime Pub/Sub](https://realtime.21no.de)
- Access token for authentication with sufficient permissions, see: [Subscribers](https://realtime.21no.de/documentation/#subscribers)
  and [Publishers](https://realtime.21no.de/documentation/#publishers)

### Working Examples

The following are working examples demonstrating how to use the `realtime-pubsub-client-go` library:

* [basic.go](main/basic.go) 
* [rpc/client](main/rpc/client.go)
* [rpc/server](main/rpc/server.go)

### Subscribing to Incoming Messages

You can handle messages for specific topics and message types by registering event listeners.
In this example we handle `text-message` messages under the `chat` topic:

```go
client.On("chat.text-message", func (args ...interface{}) {
  // Extract message and reply function
  message, _ := args[0].(realtime_pubsub.IncomingMessage)
  replyFunc, _ := args[1].(realtime_pubsub.ReplyFunc)
  
  log.Printf("Received message: %v", message)
  
  // Reply to the message
  if err := replyFunc("Got it!", "ok"); err != nil {
  log.Printf("Failed to reply to message: %v", err)
  }
})

```

#### Wildcard Subscriptions

Wildcard subscriptions are also supported, allowing you to handle multiple message types under a topic:

```go
client.On("chat.*", func (args ...interface{}) {
  // Extract message and reply function
  message, _ := args[0].(realtime_pubsub.IncomingMessage)
  replyFunc, _ := args[1].(realtime_pubsub.ReplyFunc)
  
  //...
})
```

### Publishing Messages

Publish messages to a topic using the `Publish` method:

```go
topic:= "chat"
payload := "Hello, World!"
messageType := "text-message"

waitFor, err := client.Publish(topic, payload,
  realtime_pubsub.WithPublishMessageType(messageType),
  realtime_pubsub.WithPublishCompress(true),
)
if err != nil {
  log.Fatalf("Failed to publish message: %v", err)
}

// Wait for acknowledgment
if _, err := waitFor.WaitForAck(100 * time.Millisecond); err != nil {
  log.Printf("Failed to receive acknowledgment: %v", err)
}

// Wait for a reply
reply, err := waitFor.WaitForReply(500 * time.Millisecond)
if err != nil {
  log.Printf("Failed to receive reply: %v", err)
} else {
  log.Printf("Received reply: %v", reply["data"])
}
```

### Sending Messages

Send messages to the server using the `Send` method:

```go
waitFor, err := client.Send(map[string]interface{}{
  "action": "create",
  "data": map[string]interface{}{
    "name": "John Doe",
  },
}, realtime_pubsub.WithSendMessageType("create-user"))
if err != nil {
  log.Fatalf("Failed to send message: %v", err)
}

// Wait for acknowledgment
if _, err := waitFor.WaitForAck(100 * time.Millisecond); err != nil {
  log.Printf("Failed to receive acknowledgment: %v", err)
}

// Wait for a reply
reply, err := waitFor.WaitForReply(500 * time.Millisecond)
if err != nil {
  log.Printf("Failed to receive reply: %v", err)
} else {
  log.Printf("Received reply: %v", reply.Data())
}
```

### Handling Replies

Set up event listeners to handle incoming replies, use the ReplyFunc param to send replies to messages:

```go
client.On("chat.text-message", func (args ...interface{}) {
  // Extract message and reply function
  message, _ := args[0].(realtime_pubsub.IncomingMessage)
  replyFunc, _ := args[1].(realtime_pubsub.ReplyFunc)
  
  log.Printf("Received message: %v", message)
  
  // Reply to the message
  replyPayload := "Got it!"
  replyStatus := "ok"
  if err := replyFunc(replyPayload, replyStatus); err != nil {
    log.Printf("Failed to reply to message: %v", err)
  }
})
```

### Waiting for Acknowledgements and Replies

- **Wait for Acknowledgement**: Use `WaitForAck` to wait for a gateway acknowledgement after publishing or sending a
  message.

  ```go
  if _, err := waitFor.WaitForAck(500 * time.Millisecond); err != nil {
    log.Printf("Failed to receive acknowledgment: %v", err)
  }
  ```

- **Wait for Reply**: Use `WaitForReply` to wait for a reply to your message with a specified timeout.

  ```go
  reply, err := waitFor.WaitForReply(500 * time.Millisecond)
  if err != nil {
    log.Printf("Failed to receive reply: %v", err)
  } else {
    log.Printf("Received reply: %v", reply.Data())
  }
  ```

### Error Handling

Handle errors and disconnections gracefully:

```go
client.On("error", func (args ...interface{}) {
  log.Printf("Received error: %v", args)
})

client.On("close", func (args ...interface{}) {
  log.Printf("Connection closed: %v", args)
})
```

## API Reference

### RealtimeClient

#### Constructor

```go
func NewClient(config Config) *Client
```

Creates a new `Client` instance.

- **config**: Configuration options for the client.

#### Methods

- **Connect()**

  ```go
  func (c *Client) Connect()
  ```

  Establishes a connection to the WebSocket server.

- **Disconnect() error**

  ```go
  func (c *Client) Disconnect() error
  ```

  Terminates the WebSocket connection gracefully.

- **On(event string, listener ListenerFunc)**

  ```go
  func (c *Client) On(event string, listener ListenerFunc)
  ```

  Registers a listener for a specific event. Supports wildcard patterns.

- **Off(event string, id int)**

  ```go
  func (c *Client) Off(event string, id int)
  ```

  Removes a listener for a specific event using the listener ID.

- **Once(event string, listener ListenerFunc) int**

  ```go
  func (c *Client) Once(event string, listener ListenerFunc) int
  ```

  Registers a listener that will be called at most once for the specified event.

- **SubscribeRemoteTopic(topic string) error**

  ```go
  func (c *Client) SubscribeRemoteTopic(topic string) error
  ```

  Subscribes the client to a remote topic to receive messages.

- **UnsubscribeRemoteTopic(topic string) error**

  ```go
  func (c *Client) UnsubscribeRemoteTopic(topic string) error
  ```

  Unsubscribes the client from a remote topic.

- **Publish(topic string, payload interface{}, opts ...PublishOption) (\*WaitFor, error)**

  ```go
  func (c *Client) Publish(topic string, payload interface{}, opts ...PublishOption) (*WaitFor, error)
  ```

  Publishes a message to a specified topic with optional configurations.

- **Send(payload interface{}, opts ...SendOption) (\*WaitFor, error)**

  ```go
  func (c *Client) Send(payload interface{}, opts ...SendOption) (*WaitFor, error)
  ```

  Sends a message to the server with optional configurations.

- **WaitFor(event string, timeout time.Duration) (interface{}, error)**

  ```go
  func (c *Client) WaitFor(event string, timeout time.Duration) (interface{}, error)
  ```

  Waits for a specific event to occur within the given timeout.

#### Events

- **'session.started'**

  Emitted when the session starts.

  ```go
  client.On("session.started", func(args ...interface{}) {
    info := args[0].(realtime_pubsub.ConnectionInfo)
    //...
  })
  ```

- **'error'**

  Emitted on WebSocket errors.

  ```go
  client.On("error", func(args ...interface{}) {
    // Handle error
  })
  ```

- **'close'**

  Emitted when the WebSocket connection closes.

  ```go
  client.On("close", func(args ...interface{}) {
    // Handle close event
  })
  ```

- **Custom Events**

  Handle custom events based on topic and message type.

  ```go
  client.On("topic1.action1", func(args ...interface{}) {
    // Handle specific message
  })
  ```

  **Wildcard Subscriptions**

  ```go
  client.On("topic1.*", func(args ...interface{}) {
    // Handle any message under topic1
  })
  ```

## Type Definitions

### IncomingMessage

Represents a message received from the server.

```go
type IncomingMessage map[string]interface{}
```

#### Methods

- **Topic() string**

  Extracts the "topic" from the message.

  ```go
  func (m IncomingMessage) Topic() string
  ```

- **Data() interface{}**

  Extracts the "data" from the message.

  ```go
  func (m IncomingMessage) Data() interface{}
  ```

- **MessageType() string**

  Extracts the "messageType" from the message.

  ```go
  func (m IncomingMessage) MessageType() string
  ```

- **Compression() bool**

  Extracts the "compression" flag from the message.

  ```go
  func (m IncomingMessage) Compression() bool
  ```
  
- **DataAsMap() map[string]interface{}**

  Extracts the "data" as a map from the message.

  ```go
  func (m IncomingMessage) DataAsMap() map[string]interface{}
  ```

- **DataAsPresenceMessage() PresenceMessage** 

  Extracts the "data" as a PresenceMessage from the message.

  ```go
  func (m IncomingMessage) DataAsPresenceMessage() PresenceMessage
  ```
  
### ResponseMessage

Represents a message sent by a client in response to an incoming message.

```go
type ResponseMessage map[string]interface{}
```

#### Methods

- **Id() string**

  Extracts the "id" from the response.

  ```go
  func (m ResponseMessage) Id() string
  ```

- **Data() interface{}**

  Extracts the "data" from the response.

  ```go
  func (m ResponseMessage) Data() interface{}
  ```

- **Status() string**

  Extracts the "status" from the response.

  ```go
  func (m ResponseMessage) Status() string
  ```
  
- **DataAsMap() map[string]interface{}**

  Extracts the "data" as a map from the response.

  ```go
  func (m ResponseMessage) DataAsMap() map[string]interface{}
  ```

### ReplyFunc

Defines the signature for reply functions used in event listeners.

```go
type ReplyFunc func (data interface{}, status string, opts ...ReplyOption) error
```

## Configuration Options

### Config

Configuration options for initializing the client.

```go
type Config struct {
  Logger           *logrus.Logger
  WebSocketOptions WebSocketOptions
}
```

- **Logger**: Optional. Pass a custom logger implementing the `logrus.Logger` interface for logging purposes. If not
  provided, the client uses the standard logger.
- **WebSocketOptions**: Configuration options for the WebSocket connection.

### WebSocketOptions

Options for configuring the WebSocket connection.

```go
type WebSocketOptions struct {
  URLProvider func () (string, error)
}
```

- **URLProvider**: A function that returns the WebSocket URL for connecting to the Realtime Pub/Sub server. It can
  include dynamic parameters like access tokens.

## API Components

### WaitFor

Represents a mechanism to wait for acknowledgements or replies to messages.

```go
type WaitFor struct {
  // Internal fields
}
```

#### Methods

- **WaitForAck(timeout time.Duration) (interface{}, error)**

  Waits for an acknowledgement of the published or sent message within the specified timeout.

  ```go
  func (w *WaitFor) WaitForAck(timeout time.Duration) (interface{}, error)
  ```

- **WaitForReply(timeout time.Duration) (map[string]interface{}, error)**

  Waits for a reply to the published or sent message within the specified timeout.

  ```go
  func (w *WaitFor) WaitForReply(timeout time.Duration) (map[string]interface{}, error)
  ```

## License

This library is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

---

For more detailed examples and advanced configurations, please refer to
the [documentation](https://realtime.21no.de/docs).

## Notes

- **Environment Setup**: Ensure that you have an account and an app set up
  with [Realtime Pub/Sub](https://realtime.21no.de).
- **Authentication**: Customize the `urlProvider` function to retrieve the access token for connecting to your realtime
  application.
- **Logging**: Use the `Logger` option to integrate with your application's logging system.
- **Error Handling**: Handle errors and disconnections gracefully to improve the robustness of your application.
- **Timeouts**: Handle timeouts when waiting for replies to avoid hanging operations.
- **Concurrency**: The client is designed to be safe for use in concurrent environments, with thread-safe operations on
  its internal data structures. However, keep in mind that listeners are executed sequentially by default within the
  event loop. If your application requires concurrent handling of events, you can spawn goroutines within your listeners
  or implement custom thread pools for parallel processing. Additionally, if your listeners modify shared state, ensure
  that you manage synchronization (e.g., using mutexes or channels) to avoid race conditions or data corruption.
- **Wildcard Subscriptions**: Utilize wildcard subscriptions (`*` and `**`) to handle multiple message types under a
  single topic efficiently.

---

Feel free to contribute to this project by submitting issues or pull requests
on [GitHub](https://github.com/BackendStack21/realtime-pubsub-client-go).
