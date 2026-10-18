# Nuntio

Self-hosted team chat for organizations. One Go binary, one PostgreSQL, a
Telegram-style bot API. Your data stays on your server.

## Status

- Phase 1: backend core (auth, direct/group/channel chats, messages with
  gap-free sequences, read receipts, admin, audit log, bot API, WebSocket).
- Phase 2: Vue 3 + Pinia web client embedded in the binary (chat list,
  conversation, channels, read receipts, outbox with retry, reconnect + resync,
  admin page for users/bots/audit, light + dark, phone layout). Optional
  Vindice observability (traces, metrics, logs) via `NUNTIO_VINDICE_ENDPOINT`.

## Run locally

```sh
make db-up                                   # PostgreSQL 17 in Docker
make init ORG="My Company" ADMIN=admin PASSWORD=changeme-now
make web                                     # build the Vue client into internal/web/dist
make run                                     # http://localhost:8080 (API + embedded client)
```

For UI work run `make web-dev` in a second terminal and open
http://localhost:5173; Vite proxies `/api`, `/ws` and `/bot` to :8080.

Every setting is an environment variable with a default; see `.env.example`
and `internal/config/config.go`.

## Layout (hexagonal, kept light)

```
cmd/nuntio            main: wires config → adapters → app
internal/domain       entities and rules, no imports from the outside
internal/app          use cases + the ports (interfaces) they need
internal/adapters/
  postgres            repositories + embedded migrations
  httpapi             REST, auth middleware, bot API
  ws                  WebSocket adapter
  memhub              in-memory fan-out (swap for NATS when multi-instance)
  hash                bcrypt
  obs                 Vindice / OpenTelemetry (optional)
internal/web          embedded Vite build + SPA handler
web/                  Vue 3 + Pinia + TypeScript source
```

## API sketch

REST under `/api` with an httpOnly session cookie:

```
POST /api/auth/login            {username, password}
GET  /api/me
GET  /api/users?q=
GET  /api/chats
POST /api/chats                 {type: group|channel, title, member_ids}
POST /api/chats/direct          {user_id}
GET  /api/chats/{id}/messages?after_seq=&before_seq=&limit=
POST /api/chats/{id}/messages   {client_msg_id, body}
POST /api/chats/{id}/read       {seq}
GET  /api/chats/{id}/read?seq=  → {count, total, readers}
POST /api/admin/users | PATCH /api/admin/users/{id}
POST /api/admin/bots  → {bot, token}   (token shown once)
GET  /api/admin/audit
```

WebSocket at `/ws` (same cookie). Frames are JSON with a `t` field:
client sends `send`, `read`, `ping`; server sends `hello`, `ack`, `msg`,
`read`, `chat`, `left`, `err`, `pong`. Clients resync after reconnect with
`after_seq`; `client_msg_id` makes retries idempotent.

Bot API, Telegram-shaped:

```
POST /bot/<token>/sendMessage   {"chat_id": 2, "text": "deploy done"}
GET  /bot/<token>/getMe
GET  /bot/<token>/getUpdates?offset=<last id + 1>&timeout=25
GET  /bot/<token>/getChats
```

## Development

```sh
make build   # bin/nuntio
make vet
make test    # needs the dev database
```
