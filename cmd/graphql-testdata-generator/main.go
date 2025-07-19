package main

import (
	"github.com/schubergphilis/graphql-linter/internal/app/graphql-testdata-generator/presentation"
	log "github.com/sirupsen/logrus"
)

var Version string

func main() {
	cliPresent, err := presentation.NewCLI()
	if err != nil {
		log.WithError(err).Fatal("failed to construct CLIPresent")
	}

	if err := cliPresent.Run(); err != nil {
		log.WithError(err).Fatal("unable to run presentation layer")
	}
}
