package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/b1ll1eee/nuntio-message/internal/domain"
)

// ChatRepo implements app.ChatRepo.
type ChatRepo struct{ db *DB }

func NewChatRepo(db *DB) *ChatRepo { return &ChatRepo{db: db} }

const chatCols = `id, org_id, type, title, COALESCE(direct_key,''), created_by, last_seq, created_at`

func scanChat(row pgx.Row) (*domain.Chat, error) {
	var c domain.Chat
	if err := row.Scan(&c.ID, &c.OrgID, &c.Type, &c.Title, &c.DirectKey, &c.CreatedBy, &c.LastSeq, &c.CreatedAt); err != nil {
		return nil, mapErr(err)
	}
	return &c, nil
}

func (r *ChatRepo) Create(ctx context.Context, c *domain.Chat, members []domain.Member) error {
	return r.db.tx(ctx, func(tx pgx.Tx) error {
		var key any
		if c.DirectKey != "" {
			key = c.DirectKey
		}
		if err := tx.QueryRow(ctx, `
			INSERT INTO chats (org_id, type, title, direct_key, created_by, created_at)
			VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`,
			c.OrgID, c.Type, c.Title, key, c.CreatedBy, c.CreatedAt).Scan(&c.ID); err != nil {
			return mapErr(err)
		}
		return insertMembers(ctx, tx, c.ID, members)
	})
}

func insertMembers(ctx context.Context, tx pgx.Tx, chatID int64, members []domain.Member) error {
	rows := make([][]any, 0, len(members))
	for _, m := range members {
		rows = append(rows, []any{chatID, m.UserID, m.Role, m.JoinedAt})
	}
	_, err := tx.CopyFrom(ctx, pgx.Identifier{"chat_members"}, []string{"chat_id", "user_id", "role", "joined_at"}, pgx.CopyFromRows(rows))
	return mapErr(err)
}

func (r *ChatRepo) ByID(ctx context.Context, id int64) (*domain.Chat, error) {
	return scanChat(r.db.pool.QueryRow(ctx, `SELECT `+chatCols+` FROM chats WHERE id=$1`, id))
}

func (r *ChatRepo) ByDirectKey(ctx context.Context, orgID int64, key string) (*domain.Chat, error) {
	return scanChat(r.db.pool.QueryRow(ctx, `SELECT `+chatCols+` FROM chats WHERE org_id=$1 AND direct_key=$2`, orgID, key))
}

func (r *ChatRepo) UpdateTitle(ctx context.Context, id int64, title string) error {
	_, err := r.db.pool.Exec(ctx, `UPDATE chats SET title=$2 WHERE id=$1`, id, title)
	return err
}

