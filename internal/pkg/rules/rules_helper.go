package rules

import (
	"strings"

	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/data/base/models"
	log "github.com/sirupsen/logrus"
)

const (
	DefaultErrorCapacity = 32
	LevenshteinThreshold = 3
)

func SuggestDirective(directiveName, validName string) {
	if strings.Contains(directiveName, validName) ||
		LevenshteinDistance(directiveName, validName) <= LevenshteinThreshold {
		log.Errorf("  Did you mean '@%s'?", validName)
	}
}

func LevenshteinDistance(source, target string) int {
	if len(source) == 0 {
		return len(target)
	}

	if len(target) == 0 {
		return len(source)
	}

	matrix := newLevenshteinMatrix(len(source)+1, len(target)+1)
	fillLevenshteinBorders(matrix)
	computeLevenshtein(matrix, source, target)

	return matrix[len(source)][len(target)]
}

func newLevenshteinMatrix(rows, cols int) [][]int {
	matrix := make([][]int, rows)
	for i := range matrix {
		matrix[i] = make([]int, cols)
	}

	return matrix
}

func fillLevenshteinBorders(matrix [][]int) {
	for i := range matrix {
		matrix[i][0] = i
	}

	for j := range matrix[0] {
		matrix[0][j] = j
	}
}

func computeLevenshtein(matrix [][]int, source, target string) {
	for row := 1; row < len(matrix); row++ {
		sourceChar := source[row-1]

		for col := 1; col < len(matrix[0]); col++ {
			targetChar := target[col-1]

			cost := 0
			if sourceChar != targetChar {
				cost = 1
			}

			matrix[row][col] = min(
				matrix[row-1][col]+1,
				min(
					matrix[row][col-1]+1,
					matrix[row-1][col-1]+cost,
				),
			)
		}
	}
}

func IsSuppressed(
	filePath string,
	line int,
	modelsLinterConfig *models.LinterConfig,
	rule string,
	value string,
) bool {
	if modelsLinterConfig == nil || len(modelsLinterConfig.Suppressions) == 0 {
		return false
	}

	normalizedFilePath := strings.ReplaceAll(filePath, "\\", "/")
	for _, suppression := range modelsLinterConfig.Suppressions {
		if Matches(normalizedFilePath, line, rule, suppression, value) {
			log.Debugf("SUPPRESSED: %s at line %d in %s (reason: %s)",
				rule, line, filePath, suppression.Reason)

			return true
		}
	}

	return false
}

func Matches(
	filePath string,
	line int,
	rule string,
	modelsSuppression models.Suppression,
	value string,
) bool {
	normalizedSuppressionFile := strings.ReplaceAll(modelsSuppression.File, "\\", "/")
	normalizedFilePath := strings.ReplaceAll(filePath, "\\", "/")

	fileMatches := modelsSuppression.File == "" ||
		strings.HasSuffix(normalizedFilePath, normalizedSuppressionFile)
	lineMatches := modelsSuppression.Line == 0 || modelsSuppression.Line == line
	ruleMatches := modelsSuppression.Rule == "" || modelsSuppression.Rule == rule

	valueMatches := true
	if modelsSuppression.Value != "" {
		valueMatches = modelsSuppression.Value == value
	}

	return fileMatches && lineMatches && ruleMatches && valueMatches
}
