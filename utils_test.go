package realtime_pubsub

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"testing"
)

// errorReader is an io.Reader that always returns an error.
type errorReader struct{}

func (e *errorReader) Read(_ []byte) (int, error) {
	return 0, errors.New("simulated read error")
}

// TestGetRandomIDLength tests that getRandomID returns a string of the correct length.
func TestGetRandomIDLength(t *testing.T) {
	id := getRandomID(rand.Reader)
	expectedLength := 16
	if len(id) != expectedLength {
		t.Errorf("Expected ID length to be %d, got %d", expectedLength, len(id))
	}
}

// TestGetRandomIDUniqueness tests that multiple calls to getRandomID return unique values.
func TestGetRandomIDUniqueness(t *testing.T) {
	ids := make(map[string]struct{})
	count := 1000 // Number of IDs to generate for the test

	for i := 0; i < count; i++ {
		id := getRandomID(rand.Reader)
		if _, exists := ids[id]; exists {
			t.Errorf("Duplicate ID generated: %s", id)
		}
		ids[id] = struct{}{}
	}
}

// TestGetRandomIDFallback tests the fallback behavior when the reader fails.
func TestGetRandomIDFallback(t *testing.T) {
	id := getRandomID(&errorReader{})

	// Check that the ID is a numeric string (UnixNano timestamp)
	if _, err := fmt.Sscanf(id, "%d", new(int64)); err != nil {
		t.Errorf("Expected fallback ID to be a numeric string, got '%s'", id)
	}
}

// TestGetRandomIDConcurrency tests that getRandomID behaves correctly under concurrent usage.
func TestGetRandomIDConcurrency(t *testing.T) {
	ids := make(chan string, 1000)
	concurrency := 50
	perGoroutine := 20

	// Function to generate IDs and send them to the channel
	generateIDs := func() {
		for i := 0; i < perGoroutine; i++ {
			id := getRandomID(rand.Reader)
			ids <- id
		}
	}

	// Start concurrent goroutines
	for i := 0; i < concurrency; i++ {
		go generateIDs()
	}

	// Collect generated IDs
	idMap := make(map[string]struct{})
	totalIDs := concurrency * perGoroutine
	for i := 0; i < totalIDs; i++ {
		id := <-ids
		if _, exists := idMap[id]; exists {
			t.Errorf("Duplicate ID generated in concurrent execution: %s", id)
		}
		idMap[id] = struct{}{}
	}
}

// TestGetRandomIDDeterministicReader tests getRandomID with a deterministic reader.
func TestGetRandomIDDeterministicReader(t *testing.T) {
	// Use a bytes.Reader with known content
	data := []byte("1234567890abcdef1234567890abcdef")
	reader := bytes.NewReader(data)

	id := getRandomID(reader)

	expectedID := hex.EncodeToString(data[:8])
	if id != expectedID {
		t.Errorf("Expected ID to be '%s', got '%s'", expectedID, id)
	}
}
