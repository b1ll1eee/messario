package postgres

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/b1ll1eee/nuntio-message/internal/domain"
)

// Integration tests run against a real PostgreSQL when NUNTIO_TEST_DATABASE_URL
// is set (make test). Each run uses its own schema and drops it afterwards.
func testDB(t *testing.T) *DB {
	t.Helper()
	url := os.Getenv("NUNTIO_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("NUNTIO_TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	schema := fmt.Sprintf("t_%d", time.Now().UnixNano())
	admin, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	db := &DB{pool: pool}
	if _, err := db.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Close()
		_, _ = admin.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE")
		admin.Close()
	})
	return db
}

type fixture struct {
	db       *DB
	org      *domain.Org
	alice    *domain.User
	bob      *domain.User
	bot      *domain.User
	users    *UserRepo
	chats    *ChatRepo
	messages *MessageRepo
}

func setup(t *testing.T) *fixture {
	t.Helper()
	db := testDB(t)
	ctx := context.Background()
	orgs := NewOrgRepo(db)
	org := &domain.Org{Name: "Test Org", CreatedAt: time.Now()}
	if err := orgs.Create(ctx, org); err != nil {
		t.Fatal(err)
	}
	users := NewUserRepo(db)
	mk := func(name string, kind domain.UserKind) *domain.User {
		u := &domain.User{OrgID: org.ID, Kind: kind, Username: name, DisplayName: name, Role: domain.RoleMember, Status: domain.StatusActive, CreatedAt: time.Now()}
		if err := users.Create(ctx, u, "hash-"+name); err != nil {
			t.Fatal(err)
		}
		return u
	}
	return &fixture{db: db, org: org, alice: mk("alice", domain.UserHuman), bob: mk("bob", domain.UserHuman), bot: mk("bot", domain.UserBot),
		users: users, chats: NewChatRepo(db), messages: NewMessageRepo(db)}
}

func (f *fixture) group(t *testing.T, members ...*domain.User) *domain.Chat {
	t.Helper()
	c := &domain.Chat{OrgID: f.org.ID, Type: domain.ChatGroup, Title: "g", CreatedBy: members[0].ID, CreatedAt: time.Now()}
	var ms []domain.Member
	for i, u := range members {
		role := domain.MemberMember
		if i == 0 {
			role = domain.MemberOwner
		}
		ms = append(ms, domain.Member{UserID: u.ID, Role: role, JoinedAt: time.Now()})
	}
	if err := f.chats.Create(context.Background(), c, ms); err != nil {
		t.Fatal(err)
	}
	return c
}

func TestUsersLoginLookupAndSeatCount(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	u, hash, err := f.users.ByUsername(ctx, f.org.ID, "alice")
	if err != nil || u.ID != f.alice.ID || hash != "hash-alice" {
		t.Fatalf("ByUsername: %v %v %q", err, u, hash)
	}
	if _, _, err := f.users.ByUsername(ctx, f.org.ID, "nobody"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
	dup := &domain.User{OrgID: f.org.ID, Kind: domain.UserHuman, Username: "alice", DisplayName: "x", Role: domain.RoleMember, Status: domain.StatusActive, CreatedAt: time.Now()}
	if err := f.users.Create(ctx, dup, "h"); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("duplicate username: want ErrConflict, got %v", err)
	}
	n, err := f.users.CountActiveHumans(ctx, f.org.ID)
	if err != nil || n != 2 {
		t.Fatalf("CountActiveHumans = %d, %v (bots must not count)", n, err)
	}
}

func TestDirectChatKeyIsUnique(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	key := domain.DirectKey(f.alice.ID, f.bob.ID)
	mk := func() error {
		c := &domain.Chat{OrgID: f.org.ID, Type: domain.ChatDirect, DirectKey: key, CreatedBy: f.alice.ID, CreatedAt: time.Now()}
		return f.chats.Create(ctx, c, []domain.Member{{UserID: f.alice.ID, Role: domain.MemberMember}, {UserID: f.bob.ID, Role: domain.MemberMember}})
	}
	if err := mk(); err != nil {
		t.Fatal(err)
	}
	if err := mk(); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("second direct chat: want ErrConflict, got %v", err)
	}
	if _, err := f.chats.ByDirectKey(ctx, f.org.ID, key); err != nil {
		t.Fatal(err)
	}
}

