// Package logger provides a singleton structured logger backed by zerolog.
//
// Initialise once at startup with Init, then retrieve anywhere with Get.
//
//	TRACE (-1) → DEBUG (0) → INFO (1) → WARN (2) → ERROR (3)
package logger

import (
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// Options controls logger behaviour at initialisation time.
type Options struct {
	// Level is the minimum log level: trace, debug, info, warn, error.
	// Defaults to "info" when empty or unrecognised.
	Level string
	// Pretty enables human-friendly console output (coloured, text-based).
	// Use false in production to emit pure JSON.
	Pretty bool
	// Output is the writer logs are sent to. Defaults to os.Stdout.
	Output io.Writer
}

var (
	instance    zerolog.Logger
	once        sync.Once
	initialized bool
)

// Init initialises the singleton logger. Safe to call multiple times – only
// the first call has any effect (singleton guarantee via sync.Once).
func Init(opts Options) zerolog.Logger {
	once.Do(func() {
		zerolog.TimeFieldFormat = time.RFC3339Nano

		out := opts.Output
		if out == nil {
			out = os.Stdout
		}
		if opts.Pretty {
			out = zerolog.ConsoleWriter{Out: out, TimeFormat: time.RFC3339}
		}

		lvl := parseLevel(opts.Level)
		zerolog.SetGlobalLevel(lvl)

		instance = zerolog.New(out).
			Level(lvl).
			With().
			Timestamp().
			Caller().
			Logger()

		initialized = true
	})
	return instance
}

// Get returns the singleton logger. Panics if Init has not been called yet.
func Get() zerolog.Logger {
	if !initialized {
		panic("logger: Get() called before Init()")
	}
	return instance
}

// Reset tears down the singleton so that the next Init call rebuilds it.
// Intended for use in tests only.
func Reset() {
	once = sync.Once{}
	instance = zerolog.Logger{}
	initialized = false
}

// parseLevel converts a string to a zerolog.Level.
//
//	"trace" → TraceLevel (-1)
//	"debug" → DebugLevel ( 0)
//	"info"  → InfoLevel  ( 1)  ← default
//	"warn"  → WarnLevel  ( 2)
//	"error" → ErrorLevel ( 3)
func parseLevel(s string) zerolog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "trace":
		return zerolog.TraceLevel
	case "debug":
		return zerolog.DebugLevel
	case "warn", "warning":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}
