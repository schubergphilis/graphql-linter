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

type Debugger interface {
	ReadBuildInfo() (info *debug.BuildInfo, ok bool)
}

type Debug struct{}

type Execute struct {
	Debugger      Debugger
	TargetPath    string
	Verbose       bool
	VersionString string
}

func NewExecute(targetPath string, verbose bool, versionString string) (Execute, error) {
	execute := Execute{
		Debugger:      Debug{},
		TargetPath:    targetPath,
		Verbose:       verbose,
		VersionString: versionString,
	}

	return execute, nil
}

func (Debug) ReadBuildInfo() (*debug.BuildInfo, bool) {
	return debug.ReadBuildInfo()
}

func (e Execute) Run() error {
	dataStore, err := data.NewStore(e.TargetPath, e.Verbose)
	if err != nil {
		return fmt.Errorf("unable to load new store: %w", err)
	}

	linterConfig, err := dataStore.LoadConfig()
	if err != nil {
		return fmt.Errorf("unable to load config: %w", err)
	}

	log.Debugf("linter config: %v", linterConfig)
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

	if info, ok := e.Debugger.ReadBuildInfo(); ok {
		return info.Main.Version
	}

	return "(unknown)"
}
