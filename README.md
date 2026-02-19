# web-push-go

Go library for sending Web Push notifications with VAPID.

Portuguese documentation: [README.pt-BR.md](README.pt-BR.md).

## Requirements

- Go 1.25+
- A valid browser subscription (`endpoint`, `keys.p256dh`, `keys.auth`)
- VAPID credentials (subject, private key, public key)

## Installation

```bash
go get github.com/ESSantana/web-push-go
```

## Quick start

```go
package main

import (
	"encoding/json"
	"log"

	"github.com/ESSantana/web-push-go/webpush"
)

func main() {
	vapid, err := webpush.LoadVapid(
		"mailto:you@example.com",
		"<VAPID_PRIVATE_KEY>",
		"<VAPID_PUBLIC_KEY>",
	)
	if err != nil {
		log.Fatal(err)
	}

	subscription := webpush.Subscription{
		Endpoint: "https://push.service/...",
		Keys: webpush.Keys{
			P256DH: "<P256DH>",
			Auth:   "<AUTH>",
		},
	}

	message := webpush.Message{
		Title: "Hello",
		Options: webpush.MessageOptions{
			Body: "Your notification was sent successfully.",
			Tag:  "demo",
			Data: map[string]any{
				"url": "http://localhost:8080/",
			},
		},
	}

	payload, err := json.Marshal(message)
	if err != nil {
		log.Fatal(err)
	}

	client := webpush.NewWebPushClient(vapid)
	err = client.PrepareAndSendMessage(
		subscription,
		string(payload),
		webpush.NotificationOptions{Urgency: webpush.UrgencyNormal},
	)
	if err != nil {
		log.Fatal(err)
	}
}
```

## Expected subscription format

This project uses `webpush.Subscription`:

```json
{
  "endpoint": "https://...",
  "keys": {
    "p256dh": "...",
    "auth": "..."
  }
}
```

## Message payload format for the Service Worker

The library sends payloads using `webpush.Message`:

```json
{
  "title": "Title",
  "options": {
    "body": "Text",
    "icon": "https://...",
    "tag": "example-tag",
    "data": {
      "url": "http://localhost:8080/"
    }
  }
}
```

In the Service Worker, use `event.data.json()` and call `showNotification(title, options)`.

## Local example

The [example/](example/) folder includes:

- [index.html](example/index.html): creates subscription and copies JSON
- [service-worker.js](example/service-worker.js): minimal listeners (`install`, `activate`, `push`, `notificationclick`, `notificationclose`)
- [main.go](example/main.go): send example using the library

Suggested flow:

1. Open `index.html` in a local environment compatible with Service Workers.
2. Subscribe for push and copy the subscription JSON.
3. Paste it into `subscriptionString` in `example/main.go`.
4. Run:

```bash
go run ./example
```

## WebPushClient options

- `WithHttpClient(...)`
- `WithConcurrentSending(true|false)`
- `WithMaxConcurrency(n)`
- `WithPackSize(n)`

You can also queue messages with `PrepareAndPackMessage` and send them with `SendPackedMessages`.

## WebPushClient features

### Simple send

- `PrepareAndSendMessage(subscription, payload, options)`
  - validates the subscription
  - encrypts the payload
  - builds a request with Web Push + VAPID headers
  - sends the request

### Prepare without sending

- `PrepareMessage(subscription, payload, options)` returns a ready `*http.Request`.
- `SendMessage(req)` sends it later (useful for custom logging/inspection/retry).

### Batch send (pack)

- `PrepareAndPackMessage(...)` appends requests to an internal queue.
- `SendPackedMessages()` sends all queued requests:
  - sequential mode (default): stops at first error
  - concurrent mode (`WithConcurrentSending(true)`): sends in parallel up to `WithMaxConcurrency(...)`
- `CollectPackedMessages()` returns and clears the queue without sending.

### Important defaults

- `TTL`: if not set in `NotificationOptions`, defaults to 1 day.
- `Urgency`: if set, sent as the `Urgency` header.
- `Topic`: if set, sent as the `Topic` header.
- default HTTP client timeout: 10s (when `WithHttpClient` is not provided).

### Success criteria

`SendMessage` treats these HTTP status codes as success:

- `200 OK`
- `201 Created`
- `202 Accepted`

Any other status returns an error with URL, status code, and response body.

## Troubleshooting

### `invalid subscription`

Common cause: missing `endpoint`, `keys.p256dh`, or `keys.auth` in JSON.

Check [webpush/subscription.go](webpush/subscription.go).

### `LoadVapid` error

Common cause: invalid base64url keys or mismatched private/public key pair.

Validate keys and regenerate with `NewVapid` if needed.

### Push notification not showing

Quick checklist:

- notification permission granted
- Service Worker active
- subscription is current (not expired/revoked)
- payload matches SW format (`{ title, options }`)

### `notificationclick` does not open URL

Send `options.data.url` in payload.

Example:

```json
{
  "title": "Title",
  "options": {
    "body": "Message",
    "data": {
      "url": "http://localhost:8080/"
    }
  }
}
```

### Push service HTTP error (401/403/410)

- `401/403`: invalid VAPID, incorrect subject, invalid signature/token.
- `410 Gone`: subscription expired/revoked (create a new one in browser).

## License

No license file is currently defined in this repository.
