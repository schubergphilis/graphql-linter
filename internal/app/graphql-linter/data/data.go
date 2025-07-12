package data

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/schubergphilis/mcvs-golang-project-root/pkg/projectroot"
	log "github.com/sirupsen/logrus"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/ast"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/astparser"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/federation"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/lexer/position"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/operationreport"
	"gopkg.in/yaml.v3"
)

const (
	levenshteinThreshold = 2
	linesAfterContext    = 3
	linesBeforeContext   = 2
)

type Storer interface {
	LoadConfig(configPath string) (*LinterConfig, error)
	Run() error
}

type Store struct {
	LinterConfig *LinterConfig
	Verbose      bool
}

type LinterConfig struct {
	Suppressions []Suppression `yaml:"suppressions"`
	Settings     Settings      `yaml:"settings"`
}

type Suppression struct {
	File   string `yaml:"file"`
	Line   int    `yaml:"line"`
	Rule   string `yaml:"rule"`
	Value  string `yaml:"value"`
	Reason string `yaml:"reason"`
}

type Settings struct {
	StrictMode         bool `yaml:"strictMode"`
	ValidateFederation bool `yaml:"validateFederation"`
	CheckDescriptions  bool `yaml:"checkDescriptions"`
}

type DescriptionError struct {
	LineNum     int
	Message     string
	LineContent string
}

func NewStore(verbose bool) (Store, error) {
	s := Store{
		Verbose: verbose,
	}

	return s, nil
}

func (s Store) Run() error {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	configPath := filepath.Join(projectRoot, ".graphql-linter.yml")

	linterConfig, err := s.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("unable to load config: %w", err)
	}

	log.Infof("linter config: %v", linterConfig)

	targetPath := projectRoot

	schemaFiles, err := findGraphQLFiles(targetPath)
	if err != nil {
		return fmt.Errorf("unable to find graphql files: %w", err)
	}

	if len(schemaFiles) == 0 {
		return fmt.Errorf("no GraphQL schema files found in directory: %s", targetPath)
	}

	if s.Verbose {
		log.Infof("found %d GraphQL schema files:", len(schemaFiles))

		for _, file := range schemaFiles {
			log.Infof("  - %s", file)
		}
	}

	totalErrors := 0

	for _, schemaFile := range schemaFiles {
		if s.Verbose {
			log.Infof("=== Linting %s ===", schemaFile)
		}

		if !s.lintSchemaFile(schemaFile) {
			totalErrors++
		}
	}

	printReport(schemaFiles, totalErrors)

	return nil
}

