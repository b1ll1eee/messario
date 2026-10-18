package app

import (
	"context"
	"net/url"
	"strings"

	"github.com/b1ll1eee/nuntio-message/internal/domain"
)

// SetWebhook registers where a bot wants new messages pushed. Only http(s)
// URLs are accepted; the secret is echoed back in the X-Nuntio-Secret header
// so the receiver can verify the call came from this server.
func (a *App) SetWebhook(ctx context.Context, bot *domain.User, rawURL, secret string) (*domain.Webhook, error) {
	rawURL = strings.TrimSpace(rawURL)
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, domain.Invalid("url must be an absolute http(s) URL")
	}
	if len(secret) > 256 {
		return nil, domain.Invalid("secret too long")
	}
	w := &domain.Webhook{BotID: bot.ID, URL: rawURL, Secret: secret, UpdatedAt: a.Now()}
	if err := a.Webhooks.Set(ctx, w); err != nil {
		return nil, err
	}
	return w, nil
}

func (a *App) DeleteWebhook(ctx context.Context, bot *domain.User) error {
	return a.Webhooks.Delete(ctx, bot.ID)
}

func (a *App) WebhookInfo(ctx context.Context, bot *domain.User) (*domain.Webhook, error) {
	return a.Webhooks.Get(ctx, bot.ID)
}

// WebhookPayload is what a bot's webhook receives for each new message.
type WebhookPayload struct {
	Message domain.Message `json:"message"`
}

// pushToBots delivers a new message to every bot in the chat that has a
// webhook, except the bot that sent it. Called from fanOut.
func (a *App) pushToBots(ctx context.Context, m *domain.Message) {
	if a.Webhooks == nil || a.Dispatch == nil || m.Kind == domain.KindSystem {
		return
	}
	hooks, err := a.Webhooks.ForChat(ctx, m.ChatID)
	if err != nil {
		return
	}
	for _, w := range hooks {
		if w.BotID == m.SenderID {
			continue
		}
		a.Dispatch.Deliver(w, WebhookPayload{Message: *m})
	}
}