func TestMessagesSequenceAndDedupe(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	c := f.group(t, f.alice, f.bob)
	send := func(sender *domain.User, clientID, body string) (*domain.Message, bool) {
		m := &domain.Message{ChatID: c.ID, SenderID: sender.ID, ClientMsgID: clientID, Kind: domain.KindText, Body: body, CreatedAt: time.Now()}
		created, err := f.messages.Append(ctx, m)
		if err != nil {
			t.Fatal(err)
		}
		return m, created
	}
	m1, c1 := send(f.alice, "a1", "hello")
	m2, c2 := send(f.bob, "b1", "hi")
	again, c3 := send(f.alice, "a1", "hello (retry)")
	if !c1 || !c2 || c3 {
		t.Fatalf("created flags: %v %v %v", c1, c2, c3)
	}
	if m1.Seq != 1 || m2.Seq != 2 || again.Seq != 1 || again.Body != "hello" {
		t.Fatalf("seqs: %d %d retry=%d %q", m1.Seq, m2.Seq, again.Seq, again.Body)
	}
	chat, _ := f.chats.ByID(ctx, c.ID)
	if chat.LastSeq != 2 {
		t.Fatalf("last_seq = %d, want 2 (retry must not bump it)", chat.LastSeq)
	}
	hist, err := f.messages.History(ctx, c.ID, 1, 0, 10)
	if err != nil || len(hist) != 1 || hist[0].Seq != 2 {
		t.Fatalf("History after 1: %v %v", hist, err)
	}
	page, _ := f.messages.History(ctx, c.ID, 0, 2, 10)
	if len(page) != 1 || page[0].Seq != 1 {
		t.Fatalf("History before 2: %v", page)
	}
}

func TestChatListUnreadAndReadReceipts(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	c := f.group(t, f.alice, f.bob, f.bot)
	for i := 0; i < 3; i++ {
		m := &domain.Message{ChatID: c.ID, SenderID: f.alice.ID, Kind: domain.KindText, Body: "m", CreatedAt: time.Now()}
		if _, err := f.messages.Append(ctx, m); err != nil {
			t.Fatal(err)
		}
	}
	list, err := f.chats.ListForUser(ctx, f.bob.ID)
	if err != nil || len(list) != 1 {
		t.Fatalf("ListForUser: %v %v", list, err)
	}
	if list[0].Unread != 3 || list[0].LastSeq != 3 || list[0].MemberCount != 3 {
		t.Fatalf("summary: %+v", list[0])
	}
	if err := f.chats.SetReadSeq(ctx, c.ID, f.bob.ID, 2); err != nil {
		t.Fatal(err)
	}
	_ = f.chats.SetReadSeq(ctx, c.ID, f.bob.ID, 1) // must not move backwards
	m, _ := f.chats.Member(ctx, c.ID, f.bob.ID)
	if m.ReadSeq != 2 {
		t.Fatalf("read_seq = %d, want 2", m.ReadSeq)
	}
	n, _ := f.chats.ReadCount(ctx, c.ID, 2)
	if n != 1 {
		t.Fatalf("ReadCount(2) = %d, want 1", n)
	}
	list, _ = f.chats.ListForUser(ctx, f.bob.ID)
	if list[0].Unread != 1 {
		t.Fatalf("unread after read: %d", list[0].Unread)
	}
}

func TestBotUpdatesAndWebhooksForChat(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	c := f.group(t, f.alice, f.bot)
	for _, s := range []*domain.User{f.alice, f.bot, f.alice} {
		m := &domain.Message{ChatID: c.ID, SenderID: s.ID, Kind: domain.KindText, Body: "x", CreatedAt: time.Now()}
		if _, err := f.messages.Append(ctx, m); err != nil {
			t.Fatal(err)
		}
	}
	ups, err := f.messages.UpdatesForBot(ctx, f.bot.ID, 0, 10)
	if err != nil || len(ups) != 2 {
		t.Fatalf("UpdatesForBot: %d %v (own messages excluded)", len(ups), err)
	}
	hooks := NewWebhookRepo(f.db)
	if err := hooks.Set(ctx, &domain.Webhook{BotID: f.bot.ID, URL: "https://example.com/h", Secret: "s", UpdatedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	got, err := hooks.ForChat(ctx, c.ID)
	if err != nil || len(got) != 1 || got[0].Secret != "s" {
		t.Fatalf("ForChat: %v %v", got, err)
	}
	if err := hooks.Delete(ctx, f.bot.ID); err != nil {
		t.Fatal(err)
	}
	if got, _ := hooks.ForChat(ctx, c.ID); len(got) != 0 {
		t.Fatalf("after delete: %v", got)
	}
}

func TestSessionsExpire(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	sessions := NewSessionRepo(f.db)
	now := time.Now()
	if err := sessions.Create(ctx, "h1", f.alice.ID, now.Add(time.Hour), "ua"); err != nil {
		t.Fatal(err)
	}
	if u, err := sessions.UserByToken(ctx, "h1", now); err != nil || u.ID != f.alice.ID {
		t.Fatalf("valid session: %v %v", u, err)
	}
	if _, err := sessions.UserByToken(ctx, "h1", now.Add(2*time.Hour)); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expired session: want ErrNotFound, got %v", err)
	}
	if err := sessions.DeleteForUser(ctx, f.alice.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := sessions.UserByToken(ctx, "h1", now); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("deleted session: want ErrNotFound, got %v", err)
	}
}
