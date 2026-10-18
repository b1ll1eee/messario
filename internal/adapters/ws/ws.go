// Package ws is the WebSocket adapter: one connection per client device,
// JSON frames both ways. The HTTP auth middleware runs before this handler,
// so the user is already on the request context.
//
// Client → server frames:
//
//	{"t":"send","id":"<req id>","chat_id":1,"client_msg_id":"...","body":"..."}
//	{"t":"read","chat_id":1,"seq":42}
//	{"t":"ping"}
//
// Server → client frames (see app.Event): hello, ack, msg, read, chat, left, err, pong.
package ws

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/coder/websocket"

	"github.com/b1ll1eee/messario/internal/adapters/memhub"
	"github.com/b1ll1eee/messario/internal/app"
	"github.com/b1ll1eee/messario/internal/domain"
)

// Config tunes connection behaviour.
type Config struct {
	WriteTimeout   time.Duration
	PingInterval   time.Duration
	PongTimeout    time.Duration
	SendBuffer     int
	AllowedOrigins []string
}

type Handler struct {
	app *app.App
	hub *memhub.Hub
	cfg Config
	log *slog.Logger
}

func New(a *app.App, hub *memhub.Hub, cfg Config, log *slog.Logger) *Handler {
	return &Handler{app: a, hub: hub, cfg: cfg, log: log}
}

type inbound struct {
	T           string `json:"t"`
	ID          string `json:"id,omitempty"`
	ChatID      int64  `json:"chat_id,omitempty"`
	ClientMsgID string `json:"client_msg_id,omitempty"`
	Body        string `json:"body,omitempty"`
	Seq         int64  `json:"seq,omitempty"`
}

type outbound struct {
	T  string `json:"t"`
	ID string `json:"id,omitempty"`
	P  any    `json:"p,omitempty"`
}

type errPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// client is one live connection; it implements memhub.Subscriber.
type client struct {
	out chan []byte
}

func (c *client) Send(msg []byte) bool {
	select {
	case c.out <- msg:
		return true
	default:
		return false
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	user := app.UserFrom(r.Context())
	if user == nil {
		http.Error(w, "login required", http.StatusUnauthorized)
		return
	}
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: hostPatterns(h.cfg.AllowedOrigins, r.Host),
	})
	if err != nil {
		h.log.Debug("ws accept", "err", err)
		return
	}
	defer conn.CloseNow() //nolint:errcheck

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	c := &client{out: make(chan []byte, h.cfg.SendBuffer)}
	unsubscribe := h.hub.Subscribe(user.ID, c)
	defer unsubscribe()

	h.enqueue(c, outbound{T: "hello", P: map[string]any{"user_id": user.ID, "now": time.Now().UTC()}})

	go h.writeLoop(ctx, conn, c, cancel)

	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		var in inbound
		if err := json.Unmarshal(data, &in); err != nil {
			h.enqueue(c, outbound{T: "err", P: errPayload{Code: "invalid", Message: "bad frame"}})
			continue
		}
		h.handle(ctx, user, c, in)
	}
}

func (h *Handler) handle(ctx context.Context, user *domain.User, c *client, in inbound) {
	switch in.T {
	case "ping":
		h.enqueue(c, outbound{T: "pong"})
	case "send":
		m, _, err := h.app.Send(ctx, user, in.ChatID, in.ClientMsgID, in.Body)
		if err != nil {
			h.enqueue(c, outbound{T: "err", ID: in.ID, P: toErr(err)})
			return
		}
		h.enqueue(c, outbound{T: "ack", ID: in.ID, P: m})
	case "read":
		if err := h.app.MarkRead(ctx, user, in.ChatID, in.Seq); err != nil {
			h.enqueue(c, outbound{T: "err", ID: in.ID, P: toErr(err)})
		}
	default:
		h.enqueue(c, outbound{T: "err", ID: in.ID, P: errPayload{Code: "invalid", Message: "unknown frame type"}})
	}
}

// writeLoop drains the outbound queue and sends pings; a missed pong or a
// failed write ends the connection.
func (h *Handler) writeLoop(ctx context.Context, conn *websocket.Conn, c *client, cancel context.CancelFunc) {
	defer cancel()
	ticker := time.NewTicker(h.cfg.PingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-c.out:
			wctx, done := context.WithTimeout(ctx, h.cfg.WriteTimeout)
			err := conn.Write(wctx, websocket.MessageText, msg)
			done()
			if err != nil {
				return
			}
		case <-ticker.C:
			pctx, done := context.WithTimeout(ctx, h.cfg.PongTimeout)
			err := conn.Ping(pctx)
			done()
			if err != nil {
				return
			}
		}
	}
}

func (h *Handler) enqueue(c *client, o outbound) {
	b, err := json.Marshal(o)
	if err != nil {
		return
	}
	c.Send(b)
}

func toErr(err error) errPayload {
	code := "internal"
	msg := "internal error"
	switch {
	case domain.IsInvalid(err):
		code, msg = "invalid", err.Error()
	case errors.Is(err, domain.ErrForbidden):
		code, msg = "forbidden", "not allowed"
	case errors.Is(err, domain.ErrNotFound):
		code, msg = "not_found", "not found"
	}
	return errPayload{Code: code, Message: msg}
}

// hostPatterns turns allowed origins into the host patterns coder/websocket expects,
// always including the request's own host.
func hostPatterns(origins []string, selfHost string) []string {
	out := []string{selfHost}
	for _, o := range origins {
		if u, err := url.Parse(o); err == nil && u.Host != "" {
			out = append(out, u.Host)
		}
	}
	return out
}
