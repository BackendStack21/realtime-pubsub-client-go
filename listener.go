package realtime_pubsub

// ListenerFunc represents the function signature for event listeners.
type ListenerFunc func(args ...interface{})

// ReplyFunc defines the signature for reply functions used in event listeners.
type ReplyFunc func(interface{}, string, ...ReplyOption) error

// IncomingMessage represents a message received from the server.
type IncomingMessage map[string]interface{}

// Topic receiver function that extracts the "topic" from IncomingMessage.
func (m IncomingMessage) Topic() string {
	topic, _ := m["topic"].(string)
	return topic
}

// Data receiver function that extracts the "data" from IncomingMessage.
func (m IncomingMessage) Data() interface{} {
	data, _ := m["data"]
	return data
}

// MessageType receiver function that extracts the "messageType" from IncomingMessage.
func (m IncomingMessage) MessageType() string {
	messageType, _ := m["messageType"].(string)
	return messageType
}

// Compression receiver function that extracts the "compression" from IncomingMessage.
func (m IncomingMessage) Compression() bool {
	compression, _ := m["compression"].(bool)
	return compression
}

func (m IncomingMessage) DataAsPresenceMessage() PresenceMessage {
	return NewPresenceMessage(m)
}

// ResponseMessage represents a message sent by a client in response to an incoming message.
type ResponseMessage map[string]interface{}

// Id receiver function that extracts the "id" from ResponseMessage.
func (m ResponseMessage) Id() string {
	topic, _ := m["id"].(string)
	return topic
}

// Data receiver function that extracts the "data" from ResponseMessage.
func (m ResponseMessage) Data() interface{} {
	data, _ := m["data"]
	return data
}

func (m ResponseMessage) DataAsMap() map[string]interface{} {
	data, _ := m["data"].(map[string]interface{})
	return data
}

// Status receiver function that extracts the "status" from ResponseMessage.
func (m ResponseMessage) Status() string {
	status, _ := m["status"].(string)
	return status
}

// ConnectionInfo represents the connection information received from the server.
type ConnectionInfo map[string]interface{}

// AppId receiver function that extracts the "appId" from ConnectionInfo.
func (c ConnectionInfo) AppId() string {
	appId, _ := c["appId"].(string)
	return appId
}

// ConnectionId receiver function that extracts the "id" from ConnectionInfo.
func (c ConnectionInfo) ConnectionId() string {
	sessionId, _ := c["id"].(string)
	return sessionId
}

// RemoteAddress receiver function that extracts the "remoteAddress" from ConnectionInfo.
func (c ConnectionInfo) RemoteAddress() string {
	remoteAddress, _ := c["remoteAddress"].(string)
	return remoteAddress
}

// PresenceMessage represents a presence event received from the server.
type PresenceMessage struct {
	IncomingMessage
}

// NewPresenceMessage creates a PresenceMessage from an IncomingMessage.
func NewPresenceMessage(msg IncomingMessage) PresenceMessage {
	return PresenceMessage{msg}
}

// Status receiver function that extracts the "status" attribute from payload.
func (m PresenceMessage) Status() string {
	payload, _ := m.IncomingMessage.Data().(map[string]interface{})["payload"].(map[string]interface{})
	status, _ := payload["status"].(string) // Add type assertion check
	return status
}

// ConnectionId receiver function that extracts the "connectionId" attribute from client.
func (m PresenceMessage) ConnectionId() string {
	payload, _ := m.IncomingMessage.Data().(map[string]interface{})
	client, _ := payload["client"].(map[string]interface{})
	connectionId, _ := client["connectionId"].(string) // Add type assertion check
	return connectionId
}

// Permissions receiver function that extracts the "permissions" attribute from client.
func (m PresenceMessage) Permissions() []string {
	payload, _ := m.IncomingMessage.Data().(map[string]interface{})
	client, _ := payload["client"].(map[string]interface{})
	permissions, _ := client["permissions"].([]string) // Add type assertion check
	return permissions
}
