package presentation

import (
	"flag"

	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/application"
	log "github.com/sirupsen/logrus"
)

type Presenter interface {
	Run()
}

type CLI struct {
	versionFlag bool
	verboseFlag bool
}

func NewCLI() (CLI, error) {
	cli := CLI{}
	flag.BoolVar(&cli.versionFlag, "version", false, "Show version")
	flag.BoolVar(&cli.verboseFlag, "verbose", false, "Enable verbose output")
	flag.Parse()

	return cli, nil
}

func (c CLI) Run() error {
	if c.versionFlag {
		log.Info(application.VersionString())

		return nil
	}

	if c.verboseFlag {
		log.Info("Verbose output enabled")
	}

	log.Info("Running main logic...")

	return nil
}
