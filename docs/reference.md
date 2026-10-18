# Nuntio reference

Everything that is not needed on the first read of the README: configuration,
the full API, architecture, operations.

## Configuration

Every setting is an environment variable with a default
(`internal/config/config.go`). The ones you will actually touch:

| Variable | Default | Meaning |
|---|---|---|
| `NUNTIO_ADDR` | `:8080` | HTTP listen address |
| `NUNTIO_DATABASE_URL` | `postgres://nuntio:nuntio@localhost:5432/nuntio?sslmode=disable` | PostgreSQL |
| `NUNTIO_PUBLIC_URL` | `http://localhost:8080` | External URL; its origin is allowed on mutating requests and WebSocket upgrades |
| `NUNTIO_ALLOWED_ORIGINS` | empty | Extra origins, comma-separated |
| `NUNTIO_COOKIE_SECURE` | `true` | Set `false` only for plain-HTTP local development |
| `NUNTIO_SESSION_TTL` | `720h` | Login lifetime |
| `NUNTIO_LOG_LEVEL` | `info` | `debug` / `info` / `warn` / `error` |
| `NUNTIO_MAX_GROUP_MEMBERS` | `200` | Cap per group |
| `NUNTIO_MAX_CHANNEL_MEMBERS` | `5000` | Cap per channel |
| `NUNTIO_MAX_BODY_BYTES` | `8000` | Longest message |
| `NUNTIO_MAX_FILE_MB` | `25` | Largest upload |
| `NUNTIO_FILES_DIR` | `./data/files` | Local file storage (used when no S3 endpoint) |
| `NUNTIO_S3_ENDPOINT` / `_ACCESS_KEY` / `_SECRET_KEY` / `_BUCKET` / `_REGION` / `_SSL` | empty / … / `nuntio` / `us-east-1` / `true` | Any S3-compatible store (MinIO, R2, S3) |
| `NUNTIO_BOT_RATE_PER_SECOND` / `NUNTIO_BOT_BURST` | `30` / `30` | Per-bot send limit |
| `NUNTIO_BOT_LONGPOLL_TIMEOUT` | `25s` | Max wait in `getUpdates` |
| `NUNTIO_WEBHOOK_TIMEOUT` / `_ATTEMPTS` / `_BACKOFF` / `_CONCURRENCY` | `10s` / `3` / `2s` / `16` | Webhook delivery |
| `NUNTIO_LICENSE` | empty | Customer license key; empty = community edition |
| `NUNTIO_LICENSE_PUBLIC_KEY` | compiled in | Override the vendor public key |
| `NUNTIO_FREE_SEATS` | `10` | Seats without a key |
| `NUNTIO_VINDICE_ENDPOINT` / `NUNTIO_VINDICE_API_KEY` | empty | Send traces, metrics and logs to [Vindice](https://github.com/b1ll1eee/vindice-observability) |
| `NUNTIO_ENVIRONMENT` | `production` | Tag on telemetry |

WebSocket tuning: `NUNTIO_WS_WRITE_TIMEOUT` (10s), `NUNTIO_WS_PING_INTERVAL`
(25s), `NUNTIO_WS_PONG_TIMEOUT` (10s), `NUNTIO_WS_SEND_BUFFER` (64 events).

## Commands

```
nuntio serve                    run the server (default)
nuntio migrate                  apply migrations and exit
nuntio init --org "Name" --admin admin --password '...' [--domain example.com]
licensegen keygen               vendor: create the signing key pair
licensegen issue -key K -org "Customer" -seats 50 -days 365
```

## HTTP API

All `/api` routes need the session cookie set by login. Errors are
`{"error": {"code": "...", "message": "..."}}` with a matching HTTP status:
`invalid` 400, `unauthorized` 401, `forbidden` / `suspended` 403,
`not_found` 404, `conflict` 409.

### Auth and users

| Route | Body / query | Notes |
|---|---|---|
| `POST /api/auth/login` | `{username, password}` | Sets `nuntio_session` cookie |
| `POST /api/auth/logout` | | |
| `GET /api/me` | | |
| `POST /api/me/password` | `{current, next}` | |
| `GET /api/users?q=&limit=` | | Directory: people and bots |

### Chats

| Route | Body / query | Notes |
|---|---|---|
| `GET /api/chats` | | List with unread counts, newest first |
| `POST /api/chats` | `{type: group\|channel, title, member_ids}` | Channels: admins only |
| `POST /api/chats/direct` | `{user_id}` | Opens or creates the 1:1 chat |
| `GET /api/chats/{id}` | | `{chat, me}` |
| `PATCH /api/chats/{id}` | `{title}` | |
| `GET /api/chats/{id}/messages` | `after_seq`, `before_seq`, `limit` | `after_seq` fills gaps, `before_seq` pages back |
| `POST /api/chats/{id}/messages` | `{client_msg_id, body}` | 201 created, 200 if the id was seen before |
| `POST /api/chats/{id}/files` | multipart `file`, `client_msg_id` | Posts a file message |
| `GET /api/files/{id}` | | Members only; images/PDF inline |
| `POST /api/chats/{id}/read` | `{seq}` | Advance read pointer |
| `GET /api/chats/{id}/read?seq=` | | `{count, total, readers}` |
| `GET /api/chats/{id}/members` | | |
| `POST /api/chats/{id}/members` | `{user_ids}` | Owner/admin of the chat |
| `DELETE /api/chats/{id}/members/{uid}` | | Self = leave |

### Admin

| Route | Body | Notes |
|---|---|---|
| `POST /api/admin/users` | `{username, display_name, email, department, title, role, password}` | Counts against seats |
| `PATCH /api/admin/users/{id}` | any of `display_name, email, department, title, role, status, password` | Suspending ends sessions |
| `POST /api/admin/bots` | `{username, display_name}` | Returns the token once |
| `POST /api/admin/bots/{id}/token` | | New token; old one stops working |
| `GET /api/admin/audit?limit=` | | |
| `GET /api/admin/license` | | `{seats, used_seats, status}` |

### WebSocket `/ws`

Same cookie. JSON frames with a `t` field.

Client to server:

```json
{"t":"send","id":"r1","chat_id":1,"client_msg_id":"<uuid>","body":"..."}
{"t":"read","chat_id":1,"seq":42}
{"t":"ping"}
```

Server to client: `hello` (on connect), `ack` (answers `send`, same `id`),
`msg` (new message, every member), `read` (`{chat_id,user_id,seq}`; own
devices always, the peer too in direct chats), `chat` (added to or changed),
`left`, `err` (`{code,message}`), `pong`.

After a reconnect the client calls `GET /api/chats/{id}/messages?after_seq=N`
for each open chat and resends anything still in its outbox; `client_msg_id`
makes that safe.

## Bot API

Telegram-shaped so existing bot code ports with a URL change. Responses are
`{"ok": true, "result": ...}` or `{"ok": false, "error_code": N, "description": "..."}`.

```
POST /bot/<token>/sendMessage     {"chat_id": 2, "text": "...", "client_msg_id": "..."}
POST /bot/<token>/sendDocument    multipart: chat_id, document
GET  /bot/<token>/getMe
GET  /bot/<token>/getChats
GET  /bot/<token>/getUpdates?offset=<last id + 1>&limit=100&timeout=25
POST /bot/<token>/setWebhook      {"url": "https://...", "secret": "..."}
POST /bot/<token>/deleteWebhook
GET  /bot/<token>/getWebhookInfo
```

A bot only sees chats it was added to. `getUpdates` long-polls up to
`timeout` seconds and returns messages with `id` greater than `offset - 1`,
excluding the bot's own. A webhook receives `{"message": {...}}` as JSON
with the secret in `X-Nuntio-Secret`; non-2xx responses are retried with
exponential backoff. Tokens look like `<bot id>:<random>` and are stored
hashed.

## Architecture

Hexagonal, kept deliberately light: one domain package, one use-case
package that declares the ports it needs, and one adapter per technology.

```mermaid
flowchart LR
  Browser[Vue client] -->|REST + WS| HTTP[httpapi + ws]
  Bots[Bots] -->|/bot/token| HTTP
  HTTP --> App[app: use cases]
  App --> Domain[domain]
  App --> PG[(postgres)]
  App --> Hub[memhub fan-out]
  App --> Files[filestore: local / S3]
  App --> Hook[webhook dispatcher]
  App --> Lic[license]
  Main[cmd/nuntio] -.wires.-> HTTP & App & PG & Hub & Files & Hook & Lic
  Main -.optional.-> Obs[obs: Vindice / OpenTelemetry]
```

`cmd/nuntio` reads config, opens adapters and hands them to `app.New` as
interfaces. `internal/domain` holds entities and rules with no outside
imports. `internal/app` implements every use case (login, chats, messages,
read receipts, files, bots, admin) against the ports in `ports.go`, so a
Postgres repository, the in-memory hub or the local file store can be
swapped (for NATS, S3, …) without touching use cases. `httpapi`, `ws` and the
bot API are thin: decode, call the app, encode.

Data model: `users` (people and bots), `sessions`, `chats` (`direct`,
`group`, `channel`; `direct_key` makes 1:1 chats unique; `last_seq` is the
per-chat sequence), `chat_members` (role, `read_seq`), `messages` (primary
key `chat_id, seq`; global `id` for bot cursors; `client_msg_id` unique per
sender), `files`, `bot_webhooks`, `audit_log`. Migrations are embedded and run
on start.

## Operations

- **Backups**: PostgreSQL (`pg_dump` or volume snapshot) plus the files volume
  or bucket. Restore both; nothing else holds state.
- **Updates**: `docker compose pull && docker compose up -d`. Migrations are
  forward-only and run before the server listens.
- **Scaling**: one instance serves thousands of connections. Beyond one
  instance, replace `memhub` with a NATS-backed hub (the `app.Hub` port) and
  put instances behind a load balancer; the DB schema and clients need no
  change.
- **Observability**: set `NUNTIO_VINDICE_ENDPOINT` and `NUNTIO_VINDICE_API_KEY`
  to get traces for every request, request metrics and trace-linked logs.
  `GET /healthz` for liveness.
- **Security notes**: sessions and bot tokens are stored hashed; cookies are
  httpOnly + SameSite=Lax and mutating requests must carry an allowed Origin;
  passwords are bcrypt; uploads are size-limited and served only to members
  with `nosniff`.
