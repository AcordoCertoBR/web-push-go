package webpush

import (
	"net/http"
	"time"
)

type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Options struct {
	// Vapid contains the VAPID information used for authentication.
	vapid Vapid
	// HttpClient is the HTTP client used to send push notifications.
	httpClient HttpClient
	// ConcurrentSending controls whether messages are sent concurrently.
	concurrentSending bool
	// MaxConcurrency defines the maximum number of concurrent messages to send.
	maxConcurrency int
	// PackSize defines the maximum number of messages to pack in a single batch.
	packSize int
	// VapidExpiration defines the lifetime of the VAPID JWT attached to each request.
	vapidExpiration time.Duration
}

// WithHttpClient sets the HTTP client used to send push notifications.
func WithHttpClient(client HttpClient) func(*Options) {
	return func(o *Options) {
		o.httpClient = client
	}
}

// WithConcurrentSending enables or disables concurrent sending of messages.
func WithConcurrentSending(concurrent bool) func(*Options) {
	return func(o *Options) {
		o.concurrentSending = concurrent
	}
}

// WithMaxConcurrency sets the maximum number of concurrent messages to send (default is 2).
func WithMaxConcurrency(max int) func(*Options) {
	return func(o *Options) {
		if max <= 0 {
			max = 2
		}
		o.maxConcurrency = max
	}
}

// WithPackSize sets the maximum number of messages to pack in a single batch (default is 10).
func WithPackSize(size int) func(*Options) {
	return func(o *Options) {
		if size <= 0 {
			size = 10
		}
		o.packSize = size
	}
}

// WithVapidExpiration sets the lifetime of the VAPID JWT attached to each
// request (default is 3 hours). Values outside (0, 24h] — the maximum allowed
// by the VAPID spec (RFC 8292) — fall back to the default.
func WithVapidExpiration(d time.Duration) func(*Options) {
	return func(o *Options) {
		if d <= 0 || d > 24*time.Hour {
			d = 3 * time.Hour
		}
		o.vapidExpiration = d
	}
}
