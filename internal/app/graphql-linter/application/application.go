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
	Verbose       bool
	VersionString string
}

func NewExecute(verbose bool, versionString string) (Execute, error) {
	e := Execute{
		Verbose:       verbose,
		VersionString: versionString,
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
	dataStore.LinterConfig = linterConfig

	schemaFiles, err := dataStore.FindAndLogGraphQLSchemaFiles()
	if err != nil {
		return fmt.Errorf("schema file discovery failed: %w", err)
	}

	errorCount := dataStore.LintSchemaFiles(schemaFiles)

	dataStore.PrintReport(schemaFiles, errorCount)

	return nil
}

func (e Execute) Version() string {
	if e.VersionString != "" {
		return e.VersionString
	}

	if info, ok := debug.ReadBuildInfo(); ok {
		return info.Main.Version
	}

	return "(unknown)"
}
