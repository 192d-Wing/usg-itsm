# notification

Consumes ticket events (`itsm.ticket.*`) from NATS JetStream (ADR-0004) via a
durable consumer and delivers a rendered notification over the configured
channels.

## Channels

| Channel   | Enabled by                      | Behavior                       |
|-----------|---------------------------------|--------------------------------|
| `log`     | always                          | logs every notification        |
| `webhook` | `NOTIFY_WEBHOOK_URL`            | POSTs `{title, body}` JSON     |
| `email`   | `SMTP_ADDR` + `NOTIFY_EMAIL`    | plaintext email via SMTP relay |

Delivery is at-least-once: a channel failure nak's the event so JetStream
redelivers it (up to 5 times), which can duplicate notifications on channels
that already succeeded. Malformed events are dropped (not redelivered).

## Configuration

| Env                  | Notes                                          |
|----------------------|------------------------------------------------|
| `NATS_URL`           | required; JetStream endpoint                    |
| `NOTIFY_WEBHOOK_URL` | optional webhook target                         |
| `SMTP_ADDR`          | optional SMTP relay `host:port`                 |
| `SMTP_FROM`          | From address (default `usg-itsm@localhost`)     |
| `NOTIFY_EMAIL`       | comma-separated recipients                      |
| `ADDR`               | health server, default `:8446`                  |

## Tests

- `internal/notify` — rendering, webhook (httptest), SMTP build/send (injected),
  and dispatcher behavior; all run without external services.
- `pkg/events` — durable-consumer integration test, gated on `TEST_NATS_URL`.
