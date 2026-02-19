# web-push-go (Português)

> Esta é a documentação secundária em português. A documentação principal está em [README.md](README.md).

Biblioteca Go para envio de notificações Web Push com VAPID.

## Requisitos

- Go 1.25+
- Uma subscription válida do navegador (`endpoint`, `keys.p256dh`, `keys.auth`)
- Chaves VAPID (subject, private key, public key)

## Instalação

```bash
go get github.com/ESSantana/web-push-go
```

## Uso rápido

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
		subscription,
		string(payload),
		webpush.NotificationOptions{Urgency: webpush.UrgencyNormal},
	)
	if err != nil {
		log.Fatal(err)
	}
}
```

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

Também é possível empacotar mensagens com `PrepareAndPackMessage` e enviar em lote com `SendPackedMessages`.

## Features do WebPushClient

### Envio simples

- `PrepareAndSendMessage(subscription, payload, options)`
  - valida subscription
  - criptografa o payload
  - monta request com headers Web Push + VAPID
  - envia a request

### Preparar sem enviar

- `PrepareMessage(subscription, payload, options)` retorna `*http.Request` pronta.
- `SendMessage(req)` envia depois (útil para inspeção/log/retry customizado).

### Envio em lote (pack)

- `PrepareAndPackMessage(...)` adiciona mensagens em fila interna.
- `SendPackedMessages()` envia toda a fila:
  - modo sequencial (padrão): para no primeiro erro
  - modo concorrente (`WithConcurrentSending(true)`): envia em paralelo até `WithMaxConcurrency(...)`
- `CollectPackedMessages()` devolve e limpa a fila sem enviar.

### Defaults importantes

- `TTL`: se não informado em `NotificationOptions`, usa 1 dia.
- `Urgency`: se informado, vai no header `Urgency`.
- `Topic`: se informado, vai no header `Topic`.
- HTTP client padrão: timeout de 10s (se não usar `WithHttpClient`).

### Critério de sucesso no envio

`SendMessage` considera sucesso com status HTTP:

- `200 OK`
- `201 Created`
- `202 Accepted`

Fora disso, retorna erro com URL, status code e body da resposta do push service.

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

## Licença

Sem licença definida no repositório no momento.
