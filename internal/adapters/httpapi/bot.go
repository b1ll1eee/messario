package httpapi

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/b1ll1eee/nuntio-message/internal/domain"
)

// The bot API mirrors Telegram's shape so existing bot code ports easily:
//   POST /bot/<token>/sendMessage   {"chat_id": 1, "text": "...", "client_msg_id": "..."}
//   GET  /bot/<token>/getMe
//   GET  /bot/<token>/getUpdates?offset=<last id + 1>&limit=100&timeout=25
// Responses: {"ok": true, "result": ...} or {"ok": false, "error_code": 400, "description": "..."}.

type botOK struct {
	OK     bool `json:"ok"`
	Result any  `json:"result"`
}

type botErr struct {
	OK          bool   `json:"ok"`
	ErrorCode   int    `json:"error_code"`
	Description string `json:"description"`
}

func writeBotError(w http.ResponseWriter, err error) {
	status, _, msg := classify(err)
	writeJSON(w, status, botErr{OK: false, ErrorCode: status, Description: msg})
}

func (s *Server) botAPI(w http.ResponseWriter, r *http.Request) {
	bot, err := s.app.BotByToken(r.Context(), r.PathValue("token"))
	if err != nil {
		writeBotError(w, err)
		return
	}
	switch r.PathValue("method") {
	case "getMe":
		writeJSON(w, http.StatusOK, botOK{OK: true, Result: bot})
	case "sendMessage":
		if r.Method != http.MethodPost {
			writeBotError(w, domain.Invalid("sendMessage requires POST"))
			return
		}
		if !s.limiter.allow(bot.ID) {
			writeJSON(w, http.StatusTooManyRequests, botErr{OK: false, ErrorCode: 429, Description: "too many requests"})
			return
		}
		var in struct {
			ChatID      int64  `json:"chat_id"`
			Text        string `json:"text"`
			ClientMsgID string `json:"client_msg_id"`
		}
		if err := decodeJSON(w, r, &in); err != nil {
			writeBotError(w, err)
			return
		}
		m, _, err := s.app.Send(r.Context(), bot, in.ChatID, in.ClientMsgID, in.Text)
		if err != nil {
			writeBotError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, botOK{OK: true, Result: m})
	case "getUpdates":
		q := r.URL.Query()
		offset, _ := strconv.ParseInt(q.Get("offset"), 10, 64)
		limit, _ := strconv.Atoi(q.Get("limit"))
		timeoutSec, _ := strconv.Atoi(q.Get("timeout"))
		wait := time.Duration(timeoutSec) * time.Second
		if wait > s.cfg.BotLongPoll {
			wait = s.cfg.BotLongPoll
		}
		if limit <= 0 || limit > s.cfg.BotUpdatesMax {
			limit = s.cfg.BotUpdatesMax
		}
		// Telegram semantics: offset is the first update id the client has NOT seen.
		msgs, err := s.app.BotUpdates(r.Context(), bot, offset-1, limit, wait)
		if err != nil {
			writeBotError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, botOK{OK: true, Result: msgs})
	case "getChats":
		chats, err := s.app.ListChats(r.Context(), bot)
		if err != nil {
			writeBotError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, botOK{OK: true, Result: chats})
	default:
		writeJSON(w, http.StatusNotFound, botErr{OK: false, ErrorCode: 404, Description: "unknown method"})
	}
}

// botLimiter keeps one token bucket per bot.
type botLimiter struct {
	mu      sync.Mutex
	buckets map[int64]*rate.Limiter
	r       rate.Limit
	b       int
}

func newBotLimiter(perSec float64, burst int) *botLimiter {
	return &botLimiter{buckets: map[int64]*rate.Limiter{}, r: rate.Limit(perSec), b: burst}
}

func (l *botLimiter) allow(id int64) bool {
	l.mu.Lock()
	lim, ok := l.buckets[id]
	if !ok {
		lim = rate.NewLimiter(l.r, l.b)
		l.buckets[id] = lim
	}
	l.mu.Unlock()
	return lim.Allow()
}
