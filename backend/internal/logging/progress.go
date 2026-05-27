package logging

import (
	"log/slog"
	"sync/atomic"
	"time"
)

// ProgressLogger tracks iteration progress and logs at regular intervals.
// When total is zero, it logs absolute count only (no percent).
type ProgressLogger struct {
	logger    *slog.Logger
	startTime time.Time
	total     int
	interval  int
	current   atomic.Int64
	msg       string
}

// NewProgressLogger creates a ProgressLogger that logs the given message
// at each interval iteration. If total > 0, percent is included.
// Set interval to 0 for default (every 100 iterations).
func NewProgressLogger(logger *slog.Logger, total, interval int, msg string) *ProgressLogger {
	if interval <= 0 {
		interval = 100
	}
	return &ProgressLogger{
		logger:    logger,
		startTime: time.Now(),
		total:     total,
		interval:  interval,
		msg:       msg,
	}
}

// Increment increases the counter and logs if the interval boundary is crossed.
// Returns the current count.
func (p *ProgressLogger) Increment() int {
	n := int(p.current.Add(1))
	if n%p.interval == 0 {
		attrs := []any{"count", n}
		if p.total > 0 {
			pct := float64(n) / float64(p.total) * 100
			attrs = append(attrs, "total", p.total, "percent", int(pct))
		}
		p.logger.Info(p.msg, attrs...)
	}
	return n
}

// Add adds multiple iterations and logs if the interval boundary is crossed.
func (p *ProgressLogger) Add(n int) int {
	for i := 0; i < n; i++ {
		p.Increment()
	}
	return int(p.current.Load())
}

// Done logs a final completion message with total count and elapsed time.
func (p *ProgressLogger) Done() {
	n := int(p.current.Load())
	attrs := []any{"count", n, "elapsed", time.Since(p.startTime).String()}
	if p.total > 0 {
		attrs = append(attrs, "total", p.total)
	}
	p.logger.Info(p.msg+" done", attrs...)
}

// Current returns the current count without logging.
func (p *ProgressLogger) Current() int {
	return int(p.current.Load())
}
