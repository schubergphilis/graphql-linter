package data

import (
	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/data/fileutil"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/ast"
)

// This file provides backwards compatibility while functions have been modularized into:
// - ast/helpers.go: AST utility functions
// - stringutils/utils.go: String manipulation functions
// - validation.go: Type and directive validation
// - rules.go: Linting rules and checks
// - suppression.go: Error suppression functionality
// - report.go: Error reporting and logging

// Backwards compatibility aliases - delegate to the reorganized functions
var (
	findLineNumberByText             = FindLineNumberByText
	getBaseTypeName                  = GetBaseTypeName
	getAvailableTypes                = GetAvailableTypes
	isAlphaUnderOrDigit              = IsAlphaUnderOrDigit
	isValidEnumValue                 = IsValidEnumValue
	levenshteinDistance              = LevenshteinDistance
	hasSuspiciousEnumValue           = HasSuspiciousEnumValue
	hasEmbeddedDigits                = HasEmbeddedDigits
	suggestCorrectEnumValue          = SuggestCorrectEnumValue
	removeSuffixDigits               = RemoveSuffixDigits
	removeAllDigits                  = RemoveAllDigits
	validateDirectiveNames           = ValidateDirectiveNames
	validateDirectives               = ValidateDirectives
	reportDirectiveError             = ReportDirectiveError
	reportUncapitalizedDescription   = ReportUncapitalizedDescription
	findMissingArgumentDescriptions  = FindMissingArgumentDescriptions
	filterSchemaComments             = FilterSchemaComments
	getLineContent                   = GetLineContent
	findFieldDefinitionLine          = FindFieldDefinitionLine
	findUnsortedInterfaceFields      = FindUnsortedInterfaceFields
	findRelayPageInfoSpec            = FindRelayPageInfoSpec
	isCamelCase                      = IsCamelCase
	findInputObjectValuesCamelCased  = FindInputObjectValuesCamelCased
	findMissingEnumValueDescriptions = FindMissingEnumValueDescriptions
	logSchemaParseErrors             = LogSchemaParseErrors
	reportInternalErrors             = ReportInternalErrors
	reportExternalErrors             = ReportExternalErrors
	reportExternalErrorLocations     = ReportExternalErrorLocations
	reportContextLines               = ReportContextLines
) // Delegate method calls to new structure
func (s Store) lintDescriptions(doc *ast.Document, schemaString string, schemaPath string) ([]DescriptionError, bool) {
	return s.LintDescriptions(doc, schemaString, schemaPath)
}

func (s Store) validateDataTypes(doc *ast.Document, schemaContent string, schemaPath string) (bool, []int) {
	return s.ValidateDataTypes(doc, schemaContent, schemaPath)
}

func (s Store) isSuppressed(filePath string, line int, rule string, value string) bool {
	return s.IsSuppressed(filePath, line, rule, value)
}

func (s Store) validateEnumTypes(doc *ast.Document, schemaContent string, schemaPath string) ([]string, []int) {
	return s.ValidateEnumTypes(doc, schemaContent, schemaPath)
}

func readSchemaFile(schemaPath string) (string, bool) {
	return fileutil.ReadSchemaFile(schemaPath)
}
