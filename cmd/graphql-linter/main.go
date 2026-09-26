package main

import (
	"log/slog"
	"os"

	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/presentation"
	"github.com/schubergphilis/graphql-linter/internal/pkg/logging"
)

var Version string

func main() {
	logging.Setup(false)

	cliPresent := presentation.NewCLI(os.Args[1:], Version)

	err := cliPresent.Run()
	if err != nil {
		slog.Error("unable to run presentation layer", "error", err)
		os.Exit(1)
	}
}
