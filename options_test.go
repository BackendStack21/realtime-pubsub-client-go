package realtime_pubsub

import (
	"testing"
)

// TestPublishOptions tests the PublishOptions functional options.
func TestPublishOptions(t *testing.T) {
	// Create a default PublishOptions instance
	opts := PublishOptions{
		ID:          "default-id",
		MessageType: "default-type",
		Compress:    false,
	}

	// Apply functional options
	options := []PublishOption{
		WithPublishID("custom-id"),
		WithPublishMessageType("custom-type"),
		WithPublishCompress(true),
	}

	for _, opt := range options {
		opt(&opts)
	}

	// Verify that the options have been set correctly
	if opts.ID != "custom-id" {
		t.Errorf("Expected ID to be 'custom-id', got '%s'", opts.ID)
	}

	if opts.MessageType != "custom-type" {
		t.Errorf("Expected MessageType to be 'custom-type', got '%s'", opts.MessageType)
	}

	if opts.Compress != true {
		t.Errorf("Expected Compress to be 'true', got '%v'", opts.Compress)
	}
}

// TestSendOptions tests the SendOptions functional options.
func TestSendOptions(t *testing.T) {
	// Create a default SendOptions instance
	opts := SendOptions{
		ID:          "default-id",
		MessageType: "default-type",
		Compress:    false,
	}

	// Apply functional options
	options := []SendOption{
		WithSendID("custom-id"),
		WithSendMessageType("custom-type"),
		WithSendCompress(true),
	}

	for _, opt := range options {
		opt(&opts)
	}

	// Verify that the options have been set correctly
	if opts.ID != "custom-id" {
		t.Errorf("Expected ID to be 'custom-id', got '%s'", opts.ID)
	}

	if opts.MessageType != "custom-type" {
		t.Errorf("Expected MessageType to be 'custom-type', got '%s'", opts.MessageType)
	}

	if opts.Compress != true {
		t.Errorf("Expected Compress to be 'true', got '%v'", opts.Compress)
	}
}

// TestReplyOptions tests the ReplyOptions functional options.
func TestReplyOptions(t *testing.T) {
	// Create a default ReplyOptions instance
	opts := ReplyOptions{
		Compress: false,
	}

	// Apply functional options
	options := []ReplyOption{
		WithReplyCompress(true),
	}

	for _, opt := range options {
		opt(&opts)
	}

	// Verify that the options have been set correctly
	if opts.Compress != true {
		t.Errorf("Expected Compress to be 'true', got '%v'", opts.Compress)
	}
}

// TestDefaultPublishOptions tests the default values of PublishOptions when no options are applied.
func TestDefaultPublishOptions(t *testing.T) {
	// Create a default PublishOptions instance
	opts := PublishOptions{
		ID:          "",
		MessageType: "",
		Compress:    false,
	}

	// No options applied

	// Verify default values
	if opts.ID != "" {
		t.Errorf("Expected default ID to be empty, got '%s'", opts.ID)
	}

	if opts.MessageType != "" {
		t.Errorf("Expected default MessageType to be empty, got '%s'", opts.MessageType)
	}

	if opts.Compress != false {
		t.Errorf("Expected default Compress to be 'false', got '%v'", opts.Compress)
	}
}

// TestDefaultSendOptions tests the default values of SendOptions when no options are applied.
func TestDefaultSendOptions(t *testing.T) {
	// Create a default SendOptions instance
	opts := SendOptions{
		ID:          "",
		MessageType: "",
		Compress:    false,
	}

	// No options applied

	// Verify default values
	if opts.ID != "" {
		t.Errorf("Expected default ID to be empty, got '%s'", opts.ID)
	}

	if opts.MessageType != "" {
		t.Errorf("Expected default MessageType to be empty, got '%s'", opts.MessageType)
	}

	if opts.Compress != false {
		t.Errorf("Expected default Compress to be 'false', got '%v'", opts.Compress)
	}
}

// TestDefaultReplyOptions tests the default values of ReplyOptions when no options are applied.
func TestDefaultReplyOptions(t *testing.T) {
	// Create a default ReplyOptions instance
	opts := ReplyOptions{
		Compress: false,
	}

	// No options applied

	// Verify default values
	if opts.Compress != false {
		t.Errorf("Expected default Compress to be 'false', got '%v'", opts.Compress)
	}
}

// TestPartialPublishOptions tests applying some, but not all, functional options to PublishOptions.
func TestPartialPublishOptions(t *testing.T) {
	// Create a default PublishOptions instance
	opts := PublishOptions{
		ID:          "default-id",
		MessageType: "default-type",
		Compress:    false,
	}

	// Apply some options
	options := []PublishOption{
		WithPublishID("custom-id"),
		// MessageType remains default
		WithPublishCompress(true),
	}

	for _, opt := range options {
		opt(&opts)
	}

	// Verify that the options have been set correctly
	if opts.ID != "custom-id" {
		t.Errorf("Expected ID to be 'custom-id', got '%s'", opts.ID)
	}

	if opts.MessageType != "default-type" {
		t.Errorf("Expected MessageType to be 'default-type', got '%s'", opts.MessageType)
	}

	if opts.Compress != true {
		t.Errorf("Expected Compress to be 'true', got '%v'", opts.Compress)
	}
}

// TestPartialSendOptions tests applying some, but not all, functional options to SendOptions.
func TestPartialSendOptions(t *testing.T) {
	// Create a default SendOptions instance
	opts := SendOptions{
		ID:          "default-id",
		MessageType: "default-type",
		Compress:    false,
	}

	// Apply some options
	options := []SendOption{
		WithSendMessageType("custom-type"),
		// ID remains default
		// Compress remains default
	}

	for _, opt := range options {
		opt(&opts)
	}

	// Verify that the options have been set correctly
	if opts.ID != "default-id" {
		t.Errorf("Expected ID to be 'default-id', got '%s'", opts.ID)
	}

	if opts.MessageType != "custom-type" {
		t.Errorf("Expected MessageType to be 'custom-type', got '%s'", opts.MessageType)
	}

	if opts.Compress != false {
		t.Errorf("Expected Compress to be 'false', got '%v'", opts.Compress)
	}
}
