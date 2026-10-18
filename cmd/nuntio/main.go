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

	"github.com/b1ll1eee/nuntio-message/internal/adapters/filestore"
	"github.com/b1ll1eee/nuntio-message/internal/adapters/hash"
	"github.com/b1ll1eee/nuntio-message/internal/adapters/httpapi"
	"github.com/b1ll1eee/nuntio-message/internal/adapters/memhub"
	"github.com/b1ll1eee/nuntio-message/internal/adapters/obs"
	"github.com/b1ll1eee/nuntio-message/internal/adapters/postgres"
	"github.com/b1ll1eee/nuntio-message/internal/adapters/webhook"
	"github.com/b1ll1eee/nuntio-message/internal/adapters/ws"
	"github.com/b1ll1eee/nuntio-message/internal/app"
	"github.com/b1ll1eee/nuntio-message/internal/config"
	"github.com/b1ll1eee/nuntio-message/internal/license"
	"github.com/b1ll1eee/nuntio-message/internal/web"
)

// version is stamped at build time: -ldflags "-X main.version=1.2.3".
var version = "dev"

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
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg.Obs.Version = version
	tel, err := obs.Start(ctx, obs.Config{
		Endpoint: cfg.Obs.VindiceEndpoint, APIKey: cfg.Obs.VindiceAPIKey, ServiceName: cfg.Obs.ServiceName,
		Version: version, Environment: cfg.Obs.Environment,
	}, newLogger(cfg.LogLevel))
	if err != nil {
		return fmt.Errorf("observability: %w", err)
	}
	defer func() {
		flushCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = tel.Shutdown(flushCtx)
	}()
	log := tel.Logger
	if cfg.Obs.VindiceEndpoint != "" {
		log.Info("vindice telemetry on", "endpoint", cfg.Obs.VindiceEndpoint, "service", cfg.Obs.ServiceName, "env", cfg.Obs.Environment)
	}

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

	store, err := openFileStore(ctx, cfg.Files, log)
	if err != nil {
		return err
	}
	lic, err := openLicense(cfg.License)
	if err != nil {
		return err
	}
	if _, status := lic.Seats(); cmd == "serve" {
		log.Info("license", "status", status)
	}

	hub := memhub.New(log)
	a := app.New(app.Deps{
		Orgs: postgres.NewOrgRepo(db), Users: postgres.NewUserRepo(db), Sessions: postgres.NewSessionRepo(db),
		Chats: postgres.NewChatRepo(db), Messages: postgres.NewMessageRepo(db), Audit: postgres.NewAuditRepo(db),
		Files: postgres.NewFileRepo(db), Store: store,
		Webhooks: postgres.NewWebhookRepo(db),
		Dispatch: webhook.New(webhook.Config{
			Timeout: cfg.Webhook.Timeout, Attempts: cfg.Webhook.Attempts, Backoff: cfg.Webhook.Backoff, Concurrency: cfg.Webhook.Concurrency,
		}, log),
		License: lic,
		Hub:     hub, Hasher: hash.Bcrypt{},
		Limits: app.Limits{
			MaxGroupMembers: cfg.Limits.MaxGroupMembers, MaxChannelMembers: cfg.Limits.MaxChannelMembers,
			MaxBodyBytes: cfg.Limits.MaxBodyBytes, HistoryPageSize: cfg.Limits.HistoryPageSize,
			HistoryPageMax: cfg.Limits.HistoryPageMax, DirectoryPageMax: cfg.Limits.DirectoryPageMax,
			SessionTTL: cfg.SessionTTL, MaxFileBytes: cfg.Files.MaxBytes,
		},
	})

	switch cmd {
	case "migrate":
		fmt.Println("migrations up to date")
		return nil
	case "init":
		return runInit(ctx, a, args)
	case "serve":
		return serve(ctx, cfg, a, hub, tel, log)
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

func serve(ctx context.Context, cfg config.Config, a *app.App, hub *memhub.Hub, tel *obs.Telemetry, log *slog.Logger) error {
	origins := append([]string{cfg.PublicURL}, splitList(os.Getenv("NUNTIO_ALLOWED_ORIGINS"))...)
	wsHandler := ws.New(a, hub, ws.Config{
		WriteTimeout: cfg.WS.WriteTimeout, PingInterval: cfg.WS.PingInterval, PongTimeout: cfg.WS.PongTimeout,
		SendBuffer: cfg.WS.SendBuffer, AllowedOrigins: origins,
	}, log)
	api := httpapi.New(a, httpapi.Config{
		CookieName: cfg.CookieName, CookieSecure: cfg.CookieSecure, SessionTTL: cfg.SessionTTL, AllowedOrigins: origins,
		BotRatePerSec: cfg.Bot.RatePerSecond, BotBurst: cfg.Bot.Burst, BotLongPoll: cfg.Bot.LongPollTimeout, BotUpdatesMax: cfg.Bot.UpdatesPageMax,
		MaxFileBytes: cfg.Files.MaxBytes,
	}, wsHandler, web.Handler(), log)

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           tel.Wrap(api.Routes()),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	errc := make(chan error, 1)
	go func() {
		log.Info("nuntio listening", "addr", cfg.Addr, "public_url", cfg.PublicURL, "version", version)
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

// openFileStore picks S3 when an endpoint is configured, else the local directory.
func openFileStore(ctx context.Context, cfg config.Files, log *slog.Logger) (app.FileStore, error) {
	if cfg.S3Endpoint != "" {
		log.Info("files: s3", "endpoint", cfg.S3Endpoint, "bucket", cfg.S3Bucket)
		return filestore.NewS3(ctx, filestore.S3Config{
			Endpoint: cfg.S3Endpoint, AccessKey: cfg.S3AccessKey, SecretKey: cfg.S3SecretKey,
			Bucket: cfg.S3Bucket, Region: cfg.S3Region, UseSSL: cfg.S3UseSSL,
		})
	}
	log.Info("files: local", "dir", cfg.Dir)
	return filestore.NewLocal(cfg.Dir)
}

// openLicense verifies the configured key; a bad key is a startup error so
// an operator notices, an absent key means the community tier.
func openLicense(cfg config.License) (*license.License, error) {
	var pub []byte
	if cfg.PublicKey != "" {
		k, err := license.DecodePublicKey(cfg.PublicKey)
		if err != nil {
			return nil, err
		}
		pub = k
	}
	if cfg.Key != "" && pub == nil {
		return nil, fmt.Errorf("NUNTIO_LICENSE is set but no license public key is configured")
	}
	return license.Parse(cfg.Key, pub, cfg.FreeSeats)
}
