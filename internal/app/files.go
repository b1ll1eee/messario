package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"path"
	"strings"

	"github.com/b1ll1eee/messario/internal/domain"
)

// UploadFile stores the bytes, records the file and posts a file message.
// ClientMsgID keeps retried uploads idempotent at the message level.
func (a *App) UploadFile(ctx context.Context, sender *domain.User, chatID int64, name, contentType string, size int64, r io.Reader, clientMsgID string) (*domain.Message, error) {
	c, me, err := a.GetChat(ctx, sender, chatID)
	if err != nil {
		return nil, err
	}
	if !me.CanPost(c.Type) {
		return nil, domain.ErrForbidden
	}
	name = sanitizeFilename(name)
	if size <= 0 {
		return nil, domain.Invalid("file is empty")
	}
	if size > a.Limits.MaxFileBytes {
		return nil, domain.Invalid("file exceeds %d bytes", a.Limits.MaxFileBytes)
	}
	if contentType == "" {
		contentType = mime.TypeByExtension(path.Ext(name))
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	token, err := randomToken(16)
	if err != nil {
		return nil, err
	}
	key := fmt.Sprintf("org%d/chat%d/%s%s", sender.OrgID, chatID, token, path.Ext(name))
	if err := a.Store.Put(ctx, key, io.LimitReader(r, size), size, contentType); err != nil {
		return nil, err
	}
	f := &domain.File{OrgID: sender.OrgID, ChatID: chatID, UploaderID: sender.ID, Name: name, Size: size, Mime: contentType, StorageKey: key, CreatedAt: a.Now()}
	if err := a.Files.Create(ctx, f); err != nil {
		_ = a.Store.Delete(ctx, key)
		return nil, err
	}
	m := &domain.Message{
		ChatID: chatID, SenderID: sender.ID, ClientMsgID: strings.TrimSpace(clientMsgID), Kind: domain.KindFile, Body: name,
		Attachment: &domain.Attachment{FileID: f.ID, Name: name, Size: size, Mime: contentType}, CreatedAt: a.Now(),
	}
	created, err := a.Messages.Append(ctx, m)
	if err != nil {
		return nil, err
	}
	if !created {
		// A retry of an upload that already went through: drop the duplicate bytes.
		_ = a.Store.Delete(ctx, key)
		_ = a.Files.Delete(ctx, f.ID)
		return m, nil
	}
	_ = a.Chats.SetReadSeq(ctx, chatID, sender.ID, m.Seq)
	a.fanOut(ctx, chatID, m)
	return m, nil
}

// OpenFile checks membership and returns the file's metadata and bytes.
func (a *App) OpenFile(ctx context.Context, viewer *domain.User, fileID int64) (*domain.File, io.ReadCloser, error) {
	f, err := a.Files.ByID(ctx, fileID)
	if errors.Is(err, domain.ErrNotFound) || (err == nil && f.OrgID != viewer.OrgID) {
		return nil, nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, nil, err
	}
	if _, err := a.Chats.Member(ctx, f.ChatID, viewer.ID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, nil, domain.ErrForbidden
		}
		return nil, nil, err
	}
	rc, err := a.Store.Get(ctx, f.StorageKey)
	if err != nil {
		return nil, nil, err
	}
	return f, rc, nil
}

// sanitizeFilename keeps the base name only and bounds its length.
func sanitizeFilename(name string) string {
	name = path.Base(strings.ReplaceAll(strings.TrimSpace(name), "\\", "/"))
	if name == "." || name == "/" || name == "" {
		name = "file"
	}
	const max = 180
	if len(name) > max {
		ext := path.Ext(name)
		name = name[:max-len(ext)] + ext
	}
	return name
}
