// Package logging installs the process wide slog handler, so that every binary
// in this repository logs in the same format.
package logging

import (
	"io"
	"log/slog"
	"os"
)

// Setup installs a TextHandler on stderr as the default slog logger. Verbose
// raises the level to debug and adds the time and source location.
func Setup(verbose bool) {
	slog.SetDefault(slog.New(newHandler(os.Stderr, verbose)))
}

func newHandler(writer io.Writer, verbose bool) slog.Handler {
	options := &slog.HandlerOptions{
		AddSource: verbose,
		Level:     slog.LevelInfo,
	}

	if verbose {
		options.Level = slog.LevelDebug
	} else {
		// A run takes milliseconds, so the time is the same on every line.
		options.ReplaceAttr = dropTime
	}

	return slog.NewTextHandler(writer, options)
}

func dropTime(groups []string, attr slog.Attr) slog.Attr {
	if len(groups) == 0 && attr.Key == slog.TimeKey {
		return slog.Attr{}
	}

	return attr
}
