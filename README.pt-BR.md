# web-push-go (Português)

> Esta é a documentação secundária em português. A documentação principal está em [README.md](README.md).

Biblioteca Go para envio de notificações Web Push com VAPID.

## Requisitos

- Go 1.25+
- Uma subscription válida do navegador (`endpoint`, `keys.p256dh`, `keys.auth`)
- Chaves VAPID (subject, private key, public key)

## Instalação

```bash
go get github.com/AcordoCertoBR/web-push-go
```

## Uso rápido

```go
package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/AcordoCertoBR/web-push-go/webpush"
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
		Title: "Olá",
		Options: webpush.MessageOptions{
			Body: "Sua notificação foi enviada com sucesso.",
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
		context.Background(),
		subscription,
		string(payload),
		webpush.NotificationOptions{Urgency: webpush.UrgencyNormal},
	)
	if err != nil {
		log.Fatal(err)
	}
}
```

Chaves VAPID e de subscription são aceitas em base64 raw ou com padding,
URL-safe ou padrão, e normalizadas internamente para raw URL-safe.

## Formato de subscription esperado

O projeto usa `webpush.Subscription`:

```json
{
  "endpoint": "https://...",
  "keys": {
    "p256dh": "...",
    "auth": "..."
  }
}
```

## Estrutura de mensagem enviada ao Service Worker

A biblioteca envia o payload no formato de `webpush.Message`:

```json
{
  "title": "Título",
  "options": {
    "body": "Texto",
    "icon": "https://...",
    "tag": "example-tag",
    "data": {
      "url": "http://localhost:8080/"
    }
  }
}
```

No Service Worker, use `event.data.json()` e chame `showNotification(title, options)`.

## Exemplo local

A pasta [example/](example/) contém:

- [index.html](example/index.html): gera subscription e copia JSON
- [service-worker.js](example/service-worker.js): listeners mínimos (`install`, `activate`, `push`, `notificationclick`, `notificationclose`)
- [main.go](example/main.go): exemplo de envio com a biblioteca

Fluxo sugerido:

1. Abra `index.html` em ambiente local compatível com Service Worker.
2. Assine push e copie o JSON da subscription.
3. Cole em `subscriptionString` de `example/main.go`.
4. Execute:

```bash
go run ./example
```

## Opções do WebPushClient

- `WithHttpClient(...)`
- `WithConcurrentSending(true|false)`
- `WithMaxConcurrency(n)`
- `WithPackSize(n)`
- `WithVapidExpiration(d)` — validade do JWT VAPID (padrão 3h, máx. 24h pela RFC 8292)

Também é possível empacotar mensagens com `PrepareAndPackMessage` e enviar em lote com `SendPackedMessages`.

## Features do WebPushClient

Todos os métodos de preparação recebem um `context.Context` que limita a
chamada HTTP (deadline/cancelamento).

### Envio simples

- `PrepareAndSendMessage(ctx, subscription, payload, options)`
  - valida subscription
  - criptografa o payload
  - monta request com headers Web Push + VAPID
  - envia a request

### Preparar sem enviar

- `PrepareMessage(ctx, subscription, payload, options)` retorna `*http.Request` pronta.
- `SendMessage(req)` envia depois (útil para inspeção/log/retry customizado).

### Envio em lote (pack)

- `PrepareAndPackMessage(ctx, ...)` adiciona mensagens em fila interna.
- `SendPackedMessages()` envia toda a fila e a limpa. Todas as mensagens são
  tentadas mesmo quando alguma falha; as falhas voltam como um único erro
  combinado (desembrulhe com `errors.As`/`errors.Is`):
  - modo sequencial (padrão): envia uma a uma, em ordem
  - modo concorrente (`WithConcurrentSending(true)`): envia em paralelo até `WithMaxConcurrency(...)`
- `CollectPackedMessages()` devolve e limpa a fila sem enviar.

O client é seguro para uso concorrente.

### Defaults importantes

- `TTL`: se não informado em `NotificationOptions`, usa 1 dia.
- `Urgency`: se informado, vai no header `Urgency`.
- `Topic`: se informado, vai no header `Topic`.
- HTTP client padrão: timeout de 10s (se não usar `WithHttpClient`).

### Critério de sucesso e classificação de erro

`SendMessage` considera sucesso com status HTTP:

- `200 OK`
- `201 Created`
- `202 Accepted`

Fora disso, retorna um `*webpush.ResponseError` com endpoint, status code e
body da resposta, permitindo classificar o resultado sem parsear strings de
erro:

```go
var respErr *webpush.ResponseError
if errors.As(err, &respErr) {
	switch {
	case respErr.SubscriptionGone(): // 404/410 — apague a subscription armazenada
	case respErr.Unauthorized():     // 401/403 — chaves VAPID não batem com a subscription
	case respErr.PayloadTooLarge():  // 413 — reduza o payload
	case respErr.Transient():        // 429/5xx — seguro tentar de novo depois
	}
}
```

## Troubleshooting

### `invalid subscription`

Causa comum: JSON sem `endpoint`, `keys.p256dh` ou `keys.auth`.

Confirme o formato em [webpush/subscription.go](webpush/subscription.go).

### Erro ao carregar VAPID (`LoadVapid`)

Causa comum: chaves mal formatadas (base64url inválido) ou chave privada incompatível.

Valide o par de chaves e, se necessário, gere novamente com `NewVapid`.

### Push não aparece no navegador

Checklist rápido:

- permissão de notificação concedida
- Service Worker ativo
- subscription atual (não expirada)
- payload compatível com o SW (`{ title, options }`)

### `notificationclick` não abre URL

No payload, envie `options.data.url`.

Exemplo:

```json
{
  "title": "Título",
  "options": {
    "body": "Mensagem",
    "data": {
      "url": "http://localhost:8080/"
    }
  }
}
```

### Erro HTTP do push service (401/403/410)

- `401/403`: VAPID inválido, subject incorreto, assinatura/token inválido.
- `410 Gone`: subscription expirada/revogada (gere uma nova no browser).

Use `errors.As` com `*webpush.ResponseError` e seus helpers
(`SubscriptionGone`, `Unauthorized`, `Transient`, ...) para tratar esses casos
programaticamente.

## Licença

Sem licença definida no repositório no momento.
