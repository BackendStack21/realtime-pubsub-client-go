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
	if appID == "" || accessToken == "" {
		logger.Fatal("APP_ID and ACCESS_TOKEN must be set in the environment")
	}

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

	client.On("error", func(args ...interface{}) {
		logger.Infof("Received error: %v", args)
	})

	client.On("close", func(args ...interface{}) {
		logger.Infof("Connection closed: %v", args)
	})

	client.Connect()

	// Handle session started event
	client.On("session.started", func(args ...interface{}) {
		info := args[0].(realtime_pubsub.ConnectionInfo)
		logger.Infof("Connection ID: %v", info.ConnectionId())

		// IMPORTANT: Subscribe to remote topics here
		// so subscriptions are re-established on reconnection

		// Subscribe to the "chat" topic
		if err := client.SubscribeRemoteTopic("chat"); err != nil {
			logger.Errorf("Failed to subscribe to topic: %v", err)
		}
	})
	// Wait for the session to start
	_, err := client.WaitFor("session.started", 10*time.Second)
	if err != nil {
		logger.Errorf("Failed to start session: %v", err)
	}

	// Handle incoming messages on "chat.text-message"
	client.On("chat.text-message", func(args ...interface{}) {
		// Extract message and reply function
		message := args[0].(realtime_pubsub.IncomingMessage)
		replyFunc := args[1].(realtime_pubsub.ReplyFunc)

		logger.Infof("Received message: %v", message)

		// Reply to the message
		if err := replyFunc("Got it!", "ok"); err != nil {
			logger.Errorf("Failed to reply to message: %v", err)
		}
	})

	// Handle incoming messages on the "main" topic
	client.On("main.*", func(args ...interface{}) {
		message := args[0].(realtime_pubsub.IncomingMessage)
		logger.Infof("Received message on main topic: %v", message)
	})

	// Publish a message to the "chat" topic
	waitFor, err := client.Publish("chat", "Hello, World!",
		realtime_pubsub.WithPublishMessageType("text-message"),
		realtime_pubsub.WithPublishCompress(true),
	)
	if err != nil {
		logger.Errorf("Failed to publish message: %v", err)
	}

	// Wait for acknowledgment
	if _, err := waitFor.WaitForAck(100 * time.Millisecond); err != nil {
		logger.Errorf("Failed to receive acknowledgment: %v", err)
	}

	// Wait for a reply
	reply, err := waitFor.WaitForReply(500 * time.Millisecond)
	if err != nil {
		logger.Errorf("Failed to receive reply: %v", err)
	} else {
		logger.Infof("Received reply: %v", reply.Data())
	}

	_ = client.Disconnect()
}
