package app

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/b1ll1eee/messario/internal/domain"
)

// ListChats returns the viewer's chats with unread counts, newest activity first.
func (a *App) ListChats(ctx context.Context, viewer *domain.User) ([]domain.ChatSummary, error) {
	return a.Chats.ListForUser(ctx, viewer.ID)
}

// GetChat loads a chat the viewer belongs to, plus the viewer's membership.
func (a *App) GetChat(ctx context.Context, viewer *domain.User, chatID int64) (*domain.Chat, *domain.Member, error) {
	m, err := a.Chats.Member(ctx, chatID, viewer.ID)
	if errors.Is(err, domain.ErrNotFound) {
		return nil, nil, domain.ErrForbidden
	}
	if err != nil {
		return nil, nil, err
	}
	c, err := a.Chats.ByID(ctx, chatID)
	if err != nil {
		return nil, nil, err
	}
	return c, m, nil
}

// OpenDirect returns the direct chat between viewer and peer, creating it on first use.
func (a *App) OpenDirect(ctx context.Context, viewer *domain.User, peerID int64) (*domain.Chat, error) {
	if peerID == viewer.ID {
		return nil, domain.Invalid("cannot open a direct chat with yourself")
	}
	peer, err := a.Users.ByID(ctx, peerID)
	if errors.Is(err, domain.ErrNotFound) || (err == nil && peer.OrgID != viewer.OrgID) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	key := domain.DirectKey(viewer.ID, peerID)
	c, err := a.Chats.ByDirectKey(ctx, viewer.OrgID, key)
	if err == nil {
		return c, nil
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}
	c = &domain.Chat{OrgID: viewer.OrgID, Type: domain.ChatDirect, DirectKey: key, CreatedBy: viewer.ID, CreatedAt: a.Now()}
	members := []domain.Member{
		{UserID: viewer.ID, Role: domain.MemberMember, JoinedAt: c.CreatedAt},
		{UserID: peerID, Role: domain.MemberMember, JoinedAt: c.CreatedAt},
	}
	if err := a.Chats.Create(ctx, c, members); err != nil {
		if errors.Is(err, domain.ErrConflict) {
			// Lost a race with the peer opening the same chat; load theirs.
			return a.Chats.ByDirectKey(ctx, viewer.OrgID, key)
		}
		return nil, err
	}
	a.Hub.Publish([]int64{peerID}, Event{Type: "chat", Payload: c})
	return c, nil
}

// CreateGroup creates a group or channel with the viewer as owner.
func (a *App) CreateGroup(ctx context.Context, viewer *domain.User, t domain.ChatType, title string, memberIDs []int64) (*domain.Chat, error) {
	if t != domain.ChatGroup && t != domain.ChatChannel {
		return nil, domain.Invalid("type must be group or channel")
	}
	title = strings.TrimSpace(title)
	if title == "" || len(title) > 120 {
		return nil, domain.Invalid("title is required (max 120 chars)")
	}
	if t == domain.ChatChannel && !viewer.IsAdmin() {
		return nil, domain.ErrForbidden
	}
	ids := uniqueIDs(memberIDs, viewer.ID)
	if err := a.checkCap(t, len(ids)+1); err != nil {
		return nil, err
	}
	now := a.Now()
	c := &domain.Chat{OrgID: viewer.OrgID, Type: t, Title: title, CreatedBy: viewer.ID, CreatedAt: now}
	members := []domain.Member{{UserID: viewer.ID, Role: domain.MemberOwner, JoinedAt: now}}
	for _, id := range ids {
		if _, err := a.sameOrgUser(ctx, viewer, id); err != nil {
			return nil, err
		}
		members = append(members, domain.Member{UserID: id, Role: domain.MemberMember, JoinedAt: now})
	}
	if err := a.Chats.Create(ctx, c, members); err != nil {
		return nil, err
	}
	a.audit(ctx, viewer, "chat.create", fmt.Sprint(c.ID), map[string]any{"type": t, "members": len(members)})
	a.Hub.Publish(ids, Event{Type: "chat", Payload: c})
	return c, nil
}

