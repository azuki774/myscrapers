// Package logging contains the small amount of logging policy shared by the
// SBI and NRKN scrapers. Keeping the policy here makes their records
// comparable while leaving slog's JSON handler configurable at the CLI.
package logging

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"time"
)

// Logger adds the scraper name and the common event fields to slog records.
// A Logger is safe to use when the caller supplied a nil slog.Logger; in that
// case records are discarded.
type Logger struct {
	logger  *slog.Logger
	scraper string
}

// New creates a scraper-scoped structured logger. The underlying handler is
// owned by the caller, so this function does not choose stdout or stderr.
func New(logger *slog.Logger, scraper string) *Logger {
	if logger == nil {
		logger = slog.New(slog.NewJSONHandler(io.Discard, nil))
	}
	return &Logger{logger: logger, scraper: scraper}
}

// Raw returns the underlying slog logger. It is useful when passing the
// logger through an existing options struct while retaining the CLI's output
// destination.
func (l *Logger) Raw() *slog.Logger {
	if l == nil || l.logger == nil {
		return slog.New(slog.NewJSONHandler(io.Discard, nil))
	}
	return l.logger
}

// Started records the beginning of an operation and returns a timestamp for
// its matching Completed or Failed record.
func (l *Logger) Started(stage string, attrs ...any) time.Time {
	started := time.Now()
	l.event(slog.LevelInfo, "started", stage, attrs...)
	return started
}

// Completed records successful completion of an operation. Duration is
// included in milliseconds so records from both scrapers can be aggregated.
func (l *Logger) Completed(stage string, started time.Time, attrs ...any) {
	attrs = append(attrs, "duration_ms", elapsedMillis(started))
	l.event(slog.LevelInfo, "completed", stage, attrs...)
}

// Error records the terminal failure for a command. Callers should emit one
// such record per invocation, after any operation has returned and cleanup
// has run. It deliberately carries the operation stage rather than replacing
// it with a generic run label.
func (l *Logger) Error(stage, page string, err error, attrs ...any) {
	if page != "" {
		attrs = append(attrs, "page", page)
	}
	attrs = append(attrs, "error", err)
	l.event(slog.LevelError, "failed", stage, attrs...)
}

// Warning records a non-fatal condition within a stage.
func (l *Logger) Warning(stage, message string, attrs ...any) {
	attrs = append(attrs, "message", message)
	l.event(slog.LevelWarn, "warning", stage, attrs...)
}

func (l *Logger) event(level slog.Level, event, stage string, attrs ...any) {
	if l == nil {
		return
	}
	base := make([]any, 0, 4+len(attrs))
	base = append(base, "scraper", l.scraper, "stage", stage, "event", event)
	base = append(base, attrs...)
	l.Raw().Log(context.Background(), level, stage+" "+event, base...)
}

func elapsedMillis(started time.Time) int64 {
	if started.IsZero() {
		return 0
	}
	return time.Since(started).Milliseconds()
}

// ContextError preserves the operation and optional page where an error was
// observed. The CLI uses it to put useful context on its one terminal error
// record without exposing page URLs or page contents.
type ContextError struct {
	Stage string
	Page  string
	Err   error
}

func (e *ContextError) Error() string {
	if e == nil {
		return ""
	}
	if e.Page != "" {
		return e.Stage + " (page=" + e.Page + "): " + e.Err.Error()
	}
	return e.Stage + ": " + e.Err.Error()
}

func (e *ContextError) Unwrap() error { return e.Err }

// WithContext annotates err with a stable stage and optional page identifier.
// Existing ContextErrors are preserved, so a caller adding a broad wrapper
// cannot discard the more precise failure location.
func WithContext(err error, stage, page string) error {
	if err == nil {
		return nil
	}
	var existing *ContextError
	if errors.As(err, &existing) {
		return err
	}
	return &ContextError{Stage: stage, Page: page, Err: err}
}

// Context extracts a stable stage and page identifier from an error.
func Context(err error) (stage, page string, ok bool) {
	var contextual *ContextError
	if !errors.As(err, &contextual) || contextual == nil {
		return "", "", false
	}
	return contextual.Stage, contextual.Page, true
}
