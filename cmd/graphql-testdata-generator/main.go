package main

import (
	"log/slog"
	"os"

	"github.com/schubergphilis/graphql-linter/internal/app/graphql-testdata-generator/presentation"
	"github.com/schubergphilis/graphql-linter/internal/pkg/logging"
)

var Version string

func main() {
	logging.Setup(false)

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
