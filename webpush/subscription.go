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

// HasValidEndpoint checks if the subscription endpoint is an absolute
// http(s) URL with a host. url.Parse alone accepts almost any string, so the
// scheme and host are checked explicitly.
func (s *Subscription) HasValidEndpoint() bool {
	u, err := url.Parse(s.Endpoint)
	return err == nil && (u.Scheme == "https" || u.Scheme == "http") && u.Host != ""
}

// HasKeys checks if the subscription has valid keys.
func (s *Subscription) HasKeys() bool {
	return s.Keys.P256DH != "" && s.Keys.Auth != ""
}
