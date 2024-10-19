package realtime_pubsub

import (
	"encoding/hex"
	"io"
)

// getRandomID generates a random identifier string.
// Accepts an io.Reader to allow for testing.
func getRandomID(reader io.Reader) string {
	b := make([]byte, 8)
	_, _ = reader.Read(b)

	return hex.EncodeToString(b)
}
