// Package observability provides logging initialization.
package observability

import (
	"log/slog"
	"os"

	"github.com/rodaine/protoslog"
	"golang.org/x/term"

	eratov1 "github.com/stolasapp/erato/internal/gen/stolasapp/erato/v1"
)

// InitSlog initializes a logger with the given config. When running in a
// terminal, it uses a human-readable text format; otherwise it uses JSON for
// structured logging. The handler is wrapped with protoslog for better proto
// message rendering.
func InitSlog(cfg *eratov1.Config) *slog.Logger {
	opts := &slog.HandlerOptions{
		AddSource: cfg.GetDevMode(),
		Level:     toLogLevel(cfg.GetLogLevel()),
	}
	var handler slog.Handler
	if term.IsTerminal(int(os.Stdin.Fd())) {
		handler = slog.NewTextHandler(os.Stderr, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stderr, opts)
	}
	handler = protoslog.NewHandler(handler)
	return slog.New(handler)
}

func toLogLevel(lvl eratov1.Config_LogLevel) slog.Level {
	switch lvl {
	case eratov1.Config_DEBUG:
		return slog.LevelDebug
	case eratov1.Config_INFO:
		return slog.LevelInfo
	case eratov1.Config_WARN:
		return slog.LevelWarn
	case eratov1.Config_ERROR:
		return slog.LevelError
	default:
		return slog.Level(lvl.Number())
	}
}