func (s Store) lintSchemaFile(schemaPath string) bool {
	schemaBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		log.WithError(err).Error("failed to read schema file")

		return false
	}

	// Don't filter out federation directives - parse as federation schema
	schemaString := string(schemaBytes)

	// Create filtered version only for federation validation (remove comments)
	lines := strings.Split(schemaString, "\n")

	var filteredLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "//") {
			filteredLines = append(filteredLines, line)
		}
	}

	filteredSchema := strings.Join(filteredLines, "\n")

	log.Debugf("Parsing federation schema from: %s\n", schemaPath)
	log.Debugf("Schema content length: %d bytes\n", len(schemaString))

	// Parse the document using the original schema (with comments) to preserve line numbers
	doc, parseReport := astparser.ParseGraphqlDocumentString(schemaString)

	if parseReport.HasErrors() {
		log.Errorf("Failed to parse schema - found %d errors:\n",
			len(parseReport.InternalErrors)+len(parseReport.ExternalErrors))

		// Show internal parser errors
		for i, internalErr := range parseReport.InternalErrors {
			log.Errorf("Internal Error %d: %v\n", i+1, internalErr)
		}

		// Show external parser errors with more details
		for index, externalErr := range parseReport.ExternalErrors {
			log.Errorf("External Error %d:\n", index+1)
			log.Errorf("  Message: %s\n", externalErr.Message)
			log.Errorf("  Path: %s\n", externalErr.Path)

			if externalErr.Locations != nil {
				for _, location := range externalErr.Locations {
					log.Infof("  Location: Line %d, Column %d\n", location.Line, location.Column)

					// Show the problematic line
					lines := strings.Split(schemaString, "\n")
					if int(location.Line) <= len(lines) && location.Line > 0 {
						errorLineIdx := int(location.Line) - 1
						log.Infof("  Problematic line: %s\n", lines[errorLineIdx])

						// Show context around the error
						startIdx := max(0, errorLineIdx-linesBeforeContext)
						endIdx := min(len(lines), errorLineIdx+linesAfterContext)

						log.Infof("  Context:\n")

						for contextIdx := startIdx; contextIdx < endIdx; contextIdx++ {
							marker := "  "
							if contextIdx == errorLineIdx {
								marker = ">>>"
							}

							log.Infof("  %s Line %d: %s\n", marker, contextIdx+1, lines[contextIdx])
						}
					}
				}
			}
		}

		return false
	}

	log.Debug("Schema parsed successfully!")

	// Track if any validation fails
	var hasValidationErrors bool

	// Validate directive names for typos
	if !validateDirectiveNames(&doc) {
		hasValidationErrors = true
	}

	// Validate data types and type references (use original schema with comments)
	dataTypesValid, errorLines := s.validateDataTypes(&doc, schemaString, schemaPath)
	if !dataTypesValid {
		hasValidationErrors = true
	}

	// Use federation package to build and validate the schema
	var report operationreport.Report

	// Build federation schema from the filtered schema string
	federationSchema, federationErr := federation.BuildFederationSchema(filteredSchema, filteredSchema)
	if federationErr != nil {
		log.Infof("Federation schema build failed: %v\n", federationErr)

		hasValidationErrors = true
	}

	// Check for federation validation errors in the report
	if report.HasErrors() {
		log.Error("Federation validation errors:")

		for _, internalErr := range report.InternalErrors {
			log.Errorf("  - %v\n", internalErr)
		}

		for _, externalErr := range report.ExternalErrors {
			log.Errorf("  - %s\n", externalErr.Message)
		}

		hasValidationErrors = true
	} else {
		log.Debug("Federation schema validation passed")
	}

	_ = federationSchema // Use the federation schema

	// Collect all description errors first, then sort and display them
	type DescriptionError struct {
		LineNum     int
		Message     string
		LineContent string
	}

	var descriptionErrors []DescriptionError

	// Linter: Check that all type and enum definitions have a description
	for _, obj := range doc.ObjectTypeDefinitions {
		if !obj.Description.IsDefined {
			name := doc.Input.ByteSliceString(obj.Name)
			// Find the line number for this type definition
			lineNum := findLineNumberByText(schemaString, "type "+name)
			lineContent := getLineContent(schemaString, lineNum)
			message := fmt.Sprintf("ERROR: Object type '%s' is missing a description", name)
			descriptionErrors = append(descriptionErrors, DescriptionError{
				LineNum:     lineNum,
				Message:     message,
				LineContent: lineContent,
			})

			if lineNum > 0 {
				errorLines = append(errorLines, lineNum)
			}

			hasValidationErrors = true
		}

		// Check that all fields have descriptions
		for _, fieldRef := range obj.FieldsDefinition.Refs {
			fieldDef := doc.FieldDefinitions[fieldRef]
			if !fieldDef.Description.IsDefined {
				fieldName := doc.Input.ByteSliceString(fieldDef.Name)
				typeName := doc.Input.ByteSliceString(obj.Name)

				// Get the field type to make the search more specific
				fieldType := doc.Types[fieldDef.Type]
				baseType := getBaseTypeName(&doc, fieldType)

				// Find the line number for this specific field definition
				lineNum := findFieldDefinitionLine(schemaString, fieldName, baseType)
				if lineNum == 0 {
					// Fallback: look for the field name with colon
					lineNum = findLineNumberByText(schemaString, fieldName+":")
				}

				lineContent := getLineContent(schemaString, lineNum)
				message := fmt.Sprintf("ERROR: Field '%s' in type '%s' is missing a description", fieldName, typeName)
				descriptionErrors = append(descriptionErrors, DescriptionError{
					LineNum:     lineNum,
					Message:     message,
					LineContent: lineContent,
				})

				if lineNum > 0 {
					errorLines = append(errorLines, lineNum)
				}

				hasValidationErrors = true
			}
		}
	}

	for _, enum := range doc.EnumTypeDefinitions {
		if !enum.Description.IsDefined {
			name := doc.Input.ByteSliceString(enum.Name)
			// Find the line number for this enum definition
			lineNum := findLineNumberByText(schemaString, "enum "+name)
			lineContent := getLineContent(schemaString, lineNum)
			message := fmt.Sprintf("ERROR: Enum '%s' is missing a description", name)
			descriptionErrors = append(descriptionErrors, DescriptionError{
				LineNum:     lineNum,
				Message:     message,
				LineContent: lineContent,
			})

			if lineNum > 0 {
				errorLines = append(errorLines, lineNum)
			}

			hasValidationErrors = true
		}
	}

	// Sort errors by line number and display them
	if len(descriptionErrors) > 0 {
		for i := range len(descriptionErrors) - 1 {
			for j := range len(descriptionErrors) - i - 1 {
				if descriptionErrors[j].LineNum > descriptionErrors[j+1].LineNum {
					descriptionErrors[j], descriptionErrors[j+1] = descriptionErrors[j+1], descriptionErrors[j]
				}
			}
		}

		// Display sorted errors
		for _, err := range descriptionErrors {
			log.Infof("%s %s:%d\n", err.Message, schemaPath, err.LineNum)
			log.Infof("  %d: %s\n", err.LineNum, err.LineContent)
		}
	}

	if hasValidationErrors {
		// Show schema path with line numbers if we have specific errors
		if len(errorLines) > 0 {
			// Use the first error line for the main error message
			log.Infof("Schema linting FAILED: %s:%d\n", schemaPath, errorLines[0])
		} else {
			log.Infof("Schema linting FAILED: %s\n", schemaPath)
		}

		return false
	} else {

		log.Debugf("Schema linting PASSED: %s\n", schemaPath)

		return true
	}
}

