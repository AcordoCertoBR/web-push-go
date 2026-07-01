package webpush

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/ESSantana/web-push-go/webpush/constants"
)

// WebPushClient prepares and sends Web Push requests. It is safe for
// concurrent use: the internal message pack is guarded by a mutex.
type WebPushClient struct {
	options Options

	mu       sync.Mutex
	messages []*http.Request
}

type WebPushClientOptions = func(*Options)

func NewWebPushClient(vapid Vapid, optFns ...WebPushClientOptions) *WebPushClient {
	options := &Options{
		vapid:             vapid,
		concurrentSending: false,
		maxConcurrency:    2,
		packSize:          10,
		vapidExpiration:   3 * time.Hour,
	}

	for _, fn := range optFns {
		fn(options)
	}

	if options.httpClient == nil {
		options.httpClient = &http.Client{
			Timeout: time.Second * 10,
		}
	}

	client := &WebPushClient{
		options:  *options,
		messages: make([]*http.Request, 0),
	}

	return client
}

// PrepareAndSendMessage builds a Web Push HTTP request for the given subscription and payload,
// then sends it using the configured HTTP client.
//
// It validates and encrypts the payload via PrepareMessage, applies notification options
// (such as TTL, Topic, and Urgency), attaches VAPID authorization headers, and finally
// dispatches the request with SendMessage.
//
// The context bounds the whole operation, including the HTTP round trip.
//
// An error is returned if request preparation fails, encryption/signing fails, network
// delivery fails, or the push service responds with a non-success status code (see
// ResponseError).
func (w *WebPushClient) PrepareAndSendMessage(ctx context.Context, subscription Subscription, payload string, options NotificationOptions) (err error) {
	request, err := w.PrepareMessage(ctx, subscription, payload, options)
	if err != nil {
		return err
	}
	return w.SendMessage(request)
}

// PrepareAndPackMessage builds a Web Push HTTP request for the given subscription and payload,
// then appends the prepared request to the client's internal queue for later delivery.
//
// The request is created via PrepareMessage, including payload encryption, notification
// headers (TTL, Topic, Urgency), and VAPID authorization. No network request is sent by
// this method. The context is carried by the prepared request and bounds its eventual send.
//
// Use SendPackedMessages to dispatch all queued requests.
//
// It returns an error if the pack is full or if request preparation, encryption, or
// VAPID signing fails.
func (w *WebPushClient) PrepareAndPackMessage(ctx context.Context, subscription Subscription, payload string, options NotificationOptions) (err error) {
	request, err := w.PrepareMessage(ctx, subscription, payload, options)
	if err != nil {
		return err
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if len(w.messages) >= w.options.packSize {
		return errors.New("message pack is full, send or collect the current pack before adding more messages")
	}

	w.messages = append(w.messages, request)
	return nil
}

// PrepareMessage validates the subscription, encrypts the payload, and constructs
// an authenticated Web Push HTTP request bound to the given context.
//
// The method verifies that the subscription endpoint and keys are present, encrypts
// the payload using the subscription's ECDH/auth secrets, and creates a POST request
// targeting the subscription endpoint. It then applies notification headers including
// TTL (defaulting to 1 day when unset), optional Topic, optional Urgency, and the
// required Web Push encryption/content headers.
//
// A VAPID authorization token is generated using the client's configured VAPID info
// and attached to the Authorization header together with the VAPID public key.
//
// It returns the prepared request ready to be sent with SendMessage, or an error if
// validation, encryption, request creation, or VAPID signing fails.
func (w *WebPushClient) PrepareMessage(ctx context.Context, subscription Subscription, payload string, options NotificationOptions) (*http.Request, error) {
	if !subscription.HasValidEndpoint() || !subscription.HasKeys() {
		return nil, errors.New("invalid subscription")
	}

	encryptBody, err := EncryptPayload(subscription.Keys.P256DH, subscription.Keys.Auth, payload, 0)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, subscription.Endpoint, bytes.NewReader(encryptBody))
	if err != nil {
		return nil, err
	}

	if options.TTL == 0 {
		options.TTL = constants.DAY * 1
	}

	if options.Topic != "" {
		request.Header.Add("Topic", options.Topic)
	}

	if options.Urgency != "" {
		request.Header.Add("Urgency", string(options.Urgency))
	}

	signedToken, err := GetVAPIDAuthorizationHeader(subscription.Endpoint, w.options.vapid, time.Now().Add(w.options.vapidExpiration))
	if err != nil {
		return nil, err
	}

	_, publicECDH := w.options.vapid.Keys()
	request.Header.Set("Content-Encoding", "aes128gcm")
	request.Header.Set("Content-Type", "application/octet-stream")
	request.Header.Set("Authorization", fmt.Sprintf("vapid t=%s, k=%s", signedToken, publicECDH))
	request.Header.Set("TTL", fmt.Sprintf("%d", options.TTL))

	return request, nil
}

// SendMessage sends a prepared Web Push HTTP request using the client's configured HTTP client.
//
// A successful delivery is considered any response with status 200 (OK), 201 (Created),
// or 202 (Accepted). For non-success responses, it reads the response body and returns
// a *ResponseError carrying the endpoint, status code, and push service response payload,
// so callers can classify the outcome (see ResponseError) without parsing error strings.
//
// Network/transport errors from the underlying HTTP client are returned directly.
func (w *WebPushClient) SendMessage(request *http.Request) error {
	response, err := w.options.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if !slices.Contains([]int{http.StatusOK, http.StatusCreated, http.StatusAccepted}, response.StatusCode) {
		data, err := io.ReadAll(response.Body)
		if err != nil {
			return err
		}
		return &ResponseError{
			Endpoint:   request.URL.String(),
			StatusCode: response.StatusCode,
			Body:       string(data),
		}
	}
	return nil
}

// SendPackedMessages dispatches all requests previously queued via PrepareAndPackMessage.
//
// The queue is drained up front, so the pack is empty after this call regardless of the
// outcome — a failed message is never silently re-sent by a later call. Every queued
// message is attempted even when earlier ones fail.
//
// In sequential mode (concurrentSending disabled), messages are sent one-by-one in order.
// In concurrent mode (concurrentSending enabled), messages are sent in parallel up to
// maxConcurrency.
//
// All send failures are collected and returned as a single joined error (unwrap with
// errors.As / errors.Is); if every send succeeds, nil is returned.
func (w *WebPushClient) SendPackedMessages() error {
	w.mu.Lock()
	pending := w.messages
	w.messages = nil
	w.mu.Unlock()

	if len(pending) == 0 {
		return nil
	}

	if !w.options.concurrentSending {
		var sendErrors []error
		for _, request := range pending {
			if err := w.SendMessage(request); err != nil {
				sendErrors = append(sendErrors, err)
			}
		}
		return errors.Join(sendErrors...)
	}

	sem := make(chan struct{}, w.options.maxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex

	sendErrors := make([]error, 0)

	for _, request := range pending {
		req := request
		wg.Add(1)
		sem <- struct{}{}

		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			if err := w.SendMessage(req); err != nil {
				mu.Lock()
				sendErrors = append(sendErrors, err)
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	return errors.Join(sendErrors...)
}

// CollectPackedMessages returns all currently queued requests without sending them.
// The internal queue is cleared after collection.
func (w *WebPushClient) CollectPackedMessages() []*http.Request {
	w.mu.Lock()
	defer w.mu.Unlock()

	if len(w.messages) == 0 {
		return nil
	}

	collected := slices.Clone(w.messages)
	w.messages = nil
	return collected
}
