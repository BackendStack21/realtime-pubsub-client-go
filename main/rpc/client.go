package main

import (
	"fmt"
	"os"
	"time"

	realtime_pubsub "github.com/backendstack21/realtime-pubsub-client-go"
	"github.com/sirupsen/logrus"

	"github.com/joho/godotenv"
)

func main() {
	done := make(chan bool)

	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetOutput(os.Stdout)
	logger.SetLevel(logrus.DebugLevel)

	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		logger.Errorf("Error loading .env file")
	}

	appID := os.Getenv("APP_ID")
	accessToken := os.Getenv("ACCESS_TOKEN")

	// Create a URL provider function
	urlProvider := func() (string, error) {
		url := fmt.Sprintf("wss://genesis.r7.21no.de/apps/%s?access_token=%s", appID, accessToken)
		return url, nil
	}

	// Initialize the client with configuration

	client := realtime_pubsub.NewClient(realtime_pubsub.Config{
		Logger: logger,
		WebSocketOptions: realtime_pubsub.WebSocketOptions{
			URLProvider: urlProvider,
		},
	})

	client.On("session.started", func(args ...interface{}) {
		go func() {
			// Publish a message to the "chat" topic
			waitFor, _ := client.Send("",
				realtime_pubsub.WithSendMessageType("gettime"),
			)

			// Wait for acknowledgment
			reply, err := waitFor.WaitForReply(500 * time.Millisecond)
			if err != nil {
				logger.Errorf("Error waiting for reply: %v", err)
			}

			// Print reply
			logger.Infof("Received date: %v\n", reply.DataAsMap()["time"])

			done <- true
		}()
	})

	// Handle error events
	client.Connect()

	select {
	case <-done:
	}
}
