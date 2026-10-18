// Package obs wires Nuntio to Vindice (the team's observability platform)
// through its OpenTelemetry SDK: traces for every HTTP request, request
// metrics, and logs linked to traces. It is optional: with no endpoint
// configured the service logs to stdout only.
package obs

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	vindice "github.com/b1ll1eee/vindice-observability/sdk"
)

// Config selects the Vindice ingest endpoint. Empty Endpoint disables it.
type Config struct {
	Endpoint    string
	APIKey      string
	ServiceName string
	Version     string
	Environment string
}

// Telemetry is what main() uses: a logger, an HTTP wrapper and a shutdown.
type Telemetry struct {
	Logger   *slog.Logger
	Wrap     func(http.Handler) http.Handler
	Shutdown func(context.Context) error
}

// Start returns a Telemetry that sends to Vindice when cfg.Endpoint is set,
// otherwise one that only logs locally with fallback.
func Start(ctx context.Context, cfg Config, fallback *slog.Logger) (*Telemetry, error) {
	if cfg.Endpoint == "" {
		return &Telemetry{
			Logger:   fallback,
			Wrap:     func(h http.Handler) http.Handler { return h },
			Shutdown: func(context.Context) error { return nil },
		}, nil
	}
	sdk, err := vindice.Start(ctx, vindice.Config{
		ServiceName:    cfg.ServiceName,
		ServiceVersion: cfg.Version,
		Environment:    cfg.Environment,
		Endpoint:       cfg.Endpoint,
		APIKey:         cfg.APIKey,
		LogOutput:      os.Stdout,
	})
	if err != nil {
		return nil, err
	}
	return &Telemetry{
		Logger:   sdk.Logger(),
		Wrap:     func(h http.Handler) http.Handler { return vindice.Handler(h, cfg.ServiceName) },
		Shutdown: sdk.Shutdown,
	}, nil
}
