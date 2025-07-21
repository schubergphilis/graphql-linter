package main

import (
	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/presentation"
	log "github.com/sirupsen/logrus"
)

var Version string

func main() {
	cliPresent, err := presentation.NewCLI(Version)
	if err != nil {
		log.WithError(err).Fatal("failed to construct CLIPresent")
	}

	err = cliPresent.Run()
	if err != nil {
		log.WithError(err).Fatal("unable to run presentation layer")
	}
}
