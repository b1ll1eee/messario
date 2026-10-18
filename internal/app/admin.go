package app

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/b1ll1eee/nuntio-message/internal/domain"
)

// Init bootstraps an empty database with the organization and its first admin.
// It refuses to run twice.
func (a *App) Init(ctx context.Context, orgName, emailDomain, adminUsername, adminPassword string) (*domain.User, error) {
	if _, err := a.Orgs.Get(ctx); err == nil {
		return nil, domain.ErrConflict
	} else if !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}
	orgName = strings.TrimSpace(orgName)
	if orgName == "" {
		return nil, domain.Invalid("organization name is required")
	}
	adminUsername = strings.ToLower(strings.TrimSpace(adminUsername))
	if !validUsername(adminUsername) {
		return nil, domain.Invalid("admin username must be 2-32 chars of a-z, 0-9, '_' or '.'")
	}
	if len(adminPassword) < 8 {
		return nil, domain.Invalid("admin password must be at least 8 characters")
	}
	org := &domain.Org{Name: orgName, EmailDomain: strings.ToLower(strings.TrimSpace(emailDomain)), CreatedAt: a.Now()}
	if err := a.Orgs.Create(ctx, org); err != nil {
		return nil, err
	}
	hash, err := a.Hasher.Hash(adminPassword)
	if err != nil {
		return nil, err
	}
	admin := &domain.User{
		OrgID: org.ID, Kind: domain.UserHuman, Username: adminUsername, DisplayName: adminUsername,
		Role: domain.RoleAdmin, Status: domain.StatusActive, CreatedAt: a.Now(),
	}
	if err := a.Users.Create(ctx, admin, hash); err != nil {
		return nil, err
	}
	return admin, nil
}

// NewUserInput is what an admin supplies to create a person.
type NewUserInput struct {
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Department  string `json:"department"`
	Title       string `json:"title"`
	Role        string `json:"role"`
	Password    string `json:"password"`
}

// CreateUser adds a person. Admin only.
func (a *App) CreateUser(ctx context.Context, admin *domain.User, in NewUserInput) (*domain.User, error) {
	if !admin.IsAdmin() {
		return nil, domain.ErrForbidden
	}
	in.Username = strings.ToLower(strings.TrimSpace(in.Username))
	if !validUsername(in.Username) {
		return nil, domain.Invalid("username must be 2-32 chars of a-z, 0-9, '_' or '.'")
	}
	in.DisplayName = strings.TrimSpace(in.DisplayName)
	if in.DisplayName == "" {
		return nil, domain.Invalid("display_name is required")
	}
	if len(in.Password) < 8 {
		return nil, domain.Invalid("password must be at least 8 characters")
	}
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	if err := a.checkEmailDomain(ctx, in.Email); err != nil {
		return nil, err
	}
	role := domain.RoleMember
	if in.Role == string(domain.RoleAdmin) {
		role = domain.RoleAdmin
	}
	if err := a.checkSeats(ctx, admin.OrgID); err != nil {
		return nil, err
	}
	hash, err := a.Hasher.Hash(in.Password)
	if err != nil {
		return nil, err
	}
	u := &domain.User{
		OrgID: admin.OrgID, Kind: domain.UserHuman, Username: in.Username, DisplayName: in.DisplayName,
		Email: in.Email, Department: strings.TrimSpace(in.Department), Title: strings.TrimSpace(in.Title),
		Role: role, Status: domain.StatusActive, CreatedBy: admin.ID, CreatedAt: a.Now(),
	}
	if err := a.Users.Create(ctx, u, hash); err != nil {
		return nil, err
	}
	a.audit(ctx, admin, "user.create", fmt.Sprint(u.ID), map[string]any{"username": u.Username, "role": role})
	return u, nil
}

// UpdateUserInput carries optional fields; nil means unchanged.
type UpdateUserInput struct {
	DisplayName *string `json:"display_name"`
	Email       *string `json:"email"`
	Department  *string `json:"department"`
	Title       *string `json:"title"`
	Role        *string `json:"role"`
	Status      *string `json:"status"`
	Password    *string `json:"password"`
}

