// Package config reads runtime settings from the environment. Every setting
// has a default so a bare binary starts; nothing is hard-coded elsewhere.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config is the full runtime configuration.
type Config struct {
	Addr         string        // HTTP listen address
	DatabaseURL  string        // PostgreSQL connection string
	PublicURL    string        // external URL, used for cookies/CSRF origin checks
	CookieName   string        // session cookie name
	CookieSecure bool          // Secure flag on cookies (false only for local dev)
	SessionTTL   time.Duration // how long a login lasts
	LogLevel     string        // debug|info|warn|error

	Limits  Limits
	WS      WS
	Bot     Bot
	Obs     Obs
	Files   Files
	Webhook Webhook
	License License
}

// Files selects where uploads live: a local directory (default) or S3.
type Files struct {
	MaxBytes    int64  // largest upload
	Dir         string // local directory; used when S3Endpoint is empty
	S3Endpoint  string // host[:port] of S3/MinIO/R2
	S3AccessKey string
	S3SecretKey string
	S3Bucket    string
	S3Region    string
	S3UseSSL    bool
}

// Webhook bounds bot webhook deliveries.
type Webhook struct {
	Timeout     time.Duration
	Attempts    int
	Backoff     time.Duration
	Concurrency int
}

// License holds the offline license key and the vendor public key that verifies it.
type License struct {
	Key       string // the customer's key; empty = community edition
	PublicKey string // base64url ed25519 public key (compiled-in default, overridable)
	FreeSeats int    // seats allowed without a key
}

// Obs points at Vindice, the team's observability platform. Empty endpoint = off.
type Obs struct {
	VindiceEndpoint string // e.g. http://vindice-host:4318 (also honours VINDICE_ENDPOINT)
	VindiceAPIKey   string // ingest key (also honours VINDICE_API_KEY)
	Environment     string // production|staging|dev, tagged on every signal
	ServiceName     string
	Version         string // set at build time via -ldflags "-X main.version=..."
}

// Limits are product rules that an operator may tune per install.
type Limits struct {
	MaxGroupMembers   int // cap for group chats
	MaxChannelMembers int // cap for broadcast channels
	MaxBodyBytes      int // longest message body
	HistoryPageSize   int // default page size for history
	HistoryPageMax    int // hard cap a client may request
	DirectoryPageMax  int // hard cap for user directory listing
}

// WS tunes the WebSocket adapter.
type WS struct {
	WriteTimeout time.Duration
	PingInterval time.Duration // server sends pings; client must answer within PongTimeout
	PongTimeout  time.Duration
	SendBuffer   int // queued events per connection before it is dropped as slow
}

// Bot tunes the bot API.
type Bot struct {
	RatePerSecond   float64       // sendMessage calls per second per bot
	Burst           int           // burst allowance per bot
	LongPollTimeout time.Duration // max wait in getUpdates
	UpdatesPageMax  int
}

// DefaultLicensePublicKey is the vendor's ed25519 public key (base64url).
// Replace it with your own from `licensegen keygen` before selling; it can
// also be overridden at runtime with MESSARIO_LICENSE_PUBLIC_KEY.
const DefaultLicensePublicKey = "M1ukSrg_AJVMvuGP5AWjyiK6XCAamzsc9fNpan5uJ88"

