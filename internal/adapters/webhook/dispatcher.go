// Package webhook delivers bot webhook calls in the background with retries.
package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/b1ll1eee/nuntio-message/internal/domain"
)

// Config bounds delivery work.
type Config struct {
	Timeout     time.Duration // per attempt
	Attempts    int           // total tries
	Backoff     time.Duration // first retry delay, doubles each time
	Concurrency int           // in-flight deliveries across all bots
}

type Dispatcher struct {
	cfg    Config
	client *http.Client
	sem    chan struct{}
	log    *slog.Logger
}

func New(cfg Config, log *slog.Logger) *Dispatcher {
	return &Dispatcher{
		cfg:    cfg,
		client: &http.Client{Timeout: cfg.Timeout},
		sem:    make(chan struct{}, cfg.Concurrency),
		log:    log,
	}
}

// Deliver POSTs the payload as JSON. It returns immediately; the call runs
// on a goroutine bounded by Concurrency. A 2xx ends it; anything else is
// retried with exponential backoff up to Attempts.
func (d *Dispatcher) Deliver(w domain.Webhook, payload any) {
	body, err := json.Marshal(payload)
	if err != nil {
		d.log.Error("webhook: encode", "err", err)
		return
	}
	go func() {
		d.sem <- struct{}{}
		defer func() { <-d.sem }()
		delay := d.cfg.Backoff
		for attempt := 1; attempt <= d.cfg.Attempts; attempt++ {
			status, err := d.post(w, body)
			if err == nil && status >= 200 && status < 300 {
				return
			}
			d.log.Warn("webhook: delivery failed", "bot", w.BotID, "attempt", attempt, "status", status, "err", err)
			if attempt < d.cfg.Attempts {
				time.Sleep(delay)
				delay *= 2
			}
		}
	}()
}

func (d *Dispatcher) post(w domain.Webhook, body []byte) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d.cfg.Timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.URL, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Nuntio-Webhook/1.0")
	if w.Secret != "" {
		req.Header.Set("X-Nuntio-Secret", w.Secret)
	}
	resp, err := d.client.Do(req)
	if err != nil {
		return 0, err
	}
	resp.Body.Close()
	return resp.StatusCode, nil
}
