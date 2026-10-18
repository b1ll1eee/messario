// Package httpapi is the REST adapter: routing, auth middleware, JSON helpers.
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/b1ll1eee/nuntio-message/internal/app"
	"github.com/b1ll1eee/nuntio-message/internal/domain"
)

// Config is what the HTTP adapter needs from the runtime configuration.
type Config struct {
	CookieName     string
	CookieSecure   bool
	SessionTTL     time.Duration
	AllowedOrigins []string // origins accepted on mutating requests and WS upgrades
	BotRatePerSec  float64
	BotBurst       int
	BotLongPoll    time.Duration
	BotUpdatesMax  int
	MaxFileBytes   int64
}

// Server holds the handlers' dependencies.
type Server struct {
	app     *app.App
	cfg     Config
	log     *slog.Logger
	ws      http.Handler
	spa     http.Handler
	limiter *botLimiter
	origins map[string]bool
}

// New builds the server. ws is the WebSocket handler, mounted at /ws behind
// auth; spa serves the web client for every other path.
func New(a *app.App, cfg Config, ws, spa http.Handler, log *slog.Logger) *Server {
	s := &Server{app: a, cfg: cfg, log: log, ws: ws, spa: spa, limiter: newBotLimiter(cfg.BotRatePerSec, cfg.BotBurst), origins: map[string]bool{}}
	for _, o := range cfg.AllowedOrigins {
		if u, err := url.Parse(strings.TrimSpace(o)); err == nil && u.Host != "" {
			s.origins[strings.ToLower(u.Scheme+"://"+u.Host)] = true
		}
	}
	return s
}

// Routes returns the root handler.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.healthz)

	mux.HandleFunc("POST /api/auth/login", s.login)
	mux.HandleFunc("POST /api/auth/logout", s.logout)

	auth := s.requireUser
	mux.Handle("GET /api/me", auth(s.me))
	mux.Handle("POST /api/me/password", auth(s.changePassword))
	mux.Handle("GET /api/users", auth(s.directory))

	mux.Handle("GET /api/chats", auth(s.listChats))
	mux.Handle("POST /api/chats", auth(s.createChat))
	mux.Handle("POST /api/chats/direct", auth(s.openDirect))
	mux.Handle("GET /api/chats/{id}", auth(s.getChat))
	mux.Handle("PATCH /api/chats/{id}", auth(s.renameChat))
	mux.Handle("GET /api/chats/{id}/messages", auth(s.history))
	mux.Handle("POST /api/chats/{id}/messages", auth(s.sendMessage))
	mux.Handle("POST /api/chats/{id}/files", auth(s.uploadFile))
	mux.Handle("GET /api/files/{id}", auth(s.downloadFile))
	mux.Handle("POST /api/chats/{id}/read", auth(s.markRead))
	mux.Handle("GET /api/chats/{id}/read", auth(s.readInfo))
	mux.Handle("GET /api/chats/{id}/members", auth(s.members))
	mux.Handle("POST /api/chats/{id}/members", auth(s.addMembers))
	mux.Handle("DELETE /api/chats/{id}/members/{uid}", auth(s.removeMember))

	mux.Handle("POST /api/admin/users", auth(s.adminCreateUser))
	mux.Handle("PATCH /api/admin/users/{id}", auth(s.adminUpdateUser))
	mux.Handle("POST /api/admin/bots", auth(s.adminCreateBot))
	mux.Handle("POST /api/admin/bots/{id}/token", auth(s.adminBotToken))
	mux.Handle("GET /api/admin/audit", auth(s.adminAudit))
	mux.Handle("GET /api/admin/license", auth(s.adminLicense))

	mux.Handle("GET /ws", auth(s.ws.ServeHTTP))

	mux.HandleFunc("/bot/{token}/{method}", s.botAPI)

	mux.Handle("/", s.spa)

	return s.withOriginCheck(s.withRecovery(mux))
}

