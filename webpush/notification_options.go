package webpush

type Urgency string

const (
	// UrgencyVeryLow requires device state: on power and Wi-Fi
	UrgencyVeryLow Urgency = "very-low"
	// UrgencyLow requires device state: on either power or Wi-Fi
	UrgencyLow Urgency = "low"
	// UrgencyNormal excludes device state: low battery
	UrgencyNormal Urgency = "normal"
	// UrgencyHigh admits device state: low battery
	UrgencyHigh Urgency = "high"
)

type NotificationOptions struct {
	// Urgency indicates to the push service how important a message is to the user.
	// This can be used by the push service to help conserve the battery life of a user's device
	// by only waking up for important messages when battery is low.
	// (default is UrgencyNormal)
	Urgency Urgency
	// Allows pending messages to be replaced with new corresponding messages, optimizing the display of the latest information to the user when the device is online.
	// If the topic is not set, the message will be sent without a topic
	Topic string
	// How many time the message will be stored in the push service (default is 1 day)
	TTL int
}
