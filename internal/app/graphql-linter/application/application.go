package application

import (
	"fmt"
	"runtime/debug"
	"sort"
	"strings"

	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/data"
	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/data/federation"
	log "github.com/sirupsen/logrus"
)

type Executor interface {
	Run() error
	Version()
	PrintReport(schemaFiles []string, totalErrors int, passedFiles int, allErrors []data.DescriptionError)
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

	for _, schemaFile := range schemaFiles {
		schemaString, ok := dataStore.ReadAndValidateSchemaFile(schemaFile)
		if !ok {
			return fmt.Errorf("failed to read schema file: %s", schemaFile)
		}

		filteredSchema := data.FilterSchemaComments(schemaString)
		if !federation.ValidateFederationSchema(filteredSchema) {
			return fmt.Errorf("federation validation failed for: %s", schemaFile)
		}
	}

	totalErrors, errorFilesCount, dataDescriptionError := dataStore.LintSchemaFiles(schemaFiles)

	// Use PrintReport from application package instead of Store method
	PrintReport(
		schemaFiles,
		totalErrors,
		len(schemaFiles)-errorFilesCount,
		dataDescriptionError,
	)

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

// PrintReport prints a summary report of the linting results.
func PrintReport(
	schemaFiles []string,
	totalErrors int,
	passedFiles int,
	allErrors []data.DescriptionError,
) {
	summary := data.Store{}.ReportSummary(schemaFiles, totalErrors, passedFiles, allErrors)

	printDetailedErrors(summary.AllErrors)
	printErrorTypeSummary(summary.AllErrors)

	log.WithFields(log.Fields{
		"passedFiles":   summary.PassedFiles,
		"totalFiles":    summary.TotalFiles,
		"percentPassed": fmt.Sprintf("%.2f%%", summary.PercentPassed),
	}).Info("linting summary")

	if summary.TotalErrors > 0 {
		log.WithFields(log.Fields{
			"filesWithAtLeastOneError": summary.FilesWithAtLeastOneError,
			"percentage":               fmt.Sprintf("%.2f%%", summary.PercentageFilesWithErrors),
		}).Error("files with at least one error")

		log.Fatalf("totalErrors: %d", summary.TotalErrors)

		return
	}

	log.Infof("All %d schema file(s) passed linting successfully!", summary.TotalFiles)
}

func printDetailedErrors(errors []data.DescriptionError) {
	if len(errors) == 0 {
		return
	}

	for _, err := range errors {
		log.Errorf("%s:%d: %s\n  %s", err.FilePath, err.LineNum, err.Message, err.LineContent)
	}
}

func printErrorTypeSummary(errors []data.DescriptionError) {
	errorTypeCountsMap := errorTypeCounts(errors)

	if len(errorTypeCountsMap) == 0 {
		return
	}

	log.Error("Error type summary:")

	keys := make([]string, 0, len(errorTypeCountsMap))
	for k := range errorTypeCountsMap {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, k := range keys {
		log.Errorf("  %s: %d", k, errorTypeCountsMap[k])
	}
}

func errorTypeCounts(errors []data.DescriptionError) map[string]int {
	counts := make(map[string]int)

	for _, err := range errors {
		msg := err.Message

		typeKey := msg
		if idx := strings.Index(msg, ":"); idx != -1 {
			typeKey = msg[:idx]
		} else if idx := strings.Index(msg, " "); idx != -1 {
			typeKey = msg[:idx]
		}

		counts[typeKey]++
	}

	return counts
}
