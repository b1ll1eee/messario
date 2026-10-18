package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/b1ll1eee/nuntio-message/internal/domain"
)

// CreateBot registers a bot user and returns its token exactly once.
// Token shape is "<bot id>:<random>", like Telegram, so support staff can
// tell which bot a leaked token belongs to without being able to use it.
func (a *App) CreateBot(ctx context.Context, admin *domain.User, username, displayName string) (*domain.User, string, error) {
	if !admin.IsAdmin() {
		return nil, "", domain.ErrForbidden
	}
	username = strings.ToLower(strings.TrimSpace(username))
	if !validUsername(username) {
		return nil, "", domain.Invalid("username must be 2-32 chars of a-z, 0-9, '_' or '.'")
	}
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		displayName = username
	}
	b := &domain.User{
		OrgID: admin.OrgID, Kind: domain.UserBot, Username: username, DisplayName: displayName,
		Role: domain.RoleMember, Status: domain.StatusActive, CreatedBy: admin.ID, CreatedAt: a.Now(),
	}
	if err := a.Users.Create(ctx, b, ""); err != nil {
		return nil, "", err
	}
	token, err := a.RegenerateBotToken(ctx, admin, b.ID)
	if err != nil {
		return nil, "", err
	}
	a.audit(ctx, admin, "bot.create", fmt.Sprint(b.ID), map[string]any{"username": username})
	return b, token, nil
}

// RegenerateBotToken replaces a bot's token; the old one stops working at once.
func (a *App) RegenerateBotToken(ctx context.Context, admin *domain.User, botID int64) (string, error) {
	if !admin.IsAdmin() {
		return "", domain.ErrForbidden
	}
	b, err := a.Users.ByID(ctx, botID)
	if err != nil {
		return "", err
	}
	if !b.IsBot() || b.OrgID != admin.OrgID {
		return "", domain.ErrNotFound
	}
	secret, err := randomToken(24)
	if err != nil {
		return "", err
	}
	token := fmt.Sprintf("%d:%s", b.ID, secret)
	if err := a.Users.SetBotToken(ctx, b.ID, hashToken(token)); err != nil {
		return "", err
	}
	a.audit(ctx, admin, "bot.token_regenerated", fmt.Sprint(b.ID), nil)
	return token, nil
}

// BotByToken authenticates a bot API call.
func (a *App) BotByToken(ctx context.Context, token string) (*domain.User, error) {
	if token == "" {
		return nil, domain.ErrUnauthorized
	}
	b, err := a.Users.ByBotToken(ctx, hashToken(token))
	if errors.Is(err, domain.ErrNotFound) {
		return nil, domain.ErrUnauthorized
	}
	if err != nil {
		return nil, err
	}
	if !b.IsActive() {
		return nil, domain.ErrSuspended
	}
	return b, nil
}

// BotUpdates returns messages addressed to the bot (chats it belongs to,
// excluding its own) with id > offset. When none are pending it waits up to
// wait for one to arrive, so clients can long-poll without a webhook.
func (a *App) BotUpdates(ctx context.Context, bot *domain.User, offset int64, limit int, wait time.Duration) ([]domain.Message, error) {
	if limit <= 0 || limit > a.Limits.HistoryPageMax {
		limit = a.Limits.HistoryPageMax
	}
	msgs, err := a.Messages.UpdatesForBot(ctx, bot.ID, offset, limit)
	if err != nil || len(msgs) > 0 || wait <= 0 {
		return msgs, err
	}
	wctx, cancel := context.WithTimeout(ctx, wait)
	defer cancel()
	if !a.Hub.Wait(wctx, bot.ID) {
		return []domain.Message{}, nil
	}
	return a.Messages.UpdatesForBot(ctx, bot.ID, offset, limit)
}
