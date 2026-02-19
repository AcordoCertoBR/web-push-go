package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/ESSantana/web-push-go/webpush"
)

const (
	vapidSubject    = "SUBJECT_EMAIL_OR_URL"
	vapidPrivateKey = "VAPID_PRIVATE_KEY"
	vapidPublicKey  = "VAPID_PUBLIC_KEY"
)

var subscriptionString = ``

func main() {
	vapid, err := loadVapidCredentials()
	if err != nil {
		log.Fatal(err)
	}

	subscription, err := parseSubscription(subscriptionString)
	if err != nil {
		log.Fatal(err)
	}

	client := newWebPushClient(vapid)
	message := buildExampleMessage()
	notificationOptions := buildNotificationOptions()

	err = client.PrepareAndSendMessage(subscription, message, notificationOptions)
	if err != nil {
		log.Fatalf("failed to send web push message: %v", err)
	}

	log.Println("notification sent")
}

func loadVapidCredentials() (webpush.Vapid, error) {
	vapid, err := webpush.LoadVapid(vapidSubject, vapidPrivateKey, vapidPublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to load VAPID credentials: %w", err)
	}

	return vapid, nil
}

func parseSubscription(raw string) (webpush.Subscription, error) {
	var sub webpush.Subscription
	if err := json.Unmarshal([]byte(raw), &sub); err != nil {
		return webpush.Subscription{}, fmt.Errorf("failed to parse subscription JSON: %w", err)
	}

	return sub, nil
}

func newWebPushClient(vapid webpush.Vapid) *webpush.WebPushClient {
	httpClient := &http.Client{Timeout: time.Second}
	return webpush.NewWebPushClient(
		vapid,
		webpush.WithHttpClient(httpClient),
		webpush.WithConcurrentSending(true),
		webpush.WithMaxConcurrency(5),
		webpush.WithPackSize(10),
	)
}

func buildNotificationOptions() webpush.NotificationOptions {
	return webpush.NotificationOptions{
		Urgency: webpush.UrgencyNormal,
	}
}

func buildExampleMessage() string {
	message := webpush.Message{
		Title: "Hello",
		Options: webpush.MessageOptions{
			Actions: []webpush.MessageActions{
				{
					Action: "action1",
					Title:  "Action 1",
					Icon:   "https://example.com/icon1.jpg",
				},
				{
					Action: "action2",
					Title:  "Action 2",
					Icon:   "https://example.com/icon2.jpg",
				},
			},
			Body:               "World!",
			Icon:               "https://example.com/icon.jpg",
			Renotify:           true,
			RequireInteraction: true,
			Tag:                "example-tag",
		},
	}

	payload, err := json.Marshal(message)
	if err != nil {
		log.Fatalf("failed to marshal message: %v", err)
	}

	return string(payload)
}

// Optional helper to generate a new VAPID pair.
// Execute manually when needed.
func createVapidCredentials() {
	vapid, err := webpush.NewVapid(vapidSubject)
	if err != nil {
		log.Fatal("failed to generate VAPID keys")
	}

	privateECDH, publicECDH := vapid.Keys()
	fmt.Println(vapid.Subject(), privateECDH, publicECDH)
}
