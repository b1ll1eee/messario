package app

import (
	"context"
	"strings"

	"github.com/b1ll1eee/nuntio-message/internal/domain"
)

// Send appends a text message. ClientMsgID makes retries idempotent: a
// duplicate returns the original message with created == false.
func (a *App) Send(ctx context.Context, sender *domain.User, chatID int64, clientMsgID, body string) (*domain.Message, bool, error) {
	c, me, err := a.GetChat(ctx, sender, chatID)
	if err != nil {
		return nil, false, err
	}
	if !me.CanPost(c.Type) {
		return nil, false, domain.ErrForbidden
	}
	body, err = a.validateBody(body)
	if err != nil {
		return nil, false, err
	}
	clientMsgID = strings.TrimSpace(clientMsgID)
	if len(clientMsgID) > 64 {
		return nil, false, domain.Invalid("client_msg_id too long")
	}
	m := &domain.Message{
		ChatID:      chatID,
		SenderID:    sender.ID,
		ClientMsgID: clientMsgID,
		Kind:        domain.KindText,
		Body:        body,
		CreatedAt:   a.Now(),
	}
	created, err := a.Messages.Append(ctx, m)
	if err != nil {
		return nil, false, err
	}
	if created {
		a.fanOut(ctx, chatID, m)
	}
	return m, created, nil
}

// History pages through a chat. afterSeq > 0 syncs forward (gap fill);
// beforeSeq > 0 pages backward from a point; both zero returns the newest page.
func (a *App) History(ctx context.Context, viewer *domain.User, chatID, afterSeq, beforeSeq int64, limit int) ([]domain.Message, error) {
	if _, _, err := a.GetChat(ctx, viewer, chatID); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = a.Limits.HistoryPageSize
	}
	if limit > a.Limits.HistoryPageMax {
		limit = a.Limits.HistoryPageMax
	}
	return a.Messages.History(ctx, chatID, afterSeq, beforeSeq, limit)
}

// ReadEvent is what the reader's own devices (and the peer, in direct chats) receive.
type ReadEvent struct {
	ChatID int64 `json:"chat_id"`
	UserID int64 `json:"user_id"`
	Seq    int64 `json:"seq"`
}

// MarkRead advances the viewer's read pointer. Clients debounce this call.
func (a *App) MarkRead(ctx context.Context, viewer *domain.User, chatID, seq int64) error {
	c, me, err := a.GetChat(ctx, viewer, chatID)
	if err != nil {
		return err
	}
	if seq > c.LastSeq {
		seq = c.LastSeq
	}
	if seq <= me.ReadSeq {
		return nil
	}
	if err := a.Chats.SetReadSeq(ctx, chatID, viewer.ID, seq); err != nil {
		return err
	}
	ev := Event{Type: "read", Payload: ReadEvent{ChatID: chatID, UserID: viewer.ID, Seq: seq}}
	targets := []int64{viewer.ID}
	if c.Type == domain.ChatDirect {
		// 1:1 receipts are live; group receipts are fetched on demand (ReadInfo)
		// to avoid N×N fan-out.
		if ids, err := a.Chats.MemberIDs(ctx, chatID); err == nil {
			targets = ids
		}
	}
	a.Hub.Publish(targets, ev)
	return nil
}

// ReadInfo tells how many (and which) members have read up to seq.
type ReadInfo struct {
	Count   int             `json:"count"`
	Total   int             `json:"total"`
	Readers []domain.Member `json:"readers"`
}

func (a *App) ReadInfo(ctx context.Context, viewer *domain.User, chatID, seq int64, limit int) (*ReadInfo, error) {
	if _, _, err := a.GetChat(ctx, viewer, chatID); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > a.Limits.HistoryPageMax {
		limit = a.Limits.HistoryPageMax
	}
	count, err := a.Chats.ReadCount(ctx, chatID, seq)
	if err != nil {
		return nil, err
	}
	total, err := a.Chats.MemberCount(ctx, chatID)
	if err != nil {
		return nil, err
	}
	readers, err := a.Chats.Readers(ctx, chatID, seq, limit)
	if err != nil {
		return nil, err
	}
	return &ReadInfo{Count: count, Total: total, Readers: readers}, nil
}

// postSystem appends a system message (member joined, renamed, ...).
func (a *App) postSystem(ctx context.Context, c *domain.Chat, actorID int64, body string) {
	m := &domain.Message{ChatID: c.ID, SenderID: actorID, Kind: domain.KindSystem, Body: body, CreatedAt: a.Now()}
	if created, err := a.Messages.Append(ctx, m); err == nil && created {
		a.fanOut(ctx, c.ID, m)
	}
}

// fanOut pushes a new message to every member's live connections.
func (a *App) fanOut(ctx context.Context, chatID int64, m *domain.Message) {
	ids, err := a.Chats.MemberIDs(ctx, chatID)
	if err != nil {
		return
	}
	a.Hub.Publish(ids, Event{Type: "msg", Payload: m})
}
