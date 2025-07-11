package presentation

import (
	"flag"
	"fmt"

	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/application"
	log "github.com/sirupsen/logrus"
)

type Presenter interface {
	Run() error
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
	applicationExecute, err := application.NewExecute(c.verboseFlag)
	if err != nil {
		return fmt.Errorf("unable to load new execute: %w", err)
	}

	if c.versionFlag {
		log.Info(applicationExecute.Version())

		return nil
	}

	if c.verboseFlag {
		log.Info("Verbose output enabled")
	}

	log.Info("Running main logic...")

	if err := applicationExecute.Run(); err != nil {
		return fmt.Errorf("unable to run execute: %w", err)
	}

	return nil
}
