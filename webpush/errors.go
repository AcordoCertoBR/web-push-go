package webpush

import (
	"fmt"
	"net/http"
)

// ResponseError is returned by SendMessage when the push service responds
// with a non-success status code. It exposes the status code and response
// body so callers can decide what to do with the subscription (remove it,
// retry later, shrink the payload) without parsing error strings.
//
// Use errors.As to retrieve it:
//
//	var respErr *webpush.ResponseError
//	if errors.As(err, &respErr) && respErr.SubscriptionGone() {
//		// delete the stored subscription
//	}
type ResponseError struct {
	// Endpoint is the push service URL the request was sent to.
	Endpoint string
	// StatusCode is the HTTP status code returned by the push service.
	StatusCode int
	// Body is the response body returned by the push service, useful for
	// logging and diagnostics.
	Body string
}

func (e *ResponseError) Error() string {
	return fmt.Sprintf("failed to send notification: %s, status code: %d, response: %s", e.Endpoint, e.StatusCode, e.Body)
}

// SubscriptionGone reports whether the subscription no longer exists on the
// push service (expired, unsubscribed or unregistered). The stored
// subscription should be deleted; retrying will never succeed.
func (e *ResponseError) SubscriptionGone() bool {
	return e.StatusCode == http.StatusGone || e.StatusCode == http.StatusNotFound
}

// Unauthorized reports whether the push service rejected the VAPID
// credentials — typically the subscription was created with a different key
// pair than the one signing the request. Retrying with the same keys will
// never succeed.
func (e *ResponseError) Unauthorized() bool {
	return e.StatusCode == http.StatusUnauthorized || e.StatusCode == http.StatusForbidden
}

// PayloadTooLarge reports whether the push service rejected the message
// because the encrypted body exceeds its size limit.
func (e *ResponseError) PayloadTooLarge() bool {
	return e.StatusCode == http.StatusRequestEntityTooLarge
}

// RateLimited reports whether the push service throttled the request.
func (e *ResponseError) RateLimited() bool {
	return e.StatusCode == http.StatusTooManyRequests
}

// Transient reports whether the failure is likely temporary on the push
// service side (5xx or rate limiting) and the message may be retried later.
func (e *ResponseError) Transient() bool {
	return e.StatusCode >= 500 || e.RateLimited()
}
