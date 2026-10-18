package postgres

import (
	"context"

	"github.com/b1ll1eee/messario/internal/domain"
)

// FileRepo implements app.FileRepo.
type FileRepo struct{ db *DB }

func NewFileRepo(db *DB) *FileRepo { return &FileRepo{db: db} }

func (r *FileRepo) Create(ctx context.Context, f *domain.File) error {
	return mapErr(r.db.pool.QueryRow(ctx, `
		INSERT INTO files (org_id, chat_id, uploader_id, name, size, mime, storage_key, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id`,
		f.OrgID, f.ChatID, f.UploaderID, f.Name, f.Size, f.Mime, f.StorageKey, f.CreatedAt).Scan(&f.ID))
}

func (r *FileRepo) ByID(ctx context.Context, id int64) (*domain.File, error) {
	var f domain.File
	err := r.db.pool.QueryRow(ctx, `SELECT id, org_id, chat_id, uploader_id, name, size, mime, storage_key, created_at FROM files WHERE id=$1`, id).
		Scan(&f.ID, &f.OrgID, &f.ChatID, &f.UploaderID, &f.Name, &f.Size, &f.Mime, &f.StorageKey, &f.CreatedAt)
	if err != nil {
		return nil, mapErr(err)
	}
	return &f, nil
}

func (r *FileRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.pool.Exec(ctx, `DELETE FROM files WHERE id=$1`, id)
	return err
}

// WebhookRepo implements app.WebhookRepo.
type WebhookRepo struct{ db *DB }

func NewWebhookRepo(db *DB) *WebhookRepo { return &WebhookRepo{db: db} }

func (r *WebhookRepo) Set(ctx context.Context, w *domain.Webhook) error {
	_, err := r.db.pool.Exec(ctx, `
		INSERT INTO bot_webhooks (bot_id, url, secret, updated_at) VALUES ($1,$2,$3,$4)
		ON CONFLICT (bot_id) DO UPDATE SET url=EXCLUDED.url, secret=EXCLUDED.secret, updated_at=EXCLUDED.updated_at`,
		w.BotID, w.URL, w.Secret, w.UpdatedAt)
	return mapErr(err)
}

func (r *WebhookRepo) Get(ctx context.Context, botID int64) (*domain.Webhook, error) {
	var w domain.Webhook
	err := r.db.pool.QueryRow(ctx, `SELECT bot_id, url, secret, updated_at FROM bot_webhooks WHERE bot_id=$1`, botID).
		Scan(&w.BotID, &w.URL, &w.Secret, &w.UpdatedAt)
	if err != nil {
		return nil, mapErr(err)
	}
	return &w, nil
}

func (r *WebhookRepo) Delete(ctx context.Context, botID int64) error {
	_, err := r.db.pool.Exec(ctx, `DELETE FROM bot_webhooks WHERE bot_id=$1`, botID)
	return err
}

func (r *WebhookRepo) ForChat(ctx context.Context, chatID int64) ([]domain.Webhook, error) {
	rows, err := r.db.pool.Query(ctx, `
		SELECT w.bot_id, w.url, w.secret, w.updated_at
		FROM bot_webhooks w
		JOIN chat_members m ON m.user_id = w.bot_id
		JOIN users u ON u.id = w.bot_id AND u.status = 'active'
		WHERE m.chat_id = $1`, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Webhook
	for rows.Next() {
		var w domain.Webhook
		if err := rows.Scan(&w.BotID, &w.URL, &w.Secret, &w.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}
