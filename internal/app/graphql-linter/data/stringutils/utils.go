package stringutils

import (
	"strings"
	"unicode"
)

// FindLineNumberByText finds the line number where the given text appears in the content
func FindLineNumberByText(schemaContent string, searchText string) int {
	lines := strings.Split(schemaContent, "\n")
	for i, line := range lines {
		if strings.Contains(line, searchText) {
			return i + 1
		}
	}

	return 0
}

// IsAlphaUnderOrDigit checks if a rune is a letter, digit, or underscore
func IsAlphaUnderOrDigit(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

// IsValidEnumValue validates if a string is a valid GraphQL enum value
func IsValidEnumValue(value string) bool {
	if len(value) == 0 {
		return false
	}

	r := rune(value[0])
	if !unicode.IsLetter(r) && r != '_' {
		return false
	}

	for _, r := range value[1:] {
		if !IsAlphaUnderOrDigit(r) {
			return false
		}
	}

	return true
}

// LevenshteinDistance calculates the edit distance between two strings
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

// HasSuspiciousEnumValue checks if an enum value ends with a digit
func HasSuspiciousEnumValue(value string) bool {
	if len(value) == 0 {
		return false
	}

	lastChar := value[len(value)-1]

	return lastChar >= '0' && lastChar <= '9'
}

// HasEmbeddedDigits checks if a string contains any digits
func HasEmbeddedDigits(value string) bool {
	for _, char := range value {
		if char >= '0' && char <= '9' {
			return true
		}
	}

	return false
}

// RemoveSuffixDigits removes trailing digits from a string
func RemoveSuffixDigits(value string) string {
	result := value
	for len(result) > 0 {
		lastChar := result[len(result)-1]
		if lastChar >= '0' && lastChar <= '9' {
			result = result[:len(result)-1]
		} else {
			break
		}
	}

	return result
}

// RemoveAllDigits removes all digits from a string
func RemoveAllDigits(value string) string {
	result := ""

	for _, char := range value {
		if char < '0' || char > '9' {
			result += string(char)
		}
	}

	return result
}

// IsCamelCase checks if a string follows camelCase convention
func IsCamelCase(str string) bool {
	if str == "" {
		return false
	}

	if strings.Contains(str, "_") {
		return false
	}

	return str[0] >= 'a' && str[0] <= 'z'
}

// GetLineContent extracts a specific line from multi-line content
func GetLineContent(schemaContent string, lineNum int) string {
	if lineNum <= 0 {
		return ""
	}

	lines := strings.Split(schemaContent, "\n")
	if lineNum > len(lines) {
		return ""
	}

	return strings.TrimSpace(lines[lineNum-1])
}

// FilterSchemaComments removes comment lines from schema content
func FilterSchemaComments(schemaString string) string {
	lines := strings.Split(schemaString, "\n")

	var filteredLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "//") {
			filteredLines = append(filteredLines, line)
		}
	}

	return strings.Join(filteredLines, "\n")
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}

	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}

	return b
}
