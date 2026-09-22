package main

import (
	"log/slog"
	"os"

	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/presentation"
)

var Version string

func main() {
	cliPresent := presentation.NewCLI(presentation.NewFlag(), Version)

	err := cliPresent.Run()
	if err != nil {
		slog.Error("unable to run presentation layer", "error", err)
		os.Exit(1)
	}
}
