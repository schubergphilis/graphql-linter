package main

import (
	"log/slog"
	"os"

	"github.com/schubergphilis/graphql-linter/internal/app/graphql-testdata-generator/presentation"
)

var Version string

func main() {
	cliPresent, err := presentation.NewCLI()
	if err != nil {
		slog.Error("failed to construct CLIPresent", "error", err)
		os.Exit(1)
	}

	err = cliPresent.Run()
	if err != nil {
		slog.Error("unable to run presentation layer", "error", err)
		os.Exit(1)
	}
}
