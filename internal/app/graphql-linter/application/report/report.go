package report

import (
	"fmt"
	"sort"
	"strings"

	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/data/base/models"
	log "github.com/sirupsen/logrus"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/operationreport"
)

const (
	percentMultiplier = 100
)

type Summary struct {
	TotalFiles                int
	PassedFiles               int
	TotalErrors               int
	PercentPassed             float64
	PercentageFilesWithErrors float64
	FilesWithAtLeastOneError  int
	AllErrors                 []models.DescriptionError
}

func SummarizeLintResults(
	unsuppressedDescErrs int,
	hasUnsuppressedDeprecationReasonError bool,
	unsuppressedDataTypeErrors int,
	unsuppressedDirectiveOrFederationError bool,
) (int, int) {
	totalErrors := 0
	errorFilesCount := 0

	if unsuppressedDescErrs > 0 || hasUnsuppressedDeprecationReasonError ||
		unsuppressedDataTypeErrors > 0 ||
		unsuppressedDirectiveOrFederationError {
		totalErrors += unsuppressedDescErrs + unsuppressedDataTypeErrors
		if unsuppressedDirectiveOrFederationError {
			totalErrors++
		}

		errorFilesCount++
	}

	return totalErrors, errorFilesCount
}

func ReportSummary(
	schemaFiles []string,
	totalErrors int,
	passedFiles int,
	allErrors []models.DescriptionError,
) Summary {
	totalFiles := len(schemaFiles)

	percentPassed := 0.0
	if totalFiles > 0 {
		percentPassed = float64(passedFiles) / float64(totalFiles) * percentMultiplier
	}

	percentageFilesWithErrors := 0.0

	filesWithAtLeastOneError := totalFiles - passedFiles
	if totalFiles > 0 {
		percentageFilesWithErrors = float64(
			filesWithAtLeastOneError,
		) / float64(
			totalFiles,
		) * percentMultiplier
	}

	return Summary{
		TotalFiles:                totalFiles,
		PassedFiles:               passedFiles,
		TotalErrors:               totalErrors,
		PercentPassed:             percentPassed,
		PercentageFilesWithErrors: percentageFilesWithErrors,
		FilesWithAtLeastOneError:  filesWithAtLeastOneError,
		AllErrors:                 allErrors,
	}
}

func Print(
	schemaFiles []string,
	totalErrors int,
	passedFiles int,
	allErrors []models.DescriptionError,
) {
	summary := ReportSummary(schemaFiles, totalErrors, passedFiles, allErrors)

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

func printDetailedErrors(errors []models.DescriptionError) {
	if len(errors) == 0 {
		return
	}

	for _, err := range errors {
		log.Errorf("%s:%d: %s\n  %s", err.FilePath, err.LineNum, err.Message, err.LineContent)
	}
}

func printErrorTypeSummary(errors []models.DescriptionError) {
	errorTypeCountsMap := ErrorTypeCounts(errors)

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

func ErrorTypeCounts(errors []models.DescriptionError) map[string]int {
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

func ReportInternalErrors(parseReport *operationreport.Report) {
	for i, internalErr := range parseReport.InternalErrors {
		log.Errorf("Internal Error %d: %v\n", i+1, internalErr)
	}
}

func ReportExternalErrors(
	schemaString string,
	parseReport *operationreport.Report,
	linesBeforeContext, linesAfterContext int,
) {
	lines := strings.Split(schemaString, "\n")

	for index, externalErr := range parseReport.ExternalErrors {
		log.Errorf("External Error %d:\n", index+1)
		log.Errorf("  Message: %s\n", externalErr.Message)
		log.Errorf("  Path: %s\n", externalErr.Path)
		reportExternalErrorLocations(lines, externalErr, linesBeforeContext, linesAfterContext)
	}
}

func reportExternalErrorLocations(
	lines []string,
	externalErr operationreport.ExternalError,
	linesBeforeContext, linesAfterContext int,
) {
	if externalErr.Locations == nil {
		return
	}

	for _, location := range externalErr.Locations {
		log.Infof("  Location: Line %d, Column %d\n", location.Line, location.Column)
		reportContextLines(lines, int(location.Line), linesBeforeContext, linesAfterContext)
	}
}

func reportContextLines(
	lines []string,
	lineNumber int,
	linesBeforeContext, linesAfterContext int,
) {
	errorLineIdx := lineNumber - 1
	if errorLineIdx < 0 || errorLineIdx >= len(lines) {
		return
	}

	log.Infof("  Problematic line: %s\n", lines[errorLineIdx])

	startIdx := max(0, errorLineIdx-linesBeforeContext)
	endIdx := min(len(lines), errorLineIdx+linesAfterContext+1)

	log.Infof("  Context:")

	for contextIdx := startIdx; contextIdx < endIdx; contextIdx++ {
		marker := "  "
		if contextIdx == errorLineIdx {
			marker = ">>>"
		}

		log.Infof("  %s Line %d: %s\n", marker, contextIdx+1, lines[contextIdx])
	}
}
