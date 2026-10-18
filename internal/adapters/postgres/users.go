package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/b1ll1eee/messario/internal/domain"
)

// OrgRepo implements app.OrgRepo.
type OrgRepo struct{ db *DB }

func NewOrgRepo(db *DB) *OrgRepo { return &OrgRepo{db: db} }

func (r *OrgRepo) Get(ctx context.Context) (*domain.Org, error) {
	var o domain.Org
	err := r.db.pool.QueryRow(ctx, `SELECT id, name, email_domain, created_at FROM orgs ORDER BY id LIMIT 1`).
		Scan(&o.ID, &o.Name, &o.EmailDomain, &o.CreatedAt)
	if err != nil {
		return nil, mapErr(err)
	}
	return &o, nil
}

func (r *OrgRepo) Create(ctx context.Context, o *domain.Org) error {
	return mapErr(r.db.pool.QueryRow(ctx,
		`INSERT INTO orgs (name, email_domain, created_at) VALUES ($1,$2,$3) RETURNING id`,
		o.Name, o.EmailDomain, o.CreatedAt).Scan(&o.ID))
}

// UserRepo implements app.UserRepo.
type UserRepo struct{ db *DB }

func NewUserRepo(db *DB) *UserRepo { return &UserRepo{db: db} }

const userCols = `id, org_id, kind, username, display_name, email, department, title, role, status, COALESCE(created_by,0), created_at`

// userColsU is userCols qualified with the alias "u" for joins.
const userColsU = `u.id, u.org_id, u.kind, u.username, u.display_name, u.email, u.department, u.title, u.role, u.status, COALESCE(u.created_by,0), u.created_at`

func scanUser(row pgx.Row) (*domain.User, error) {
	var u domain.User
	err := row.Scan(&u.ID, &u.OrgID, &u.Kind, &u.Username, &u.DisplayName, &u.Email, &u.Department, &u.Title, &u.Role, &u.Status, &u.CreatedBy, &u.CreatedAt)
	if err != nil {
		return nil, mapErr(err)
	}
	return &u, nil
}

func (r *UserRepo) Create(ctx context.Context, u *domain.User, passwordHash string) error {
	var hash, createdBy any
	if passwordHash != "" {
		hash = passwordHash
	}
	if u.CreatedBy != 0 {
		createdBy = u.CreatedBy
	}
	return mapErr(r.db.pool.QueryRow(ctx, `
		INSERT INTO users (org_id, kind, username, display_name, email, department, title, role, status, password_hash, created_by, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) RETURNING id`,
		u.OrgID, u.Kind, u.Username, u.DisplayName, u.Email, u.Department, u.Title, u.Role, u.Status, hash, createdBy, u.CreatedAt,
	).Scan(&u.ID))
}

func (r *UserRepo) ByID(ctx context.Context, id int64) (*domain.User, error) {
	return scanUser(r.db.pool.QueryRow(ctx, `SELECT `+userCols+` FROM users WHERE id=$1`, id))
}

func (r *UserRepo) ByUsername(ctx context.Context, orgID int64, username string) (*domain.User, string, error) {
	var u domain.User
	var hash *string
	err := r.db.pool.QueryRow(ctx, `SELECT `+userCols+`, password_hash FROM users WHERE org_id=$1 AND username=$2`, orgID, username).
		Scan(&u.ID, &u.OrgID, &u.Kind, &u.Username, &u.DisplayName, &u.Email, &u.Department, &u.Title, &u.Role, &u.Status, &u.CreatedBy, &u.CreatedAt, &hash)
	if err != nil {
		return nil, "", mapErr(err)
	}
	if hash == nil {
		return &u, "", nil
	}
	return &u, *hash, nil
}

func (r *UserRepo) List(ctx context.Context, orgID int64, query string, limit int) ([]domain.User, error) {
	rows, err := r.db.pool.Query(ctx, `
		SELECT `+userCols+` FROM users
		WHERE org_id=$1 AND ($2 = '' OR username ILIKE $2 || '%' OR display_name ILIKE '%' || $2 || '%')
		ORDER BY display_name, id LIMIT $3`, orgID, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *u)
	}
	if out == nil {
		out = []domain.User{}
	}
	return out, rows.Err()
}

func (r *UserRepo) Update(ctx context.Context, u *domain.User) error {
	_, err := r.db.pool.Exec(ctx, `
		UPDATE users SET display_name=$2, email=$3, department=$4, title=$5, role=$6, status=$7 WHERE id=$1`,
		u.ID, u.DisplayName, u.Email, u.Department, u.Title, u.Role, u.Status)
	return mapErr(err)
}

func (r *UserRepo) SetPassword(ctx context.Context, id int64, hash string) error {
	_, err := r.db.pool.Exec(ctx, `UPDATE users SET password_hash=$2 WHERE id=$1`, id, hash)
	return err
}

func (r *UserRepo) SetBotToken(ctx context.Context, id int64, tokenHash string) error {
	_, err := r.db.pool.Exec(ctx, `UPDATE users SET bot_token_hash=$2 WHERE id=$1 AND kind='bot'`, id, tokenHash)
	return err
}

func (r *UserRepo) ByBotToken(ctx context.Context, tokenHash string) (*domain.User, error) {
	return scanUser(r.db.pool.QueryRow(ctx, `SELECT `+userCols+` FROM users WHERE bot_token_hash=$1`, tokenHash))
}

func (r *UserRepo) CountActiveHumans(ctx context.Context, orgID int64) (int, error) {
	var n int
	err := r.db.pool.QueryRow(ctx, `SELECT count(*) FROM users WHERE org_id=$1 AND kind='user' AND status='active'`, orgID).Scan(&n)
	return n, err
}

// SessionRepo implements app.SessionRepo.
type SessionRepo struct{ db *DB }

func NewSessionRepo(db *DB) *SessionRepo { return &SessionRepo{db: db} }

func (r *SessionRepo) Create(ctx context.Context, tokenHash string, userID int64, expiresAt time.Time, userAgent string) error {
	_, err := r.db.pool.Exec(ctx, `INSERT INTO sessions (token_hash, user_id, expires_at, user_agent) VALUES ($1,$2,$3,$4)`,
		tokenHash, userID, expiresAt, userAgent)
	return mapErr(err)
}

func (r *SessionRepo) UserByToken(ctx context.Context, tokenHash string, now time.Time) (*domain.User, error) {
	u, err := scanUser(r.db.pool.QueryRow(ctx, `
		SELECT `+userColsU+` FROM sessions s JOIN users u ON u.id=s.user_id
		WHERE s.token_hash=$1 AND s.expires_at > $2`, tokenHash, now))
	if err != nil {
		return nil, err
	}
	// Touch last_seen at most once a minute; best effort.
	_, _ = r.db.pool.Exec(ctx, `UPDATE sessions SET last_seen_at=$2 WHERE token_hash=$1 AND last_seen_at < $2 - interval '1 minute'`, tokenHash, now)
	return u, nil
}

func (r *SessionRepo) Delete(ctx context.Context, tokenHash string) error {
	_, err := r.db.pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash=$1`, tokenHash)
	return err
}

func (r *SessionRepo) DeleteForUser(ctx context.Context, userID int64) error {
	_, err := r.db.pool.Exec(ctx, `DELETE FROM sessions WHERE user_id=$1`, userID)
	return err
}
