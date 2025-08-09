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

	matrix := make([][]int, len(source)+1)
	for row := range matrix {
		matrix[row] = make([]int, len(target)+1)
	}

	for row := 0; row <= len(source); row++ {
		matrix[row][0] = row
	}

	for col := 0; col <= len(target); col++ {
		matrix[0][col] = col
	}

	for row := 1; row <= len(source); row++ {
		for col := 1; col <= len(target); col++ {
			cost := 0
			if source[row-1] != target[col-1] {
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

	return matrix[len(source)][len(target)]
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
