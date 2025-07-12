package application

import (
	"fmt"
	"runtime/debug"

	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/data"
	log "github.com/sirupsen/logrus"
)

type Executor interface {
	Run() error
	Version()
}

type Execute struct {
	Verbose bool
}

func NewExecute(verbose bool) (Execute, error) {
	e := Execute{
		Verbose: verbose,
	}

	return e, nil
}

func (e Execute) Run() error {
	dataStore, err := data.NewStore(e.Verbose)
	if err != nil {
		return fmt.Errorf("unable to load new store: %w", err)
	}

	linterConfig, err := dataStore.LoadConfig()
	if err != nil {
		return fmt.Errorf("unable to load config: %w", err)
	}

	log.Infof("linter config: %v", linterConfig)

	schemaFiles, err := dataStore.FindAndLogGraphQLSchemaFiles()
	if err != nil {
		return fmt.Errorf("schema file discovery failed: %w", err)
	}

	errorCount := dataStore.LintSchemaFiles(schemaFiles)

	dataStore.PrintReport(schemaFiles, errorCount)

	return nil
}

func (e Execute) Version() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		return info.Main.Version
	}

	return "(unknown)"
}