func (s *Server) healthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ---- middleware ----

func (s *Server) withRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				s.log.Error("panic", "err", rec, "path", r.URL.Path)
				writeError(w, http.StatusInternalServerError, "internal", "internal error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// withOriginCheck rejects cross-site mutating requests and WS upgrades.
// Together with SameSite=Lax cookies this is the CSRF defence.
func (s *Server) withOriginCheck(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mutating := r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions
		isWS := r.URL.Path == "/ws"
		if (mutating || isWS) && !strings.HasPrefix(r.URL.Path, "/bot/") {
			if origin := r.Header.Get("Origin"); origin != "" && !s.originAllowed(origin, r) {
				writeError(w, http.StatusForbidden, "bad_origin", "request origin not allowed")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) originAllowed(origin string, r *http.Request) bool {
	origin = strings.ToLower(origin)
	if s.origins[origin] {
		return true
	}
	// Same host as the request itself (covers direct access by IP or hostname).
	if u, err := url.Parse(origin); err == nil && strings.EqualFold(u.Host, r.Host) {
		return true
	}
	return false
}

// requireUser authenticates via the session cookie and stores the user on the context.
func (s *Server) requireUser(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(s.cfg.CookieName)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized", "login required")
			return
		}
		u, err := s.app.Authenticate(r.Context(), c.Value)
		if err != nil {
			s.writeAppError(w, err)
			return
		}
		next.ServeHTTP(w, r.WithContext(app.WithUser(r.Context(), u)))
	})
}

// ---- helpers ----

func (s *Server) setSessionCookie(w http.ResponseWriter, token string, ttl time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name: s.cfg.CookieName, Value: token, Path: "/", HttpOnly: true,
		Secure: s.cfg.CookieSecure, SameSite: http.SameSiteLaxMode, MaxAge: int(ttl.Seconds()),
	})
}

func (s *Server) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: s.cfg.CookieName, Value: "", Path: "/", HttpOnly: true, Secure: s.cfg.CookieSecure, SameSite: http.SameSiteLaxMode, MaxAge: -1})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

type errorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	var b errorBody
	b.Error.Code, b.Error.Message = code, msg
	writeJSON(w, status, b)
}

// writeAppError maps domain errors to HTTP statuses.
func (s *Server) writeAppError(w http.ResponseWriter, err error) {
	status, code, msg := classify(err)
	if status == http.StatusInternalServerError {
		s.log.Error("request failed", "err", err)
	}
	writeError(w, status, code, msg)
}

func classify(err error) (int, string, string) {
	switch {
	case domain.IsInvalid(err):
		return http.StatusBadRequest, "invalid", err.Error()
	case errors.Is(err, domain.ErrUnauthorized):
		return http.StatusUnauthorized, "unauthorized", "invalid credentials or session"
	case errors.Is(err, domain.ErrSuspended):
		return http.StatusForbidden, "suspended", "account is suspended"
	case errors.Is(err, domain.ErrForbidden):
		return http.StatusForbidden, "forbidden", "not allowed"
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound, "not_found", "not found"
	case errors.Is(err, domain.ErrConflict):
		return http.StatusConflict, "conflict", "already exists"
	case errors.Is(err, context.Canceled):
		return 499, "canceled", "client went away"
	}
	return http.StatusInternalServerError, "internal", "internal error"
}

const maxJSONBody = 1 << 20 // 1 MiB

func decodeJSON(w http.ResponseWriter, r *http.Request, v any) error {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxJSONBody))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return domain.Invalid("invalid JSON body: %v", err)
	}
	return nil
}

func pathInt(r *http.Request, name string) (int64, error) {
	n, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil || n <= 0 {
		return 0, domain.Invalid("invalid %s", name)
	}
	return n, nil
}

func queryInt(r *http.Request, name string) int64 {
	n, _ := strconv.ParseInt(r.URL.Query().Get(name), 10, 64)
	return n
}
