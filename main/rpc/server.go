package main

import (
	"fmt"
	"github.com/backendstack21/realtime-pubsub-client-go"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"os"
	"time"
)

func main() {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetOutput(os.Stdout)
	logger.SetLevel(logrus.InfoLevel)

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

	client.On("secure/inbound.gettime", func(args ...interface{}) {
		logger.Infof("Responding to gettime request...")
		replyFunc := args[1].(realtime_pubsub.ReplyFunc)

		go func() {
			if err := replyFunc(map[string]interface{}{
				"time": time.Now().Format(time.RFC3339),
			}, "ok"); err != nil {
				logger.Errorf("Failed to reply to message: %v", err)
			}
		}()
	})

	client.On("secure/inbound.presence", func(args ...interface{}) {
		message := args[0].(realtime_pubsub.IncomingMessage)
		presence := message.DataAsPresenceMessage()

		if presence.Status() == "connected" {
			logger.Infof("Client %v connected with %v permissions...", presence.ConnectionId(), presence.Permissions())
		} else if presence.Status() == "disconnected" {
			logger.Infof("Client %v disconnected...", presence.ConnectionId())
		}
	})

	client.On("session.started", func(args ...interface{}) {
		err := client.SubscribeRemoteTopic("secure/inbound")
		if err != nil {
			logger.Errorf("Error subscribing to remote topic: %v", err)
		}
	})

	// Handle error events
	client.Connect()

	select {}
}
