// Package domain holds the core types and rules of Messario. It has no
// dependency on any framework, database or transport.
package domain

import (
	"errors"
	"fmt"
	"time"
)

// Sentinel errors. Adapters map these to transport-specific responses.
var (
	ErrNotFound     = errors.New("not found")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrConflict     = errors.New("conflict")
	ErrSuspended    = errors.New("account suspended")
)

// ValidationError describes a bad input. It is safe to show to the client.
type ValidationError struct{ Msg string }

func (e *ValidationError) Error() string { return e.Msg }

// Invalid builds a ValidationError.
func Invalid(format string, args ...any) error {
	return &ValidationError{Msg: fmt.Sprintf(format, args...)}
}

// IsInvalid reports whether err is a ValidationError.
func IsInvalid(err error) bool {
	var ve *ValidationError
	return errors.As(err, &ve)
}

// Org is one installation's organization. Every row carries an org_id so a
// multi-tenant deployment stays possible even though v1 is one org per install.
type Org struct {
	ID          int64
	Name        string
	EmailDomain string // when set, invited emails must end with this domain
	CreatedAt   time.Time
}

type UserKind string

const (
	UserHuman UserKind = "user"
	UserBot   UserKind = "bot"
)

type Role string

const (
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
)

type UserStatus string

const (
	StatusActive    UserStatus = "active"
	StatusSuspended UserStatus = "suspended"
)

// User is a person or a bot. Bots have Kind == UserBot and no password.
type User struct {
	ID          int64      `json:"id"`
	OrgID       int64      `json:"-"`
	Kind        UserKind   `json:"kind"`
	Username    string     `json:"username"`
	DisplayName string     `json:"display_name"`
	Email       string     `json:"email,omitempty"`
	Department  string     `json:"department,omitempty"`
	Title       string     `json:"title,omitempty"`
	Role        Role       `json:"role"`
	Status      UserStatus `json:"status"`
	CreatedBy   int64      `json:"-"`
	CreatedAt   time.Time  `json:"created_at"`
}

func (u *User) IsAdmin() bool  { return u.Role == RoleAdmin }
func (u *User) IsBot() bool    { return u.Kind == UserBot }
func (u *User) IsActive() bool { return u.Status == StatusActive }

type ChatType string

const (
	ChatDirect  ChatType = "direct"
	ChatGroup   ChatType = "group"
	ChatChannel ChatType = "channel"
)

// Chat is a direct conversation, a group, or a broadcast channel.
// LastSeq is the per-chat sequence of the newest message; clients sync with
// "give me everything after seq N".
type Chat struct {
	ID        int64     `json:"id"`
	OrgID     int64     `json:"-"`
	Type      ChatType  `json:"type"`
	Title     string    `json:"title"`
	DirectKey string    `json:"-"`
	CreatedBy int64     `json:"created_by"`
	LastSeq   int64     `json:"last_seq"`
	CreatedAt time.Time `json:"created_at"`
}

// DirectKey is the canonical pair key "minID:maxID" that makes a direct
// chat between two users unique.
func DirectKey(a, b int64) string {
	if a > b {
		a, b = b, a
	}
	return fmt.Sprintf("%d:%d", a, b)
}

type MemberRole string

const (
	MemberOwner  MemberRole = "owner"
	MemberAdmin  MemberRole = "admin"
	MemberMember MemberRole = "member"
)

// Member links a user to a chat. ReadSeq is the newest seq the user has read.
type Member struct {
	ChatID   int64      `json:"chat_id"`
	UserID   int64      `json:"user_id"`
	Role     MemberRole `json:"role"`
	ReadSeq  int64      `json:"read_seq"`
	JoinedAt time.Time  `json:"joined_at"`
}

// CanPost reports whether this member may send messages given the chat type.
// Channels are broadcast-only: owners and admins post, members read.
func (m *Member) CanPost(t ChatType) bool {
	if t != ChatChannel {
		return true
	}
	return m.Role == MemberOwner || m.Role == MemberAdmin
}

// CanManage reports whether this member may add/remove members or edit the chat.
func (m *Member) CanManage() bool {
	return m.Role == MemberOwner || m.Role == MemberAdmin
}

type MessageKind string

const (
	KindText   MessageKind = "text"
	KindSystem MessageKind = "system"
	KindFile   MessageKind = "file"
)

// Attachment describes a file carried by a message.
type Attachment struct {
	FileID int64  `json:"file_id"`
	Name   string `json:"name"`
	Size   int64  `json:"size"`
	Mime   string `json:"mime"`
}

// Message is one entry in a chat. (ChatID, Seq) is the primary identity;
// ID is a global, monotonically increasing id used for cross-chat cursors
// such as a bot's getUpdates offset.
type Message struct {
	ID          int64       `json:"id"`
	ChatID      int64       `json:"chat_id"`
	Seq         int64       `json:"seq"`
	SenderID    int64       `json:"sender_id"`
	ClientMsgID string      `json:"client_msg_id,omitempty"`
	Kind        MessageKind `json:"kind"`
	Body        string      `json:"body"`
	Attachment  *Attachment `json:"attachment,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
}

// ChatSummary is what a chat list row needs: the chat, the viewer's unread
// count and a preview of the newest message.
type ChatSummary struct {
	Chat
	PeerID      int64     `json:"peer_id,omitempty"` // direct chats: the other user
	ReadSeq     int64     `json:"read_seq"`
	Unread      int64     `json:"unread"`
	LastBody    string    `json:"last_body"`
	LastKind    string    `json:"last_kind"`
	LastSender  int64     `json:"last_sender_id"`
	LastAt      time.Time `json:"last_at"`
	MemberCount int       `json:"member_count"`
}

// File is an uploaded attachment. StorageKey locates the bytes in the FileStore.
type File struct {
	ID         int64     `json:"id"`
	OrgID      int64     `json:"-"`
	ChatID     int64     `json:"chat_id"`
	UploaderID int64     `json:"uploader_id"`
	Name       string    `json:"name"`
	Size       int64     `json:"size"`
	Mime       string    `json:"mime"`
	StorageKey string    `json:"-"`
	CreatedAt  time.Time `json:"created_at"`
}

// Webhook is where a bot wants incoming messages pushed.
type Webhook struct {
	BotID     int64     `json:"bot_id"`
	URL       string    `json:"url"`
	Secret    string    `json:"-"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AuditEntry records an administrative or security-relevant action.
type AuditEntry struct {
	ID        int64          `json:"id"`
	OrgID     int64          `json:"-"`
	ActorID   int64          `json:"actor_id"`
	Action    string         `json:"action"`
	Target    string         `json:"target"`
	Meta      map[string]any `json:"meta,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}
