package webpush

import "net/url"

type Keys struct {
	P256DH string `json:"p256dh"`
	Auth   string `json:"auth"`
}

type Subscription struct {
	Endpoint string `json:"endpoint"`
	Keys     Keys   `json:"keys"`
}

// HasValidEndpoint checks if the subscription has a valid endpoint URL.
func (s *Subscription) HasValidEndpoint() bool {
	_, err := url.Parse(s.Endpoint)
	return err == nil && s.Endpoint != ""
}

// HasKeys checks if the subscription has valid keys.
func (s *Subscription) HasKeys() bool {
	return s.Keys.P256DH != "" && s.Keys.Auth != ""
}