func (s Store) validateDataTypes(doc *ast.Document, schemaContent string, schemaPath string) (bool, []int) {
	log.Info("Validating data types...")

	var errorLines []int

	// Built-in GraphQL scalar types
	builtInScalars := map[string]bool{
		"String":  true,
		"Int":     true,
		"Float":   true,
		"Boolean": true,
		"ID":      true,
	}

	// Collect all defined types from the schema
	definedTypes := make(map[string]bool)

	// Add object types
	for _, obj := range doc.ObjectTypeDefinitions {
		typeName := doc.Input.ByteSliceString(obj.Name)
		definedTypes[typeName] = true
	}

	// Add enum types
	for _, enum := range doc.EnumTypeDefinitions {
		typeName := doc.Input.ByteSliceString(enum.Name)
		definedTypes[typeName] = true
	}

	// Add input types
	for _, input := range doc.InputObjectTypeDefinitions {
		typeName := doc.Input.ByteSliceString(input.Name)
		definedTypes[typeName] = true
	}

	// Add interface types
	for _, iface := range doc.InterfaceTypeDefinitions {
		typeName := doc.Input.ByteSliceString(iface.Name)
		definedTypes[typeName] = true
	}

	// Add union types
	for _, union := range doc.UnionTypeDefinitions {
		typeName := doc.Input.ByteSliceString(union.Name)
		definedTypes[typeName] = true
	}

	// Add scalar types
	for _, scalar := range doc.ScalarTypeDefinitions {
		typeName := doc.Input.ByteSliceString(scalar.Name)
		definedTypes[typeName] = true
	}

	hasErrors := false // Validate field types

	for _, fieldDef := range doc.FieldDefinitions {
		fieldName := doc.Input.ByteSliceString(fieldDef.Name)

		// Get the base type (unwrap lists and non-nulls)
		fieldType := doc.Types[fieldDef.Type]
		baseType := getBaseTypeName(doc, fieldType)

		if !builtInScalars[baseType] && !definedTypes[baseType] {
			// Find the line number in the original schema (with comments)
			lineNum := findFieldDefinitionLine(schemaContent, fieldName, baseType)
			if lineNum == 0 {
				// More specific fallback: look for "fieldName: baseType" pattern
				lineNum = findLineNumberByText(schemaContent, fieldName+": "+baseType)
			}

			if lineNum == 0 {
				// Generic fallback
				lineNum = findLineNumberByText(schemaContent, fieldName+":")
			}

			log.Errorf("ERROR: Field '%s' references undefined type '%s' (line %d)\n", fieldName, baseType, lineNum)
			log.Errorf("  Available types: %v\n", getAvailableTypes(builtInScalars, definedTypes))

			if lineNum > 0 {
				errorLines = append(errorLines, lineNum)
			}

			hasErrors = true
		}
	}

	// Validate input field types
	for _, inputFieldDef := range doc.InputValueDefinitions {
		fieldName := doc.Input.ByteSliceString(inputFieldDef.Name)

		// Get the base type (unwrap lists and non-nulls)
		inputFieldType := doc.Types[inputFieldDef.Type]
		baseType := getBaseTypeName(doc, inputFieldType)

		if !builtInScalars[baseType] && !definedTypes[baseType] {
			// Find the line number in the original schema (with comments)
			lineNum := findFieldDefinitionLine(schemaContent, fieldName, baseType)
			if lineNum == 0 {
				// More specific fallback: look for "fieldName: baseType" pattern
				lineNum = findLineNumberByText(schemaContent, fieldName+": "+baseType)
			}

			if lineNum == 0 {
				// Generic fallback
				lineNum = findLineNumberByText(schemaContent, fieldName+":")
			}

			log.Errorf("ERROR: Input field '%s' references undefined type '%s' (line %d)\n", fieldName, baseType, lineNum)
			log.Errorf("  Available types: %v\n", getAvailableTypes(builtInScalars, definedTypes))

			if lineNum > 0 {
				errorLines = append(errorLines, lineNum)
			}

			hasErrors = true
		}
	}

	for _, enumDef := range doc.EnumTypeDefinitions {
		enumName := doc.Input.ByteSliceString(enumDef.Name)

		for _, valueRef := range enumDef.EnumValuesDefinition.Refs {
			valueDef := doc.EnumValueDefinitions[valueRef]
			valueName := doc.Input.ByteSliceString(valueDef.EnumValue)

			// Check for invalid enum value names (ending with numbers, etc.)
			if !isValidEnumValue(valueName) {
				lineNum := findLineNumberByText(schemaContent, valueName)
				log.Infof("ERROR: Enum '%s' has invalid value '%s' (line %d)\n", enumName, valueName, lineNum)
				log.Infof("  Enum values should be valid GraphQL identifiers (letters, digits, underscores, no leading digits)\n")

				if lineNum > 0 {
					errorLines = append(errorLines, lineNum)
				}

				hasErrors = true
			}

			if hasSuspiciousEnumValue(valueName) || hasEmbeddedDigits(valueName) {
				lineNum := findLineNumberByText(schemaContent, valueName)

				// Check if this error is suppressed
				if s.IsSuppressed(schemaPath, lineNum, "suspicious_enum_value", valueName) {
					continue // Skip this error as it's suppressed
				}

				log.Infof("ERROR: Enum '%s' has suspicious value '%s' (line %d)\n", enumName, valueName, lineNum)

				// Suggest common fixes for standard GraphQL scalar types
				if suggestion := suggestCorrectEnumValue(valueName); suggestion != "" {
					log.Infof("  Did you mean '%s'?\n", suggestion)
				} else {
					// Generic suggestion for values ending with digits
					suggestedValue := removeSuffixDigits(valueName)
					log.Errorf("  Did you mean '%s'? Enum values typically don't contain numbers.\n", suggestedValue)
				}

				if lineNum > 0 {
					errorLines = append(errorLines, lineNum)
				}

				hasErrors = true
			}
		}
	}

	if hasErrors {
		log.Error("Data type validation FAILED - schema contains invalid type references")

		return false, errorLines
	}

	log.Info("Data type validation PASSED")

	return true, errorLines
}

