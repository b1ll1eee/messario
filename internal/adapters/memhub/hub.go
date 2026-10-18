// Package memhub is the single-instance implementation of app.Hub: a map
// from user id to that user's live subscribers. Swap for a NATS-backed hub
// when running more than one instance.
package memhub

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/b1ll1eee/messario/internal/app"
)

// Subscriber receives pre-encoded JSON events. Send must not block: a
// subscriber that cannot keep up returns false and is dropped; the client
// then resyncs on reconnect using after_seq.
type Subscriber interface {
	Send(msg []byte) bool
}

type Hub struct {
	mu      sync.RWMutex
	subs    map[int64]map[Subscriber]struct{}
	waiters map[int64]map[chan struct{}]struct{}
	log     *slog.Logger
}

func New(log *slog.Logger) *Hub {
	return &Hub{subs: map[int64]map[Subscriber]struct{}{}, waiters: map[int64]map[chan struct{}]struct{}{}, log: log}
}

// Subscribe registers s for userID and returns the unsubscribe func.
func (h *Hub) Subscribe(userID int64, s Subscriber) func() {
	h.mu.Lock()
	if h.subs[userID] == nil {
		h.subs[userID] = map[Subscriber]struct{}{}
	}
	h.subs[userID][s] = struct{}{}
	h.mu.Unlock()
	return func() {
		h.mu.Lock()
		delete(h.subs[userID], s)
		if len(h.subs[userID]) == 0 {
			delete(h.subs, userID)
		}
		h.mu.Unlock()
	}
}

// Publish encodes once and delivers to every subscriber of every user.
func (h *Hub) Publish(userIDs []int64, ev app.Event) {
	if len(userIDs) == 0 {
		return
	}
	msg, err := json.Marshal(ev)
	if err != nil {
		h.log.Error("hub: encode event", "err", err)
		return
	}
	var dropped []struct {
		uid int64
		s   Subscriber
	}
	h.mu.RLock()
	for _, uid := range userIDs {
		for s := range h.subs[uid] {
			if !s.Send(msg) {
				dropped = append(dropped, struct {
					uid int64
					s   Subscriber
				}{uid, s})
			}
		}
		for ch := range h.waiters[uid] {
			select {
			case ch <- struct{}{}:
			default:
			}
		}
	}
	h.mu.RUnlock()
	if len(dropped) > 0 {
		h.mu.Lock()
		for _, d := range dropped {
			delete(h.subs[d.uid], d.s)
		}
		h.mu.Unlock()
		h.log.Warn("hub: dropped slow subscribers", "count", len(dropped))
	}
}

// Wait blocks until something is published for userID or ctx ends.
func (h *Hub) Wait(ctx context.Context, userID int64) bool {
	ch := make(chan struct{}, 1)
	h.mu.Lock()
	if h.waiters[userID] == nil {
		h.waiters[userID] = map[chan struct{}]struct{}{}
	}
	h.waiters[userID][ch] = struct{}{}
	h.mu.Unlock()
	defer func() {
		h.mu.Lock()
		delete(h.waiters[userID], ch)
		if len(h.waiters[userID]) == 0 {
			delete(h.waiters, userID)
		}
		h.mu.Unlock()
	}()
	select {
	case <-ch:
		return true
	case <-ctx.Done():
		return false
	}
}

// Online reports how many connections userID has. Used by tests and /healthz.
func (h *Hub) Online(userID int64) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.subs[userID])
}
