package webpush

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
)

type fakeHTTPClient struct {
	mu       sync.Mutex
	requests []*http.Request
	status   int
	body     string
}

func (f *fakeHTTPClient) Do(req *http.Request) (*http.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests = append(f.requests, req)
	return &http.Response{
		StatusCode: f.status,
		Body:       io.NopCloser(strings.NewReader(f.body)),
	}, nil
}

func (f *fakeHTTPClient) requestCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.requests)
}

func newTestClient(t *testing.T, fake *fakeHTTPClient, opts ...WebPushClientOptions) *WebPushClient {
	t.Helper()

	vapid, err := NewVapid("dev@example.com")
	if err != nil {
		t.Fatalf("NewVapid returned error: %v", err)
	}

	opts = append(opts, WithHttpClient(fake))
	return NewWebPushClient(vapid, opts...)
}

func testSubscription(t *testing.T) Subscription {
	t.Helper()

	_, priv, _, err := generateServerKeys()
	if err != nil {
		t.Fatalf("failed to generate subscription keys: %v", err)
	}

	return Subscription{
		Endpoint: "https://push.example.com/send/abc",
		Keys: Keys{
			P256DH: base64.RawURLEncoding.EncodeToString(priv.PublicKey().Bytes()),
			Auth:   base64.RawURLEncoding.EncodeToString([]byte("0123456789ABCDEF")),
		},
	}
}

func TestSendMessageReturnsTypedResponseError(t *testing.T) {
	fake := &fakeHTTPClient{status: http.StatusGone, body: "push subscription has unsubscribed or expired"}
	client := newTestClient(t, fake)

	err := client.PrepareAndSendMessage(context.Background(), testSubscription(t), "hello", NotificationOptions{})
	if err == nil {
		t.Fatal("expected error for 410 response")
	}

	var respErr *ResponseError
	if !errors.As(err, &respErr) {
		t.Fatalf("expected *ResponseError, got %T: %v", err, err)
	}
	if respErr.StatusCode != http.StatusGone {
		t.Fatalf("expected status 410, got %d", respErr.StatusCode)
	}
	if !respErr.SubscriptionGone() {
		t.Fatal("expected SubscriptionGone to be true for 410")
	}
	if respErr.Transient() {
		t.Fatal("expected Transient to be false for 410")
	}
	if !strings.Contains(respErr.Body, "unsubscribed") {
		t.Fatalf("expected response body to be captured, got %q", respErr.Body)
	}
}

func TestSendMessageTreatsCreatedAsSuccess(t *testing.T) {
	fake := &fakeHTTPClient{status: http.StatusCreated}
	client := newTestClient(t, fake)

	err := client.PrepareAndSendMessage(context.Background(), testSubscription(t), "hello", NotificationOptions{})
	if err != nil {
		t.Fatalf("expected success for 201 response, got %v", err)
	}
}

func TestResponseErrorClassification(t *testing.T) {
	tests := []struct {
		status    int
		gone      bool
		unauth    bool
		tooLarge  bool
		limited   bool
		transient bool
	}{
		{status: http.StatusGone, gone: true},
		{status: http.StatusNotFound, gone: true},
		{status: http.StatusUnauthorized, unauth: true},
		{status: http.StatusForbidden, unauth: true},
		{status: http.StatusRequestEntityTooLarge, tooLarge: true},
		{status: http.StatusTooManyRequests, limited: true, transient: true},
		{status: http.StatusInternalServerError, transient: true},
		{status: http.StatusBadGateway, transient: true},
	}

	for _, tt := range tests {
		e := &ResponseError{StatusCode: tt.status}
		if e.SubscriptionGone() != tt.gone {
			t.Errorf("status %d: SubscriptionGone = %v, want %v", tt.status, e.SubscriptionGone(), tt.gone)
		}
		if e.Unauthorized() != tt.unauth {
			t.Errorf("status %d: Unauthorized = %v, want %v", tt.status, e.Unauthorized(), tt.unauth)
		}
		if e.PayloadTooLarge() != tt.tooLarge {
			t.Errorf("status %d: PayloadTooLarge = %v, want %v", tt.status, e.PayloadTooLarge(), tt.tooLarge)
		}
		if e.RateLimited() != tt.limited {
			t.Errorf("status %d: RateLimited = %v, want %v", tt.status, e.RateLimited(), tt.limited)
		}
		if e.Transient() != tt.transient {
			t.Errorf("status %d: Transient = %v, want %v", tt.status, e.Transient(), tt.transient)
		}
	}
}

