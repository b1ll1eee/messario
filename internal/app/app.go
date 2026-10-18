package app

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/b1ll1eee/nuntio-message/internal/domain"
)

// Limits are the product rules the use cases enforce. main() fills them from config.
type Limits struct {
	MaxGroupMembers   int
	MaxChannelMembers int
	MaxBodyBytes      int
	HistoryPageSize   int
	HistoryPageMax    int
	DirectoryPageMax  int
	SessionTTL        time.Duration
	MaxFileBytes      int64
}

// Deps are the ports the use cases need.
type Deps struct {
	Orgs     OrgRepo
	Users    UserRepo
	Sessions SessionRepo
	Chats    ChatRepo
	Messages MessageRepo
	Audit    AuditRepo
	Files    FileRepo
	Store    FileStore
	Webhooks WebhookRepo
	Dispatch WebhookDispatcher
	License  Licensor
	Hub      Hub
	Hasher   PasswordHasher
	Now      Clock
	Limits   Limits
}

// App is the single entry point to all use cases. Methods are grouped by
// file: auth.go, chats.go, messages.go, bots.go, admin.go.
type App struct {
	Deps
}

// New wires the use cases. A nil Now defaults to time.Now.
func New(d Deps) *App {
	if d.Now == nil {
		d.Now = time.Now
	}
	return &App{Deps: d}
}

var usernameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9_.]{1,31}$`)

func validUsername(s string) bool { return usernameRe.MatchString(s) }

func (a *App) validateBody(body string) (string, error) {
	body = strings.TrimRight(body, " \t\r\n")
	if strings.TrimSpace(body) == "" {
		return "", domain.Invalid("message body is empty")
	}
	if !utf8.ValidString(body) {
		return "", domain.Invalid("message body is not valid UTF-8")
	}
	if len(body) > a.Limits.MaxBodyBytes {
		return "", domain.Invalid("message body exceeds %d bytes", a.Limits.MaxBodyBytes)
	}
	return body, nil
}

// randomToken returns a URL-safe random string of n bytes of entropy.
func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// hashToken is how session and bot tokens are stored: never in clear text.
func hashToken(t string) string {
	sum := sha256.Sum256([]byte(t))
	return hex.EncodeToString(sum[:])
}

func (a *App) audit(ctx context.Context, actor *domain.User, action, target string, meta map[string]any) {
	// Audit failures must not fail the user's action; the adapter logs them.
	_ = a.Audit.Record(ctx, &domain.AuditEntry{
		OrgID:     actor.OrgID,
		ActorID:   actor.ID,
		Action:    action,
		Target:    target,
		Meta:      meta,
		CreatedAt: a.Now(),
	})
}
