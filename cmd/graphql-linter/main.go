package main

import (
	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/presentation"
	log "github.com/sirupsen/logrus"
)

var Version string

func main() {
	cliPresent := presentation.NewCLI(presentation.NewFlag(), Version)

	err := cliPresent.Run()
	if err != nil {
		log.WithError(err).Fatal("unable to run presentation layer")
	}
}
