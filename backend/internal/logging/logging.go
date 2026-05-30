package logging

import (
	"log/slog"
	"os"
	"owl/internal/config"
	"path/filepath"
	"strings"
	"time"

	"github.com/lmittmann/tint"
	isatty "github.com/mattn/go-isatty"
)

var logFile *os.File

// Init sets up the default slog logger with tint for colored text output to stderr.
// Also writes logs to a timestamped file in the data/logs directory.
// Log level is read from LOG_LEVEL env var (debug, info, warn, error).
// Defaults to info. Colors are automatically disabled when stderr is not a TTY.
func Init(config *config.Config) {
	var slogLevel slog.Level
	switch strings.ToLower(config.LogLevel) {
	case "debug":
		slogLevel = slog.LevelDebug
	case "warn":
		slogLevel = slog.LevelWarn
	case "error":
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	isTTY := isatty.IsTerminal(os.Stderr.Fd())

	var handlers []slog.Handler

	tintHandler := tint.NewHandler(os.Stderr, &tint.Options{
		Level:      slogLevel,
		TimeFormat: time.DateTime,
		NoColor:    !isTTY,
		AddSource:  slogLevel <= slog.LevelDebug,
	})
	handlers = append(handlers, tintHandler)

	logsDir := filepath.Join(config.DataDir, "logs")
	if err := os.MkdirAll(logsDir, 0755); err == nil {
		logFile, err = os.OpenFile(
			filepath.Join(logsDir, time.Now().Format("2006-01-02_15-04-05")+".log"),
			os.O_CREATE|os.O_WRONLY|os.O_APPEND,
			0644,
		)
		if err == nil {
			textHandler := slog.NewTextHandler(logFile, &slog.HandlerOptions{
				Level:     slogLevel,
				AddSource: slogLevel <= slog.LevelDebug,
			})
			handlers = append(handlers, textHandler)
		}
	}

	slog.SetDefault(slog.New(slog.NewMultiHandler(handlers...)))
}

// Close closes the log file if open
func Close() {
	if logFile != nil {
		logFile.Close()
	}
}