// Load reads MESSARIO_* variables with defaults.
func Load() (Config, error) {
	c := Config{
		Addr:         env("MESSARIO_ADDR", ":8080"),
		DatabaseURL:  env("MESSARIO_DATABASE_URL", "postgres://messario:messario@localhost:5432/messario?sslmode=disable"),
		PublicURL:    env("MESSARIO_PUBLIC_URL", "http://localhost:8080"),
		CookieName:   env("MESSARIO_COOKIE_NAME", "messario_session"),
		CookieSecure: envBool("MESSARIO_COOKIE_SECURE", true),
		SessionTTL:   envDuration("MESSARIO_SESSION_TTL", 30*24*time.Hour),
		LogLevel:     env("MESSARIO_LOG_LEVEL", "info"),
		Limits: Limits{
			MaxGroupMembers:   envInt("MESSARIO_MAX_GROUP_MEMBERS", 200),
			MaxChannelMembers: envInt("MESSARIO_MAX_CHANNEL_MEMBERS", 5000),
			MaxBodyBytes:      envInt("MESSARIO_MAX_BODY_BYTES", 8000),
			HistoryPageSize:   envInt("MESSARIO_HISTORY_PAGE_SIZE", 50),
			HistoryPageMax:    envInt("MESSARIO_HISTORY_PAGE_MAX", 200),
			DirectoryPageMax:  envInt("MESSARIO_DIRECTORY_PAGE_MAX", 500),
		},
		WS: WS{
			WriteTimeout: envDuration("MESSARIO_WS_WRITE_TIMEOUT", 10*time.Second),
			PingInterval: envDuration("MESSARIO_WS_PING_INTERVAL", 25*time.Second),
			PongTimeout:  envDuration("MESSARIO_WS_PONG_TIMEOUT", 10*time.Second),
			SendBuffer:   envInt("MESSARIO_WS_SEND_BUFFER", 64),
		},
		Bot: Bot{
			RatePerSecond:   envFloat("MESSARIO_BOT_RATE_PER_SECOND", 30),
			Burst:           envInt("MESSARIO_BOT_BURST", 30),
			LongPollTimeout: envDuration("MESSARIO_BOT_LONGPOLL_TIMEOUT", 25*time.Second),
			UpdatesPageMax:  envInt("MESSARIO_BOT_UPDATES_PAGE_MAX", 100),
		},
		Obs: Obs{
			VindiceEndpoint: env("MESSARIO_VINDICE_ENDPOINT", env("VINDICE_ENDPOINT", "")),
			VindiceAPIKey:   env("MESSARIO_VINDICE_API_KEY", env("VINDICE_API_KEY", "")),
			Environment:     env("MESSARIO_ENVIRONMENT", "production"),
			ServiceName:     env("MESSARIO_SERVICE_NAME", "messario"),
		},
		Files: Files{
			MaxBytes:    int64(envInt("MESSARIO_MAX_FILE_MB", 25)) << 20,
			Dir:         env("MESSARIO_FILES_DIR", "./data/files"),
			S3Endpoint:  env("MESSARIO_S3_ENDPOINT", ""),
			S3AccessKey: env("MESSARIO_S3_ACCESS_KEY", ""),
			S3SecretKey: env("MESSARIO_S3_SECRET_KEY", ""),
			S3Bucket:    env("MESSARIO_S3_BUCKET", "messario"),
			S3Region:    env("MESSARIO_S3_REGION", "us-east-1"),
			S3UseSSL:    envBool("MESSARIO_S3_SSL", true),
		},
		Webhook: Webhook{
			Timeout:     envDuration("MESSARIO_WEBHOOK_TIMEOUT", 10*time.Second),
			Attempts:    envInt("MESSARIO_WEBHOOK_ATTEMPTS", 3),
			Backoff:     envDuration("MESSARIO_WEBHOOK_BACKOFF", 2*time.Second),
			Concurrency: envInt("MESSARIO_WEBHOOK_CONCURRENCY", 16),
		},
		License: License{
			Key:       env("MESSARIO_LICENSE", ""),
			PublicKey: env("MESSARIO_LICENSE_PUBLIC_KEY", DefaultLicensePublicKey),
			FreeSeats: envInt("MESSARIO_FREE_SEATS", 10),
		},
	}
	if c.DatabaseURL == "" {
		return c, fmt.Errorf("MESSARIO_DATABASE_URL is required")
	}
	return c, nil
}

func env(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func envFloat(key string, def float64) float64 {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil {
			return n
		}
	}
	return def
}

func envBool(key string, def bool) bool {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}

func envDuration(key string, def time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
