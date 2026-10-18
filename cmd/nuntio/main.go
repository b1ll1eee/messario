// Command nuntio is the single binary: API, WebSocket, bot API and (later)
// the embedded web client.
//
//	nuntio serve              run the server (default)
//	nuntio migrate            apply database migrations and exit
//	nuntio init --org ... --admin ... --password ...   create the org and first admin
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/b1ll1eee/nuntio-message/internal/adapters/hash"
	"github.com/b1ll1eee/nuntio-message/internal/adapters/httpapi"
	"github.com/b1ll1eee/nuntio-message/internal/adapters/memhub"
	"github.com/b1ll1eee/nuntio-message/internal/adapters/postgres"
	"github.com/b1ll1eee/nuntio-message/internal/adapters/ws"
	"github.com/b1ll1eee/nuntio-message/internal/app"
	"github.com/b1ll1eee/nuntio-message/internal/config"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	cmd := "serve"
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		cmd, args = args[0], args[1:]
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log := newLogger(cfg.LogLevel)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	applied, err := db.Migrate(ctx)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	if len(applied) > 0 {
		log.Info("migrations applied", "versions", applied)
	}

	hub := memhub.New(log)
	a := app.New(app.Deps{
		Orgs: postgres.NewOrgRepo(db), Users: postgres.NewUserRepo(db), Sessions: postgres.NewSessionRepo(db),
		Chats: postgres.NewChatRepo(db), Messages: postgres.NewMessageRepo(db), Audit: postgres.NewAuditRepo(db),
		Hub: hub, Hasher: hash.Bcrypt{},
		Limits: app.Limits{
			MaxGroupMembers: cfg.Limits.MaxGroupMembers, MaxChannelMembers: cfg.Limits.MaxChannelMembers,
			MaxBodyBytes: cfg.Limits.MaxBodyBytes, HistoryPageSize: cfg.Limits.HistoryPageSize,
			HistoryPageMax: cfg.Limits.HistoryPageMax, DirectoryPageMax: cfg.Limits.DirectoryPageMax,
			SessionTTL: cfg.SessionTTL,
		},
	})

	switch cmd {
	case "migrate":
		fmt.Println("migrations up to date")
		return nil
	case "init":
		return runInit(ctx, a, args)
	case "serve":
		return serve(ctx, cfg, a, hub, log)
	default:
		return fmt.Errorf("unknown command %q (serve|migrate|init)", cmd)
	}
}

func runInit(ctx context.Context, a *app.App, args []string) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	org := fs.String("org", "", "organization name")
	domainName := fs.String("domain", "", "email domain allowed for members (optional)")
	admin := fs.String("admin", "admin", "first admin username")
	password := fs.String("password", "", "first admin password (or NUNTIO_INIT_PASSWORD)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *password == "" {
		*password = os.Getenv("NUNTIO_INIT_PASSWORD")
	}
	u, err := a.Init(ctx, *org, *domainName, *admin, *password)
	if err != nil {
		return err
	}
	fmt.Printf("initialized organization %q with admin %q (id %d)\n", *org, u.Username, u.ID)
	return nil
}

func serve(ctx context.Context, cfg config.Config, a *app.App, hub *memhub.Hub, log *slog.Logger) error {
	origins := append([]string{cfg.PublicURL}, splitList(os.Getenv("NUNTIO_ALLOWED_ORIGINS"))...)
	wsHandler := ws.New(a, hub, ws.Config{
		WriteTimeout: cfg.WS.WriteTimeout, PingInterval: cfg.WS.PingInterval, PongTimeout: cfg.WS.PongTimeout,
		SendBuffer: cfg.WS.SendBuffer, AllowedOrigins: origins,
	}, log)
	api := httpapi.New(a, httpapi.Config{
		CookieName: cfg.CookieName, CookieSecure: cfg.CookieSecure, SessionTTL: cfg.SessionTTL, AllowedOrigins: origins,
		BotRatePerSec: cfg.Bot.RatePerSecond, BotBurst: cfg.Bot.Burst, BotLongPoll: cfg.Bot.LongPollTimeout, BotUpdatesMax: cfg.Bot.UpdatesPageMax,
	}, wsHandler, log)

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           api.Routes(),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	errc := make(chan error, 1)
	go func() {
		log.Info("nuntio listening", "addr", cfg.Addr, "public_url", cfg.PublicURL)
		errc <- srv.ListenAndServe()
	}()
	select {
	case err := <-errc:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		log.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}

func newLogger(level string) *slog.Logger {
	var l slog.Level
	if err := l.UnmarshalText([]byte(level)); err != nil {
		l = slog.LevelInfo
	}
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: l}))
}

func splitList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
