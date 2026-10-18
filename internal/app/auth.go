package app

import (
	"context"
	"errors"
	"strings"

	"github.com/b1ll1eee/messario/internal/domain"
)

// Login checks credentials and opens a session. The returned token goes into
// an httpOnly cookie; only its hash is stored.
func (a *App) Login(ctx context.Context, username, password, userAgent string) (*domain.User, string, error) {
	username = strings.ToLower(strings.TrimSpace(username))
	if username == "" || password == "" {
		return nil, "", domain.Invalid("username and password are required")
	}
	org, err := a.Orgs.Get(ctx)
	if err != nil {
		return nil, "", err
	}
	u, hash, err := a.Users.ByUsername(ctx, org.ID, username)
	if errors.Is(err, domain.ErrNotFound) || (err == nil && (u.IsBot() || hash == "")) {
		// Same cost path as a wrong password so usernames can't be probed by timing.
		a.Hasher.Compare("$2a$10$invalidinvalidinvalidinvalidinvalidinvalidinvalidinv", password)
		return nil, "", domain.ErrUnauthorized
	}
	if err != nil {
		return nil, "", err
	}
	if !a.Hasher.Compare(hash, password) {
		return nil, "", domain.ErrUnauthorized
	}
	if !u.IsActive() {
		return nil, "", domain.ErrSuspended
	}
	token, err := randomToken(32)
	if err != nil {
		return nil, "", err
	}
	now := a.Now()
	if err := a.Sessions.Create(ctx, hashToken(token), u.ID, now.Add(a.Limits.SessionTTL), userAgent); err != nil {
		return nil, "", err
	}
	a.audit(ctx, u, "auth.login", "", nil)
	return u, token, nil
}

// Authenticate resolves a session token to its active user.
func (a *App) Authenticate(ctx context.Context, token string) (*domain.User, error) {
	if token == "" {
		return nil, domain.ErrUnauthorized
	}
	u, err := a.Sessions.UserByToken(ctx, hashToken(token), a.Now())
	if errors.Is(err, domain.ErrNotFound) {
		return nil, domain.ErrUnauthorized
	}
	if err != nil {
		return nil, err
	}
	if !u.IsActive() {
		return nil, domain.ErrSuspended
	}
	return u, nil
}

// Logout ends one session.
func (a *App) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return a.Sessions.Delete(ctx, hashToken(token))
}

// ChangePassword lets a user change their own password.
func (a *App) ChangePassword(ctx context.Context, u *domain.User, current, next string) error {
	if len(next) < 8 {
		return domain.Invalid("password must be at least 8 characters")
	}
	_, hash, err := a.Users.ByUsername(ctx, u.OrgID, u.Username)
	if err != nil {
		return err
	}
	if !a.Hasher.Compare(hash, current) {
		return domain.ErrUnauthorized
	}
	newHash, err := a.Hasher.Hash(next)
	if err != nil {
		return err
	}
	if err := a.Users.SetPassword(ctx, u.ID, newHash); err != nil {
		return err
	}
	a.audit(ctx, u, "auth.password_changed", "", nil)
	return nil
}
