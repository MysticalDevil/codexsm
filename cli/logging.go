package cli

import (
	"fmt"
	"io"
	"log/slog"
	"strings"
)

var appLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

func logger() *slog.Logger {
	return appLogger
}

func configureLogger(format, level string, out io.Writer) error {
	lv, err := parseLogLevel(level)
	if err != nil {
		return err
	}
	if out == nil {
		out = io.Discard
	}
	opts := &slog.HandlerOptions{Level: lv}
	var h slog.Handler
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "text":
		h = slog.NewTextHandler(out, opts)
	case "json":
		h = slog.NewJSONHandler(out, opts)
	default:
		return fmt.Errorf("invalid --log-format %q (allowed: text, json)", format)
	}
	appLogger = slog.New(h)
	slog.SetDefault(appLogger)
	return nil
}

func parseLogLevel(v string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "warn", "warning":
		return slog.LevelWarn, nil
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("invalid --log-level %q (allowed: debug, info, warn, error)", v)
	}
}