// UpdateUser edits a person or bot. Suspending a user ends their sessions.
func (a *App) UpdateUser(ctx context.Context, admin *domain.User, id int64, in UpdateUserInput) (*domain.User, error) {
	if !admin.IsAdmin() {
		return nil, domain.ErrForbidden
	}
	u, err := a.Users.ByID(ctx, id)
	if errors.Is(err, domain.ErrNotFound) || (err == nil && u.OrgID != admin.OrgID) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	changes := map[string]any{}
	if in.DisplayName != nil {
		v := strings.TrimSpace(*in.DisplayName)
		if v == "" {
			return nil, domain.Invalid("display_name cannot be empty")
		}
		u.DisplayName, changes["display_name"] = v, v
	}
	if in.Email != nil {
		v := strings.ToLower(strings.TrimSpace(*in.Email))
		if err := a.checkEmailDomain(ctx, v); err != nil {
			return nil, err
		}
		u.Email, changes["email"] = v, v
	}
	if in.Department != nil {
		u.Department, changes["department"] = strings.TrimSpace(*in.Department), *in.Department
	}
	if in.Title != nil {
		u.Title, changes["title"] = strings.TrimSpace(*in.Title), *in.Title
	}
	if in.Role != nil {
		if u.IsBot() {
			return nil, domain.Invalid("bots cannot have a role")
		}
		switch domain.Role(*in.Role) {
		case domain.RoleAdmin, domain.RoleMember:
			if u.ID == admin.ID && domain.Role(*in.Role) != domain.RoleAdmin {
				return nil, domain.Invalid("you cannot remove your own admin role")
			}
			u.Role, changes["role"] = domain.Role(*in.Role), *in.Role
		default:
			return nil, domain.Invalid("role must be admin or member")
		}
	}
	if in.Status != nil {
		switch domain.UserStatus(*in.Status) {
		case domain.StatusActive, domain.StatusSuspended:
			if u.ID == admin.ID && domain.UserStatus(*in.Status) == domain.StatusSuspended {
				return nil, domain.Invalid("you cannot suspend yourself")
			}
			if domain.UserStatus(*in.Status) == domain.StatusActive && u.Status != domain.StatusActive && !u.IsBot() {
				if err := a.checkSeats(ctx, admin.OrgID); err != nil {
					return nil, err
				}
			}
			u.Status, changes["status"] = domain.UserStatus(*in.Status), *in.Status
		default:
			return nil, domain.Invalid("status must be active or suspended")
		}
	}
	if err := a.Users.Update(ctx, u); err != nil {
		return nil, err
	}
	if in.Password != nil {
		if u.IsBot() {
			return nil, domain.Invalid("bots have tokens, not passwords")
		}
		if len(*in.Password) < 8 {
			return nil, domain.Invalid("password must be at least 8 characters")
		}
		hash, err := a.Hasher.Hash(*in.Password)
		if err != nil {
			return nil, err
		}
		if err := a.Users.SetPassword(ctx, u.ID, hash); err != nil {
			return nil, err
		}
		changes["password"] = "changed"
	}
	if u.Status == domain.StatusSuspended {
		_ = a.Sessions.DeleteForUser(ctx, u.ID)
	}
	a.audit(ctx, admin, "user.update", fmt.Sprint(u.ID), changes)
	return u, nil
}

// Directory lists people and bots in the org for the chat picker.
func (a *App) Directory(ctx context.Context, viewer *domain.User, query string, limit int) ([]domain.User, error) {
	if limit <= 0 || limit > a.Limits.DirectoryPageMax {
		limit = a.Limits.DirectoryPageMax
	}
	return a.Users.List(ctx, viewer.OrgID, strings.TrimSpace(query), limit)
}

// AuditLog lists recent audit entries. Admin only.
func (a *App) AuditLog(ctx context.Context, admin *domain.User, limit int) ([]domain.AuditEntry, error) {
	if !admin.IsAdmin() {
		return nil, domain.ErrForbidden
	}
	if limit <= 0 || limit > a.Limits.DirectoryPageMax {
		limit = a.Limits.DirectoryPageMax
	}
	return a.Audit.List(ctx, admin.OrgID, limit)
}

// LicenseStatus is what the admin page shows.
type LicenseStatus struct {
	Seats     int    `json:"seats"` // 0 = unlimited
	UsedSeats int    `json:"used_seats"`
	Status    string `json:"status"`
}

// License reports seat usage against the license. Admin only.
func (a *App) LicenseInfo(ctx context.Context, admin *domain.User) (*LicenseStatus, error) {
	if !admin.IsAdmin() {
		return nil, domain.ErrForbidden
	}
	used, err := a.Users.CountActiveHumans(ctx, admin.OrgID)
	if err != nil {
		return nil, err
	}
	seats, status := 0, "unlimited"
	if a.License != nil {
		seats, status = a.License.Seats()
	}
	return &LicenseStatus{Seats: seats, UsedSeats: used, Status: status}, nil
}

// checkSeats refuses to add a human user beyond the licensed seat count.
func (a *App) checkSeats(ctx context.Context, orgID int64) error {
	if a.License == nil {
		return nil
	}
	seats, status := a.License.Seats()
	if seats <= 0 {
		return nil
	}
	used, err := a.Users.CountActiveHumans(ctx, orgID)
	if err != nil {
		return err
	}
	if used >= seats {
		return domain.Invalid("seat limit reached (%d of %d, %s)", used, seats, status)
	}
	return nil
}

func (a *App) checkEmailDomain(ctx context.Context, email string) error {
	if email == "" {
		return nil
	}
	if !strings.Contains(email, "@") {
		return domain.Invalid("email is not valid")
	}
	org, err := a.Orgs.Get(ctx)
	if err != nil {
		return err
	}
	if org.EmailDomain != "" && !strings.HasSuffix(email, "@"+org.EmailDomain) {
		return domain.Invalid("email must belong to %s", org.EmailDomain)
	}
	return nil
}
