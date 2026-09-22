package report

import (
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"

	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/data/base/models"
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

func NewSummary(
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
	summary := NewSummary(schemaFiles, totalErrors, passedFiles, allErrors)

	printDetailedErrors(summary.AllErrors)
	printErrorTypeSummary(summary.AllErrors)

	slog.Info(
		"linting summary",
		"passedFiles", summary.PassedFiles,
		"totalFiles", summary.TotalFiles,
		"percentPassed", fmt.Sprintf("%.2f%%", summary.PercentPassed),
	)

	if summary.TotalErrors > 0 {
		slog.Error(
			"files with at least one error",
			"filesWithAtLeastOneError", summary.FilesWithAtLeastOneError,
			"percentage", fmt.Sprintf("%.2f%%", summary.PercentageFilesWithErrors),
		)
		slog.Error(fmt.Sprintf("totalErrors: %d", summary.TotalErrors))
		os.Exit(1)
	}

	slog.Info(fmt.Sprintf("All %d schema file(s) passed linting successfully!", summary.TotalFiles))
}

func printDetailedErrors(errors []models.DescriptionError) {
	if len(errors) == 0 {
		return
	}

	for _, err := range errors {
		slog.Error(fmt.Sprintf("%s:%d: %s\n  %s", err.FilePath, err.LineNum, err.Message, err.LineContent))
	}
}

func printErrorTypeSummary(errors []models.DescriptionError) {
	errorTypeCountsMap := ErrorTypeCounts(errors)

	if len(errorTypeCountsMap) == 0 {
		return
	}

	slog.Error("Error type summary:")

	keys := make([]string, 0, len(errorTypeCountsMap))
	for k := range errorTypeCountsMap {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, k := range keys {
		slog.Error(fmt.Sprintf("  %s: %d", k, errorTypeCountsMap[k]))
	}
}

func ErrorTypeCounts(errors []models.DescriptionError) map[string]int {
	counts := make(map[string]int)

	for _, err := range errors {
		msg := err.Message

		typeKey := msg
		if before, _, found := strings.Cut(msg, ":"); found {
			typeKey = before
		} else if before, _, found := strings.Cut(msg, " "); found {
			typeKey = before
		}

		counts[typeKey]++
	}

	return counts
}

func InternalErrors(parseReport *operationreport.Report) {
	for i, internalErr := range parseReport.InternalErrors {
		slog.Error(fmt.Sprintf("Internal Error %d: %v", i+1, internalErr))
	}
}

func ExternalErrors(
	schemaString string,
	parseReport *operationreport.Report,
	linesBeforeContext, linesAfterContext int,
) {
	lines := strings.Split(schemaString, "\n")

	for index, externalErr := range parseReport.ExternalErrors {
		slog.Error(fmt.Sprintf("External Error %d:", index+1))
		slog.Error("  Message: " + externalErr.Message)
		slog.Error(fmt.Sprintf("  Path: %s", externalErr.Path))
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
		slog.Info(fmt.Sprintf("  Location: Line %d, Column %d", location.Line, location.Column))
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

	slog.Info("  Problematic line: " + lines[errorLineIdx])

	startIdx := max(0, errorLineIdx-linesBeforeContext)
	endIdx := min(len(lines), errorLineIdx+linesAfterContext+1)

	slog.Info("  Context:")

	for contextIdx := startIdx; contextIdx < endIdx; contextIdx++ {
		marker := "  "
		if contextIdx == errorLineIdx {
			marker = ">>>"
		}

		slog.Info(fmt.Sprintf("  %s Line %d: %s", marker, contextIdx+1, lines[contextIdx]))
	}
}