// AddMembers adds users to a group or channel. Requires a manager role.
func (a *App) AddMembers(ctx context.Context, viewer *domain.User, chatID int64, memberIDs []int64) error {
	c, me, err := a.GetChat(ctx, viewer, chatID)
	if err != nil {
		return err
	}
	if c.Type == domain.ChatDirect || !me.CanManage() {
		return domain.ErrForbidden
	}
	ids := uniqueIDs(memberIDs, 0)
	if len(ids) == 0 {
		return domain.Invalid("no members given")
	}
	count, err := a.Chats.MemberCount(ctx, chatID)
	if err != nil {
		return err
	}
	if err := a.checkCap(c.Type, count+len(ids)); err != nil {
		return err
	}
	now := a.Now()
	var members []domain.Member
	for _, id := range ids {
		if _, err := a.sameOrgUser(ctx, viewer, id); err != nil {
			return err
		}
		members = append(members, domain.Member{UserID: id, Role: domain.MemberMember, JoinedAt: now})
	}
	if err := a.Chats.AddMembers(ctx, chatID, members); err != nil {
		return err
	}
	a.Hub.Publish(ids, Event{Type: "chat", Payload: c})
	a.postSystem(ctx, c, SystemEvent{Type: "added", ActorID: viewer.ID, UserIDs: ids})
	return nil
}

// RemoveMember removes a user from a group/channel. Managers may remove anyone
// but the owner; any member may remove themselves (leave).
func (a *App) RemoveMember(ctx context.Context, viewer *domain.User, chatID, userID int64) error {
	c, me, err := a.GetChat(ctx, viewer, chatID)
	if err != nil {
		return err
	}
	if c.Type == domain.ChatDirect {
		return domain.ErrForbidden
	}
	target, err := a.Chats.Member(ctx, chatID, userID)
	if err != nil {
		return err
	}
	self := userID == viewer.ID
	if !self && !me.CanManage() {
		return domain.ErrForbidden
	}
	if target.Role == domain.MemberOwner {
		return domain.Invalid("the owner cannot be removed; transfer ownership first")
	}
	if err := a.Chats.RemoveMember(ctx, chatID, userID); err != nil {
		return err
	}
	a.Hub.Publish([]int64{userID}, Event{Type: "left", Payload: map[string]any{"chat_id": chatID}})
	if self {
		a.postSystem(ctx, c, SystemEvent{Type: "left", ActorID: viewer.ID})
	} else {
		a.postSystem(ctx, c, SystemEvent{Type: "removed", ActorID: viewer.ID, UserIDs: []int64{userID}})
	}
	return nil
}

// Members lists a chat's members for a viewer who belongs to it.
func (a *App) Members(ctx context.Context, viewer *domain.User, chatID int64) ([]domain.Member, error) {
	if _, _, err := a.GetChat(ctx, viewer, chatID); err != nil {
		return nil, err
	}
	return a.Chats.Members(ctx, chatID)
}

// RenameChat changes a group/channel title.
func (a *App) RenameChat(ctx context.Context, viewer *domain.User, chatID int64, title string) error {
	c, me, err := a.GetChat(ctx, viewer, chatID)
	if err != nil {
		return err
	}
	if c.Type == domain.ChatDirect || !me.CanManage() {
		return domain.ErrForbidden
	}
	title = strings.TrimSpace(title)
	if title == "" || len(title) > 120 {
		return domain.Invalid("title is required (max 120 chars)")
	}
	if err := a.Chats.UpdateTitle(ctx, chatID, title); err != nil {
		return err
	}
	c.Title = title
	ids, _ := a.Chats.MemberIDs(ctx, chatID)
	a.Hub.Publish(ids, Event{Type: "chat", Payload: c})
	a.postSystem(ctx, c, SystemEvent{Type: "renamed", ActorID: viewer.ID, Title: title})
	return nil
}

func (a *App) checkCap(t domain.ChatType, total int) error {
	limit := a.Limits.MaxGroupMembers
	if t == domain.ChatChannel {
		limit = a.Limits.MaxChannelMembers
	}
	if total > limit {
		return domain.Invalid("%s cannot have more than %d members", t, limit)
	}
	return nil
}

func (a *App) sameOrgUser(ctx context.Context, viewer *domain.User, id int64) (*domain.User, error) {
	u, err := a.Users.ByID(ctx, id)
	if errors.Is(err, domain.ErrNotFound) || (err == nil && u.OrgID != viewer.OrgID) {
		return nil, domain.Invalid("user %d not found", id)
	}
	if err != nil {
		return nil, err
	}
	if !u.IsActive() {
		return nil, domain.Invalid("user %s is suspended", u.Username)
	}
	return u, nil
}

// uniqueIDs dedupes ids and drops zero and the excluded id.
func uniqueIDs(ids []int64, exclude int64) []int64 {
	seen := make(map[int64]bool, len(ids))
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id == 0 || id == exclude || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}