// ListForUser builds the chat list in one query: membership, unread count,
// the newest message, the peer for direct chats and the member count.
func (r *ChatRepo) ListForUser(ctx context.Context, userID int64) ([]domain.ChatSummary, error) {
	rows, err := r.db.pool.Query(ctx, `
		SELECT c.id, c.org_id, c.type,
		       CASE WHEN c.type='direct' THEN COALESCE(peer.display_name, '') ELSE c.title END AS title,
		       COALESCE(peer.id, 0) AS peer_id,
		       c.created_by, c.last_seq, c.created_at,
		       m.read_seq, GREATEST(c.last_seq - m.read_seq, 0) AS unread,
		       COALESCE(lm.body,''), COALESCE(lm.kind,''), COALESCE(lm.sender_id,0), COALESCE(lm.created_at, c.created_at) AS last_at,
		       (SELECT count(*) FROM chat_members mc WHERE mc.chat_id=c.id) AS member_count
		FROM chat_members m
		JOIN chats c ON c.id = m.chat_id
		LEFT JOIN messages lm ON lm.chat_id = c.id AND lm.seq = c.last_seq
		LEFT JOIN LATERAL (
		    SELECT u.id, u.display_name FROM chat_members m2 JOIN users u ON u.id = m2.user_id
		    WHERE c.type='direct' AND m2.chat_id = c.id AND m2.user_id <> m.user_id LIMIT 1
		) peer ON true
		WHERE m.user_id = $1
		ORDER BY last_at DESC, c.id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.ChatSummary{}
	for rows.Next() {
		var s domain.ChatSummary
		if err := rows.Scan(&s.ID, &s.OrgID, &s.Type, &s.Title, &s.PeerID, &s.CreatedBy, &s.LastSeq, &s.CreatedAt,
			&s.ReadSeq, &s.Unread, &s.LastBody, &s.LastKind, &s.LastSender, &s.LastAt, &s.MemberCount); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *ChatRepo) Members(ctx context.Context, chatID int64) ([]domain.Member, error) {
	rows, err := r.db.pool.Query(ctx, `SELECT chat_id, user_id, role, read_seq, joined_at FROM chat_members WHERE chat_id=$1 ORDER BY joined_at, user_id`, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectMembers(rows)
}

func collectMembers(rows pgx.Rows) ([]domain.Member, error) {
	out := []domain.Member{}
	for rows.Next() {
		var m domain.Member
		if err := rows.Scan(&m.ChatID, &m.UserID, &m.Role, &m.ReadSeq, &m.JoinedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *ChatRepo) MemberIDs(ctx context.Context, chatID int64) ([]int64, error) {
	rows, err := r.db.pool.Query(ctx, `SELECT user_id FROM chat_members WHERE chat_id=$1`, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *ChatRepo) Member(ctx context.Context, chatID, userID int64) (*domain.Member, error) {
	var m domain.Member
	err := r.db.pool.QueryRow(ctx, `SELECT chat_id, user_id, role, read_seq, joined_at FROM chat_members WHERE chat_id=$1 AND user_id=$2`, chatID, userID).
		Scan(&m.ChatID, &m.UserID, &m.Role, &m.ReadSeq, &m.JoinedAt)
	if err != nil {
		return nil, mapErr(err)
	}
	return &m, nil
}

func (r *ChatRepo) MemberCount(ctx context.Context, chatID int64) (int, error) {
	var n int
	err := r.db.pool.QueryRow(ctx, `SELECT count(*) FROM chat_members WHERE chat_id=$1`, chatID).Scan(&n)
	return n, err
}

func (r *ChatRepo) AddMembers(ctx context.Context, chatID int64, members []domain.Member) error {
	return r.db.tx(ctx, func(tx pgx.Tx) error {
		for _, m := range members {
			if _, err := tx.Exec(ctx, `
				INSERT INTO chat_members (chat_id, user_id, role, joined_at) VALUES ($1,$2,$3,$4)
				ON CONFLICT (chat_id, user_id) DO NOTHING`, chatID, m.UserID, m.Role, m.JoinedAt); err != nil {
				return mapErr(err)
			}
		}
		return nil
	})
}

func (r *ChatRepo) RemoveMember(ctx context.Context, chatID, userID int64) error {
	tag, err := r.db.pool.Exec(ctx, `DELETE FROM chat_members WHERE chat_id=$1 AND user_id=$2`, chatID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *ChatRepo) SetReadSeq(ctx context.Context, chatID, userID, seq int64) error {
	_, err := r.db.pool.Exec(ctx, `UPDATE chat_members SET read_seq=$3 WHERE chat_id=$1 AND user_id=$2 AND read_seq < $3`, chatID, userID, seq)
	return err
}

func (r *ChatRepo) ReadCount(ctx context.Context, chatID, seq int64) (int, error) {
	var n int
	err := r.db.pool.QueryRow(ctx, `SELECT count(*) FROM chat_members WHERE chat_id=$1 AND read_seq >= $2`, chatID, seq).Scan(&n)
	return n, err
}

func (r *ChatRepo) Readers(ctx context.Context, chatID, seq int64, limit int) ([]domain.Member, error) {
	rows, err := r.db.pool.Query(ctx, `SELECT chat_id, user_id, role, read_seq, joined_at FROM chat_members WHERE chat_id=$1 AND read_seq >= $2 ORDER BY user_id LIMIT $3`, chatID, seq, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectMembers(rows)
}
