package logging

import (
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/lmittmann/tint"
	isatty "github.com/mattn/go-isatty"
)

// Init sets up the default slog logger with tint for colored text output to stderr.
// Log level is read from LOG_LEVEL env var (debug, info, warn, error).
// Defaults to info. Colors are automatically disabled when stderr is not a TTY.
func Init(level string) {
	var slogLevel slog.Level
	switch strings.ToLower(level) {
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

	handler := tint.NewHandler(os.Stderr, &tint.Options{
		Level:      slogLevel,
		TimeFormat: time.DateTime,
		NoColor:    !isTTY,
		AddSource:  slogLevel <= slog.LevelDebug,
	})

	slog.SetDefault(slog.New(handler))
}

// NewLogger creates a child logger with a component key attached to all messages.
func NewLogger(component string) *slog.Logger {
	return slog.Default().With("component", component)
}

// InitOTEL sets up OpenTelemetry integration for structured logging and tracing.
// This is a placeholder for future implementation.
//
// To enable OTEL in the future:
// 1. go get go.opentelemetry.io/contrib/bridges/otelslog
// 2. go get go.opentelemetry.io/otel/sdk/log
// 3. go get go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc
// 4. Uncomment this function
// 5. Replace slog.NewTextHandler with slog.NewMultiHandler(jsonHandler, otelHandler)
//
//	func InitOTEL() (func(), error) {
//	    res, err := resource.Merge(
//	        resource.Default(),
//	        resource.NewWithAttributes(semconv.SchemaURL, semconv.ServiceNameKey.String("owl")),
//	    )
//	    if err != nil { return nil, err }
//
//	    exporter, err := otlploggrpc.New(context.Background(),
//	        otlploggrpc.WithEndpoint("localhost:4317"),
//	        otlploggrpc.WithInsecure(),
//	    )
//	    if err != nil { return nil, err }
//
//	    provider := sdklog.NewLoggerProvider(
//	        sdklog.WithResource(res),
//	        sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
//	    )
//
//	    otelHandler := otelslog.NewHandler("owl",
//	        otelslog.WithLoggerProvider(provider),
//	        otelslog.WithSource(true),
//	    )
//
//	    jsonHandler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})
//	    multiHandler := slog.NewMultiHandler(jsonHandler, otelHandler)
//	    slog.SetDefault(slog.New(multiHandler))
//
//	    return func() {
//	        provider.Shutdown(context.Background())
//	    }, nil
//	}
func InitOTEL() {}