func TestSendPackedMessagesConcurrentClearsQueue(t *testing.T) {
	fake := &fakeHTTPClient{status: http.StatusCreated}
	client := newTestClient(t, fake, WithConcurrentSending(true), WithMaxConcurrency(2))
	sub := testSubscription(t)

	for range 3 {
		if err := client.PrepareAndPackMessage(context.Background(), sub, "hello", NotificationOptions{}); err != nil {
			t.Fatalf("PrepareAndPackMessage returned error: %v", err)
		}
	}

	if err := client.SendPackedMessages(); err != nil {
		t.Fatalf("SendPackedMessages returned error: %v", err)
	}
	if got := fake.requestCount(); got != 3 {
		t.Fatalf("expected 3 requests, got %d", got)
	}

	// A second call must not re-send anything: the queue was drained.
	if err := client.SendPackedMessages(); err != nil {
		t.Fatalf("second SendPackedMessages returned error: %v", err)
	}
	if got := fake.requestCount(); got != 3 {
		t.Fatalf("expected no additional requests after second call, got %d total", got)
	}
}

func TestSendPackedMessagesSequentialAttemptsAllAndClearsQueue(t *testing.T) {
	fake := &fakeHTTPClient{status: http.StatusInternalServerError, body: "server error"}
	client := newTestClient(t, fake)
	sub := testSubscription(t)

	for range 3 {
		if err := client.PrepareAndPackMessage(context.Background(), sub, "hello", NotificationOptions{}); err != nil {
			t.Fatalf("PrepareAndPackMessage returned error: %v", err)
		}
	}

	err := client.SendPackedMessages()
	if err == nil {
		t.Fatal("expected joined error when all sends fail")
	}
	if got := fake.requestCount(); got != 3 {
		t.Fatalf("expected all 3 messages to be attempted, got %d", got)
	}

	var respErr *ResponseError
	if !errors.As(err, &respErr) {
		t.Fatalf("expected joined error to unwrap to *ResponseError, got %v", err)
	}

	// Failed messages are not retained for a later implicit retry.
	if err := client.SendPackedMessages(); err != nil {
		t.Fatalf("second SendPackedMessages returned error: %v", err)
	}
	if got := fake.requestCount(); got != 3 {
		t.Fatalf("expected no additional requests after second call, got %d total", got)
	}
}

func TestPrepareAndPackMessageRespectsPackSize(t *testing.T) {
	fake := &fakeHTTPClient{status: http.StatusCreated}
	client := newTestClient(t, fake, WithPackSize(1))
	sub := testSubscription(t)

	if err := client.PrepareAndPackMessage(context.Background(), sub, "hello", NotificationOptions{}); err != nil {
		t.Fatalf("first PrepareAndPackMessage returned error: %v", err)
	}
	if err := client.PrepareAndPackMessage(context.Background(), sub, "hello", NotificationOptions{}); err == nil {
		t.Fatal("expected error when pack is full")
	}
}

func TestPrepareMessageCarriesContextAndHeaders(t *testing.T) {
	fake := &fakeHTTPClient{status: http.StatusCreated}
	client := newTestClient(t, fake)

	type ctxKey struct{}
	ctx := context.WithValue(context.Background(), ctxKey{}, "sentinel")

	req, err := client.PrepareMessage(ctx, testSubscription(t), "hello", NotificationOptions{
		Topic:   "campaign-1",
		Urgency: UrgencyLow,
	})
	if err != nil {
		t.Fatalf("PrepareMessage returned error: %v", err)
	}

	if req.Context().Value(ctxKey{}) != "sentinel" {
		t.Fatal("expected prepared request to carry the provided context")
	}
	if got := req.Header.Get("TTL"); got != "86400" {
		t.Fatalf("expected default TTL of 1 day (86400), got %q", got)
	}
	if got := req.Header.Get("Topic"); got != "campaign-1" {
		t.Fatalf("expected Topic header, got %q", got)
	}
	if got := req.Header.Get("Urgency"); got != string(UrgencyLow) {
		t.Fatalf("expected Urgency header, got %q", got)
	}
	if got := req.Header.Get("Content-Encoding"); got != "aes128gcm" {
		t.Fatalf("expected aes128gcm content encoding, got %q", got)
	}
	if auth := req.Header.Get("Authorization"); !strings.HasPrefix(auth, "vapid t=") {
		t.Fatalf("expected VAPID authorization header, got %q", auth)
	}
}

func TestPrepareMessageRejectsInvalidSubscription(t *testing.T) {
	fake := &fakeHTTPClient{status: http.StatusCreated}
	client := newTestClient(t, fake)

	invalid := Subscription{Endpoint: "not-a-url", Keys: Keys{P256DH: "x", Auth: "y"}}
	if _, err := client.PrepareMessage(context.Background(), invalid, "hello", NotificationOptions{}); err == nil {
		t.Fatal("expected error for invalid endpoint")
	}
}
