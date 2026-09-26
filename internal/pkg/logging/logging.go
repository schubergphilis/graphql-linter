// Package logging installs the process wide slog handler, so that every binary
// in this repository logs in the same format.
package logging

import (
	"log/slog"
	"os"
)

// Setup installs a TextHandler on stderr as the default slog logger. Verbose
// raises the level to debug and adds the source location.
func Setup(verbose bool) {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		AddSource: verbose,
		Level:     level,
	})))
}
