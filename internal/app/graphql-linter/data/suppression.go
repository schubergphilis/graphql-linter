// Package suppression provides functionality for suppressing linting errors
package data

import "strings"

// Matches checks if this suppression applies to the given error context
func (s Suppression) Matches(filePath string, line int, rule string, value string) bool {
	// Check file match
	if s.File != "" && !strings.HasSuffix(filePath, s.File) {
		return false
	}

	// Check line match
	if s.Line != 0 && s.Line != line {
		return false
	}

	// Check rule match
	if s.Rule != "" && s.Rule != rule {
		return false
	}

	// Check value match
	if s.Value != "" && s.Value != value {
		return false
	}

	return true
}

// IsSuppressed checks if an error should be suppressed based on the configuration
func (s Store) IsSuppressed(filePath string, line int, rule string, value string) bool {
	if s.LinterConfig == nil {
		return false
	}

	for _, suppression := range s.LinterConfig.Suppressions {
		if suppression.Matches(filePath, line, rule, value) {
			return true
		}
	}

	return false
}