func printReport(schemaFiles []string, totalErrors int) {
	log.Infof("\n=== Linting Summary ===")
	log.Infof("Total files checked: %d", len(schemaFiles))

	if totalErrors > 0 {
		log.Infof("Files with errors: %d", totalErrors)
		log.Infof("Files passed: %d", len(schemaFiles)-totalErrors)
		log.Infof("Linting completed with %d file(s) containing errors", totalErrors)
		log.Fatal("Exiting due to lint errors")
	}

	log.Infof("Files passed: %d", len(schemaFiles))
	log.Infof("All %d schema file(s) passed linting successfully!", len(schemaFiles))
}

func (s Store) LoadConfig(configPath string) (*LinterConfig, error) {
	config := &LinterConfig{
		Settings: Settings{
			StrictMode:         true,
			ValidateFederation: true,
			CheckDescriptions:  true,
		},
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Debugf("no config file found at %s. Using defaults", configPath)

		return config, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	err = yaml.Unmarshal(data, config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if s.Verbose {
		log.Infof("loaded config with %d suppressions", len(config.Suppressions))
	}

	return config, nil
}

func findGraphQLFiles(rootPath string) ([]string, error) {
	var files []string

	if err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if shouldSkip(info) {
			if info.IsDir() {
				return filepath.SkipDir
			}

			return nil
		}

		if isIgnoredDir(info) {
			return filepath.SkipDir
		}

		if isGraphQLFile(info) {
			files = append(files, path)
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf("unable to walk dir for graphql files: %w", err)
	}

	return files, nil
}

func shouldSkip(info os.FileInfo) bool {
	return strings.HasPrefix(info.Name(), ".")
}

func isIgnoredDir(info os.FileInfo) bool {
	if !info.IsDir() {
		return false
	}

	switch strings.ToLower(info.Name()) {
	case "node_modules", "vendor", ".git":
		return true
	default:
		return false
	}
}

func isGraphQLFile(info os.FileInfo) bool {
	if info.IsDir() {
		return false
	}

	ext := strings.ToLower(filepath.Ext(info.Name()))

	return ext == ".graphql" || ext == ".graphqls"
}

func printFailedLintMessage(schemaPath string, errorLines []int) {
	if len(errorLines) == 0 {
		log.Errorf("schema linting failed: %s", schemaPath)

		return
	}

	log.Errorf("schema linting failed: %s:%d", schemaPath, errorLines[0])
}

func getLineNumber(pos position.Position) int {
	return int(pos.LineStart)
}

func removeSuffixDigits(value string) string {
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

func removeAllDigits(value string) string {
	result := ""

	for _, char := range value {
		if char < '0' || char > '9' {
			result += string(char)
		}
	}

	return result
}

func suggestCorrectEnumValue(value string) string {
	// Common GraphQL scalar type corrections
	corrections := map[string]string{
		"STRING2":  "STRING",
		"BOOLEAN2": "BOOLEAN",
		"BOOLE3AN": "BOOLEAN",
		"BOOL3AN":  "BOOLEAN",
		"BOOLEAN3": "BOOLEAN",
		"FLOA2T":   "FLOAT",
		"FLO2AT":   "FLOAT",
		"FLOAT2":   "FLOAT",
		"INT2":     "INT",
		"INTEGER2": "INTEGER",
		"I2NT":     "INT",
		"INTE2GER": "INTEGER",
	}

	// Direct match
	if correction, exists := corrections[value]; exists {
		return correction
	}

	// Try removing all digits and see if it matches a standard type
	cleanValue := removeAllDigits(value)
	standardTypes := []string{"STRING", "BOOLEAN", "FLOAT", "INT", "INTEGER", "ID"}

	for _, standardType := range standardTypes {
		if levenshteinDistance(cleanValue, standardType) <= levenshteinThreshold {
			return standardType
		}
	}

	return ""
}

func hasEmbeddedDigits(value string) bool {
	for _, char := range value {
		if char >= '0' && char <= '9' {
			return true
		}
	}

	return false
}

func hasSuspiciousEnumValue(value string) bool {
	if len(value) == 0 {
		return false
	}

	lastChar := value[len(value)-1]

	return lastChar >= '0' && lastChar <= '9'
}

func levenshteinDistance(source, target string) int {
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

func isValidEnumValue(value string) bool {
	if len(value) == 0 {
		return false
	}

	ch := value[0]
	if (ch < 'A' || (ch > 'Z' && ch < 'a') || ch > 'z') && ch != '_' {
		return false
	}

	for idx := 1; idx < len(value); idx++ {
		c := value[idx]
		if (c < 'A' || c > 'Z') &&
			(c < 'a' || c > 'z') &&
			(c < '0' || c > '9') &&
			c != '_' {
			return false
		}
	}

	return true
}

func getAvailableTypes(builtInScalars, definedTypes map[string]bool) []string {
	types := make([]string, 0, len(builtInScalars)+len(definedTypes))

	for t := range builtInScalars {
		types = append(types, t)
	}

	for t := range definedTypes {
		types = append(types, t)
	}

	return types
}

func getBaseTypeName(doc *ast.Document, typeRef ast.Type) string {
	switch typeRef.TypeKind {
	case ast.TypeKindNamed:
		return doc.Input.ByteSliceString(typeRef.Name)
	case ast.TypeKindList:
		return getBaseTypeName(doc, doc.Types[typeRef.OfType])
	case ast.TypeKindNonNull:
		return getBaseTypeName(doc, doc.Types[typeRef.OfType])
	case ast.TypeKindUnknown:
		// Optionally log or handle the unknown
		return ""
	default:
		// Defensive: in case a new TypeKind is added in the future
		return ""
	}
}

func findLineNumberByText(schemaContent string, searchText string) int {
	lines := strings.Split(schemaContent, "\n")

	for i, line := range lines {
		if strings.Contains(line, searchText) {
			return i + 1
		}
	}

	return 0
}

func getLineContent(schemaContent string, lineNum int) string {
	if lineNum <= 0 {
		return ""
	}

	lines := strings.Split(schemaContent, "\n")
	if lineNum > len(lines) {
		return ""
	}

	return strings.TrimSpace(lines[lineNum-1])
}

func mapFilteredLineToOriginal(originalSchema, filteredSchema string, filteredLineNum int) int {
	originalLines := strings.Split(originalSchema, "\n")
	filteredLines := strings.Split(filteredSchema, "\n")

	if filteredLineNum <= 0 || filteredLineNum > len(filteredLines) {
		return 0
	}

	filteredLine := strings.TrimSpace(filteredLines[filteredLineNum-1])
	if filteredLine == "" {
		return 0
	}

	for i, originalLine := range originalLines {
		if strings.TrimSpace(originalLine) == filteredLine {
			return i + 1 // Line numbers are 1-based
		}
	}

	return findLineNumberByText(originalSchema, filteredLine)
}

func isEntityType(obj ast.ObjectTypeDefinition, doc *ast.Document) bool {
	for _, field := range obj.FieldsDefinition.Refs {
		fieldDef := doc.FieldDefinitions[field]
		fieldName := doc.Input.ByteSliceString(fieldDef.Name)

		if fieldName == "id" || fieldName == "legacyID" {
			return true
		}
	}

	return false
}

func (sup Suppression) Matches(filePath string, line int, rule string, value string) bool {
	normalizedSuppressionFile := strings.ReplaceAll(sup.File, "\\", "/")
	fileMatches := sup.File == "" || strings.HasSuffix(filePath, normalizedSuppressionFile)
	lineMatches := sup.Line == 0 || sup.Line == line
	ruleMatches := sup.Rule == "" || sup.Rule == rule
	valueMatches := sup.Value == "" || sup.Value == value

	return fileMatches && lineMatches && ruleMatches && valueMatches
}

func (s Store) IsSuppressed(filePath string, line int, rule string, value string) bool {
	if s.LinterConfig == nil {
		return false
	}

	normalizedFilePath := strings.ReplaceAll(filePath, "\\", "/")
	for _, suppression := range s.LinterConfig.Suppressions {
		if suppression.Matches(normalizedFilePath, line, rule, value) {
			log.Debugf("SUPPRESSED: %s at line %d in %s (reason: %s)",
				rule, line, filePath, suppression.Reason)

			return true
		}
	}

	return false
}

func findFieldDefinitionLine(schemaContent string, fieldName string, typeName string) int {
	lines := strings.Split(schemaContent, "\n")
	for index, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		if strings.Contains(trimmedLine, fieldName+":") {
			if strings.Contains(trimmedLine, typeName+"!") ||
				strings.Contains(trimmedLine, typeName+"]") ||
				strings.Contains(trimmedLine, "["+typeName) ||
				strings.HasSuffix(trimmedLine, typeName) ||
				strings.Contains(trimmedLine, typeName+" ") {
				return index + 1
			}
		}
	}

	return 0
}

func validateDirectiveNames(doc *ast.Document) bool {
	// Official federation directives as per federation spec
	validFederationDirectives := map[string]bool{
		"key":              true,
		"external":         true,
		"requires":         true,
		"provides":         true,
		"extends":          true,
		"shareable":        true,
		"inaccessible":     true,
		"override":         true,
		"composeDirective": true,
		"interfaceObject":  true,
		"tag":              true,
		"deprecated":       true, // Standard GraphQL directive
		"specifiedBy":      true, // Standard GraphQL directive
		"oneOf":            true, // Standard GraphQL directive
	}

	log.Debug("Validating federation directive names...")

	hasErrors := false

	for _, obj := range doc.ObjectTypeDefinitions {
		typeName := doc.Input.ByteSliceString(obj.Name)

		for _, directiveRef := range obj.Directives.Refs {
			directive := doc.Directives[directiveRef]
			directiveName := doc.Input.ByteSliceString(directive.Name)

			if !validFederationDirectives[directiveName] {
				log.Errorf("ERROR: Invalid federation directive '@%s' on type '%s'\n", directiveName, typeName)
				log.Errorf(`  Federation only allows these directives: @key, @external, @requires, @provides, @extends,
				  @shareable, @inaccessible, @override, @composeDirective, @interfaceObject, @tag, @deprecated, @specifiedBy,
				  @oneOf`)

				if strings.Contains(directiveName, "key") || levenshteinDistance(directiveName, "key") <= levenshteinThreshold {
					log.Errorf("  Did you mean '@key'?\n")
				} else if strings.Contains(directiveName, "external") ||
					levenshteinDistance(directiveName, "external") <= levenshteinThreshold {
					log.Errorf("  Did you mean '@external'?\n")
				}

				hasErrors = true
			}
		}
	}

	for _, fieldDef := range doc.FieldDefinitions {
		for _, directiveRef := range fieldDef.Directives.Refs {
			directive := doc.Directives[directiveRef]
			directiveName := doc.Input.ByteSliceString(directive.Name)

			if !validFederationDirectives[directiveName] {
				fieldName := doc.Input.ByteSliceString(fieldDef.Name)
				log.Errorf("ERROR: Invalid federation directive '@%s' on field '%s'\n", directiveName, fieldName)
				log.Errorf(`  Federation only allows these directives on fields: @external, @requires, @provides, @shareable,
				  @inaccessible, @override, @tag, @deprecated`)

				hasErrors = true
			}
		}
	}

	if hasErrors {
		log.Error("Federation directive validation FAILED - schema contains invalid directives")

		return false
	}

	log.Debug("federation directive validation PASSED")

	return true
}

func validateFederationSchema(doc *ast.Document) {
	log.Info("Validating federation schema...")

	federationDirectives := []string{"key", "external", "requires", "provides", "extends"}

	for _, obj := range doc.ObjectTypeDefinitions {
		typeName := doc.Input.ByteSliceString(obj.Name)
		validateObjectDirectives(obj, doc, typeName, federationDirectives)
		checkEntityKey(obj, doc, typeName)
	}

	log.Info("Federation validation completed")
}

func validateObjectDirectives(
	obj ast.ObjectTypeDefinition,
	doc *ast.Document,
	typeName string,
	federationDirectives []string,
) {
	for _, directiveRef := range obj.Directives.Refs {
		directive := doc.Directives[directiveRef]
		directiveName := doc.Input.ByteSliceString(directive.Name)

		if directiveName == "key" {
			validateKeyDirective(directive, doc, typeName)
		}

		reportFederationDirectiveUsage(directiveName, typeName, federationDirectives)
	}
}

func validateKeyDirective(
	directive ast.Directive,
	doc *ast.Document,
	typeName string,
) {
	hasFieldsArg := false

	for _, argRef := range directive.Arguments.Refs {
		arg := doc.Arguments[argRef]
		argName := doc.Input.ByteSliceString(arg.Name)

		if argName == "fields" {
			hasFieldsArg = true

			break
		}
	}

	if !hasFieldsArg {
		log.Errorf("Error: @key directive on type '%s' is missing 'fields' argument", typeName)
	}
}

func reportFederationDirectiveUsage(
	directiveName, typeName string,
	federationDirectives []string,
) {
	for _, fedDirective := range federationDirectives {
		if directiveName == fedDirective {
			log.Infof("Info: Type '%s' uses federation directive @%s", typeName, fedDirective)
		}
	}
}

func checkEntityKey(obj ast.ObjectTypeDefinition, doc *ast.Document, typeName string) {
	if typeName == "Query" || typeName == "Mutation" || typeName == "Subscription" {
		return
	}

	if !hasKeyDirective(obj, doc) && isEntityType(obj, doc) {
		log.Warnf(`Warning: Type '%s' has ID fields but no @key directive - consider adding @key if this is a federated
		  entity`, typeName)
	}
}

func hasKeyDirective(obj ast.ObjectTypeDefinition, doc *ast.Document) bool {
	for _, directiveRef := range obj.Directives.Refs {
		directive := doc.Directives[directiveRef]

		directiveName := doc.Input.ByteSliceString(directive.Name)

		if directiveName == "key" {
			return true
		}
	}

	return false
}
