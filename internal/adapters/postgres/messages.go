package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/b1ll1eee/messario/internal/domain"
)

// MessageRepo implements app.MessageRepo.
type MessageRepo struct{ db *DB }

func NewMessageRepo(db *DB) *MessageRepo { return &MessageRepo{db: db} }

const msgCols = `id, chat_id, seq, sender_id, COALESCE(client_msg_id,''), kind, body, attachment, created_at`

func scanMessage(row pgx.Row) (*domain.Message, error) {
	var m domain.Message
	var att []byte
	if err := row.Scan(&m.ID, &m.ChatID, &m.Seq, &m.SenderID, &m.ClientMsgID, &m.Kind, &m.Body, &att, &m.CreatedAt); err != nil {
		return nil, mapErr(err)
	}
	if len(att) > 0 {
		m.Attachment = &domain.Attachment{}
		if err := json.Unmarshal(att, m.Attachment); err != nil {
			return nil, err
		}
	}
	return &m, nil
}

// Append assigns the next seq under a row lock on the chat so sequences are
// gap-free, and returns the existing row for a repeated client_msg_id.
func (r *MessageRepo) Append(ctx context.Context, m *domain.Message) (bool, error) {
	created := false
	err := r.db.tx(ctx, func(tx pgx.Tx) error {
		var lastSeq int64
		if err := tx.QueryRow(ctx, `SELECT last_seq FROM chats WHERE id=$1 FOR UPDATE`, m.ChatID).Scan(&lastSeq); err != nil {
			return mapErr(err)
		}
		if m.ClientMsgID != "" {
			existing, err := scanMessage(tx.QueryRow(ctx, `SELECT `+msgCols+` FROM messages WHERE sender_id=$1 AND client_msg_id=$2`, m.SenderID, m.ClientMsgID))
			if err == nil {
				*m = *existing
				return nil
			}
			if !errors.Is(err, domain.ErrNotFound) {
				return err
			}
		}
		var clientID, att any
		if m.ClientMsgID != "" {
			clientID = m.ClientMsgID
		}
		if m.Attachment != nil {
			b, err := json.Marshal(m.Attachment)
			if err != nil {
				return err
			}
			att = b
		}
		m.Seq = lastSeq + 1
		if err := tx.QueryRow(ctx, `
			INSERT INTO messages (chat_id, seq, sender_id, client_msg_id, kind, body, attachment, created_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id`,
			m.ChatID, m.Seq, m.SenderID, clientID, m.Kind, m.Body, att, m.CreatedAt).Scan(&m.ID); err != nil {
			return mapErr(err)
		}
		if _, err := tx.Exec(ctx, `UPDATE chats SET last_seq=$2 WHERE id=$1`, m.ChatID, m.Seq); err != nil {
			return err
		}
		created = true
		return nil
	})
	return created, err
}

func (r *MessageRepo) History(ctx context.Context, chatID, afterSeq, beforeSeq int64, limit int) ([]domain.Message, error) {
	var rows pgx.Rows
	var err error
	switch {
	case afterSeq > 0:
		rows, err = r.db.pool.Query(ctx, `SELECT `+msgCols+` FROM messages WHERE chat_id=$1 AND seq > $2 ORDER BY seq LIMIT $3`, chatID, afterSeq, limit)
	case beforeSeq > 0:
		rows, err = r.db.pool.Query(ctx, `SELECT * FROM (SELECT `+msgCols+` FROM messages WHERE chat_id=$1 AND seq < $2 ORDER BY seq DESC LIMIT $3) t ORDER BY seq`, chatID, beforeSeq, limit)
	default:
		rows, err = r.db.pool.Query(ctx, `SELECT * FROM (SELECT `+msgCols+` FROM messages WHERE chat_id=$1 ORDER BY seq DESC LIMIT $2) t ORDER BY seq`, chatID, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectMessages(rows)
}

func (r *MessageRepo) UpdatesForBot(ctx context.Context, botID, afterID int64, limit int) ([]domain.Message, error) {
	rows, err := r.db.pool.Query(ctx, `
		SELECT `+msgCols+` FROM messages
		WHERE id > $2 AND sender_id <> $1 AND kind <> 'system'
		  AND chat_id IN (SELECT chat_id FROM chat_members WHERE user_id=$1)
		ORDER BY id LIMIT $3`, botID, afterID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectMessages(rows)
}

func collectMessages(rows pgx.Rows) ([]domain.Message, error) {
	out := []domain.Message{}
	for rows.Next() {
		m, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *m)
	}
	return out, rows.Err()
}

// AuditRepo implements app.AuditRepo.
type AuditRepo struct{ db *DB }

func NewAuditRepo(db *DB) *AuditRepo { return &AuditRepo{db: db} }

func (r *AuditRepo) Record(ctx context.Context, e *domain.AuditEntry) error {
	var meta any
	if len(e.Meta) > 0 {
		b, err := json.Marshal(e.Meta)
		if err != nil {
			return err
		}
		meta = b
	}
	return r.db.pool.QueryRow(ctx, `
		INSERT INTO audit_log (org_id, actor_id, action, target, meta, created_at) VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`,
		e.OrgID, e.ActorID, e.Action, e.Target, meta, e.CreatedAt).Scan(&e.ID)
}

func (r *AuditRepo) List(ctx context.Context, orgID int64, limit int) ([]domain.AuditEntry, error) {
	rows, err := r.db.pool.Query(ctx, `SELECT id, org_id, actor_id, action, target, meta, created_at FROM audit_log WHERE org_id=$1 ORDER BY id DESC LIMIT $2`, orgID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.AuditEntry{}
	for rows.Next() {
		var e domain.AuditEntry
		var meta []byte
		if err := rows.Scan(&e.ID, &e.OrgID, &e.ActorID, &e.Action, &e.Target, &meta, &e.CreatedAt); err != nil {
			return nil, err
		}
		if len(meta) > 0 {
			_ = json.Unmarshal(meta, &e.Meta)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
