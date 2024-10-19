// listener_test.go

package realtime_pubsub

import (
	"reflect"
	"testing"
)

func TestIncomingMessage_Topic(t *testing.T) {
	msg := IncomingMessage{
		"topic": "test/topic",
	}

	expected := "test/topic"
	actual := msg.Topic()

	if actual != expected {
		t.Errorf("Expected topic '%s', got '%s'", expected, actual)
	}
}

func TestIncomingMessage_Data(t *testing.T) {
	data := map[string]interface{}{
		"key": "value",
	}

	msg := IncomingMessage{
		"data": data,
	}

	actual := msg.Data()
	if !reflect.DeepEqual(actual, data) {
		t.Errorf("Expected data '%v', got '%v'", data, actual)
	}
}

func TestIncomingMessage_MessageType(t *testing.T) {
	msg := IncomingMessage{
		"messageType": "testType",
	}

	expected := "testType"
	actual := msg.MessageType()

	if actual != expected {
		t.Errorf("Expected messageType '%s', got '%s'", expected, actual)
	}
}

func TestIncomingMessage_Compression(t *testing.T) {
	msg := IncomingMessage{
		"compression": true,
	}

	expected := true
	actual := msg.Compression()

	if actual != expected {
		t.Errorf("Expected compression '%v', got '%v'", expected, actual)
	}
}

func TestIncomingMessage_DataAsPresenceMessage(t *testing.T) {
	data := map[string]interface{}{
		"payload": map[string]interface{}{
			"status": "connected",
		},
		"client": map[string]interface{}{
			"connectionId": "12345",
			"permissions":  []string{"read", "write"},
		},
	}

	msg := IncomingMessage{
		"data": data,
	}

	presenceMsg := msg.DataAsPresenceMessage()

	status := presenceMsg.Status()
	if status != "connected" {
		t.Errorf("Expected status 'connected', got '%s'", status)
	}

	connectionId := presenceMsg.ConnectionId()
	if connectionId != "12345" {
		t.Errorf("Expected connectionId '12345', got '%s'", connectionId)
	}

	expectedPermissions := []string{"read", "write"}
	permissions := presenceMsg.Permissions()
	if !reflect.DeepEqual(permissions, expectedPermissions) {
		t.Errorf("Expected permissions '%v', got '%v'", expectedPermissions, permissions)
	}
}

func TestResponseMessage_Id(t *testing.T) {
	msg := ResponseMessage{
		"id": "response-123",
	}

	expected := "response-123"
	actual := msg.Id()

	if actual != expected {
		t.Errorf("Expected id '%s', got '%s'", expected, actual)
	}
}

func TestResponseMessage_Data(t *testing.T) {
	data := map[string]interface{}{
		"result": "success",
	}

	msg := ResponseMessage{
		"data": data,
	}

	actual := msg.Data()
	if !reflect.DeepEqual(actual, data) {
		t.Errorf("Expected data '%v', got '%v'", data, actual)
	}
}

func TestResponseMessage_DataAsMap(t *testing.T) {
	data := map[string]interface{}{
		"result": "success",
	}

	msg := ResponseMessage{
		"data": data,
	}

	actual := msg.DataAsMap()
	if !reflect.DeepEqual(actual, data) {
		t.Errorf("Expected data as map '%v', got '%v'", data, actual)
	}
}

func TestResponseMessage_Status(t *testing.T) {
	msg := ResponseMessage{
		"status": "ok",
	}

	expected := "ok"
	actual := msg.Status()

	if actual != expected {
		t.Errorf("Expected status '%s', got '%s'", expected, actual)
	}
}

func TestConnectionInfo_AppId(t *testing.T) {
	connInfo := ConnectionInfo{
		"appId": "app-123",
	}

	expected := "app-123"
	actual := connInfo.AppId()

	if actual != expected {
		t.Errorf("Expected appId '%s', got '%s'", expected, actual)
	}
}

func TestConnectionInfo_ConnectionId(t *testing.T) {
	connInfo := ConnectionInfo{
		"id": "conn-456",
	}

	expected := "conn-456"
	actual := connInfo.ConnectionId()

	if actual != expected {
		t.Errorf("Expected connectionId '%s', got '%s'", expected, actual)
	}
}

func TestConnectionInfo_RemoteAddress(t *testing.T) {
	connInfo := ConnectionInfo{
		"remoteAddress": "192.168.1.1",
	}

	expected := "192.168.1.1"
	actual := connInfo.RemoteAddress()

	if actual != expected {
		t.Errorf("Expected remoteAddress '%s', got '%s'", expected, actual)
	}
}

func TestPresenceMessage_Status(t *testing.T) {
	data := map[string]interface{}{
		"payload": map[string]interface{}{
			"status": "connected",
		},
	}

	msg := IncomingMessage{
		"data": data,
	}

	presenceMsg := NewPresenceMessage(msg)

	expected := "connected"
	actual := presenceMsg.Status()

	if actual != expected {
		t.Errorf("Expected status '%s', got '%s'", expected, actual)
	}
}

func TestPresenceMessage_ConnectionId(t *testing.T) {
	data := map[string]interface{}{
		"client": map[string]interface{}{
			"connectionId": "conn-789",
		},
	}

	msg := IncomingMessage{
		"data": data,
	}

	presenceMsg := NewPresenceMessage(msg)

	expected := "conn-789"
	actual := presenceMsg.ConnectionId()

	if actual != expected {
		t.Errorf("Expected connectionId '%s', got '%s'", expected, actual)
	}
}

func TestPresenceMessage_Permissions(t *testing.T) {
	data := map[string]interface{}{
		"client": map[string]interface{}{
			"permissions": []string{"read", "write"},
		},
	}

	msg := IncomingMessage{
		"data": data,
	}

	presenceMsg := NewPresenceMessage(msg)

	expected := []string{"read", "write"}
	actual := presenceMsg.Permissions()

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Expected permissions '%v', got '%v'", expected, actual)
	}
}

func TestPresenceMessage_InvalidTypeAssertions(t *testing.T) {
	// Test with incorrect types to ensure type assertions are handled safely
	data := map[string]interface{}{
		"payload": "invalid_payload_type",
		"client":  "invalid_client_type",
	}

	msg := IncomingMessage{
		"data": data,
	}

	presenceMsg := NewPresenceMessage(msg)

	// Expect empty string due to failed type assertion
	expectedStatus := ""
	actualStatus := presenceMsg.Status()
	if actualStatus != expectedStatus {
		t.Errorf("Expected status '%s', got '%s'", expectedStatus, actualStatus)
	}

	// Expect empty string due to failed type assertion
	expectedConnectionId := ""
	actualConnectionId := presenceMsg.ConnectionId()
	if actualConnectionId != expectedConnectionId {
		t.Errorf("Expected connectionId '%s', got '%s'", expectedConnectionId, actualConnectionId)
	}

	// Expect nil slice due to failed type assertion
	expectedPermissions := []string(nil)
	actualPermissions := presenceMsg.Permissions()
	if actualPermissions != nil {
		t.Errorf("Expected permissions '%v', got '%v'", expectedPermissions, actualPermissions)
	}
}
