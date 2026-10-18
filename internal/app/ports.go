// Package app holds the use cases. It depends only on domain and on the
// ports declared here; adapters (postgres, http, ws, ...) implement them.
package app

import (
	"context"
	"io"
	"time"

	"github.com/b1ll1eee/messario/internal/domain"
)

// OrgRepo stores the installation's organization.
type OrgRepo interface {
	Get(ctx context.Context) (*domain.Org, error) // ErrNotFound before init
	Create(ctx context.Context, org *domain.Org) error
}

// UserRepo stores people and bots.
type UserRepo interface {
	Create(ctx context.Context, u *domain.User, passwordHash string) error
	ByID(ctx context.Context, id int64) (*domain.User, error)
	// ByUsername returns the user and its password hash for login.
	ByUsername(ctx context.Context, orgID int64, username string) (*domain.User, string, error)
	// List returns active and suspended humans and bots matching query (prefix on
	// username/display name), ordered by display name.
	List(ctx context.Context, orgID int64, query string, limit int) ([]domain.User, error)
	Update(ctx context.Context, u *domain.User) error
	SetPassword(ctx context.Context, id int64, hash string) error
	SetBotToken(ctx context.Context, id int64, tokenHash string) error
	ByBotToken(ctx context.Context, tokenHash string) (*domain.User, error)
	// CountActiveHumans counts non-bot users with status active (licensed seats in use).
	CountActiveHumans(ctx context.Context, orgID int64) (int, error)
}

// SessionRepo stores login sessions keyed by a hash of the cookie token.
type SessionRepo interface {
	Create(ctx context.Context, tokenHash string, userID int64, expiresAt time.Time, userAgent string) error
	UserByToken(ctx context.Context, tokenHash string, now time.Time) (*domain.User, error)
	Delete(ctx context.Context, tokenHash string) error
	DeleteForUser(ctx context.Context, userID int64) error
}

// ChatRepo stores chats and memberships.
type ChatRepo interface {
	Create(ctx context.Context, c *domain.Chat, members []domain.Member) error
	ByID(ctx context.Context, id int64) (*domain.Chat, error)
	ByDirectKey(ctx context.Context, orgID int64, key string) (*domain.Chat, error)
	UpdateTitle(ctx context.Context, id int64, title string) error
	ListForUser(ctx context.Context, userID int64) ([]domain.ChatSummary, error)
	Members(ctx context.Context, chatID int64) ([]domain.Member, error)
	MemberIDs(ctx context.Context, chatID int64) ([]int64, error)
	Member(ctx context.Context, chatID, userID int64) (*domain.Member, error)
	MemberCount(ctx context.Context, chatID int64) (int, error)
	AddMembers(ctx context.Context, chatID int64, members []domain.Member) error
	RemoveMember(ctx context.Context, chatID, userID int64) error
	// SetReadSeq advances read_seq (never moves it backwards).
	SetReadSeq(ctx context.Context, chatID, userID, seq int64) error
	// ReadCount counts members whose read_seq >= seq.
	ReadCount(ctx context.Context, chatID, seq int64) (int, error)
	// Readers lists members whose read_seq >= seq.
	Readers(ctx context.Context, chatID, seq int64, limit int) ([]domain.Member, error)
}

// MessageRepo stores messages.
type MessageRepo interface {
	// Append assigns Seq, ID and CreatedAt. When a message with the same
	// (SenderID, ClientMsgID) already exists it returns that message and
	// created == false, so client retries are idempotent.
	Append(ctx context.Context, m *domain.Message) (created bool, err error)
	// History returns messages with seq > afterSeq and (beforeSeq == 0 or seq < beforeSeq),
	// oldest first, at most limit rows.
	History(ctx context.Context, chatID, afterSeq, beforeSeq int64, limit int) ([]domain.Message, error)
	// UpdatesForBot returns messages with id > afterID in chats the bot belongs to,
	// excluding the bot's own messages, oldest first.
	UpdatesForBot(ctx context.Context, botID, afterID int64, limit int) ([]domain.Message, error)
}

// FileRepo stores file metadata.
type FileRepo interface {
	Create(ctx context.Context, f *domain.File) error
	ByID(ctx context.Context, id int64) (*domain.File, error)
	Delete(ctx context.Context, id int64) error
}

// FileStore stores file bytes (local disk, S3, MinIO, R2).
type FileStore interface {
	Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
}

// WebhookRepo stores bot webhook settings.
type WebhookRepo interface {
	Set(ctx context.Context, w *domain.Webhook) error
	Get(ctx context.Context, botID int64) (*domain.Webhook, error)
	Delete(ctx context.Context, botID int64) error
	// ForChat returns the webhooks of every bot that is a member of chatID.
	ForChat(ctx context.Context, chatID int64) ([]domain.Webhook, error)
}

// WebhookDispatcher delivers a payload to a bot's webhook asynchronously
// (with retries); it must not block the caller.
type WebhookDispatcher interface {
	Deliver(w domain.Webhook, payload any)
}

// Licensor answers how many human seats this install may have.
type Licensor interface {
	// Seats returns the allowed number of active human users (0 = unlimited)
	// and a human-readable status for admins.
	Seats() (seats int, status string)
}

// AuditRepo appends audit entries.
type AuditRepo interface {
	Record(ctx context.Context, e *domain.AuditEntry) error
	List(ctx context.Context, orgID int64, limit int) ([]domain.AuditEntry, error)
}

// Event is what gets pushed to connected clients.
type Event struct {
	Type    string `json:"t"`
	Payload any    `json:"p,omitempty"`
}

// Hub delivers events to the live connections of the given users. An
// in-memory implementation serves one instance; a NATS one serves many.
type Hub interface {
	Publish(userIDs []int64, ev Event)
	// Wait blocks until an event arrives for userID or ctx is done.
	// Used by bot long-polling.
	Wait(ctx context.Context, userID int64) bool
}

// PasswordHasher abstracts bcrypt so tests can use a cheap hasher.
type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) bool
}

// Clock returns the current time; injectable for tests.
type Clock func() time.Time
