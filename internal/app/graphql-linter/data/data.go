package data

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/schubergphilis/mcvs-golang-project-root/pkg/projectroot"
	log "github.com/sirupsen/logrus"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/ast"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/astparser"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/federation"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/operationreport"
	"gopkg.in/yaml.v3"
)

const (
	levenshteinThreshold = 2
	linesAfterContext    = 3
	linesBeforeContext   = 2
	defaultErrorCapacity = 32
)

type Storer interface {
	FindAndLogGraphQLSchemaFiles() ([]string, error)
	LintSchemaFiles(schemaFiles []string) int
	LoadConfig(configPath string) (*LinterConfig, error)
	PrintReport(schemaFiles []string, totalErrors int)
}

type Store struct {
	LinterConfig *LinterConfig
	TargetPath   string
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

func NewStore(targetPath string, verbose bool) (Store, error) {
	s := Store{
		TargetPath: targetPath,
		Verbose:    verbose,
	}

	return s, nil
}

func (s Store) FindAndLogGraphQLSchemaFiles() ([]string, error) {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to determine project root: %w", err)
	}

	if s.TargetPath == "" {
		s.TargetPath = projectRoot
	}

	schemaFiles, err := findGraphQLFiles(s.TargetPath)
	if err != nil {
		return nil, fmt.Errorf("unable to find graphql files: %w", err)
	}

	if len(schemaFiles) == 0 {
		return nil, fmt.Errorf("no GraphQL schema files found in directory: %s", s.TargetPath)
	}

	if s.Verbose {
		log.Infof("found %d GraphQL schema files:", len(schemaFiles))

		for _, file := range schemaFiles {
			log.Infof("  - %s", file)
		}
	}

	return schemaFiles, nil
}

func (s Store) LintSchemaFiles(schemaFiles []string) int {
	totalErrors := 0

	for _, schemaFile := range schemaFiles {
		if s.Verbose {
			log.Infof("=== Linting %s ===", schemaFile)
		}

		if !s.lintSchemaFile(schemaFile) {
			totalErrors++
		}
	}

	return totalErrors
}

func (s Store) LoadConfig() (*LinterConfig, error) {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to determine project root: %w", err)
	}

	configPath := filepath.Join(projectRoot, ".graphql-linter.yml")

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

func LogSchemaParseErrors(
	schemaString string,
	parseReport *operationreport.Report,
) {
	if !parseReport.HasErrors() {
		return
	}

	log.Errorf("Failed to parse schema - found %d errors:\n",
		len(parseReport.InternalErrors)+len(parseReport.ExternalErrors))

	reportInternalErrors(parseReport)
	reportExternalErrors(schemaString, parseReport, linesBeforeContext, linesAfterContext)
}

func reportInternalErrors(parseReport *operationreport.Report) {
	for i, internalErr := range parseReport.InternalErrors {
		log.Errorf("Internal Error %d: %v\n", i+1, internalErr)
	}
}

func reportExternalErrors(
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

func (s Store) PrintReport(schemaFiles []string, totalErrors int) {
	log.WithFields(log.Fields{
		"totalFiles":  len(schemaFiles),
		"passedFiles": len(schemaFiles) - totalErrors,
	}).Info("linting summary")

	if totalErrors > 0 {
		log.WithFields(log.Fields{
			"totalErrors": totalErrors,
		}).Fatal("linting failed with errors")

		return
	}

	log.Infof("All %d schema file(s) passed linting successfully!", len(schemaFiles))
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
	if len(value) == 0 {
		return ""
	}

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

	if correction, exists := corrections[value]; exists {
		return correction
	}

	cleanValue := removeAllDigits(value)
	standardTypes := []string{"STRING", "BOOLEAN", "FLOAT", "INT", "INTEGER", "ID"}

	for _, standardType := range standardTypes {
		if cleanValue == standardType {
			return standardType
		}
	}

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

func isAlphaUnderOrDigit(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

func isValidEnumValue(value string) bool {
	if len(value) == 0 {
		return false
	}

	r := rune(value[0])
	if !unicode.IsLetter(r) && r != '_' {
		return false
	}

	for _, r := range value[1:] {
		if !isAlphaUnderOrDigit(r) {
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

func (sup Suppression) Matches(filePath string, line int, rule string, value string) bool {
	normalizedSuppressionFile := strings.ReplaceAll(sup.File, "\\", "/")
	fileMatches := sup.File == "" || strings.HasSuffix(filePath, normalizedSuppressionFile)
	lineMatches := sup.Line == 0 || sup.Line == line
	ruleMatches := sup.Rule == "" || sup.Rule == rule
	valueMatches := sup.Value == "" || sup.Value == value

	return fileMatches && lineMatches && ruleMatches && valueMatches
}

func (s Store) isSuppressed(filePath string, line int, rule string, value string) bool {
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
		if !validateDirectives(
			doc,
			obj.Directives.Refs,
			validFederationDirectives,
			typeName,
			"type",
		) {
			hasErrors = true
		}
	}

	for _, fieldDef := range doc.FieldDefinitions {
		fieldName := doc.Input.ByteSliceString(fieldDef.Name)
		if !validateDirectives(
			doc,
			fieldDef.Directives.Refs,
			validFederationDirectives,
			fieldName,
			"field",
		) {
			hasErrors = true
		}
	}

	if hasErrors {
		log.Error("Federation directive validation FAILED - schema contains invalid directives")

		return false
	}

	log.Debug("federation directive validation PASSED")

	return true
}

func validateDirectives(
	doc *ast.Document,
	directiveRefs []int,
	validDirectives map[string]bool,
	parentName, parentKind string,
) bool {
	hasErrors := false

	for _, directiveRef := range directiveRefs {
		directive := doc.Directives[directiveRef]

		directiveName := doc.Input.ByteSliceString(directive.Name)
		if !validDirectives[directiveName] {
			reportDirectiveError(directiveName, parentName, parentKind)

			hasErrors = true
		}
	}

	return !hasErrors
}

func reportDirectiveError(directiveName, parentName, parentKind string) {
	log.Errorf(
		"ERROR: Invalid federation directive '@%s' on %s '%s'",
		directiveName,
		parentKind,
		parentName,
	)

	switch parentKind {
	case "type":
		log.Errorf(
			`  Federation only allows these directives: @key, @external, @requires, @provides, @extends,
          @shareable, @inaccessible, @override, @composeDirective, @interfaceObject, @tag, @deprecated, @specifiedBy,
          @oneOf`,
		)
		suggestDirective(directiveName, "key")
		suggestDirective(directiveName, "external")
	case "field":
		log.Errorf(
			`  Federation only allows these directives on fields: @external, @requires, @provides, @shareable,
          @inaccessible, @override, @tag, @deprecated`,
		)
	}
}

func suggestDirective(directiveName, validName string) {
	if strings.Contains(directiveName, validName) ||
		levenshteinDistance(directiveName, validName) <= levenshteinThreshold {
		log.Errorf("  Did you mean '@%s'?", validName)
	}
}

func lintSchemaValidation(
	store Store,
	doc *ast.Document,
	schemaString, filteredSchema, schemaPath string,
) (bool, []int) {
	var (
		hasValidationErrors bool
		errorLines          []int
	)

	if !validateDirectiveNames(doc) {
		hasValidationErrors = true
	}

	dataTypesValid, errorLinesDT := store.validateDataTypes(doc, schemaString, schemaPath)
	if !dataTypesValid {
		hasValidationErrors = true
	}

	if !validateFederationSchema(filteredSchema) {
		hasValidationErrors = true
	}

	descriptionErrors, errorLinesDesc, hasDeprecationReasonError := lintDescriptions(
		doc,
		schemaString,
	)
	if len(descriptionErrors) > 0 {
		printDescriptionErrors(descriptionErrors, schemaPath)

		hasValidationErrors = true

		errorLines = append(errorLines, errorLinesDesc...)
	}

	if hasDeprecationReasonError {
		hasValidationErrors = true
	}

	errorLines = append(errorLines, errorLinesDT...)

	return hasValidationErrors, errorLines
}

func (s Store) lintSchemaFile(schemaPath string) bool {
	schemaString, ok := readSchemaFile(schemaPath)
	if !ok {
		return false
	}

	filteredSchema := filterSchemaComments(schemaString)

	log.Debugf("Parsing federation schema from: %s\n", schemaPath)
	log.Debugf("Schema content length: %d bytes\n", len(schemaString))

	doc, parseReport := astparser.ParseGraphqlDocumentString(schemaString)
	LogSchemaParseErrors(schemaString, &parseReport)
	log.Debug("Schema parsed successfully!")

	hasValidationErrors, errorLines := lintSchemaValidation(
		s,
		&doc,
		schemaString,
		filteredSchema,
		schemaPath,
	)

	if hasValidationErrors {
		if len(errorLines) > 0 {
			log.WithFields(log.Fields{
				"numberOfErrors":                len(errorLines),
				"schemaPathIncludingLineNumber": schemaPath + fmt.Sprintf(":%d", errorLines[0]),
			}).Error("schema linting FAILED with errors")
		} else {
			log.WithFields(log.Fields{
				"schemaPath": schemaPath,
			}).Error("schema linting FAILED with no specific errors reported")
		}

		return false
	}

	log.Debugf("Schema linting PASSED: %s", schemaPath)

	return true
}

func readSchemaFile(schemaPath string) (string, bool) {
	schemaBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		log.WithError(err).Error("failed to read schema file")

		return "", false
	}

	return string(schemaBytes), true
}

func filterSchemaComments(schemaString string) string {
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

func validateFederationSchema(filteredSchema string) bool {
	var report operationreport.Report

	federationSchema, federationErr := federation.BuildFederationSchema(
		filteredSchema,
		filteredSchema,
	)
	if federationErr != nil {
		log.Infof("Federation schema build failed: %v\n", federationErr)

		return false
	}

	_ = federationSchema

	if report.HasErrors() {
		log.Error("Federation validation errors:")

		for _, internalErr := range report.InternalErrors {
			log.Errorf("  - %v\n", internalErr)
		}

		for _, externalErr := range report.ExternalErrors {
			log.Errorf("  - %s\n", externalErr.Message)
		}

		return false
	}

	log.Debug("Federation schema validation passed")

	return true
}

func collectDefinedTypes(doc *ast.Document) map[string]bool {
	definedTypes := make(map[string]bool)

	for _, obj := range doc.ObjectTypeDefinitions {
		typeName := doc.Input.ByteSliceString(obj.Name)
		definedTypes[typeName] = true
	}

	for _, enum := range doc.EnumTypeDefinitions {
		typeName := doc.Input.ByteSliceString(enum.Name)
		definedTypes[typeName] = true
	}

	for _, input := range doc.InputObjectTypeDefinitions {
		typeName := doc.Input.ByteSliceString(input.Name)
		definedTypes[typeName] = true
	}

	for _, iface := range doc.InterfaceTypeDefinitions {
		typeName := doc.Input.ByteSliceString(iface.Name)
		definedTypes[typeName] = true
	}

	for _, union := range doc.UnionTypeDefinitions {
		typeName := doc.Input.ByteSliceString(union.Name)
		definedTypes[typeName] = true
	}

	for _, scalar := range doc.ScalarTypeDefinitions {
		typeName := doc.Input.ByteSliceString(scalar.Name)
		definedTypes[typeName] = true
	}

	return definedTypes
}

func validateTypeReferences(
	doc *ast.Document,
	schemaContent string,
	builtInScalars, definedTypes map[string]bool,
	typeRefs []int,
	getName func(int) string,
	getType func(int) ast.Type,
	errorPrefix string,
) ([]string, []int) {
	var (
		errors     []string
		errorLines []int
	)

	for _, ref := range typeRefs {
		fieldName := getName(ref)
		fieldType := getType(ref)

		baseType := getBaseTypeName(doc, fieldType)
		if !builtInScalars[baseType] && !definedTypes[baseType] {
			lineNum := findFieldDefinitionLine(schemaContent, fieldName, baseType)
			if lineNum == 0 {
				lineNum = findLineNumberByText(schemaContent, fieldName+": "+baseType)
			}

			if lineNum == 0 {
				lineNum = findLineNumberByText(schemaContent, fieldName+":")
			}

			log.Errorf(
				"%s '%s' references undefined type '%s' (line %d)\n",
				errorPrefix,
				fieldName,
				baseType,
				lineNum,
			)
			log.Errorf("  Available types: %v\n", getAvailableTypes(builtInScalars, definedTypes))

			if lineNum > 0 {
				errorLines = append(errorLines, lineNum)
			}

			errors = append(errors, fieldName)
		}
	}

	return errors, errorLines
}

func indexSlice(n int) []int {
	indices := make([]int, n)
	for i := range indices {
		indices[i] = i
	}

	return indices
}

func validateFieldTypes(
	doc *ast.Document,
	schemaContent string,
	builtInScalars, definedTypes map[string]bool,
) ([]string, []int) {
	return validateTypeReferences(
		doc,
		schemaContent,
		builtInScalars,
		definedTypes,
		indexSlice(len(doc.FieldDefinitions)),
		func(i int) string { return doc.Input.ByteSliceString(doc.FieldDefinitions[i].Name) },
		func(i int) ast.Type { return doc.Types[doc.FieldDefinitions[i].Type] },
		"ERROR: Field",
	)
}

func validateInputFieldTypes(
	doc *ast.Document,
	schemaContent string,
	builtInScalars, definedTypes map[string]bool,
) ([]string, []int) {
	return validateTypeReferences(
		doc,
		schemaContent,
		builtInScalars,
		definedTypes,
		indexSlice(len(doc.InputValueDefinitions)),
		func(i int) string { return doc.Input.ByteSliceString(doc.InputValueDefinitions[i].Name) },
		func(i int) ast.Type { return doc.Types[doc.InputValueDefinitions[i].Type] },
		"ERROR: Input field",
	)
}

func checkInvalidEnumValue(enumName, valueName, schemaContent string) (string, int) {
	if isValidEnumValue(valueName) {
		return "", 0
	}

	lineNum := findLineNumberByText(schemaContent, valueName)
	log.Infof(
		"ERROR: Enum '%s' has invalid value '%s' (line %d)\n",
		enumName,
		valueName,
		lineNum,
	)
	log.Infof(
		"  Enum values should be valid GraphQL identifiers (letters, digits, underscores, no leading digits)\n",
	)

	return valueName, lineNum
}

func (s Store) checkSuspiciousEnumValue(
	enumName,
	valueName,
	schemaContent,
	schemaPath string,
) (string, int) {
	if !hasSuspiciousEnumValue(valueName) && !hasEmbeddedDigits(valueName) {
		return "", 0
	}

	lineNum := findLineNumberByText(schemaContent, valueName)
	if s.isSuppressed(schemaPath, lineNum, "suspicious_enum_value", valueName) {
		return "", 0
	}

	log.Errorf(
		"ERROR: Enum '%s' has suspicious value '%s' (line %d)\n",
		enumName,
		valueName,
		lineNum,
	)

	if suggestion := suggestCorrectEnumValue(valueName); suggestion != "" {
		log.Errorf("  Did you mean '%s'?\n", suggestion)
	} else {
		suggestedValue := removeSuffixDigits(valueName)
		log.Errorf("  Did you mean '%s'? Enum values typically don't contain numbers.\n", suggestedValue)
	}

	return valueName, lineNum
}

func (s Store) validateEnumTypes(
	doc *ast.Document,
	schemaContent string,
	schemaPath string,
) ([]string, []int) {
	var (
		errors     []string
		errorLines []int
	)

	for _, enumDef := range doc.EnumTypeDefinitions {
		enumName := doc.Input.ByteSliceString(enumDef.Name)

		for _, valueRef := range enumDef.EnumValuesDefinition.Refs {
			valueDef := doc.EnumValueDefinitions[valueRef]
			valueName := doc.Input.ByteSliceString(valueDef.EnumValue)

			if err, line := checkInvalidEnumValue(enumName, valueName, schemaContent); err != "" {
				errors = append(errors, valueName)

				if line > 0 {
					errorLines = append(errorLines, line)
				}
			}

			if err, line := s.checkSuspiciousEnumValue(enumName, valueName, schemaContent, schemaPath); err != "" {
				errors = append(errors, valueName)

				if line > 0 {
					errorLines = append(errorLines, line)
				}
			}
		}
	}

	return errors, errorLines
}

func (s Store) validateDataTypes(
	doc *ast.Document,
	schemaContent string,
	schemaPath string,
) (bool, []int) {
	builtInScalars := map[string]bool{
		"String":  true,
		"Int":     true,
		"Float":   true,
		"Boolean": true,
		"ID":      true,
	}
	definedTypes := collectDefinedTypes(doc)
	hasErrors := false

	var errorLines []int

	fieldErrors, fieldErrorLines := validateFieldTypes(
		doc,
		schemaContent,
		builtInScalars,
		definedTypes,
	)
	if len(fieldErrors) > 0 {
		hasErrors = true

		errorLines = append(errorLines, fieldErrorLines...)
	}

	inputErrors, inputErrorLines := validateInputFieldTypes(
		doc,
		schemaContent,
		builtInScalars,
		definedTypes,
	)
	if len(inputErrors) > 0 {
		hasErrors = true

		errorLines = append(errorLines, inputErrorLines...)
	}

	enumErrors, enumErrorLines := s.validateEnumTypes(doc, schemaContent, schemaPath)
	if len(enumErrors) > 0 {
		hasErrors = true

		errorLines = append(errorLines, enumErrorLines...)
	}

	if hasErrors {
		log.Error("Data type validation FAILED - schema contains invalid type references")

		return false, errorLines
	}

	log.Debug("Data type validation PASSED")

	return true, errorLines
}

func findUnusedTypes(doc *ast.Document, schemaString string) []DescriptionError {
	var unusedTypeErrors []DescriptionError

	for _, obj := range doc.ObjectTypeDefinitions {
		name := doc.Input.ByteSliceString(obj.Name)
		if name == "Query" || name == "Mutation" || name == "Subscription" {
			continue
		}

		isUsed := false

		for _, fieldDef := range doc.FieldDefinitions {
			baseType := getBaseTypeName(doc, doc.Types[fieldDef.Type])
			if baseType == name {
				isUsed = true

				break
			}
		}

		if !isUsed {
			lineNum := findLineNumberByText(schemaString, "type "+name)
			lineContent := getLineContent(schemaString, lineNum)
			message := fmt.Sprintf("ERROR: Type '%s' is defined but not used", name)
			unusedTypeErrors = append(unusedTypeErrors, DescriptionError{
				LineNum:     lineNum,
				Message:     message,
				LineContent: lineContent,
			})
		}
	}

	return unusedTypeErrors
}

func checkTypeDescription(
	doc *ast.Document,
	schemaString string,
	obj ast.ObjectTypeDefinition,
) (*DescriptionError, int) {
	if obj.Description.IsDefined {
		return nil, 0
	}

	name := doc.Input.ByteSliceString(obj.Name)
	lineNum := findLineNumberByText(schemaString, "type "+name)
	lineContent := getLineContent(schemaString, lineNum)
	message := fmt.Sprintf("ERROR: Object type '%s' is missing a description", name)

	return &DescriptionError{
		LineNum:     lineNum,
		Message:     message,
		LineContent: lineContent,
	}, lineNum
}

func checkFieldDescription(
	doc *ast.Document,
	schemaString string,
	obj ast.ObjectTypeDefinition,
	fieldRef int,
) (*DescriptionError, int) {
	fieldDef := doc.FieldDefinitions[fieldRef]
	if fieldDef.Description.IsDefined {
		return nil, 0
	}

	fieldName := doc.Input.ByteSliceString(fieldDef.Name)
	typeName := doc.Input.ByteSliceString(obj.Name)
	fieldType := doc.Types[fieldDef.Type]
	baseType := getBaseTypeName(doc, fieldType)

	lineNum := findFieldDefinitionLine(schemaString, fieldName, baseType)
	if lineNum == 0 {
		lineNum = findLineNumberByText(schemaString, fieldName+":")
	}

	lineContent := getLineContent(schemaString, lineNum)
	message := fmt.Sprintf(
		"ERROR: Field '%s' in type '%s' is missing a description",
		fieldName,
		typeName,
	)

	return &DescriptionError{
		LineNum:     lineNum,
		Message:     message,
		LineContent: lineContent,
	}, lineNum
}

func checkEnumDescription(
	doc *ast.Document,
	schemaString string,
	enum ast.EnumTypeDefinition,
) (*DescriptionError, int) {
	if enum.Description.IsDefined {
		return nil, 0
	}

	name := doc.Input.ByteSliceString(enum.Name)
	lineNum := findLineNumberByText(schemaString, "enum "+name)
	lineContent := getLineContent(schemaString, lineNum)
	message := fmt.Sprintf("ERROR: Enum '%s' is missing a description", name)

	return &DescriptionError{
		LineNum:     lineNum,
		Message:     message,
		LineContent: lineContent,
	}, lineNum
}

func sortDescriptionErrors(errors []DescriptionError) []DescriptionError {
	for i := range len(errors) - 1 {
		for j := range len(errors) - i - 1 {
			if errors[j].LineNum > errors[j+1].LineNum {
				errors[j], errors[j+1] = errors[j+1], errors[j]
			}
		}
	}

	return errors
}

func printDescriptionErrors(errors []DescriptionError, schemaPath string) {
	for _, err := range errors {
		log.Errorf("%s:%d: %s\n  %s", schemaPath, err.LineNum, err.Message, err.LineContent)
	}
}

func findMissingTypeDescriptions(doc *ast.Document, schemaString string) []DescriptionError {
	var errors []DescriptionError

	for _, obj := range doc.ObjectTypeDefinitions {
		if err, _ := checkTypeDescription(doc, schemaString, obj); err != nil {
			errors = append(errors, *err)
		}
	}

	return errors
}

func findMissingFieldDescriptions(doc *ast.Document, schemaString string) []DescriptionError {
	var errors []DescriptionError

	for _, obj := range doc.ObjectTypeDefinitions {
		for _, fieldRef := range obj.FieldsDefinition.Refs {
			if err, _ := checkFieldDescription(doc, schemaString, obj, fieldRef); err != nil {
				errors = append(errors, *err)
			}
		}
	}

	return errors
}

func findMissingArgumentDescriptions(doc *ast.Document, schemaString string) []DescriptionError {
	var errors []DescriptionError

	for _, obj := range doc.ObjectTypeDefinitions {
		for _, fieldRef := range obj.FieldsDefinition.Refs {
			fieldDef := doc.FieldDefinitions[fieldRef]
			for _, argRef := range fieldDef.ArgumentsDefinition.Refs {
				argDef := doc.InputValueDefinitions[argRef]
				if !argDef.Description.IsDefined {
					argName := doc.Input.ByteSliceString(argDef.Name)
					fieldName := doc.Input.ByteSliceString(fieldDef.Name)
					lineNum := findLineNumberByText(schemaString, argName+":")
					lineContent := getLineContent(schemaString, lineNum)
					message := fmt.Sprintf(
						"ERROR: Argument '%s' of field '%s' is missing a description",
						argName,
						fieldName,
					)
					errors = append(errors, DescriptionError{
						LineNum:     lineNum,
						Message:     message,
						LineContent: lineContent,
					})
				}
			}
		}
	}

	return errors
}

func findMissingEnumDescriptions(doc *ast.Document, schemaString string) []DescriptionError {
	var errors []DescriptionError

	for _, enum := range doc.EnumTypeDefinitions {
		if err, _ := checkEnumDescription(doc, schemaString, enum); err != nil {
			errors = append(errors, *err)
		}
	}

	return errors
}

func checkEnumValueDeprecationReason(
	doc *ast.Document,
	enumName, valueName string,
	valueDef ast.EnumValueDefinition,
	schemaString string,
) *DescriptionError {
	for _, dirRef := range valueDef.Directives.Refs {
		dir := doc.Directives[dirRef]

		dirName := doc.Input.ByteSliceString(dir.Name)
		if dirName == "deprecated" {
			deprecationReason := ""

			for _, argRef := range dir.Arguments.Refs {
				arg := doc.InputValueDefinitions[argRef]

				argName := doc.Input.ByteSliceString(arg.Name)
				if argName == "reason" {
					if arg.DefaultValue.IsDefined {
						deprecationReason = fmt.Sprintf("%v", arg.DefaultValue.Value)
					}
				}
			}

			if deprecationReason == "" {
				lineNum := findLineNumberByText(schemaString, valueName)
				lineContent := getLineContent(schemaString, lineNum)
				message := fmt.Sprintf(
					"deprecations-have-a-reason: Enum value '%s.%s' is deprecated but has no deprecation reason.",
					enumName,
					valueName,
				)

				return &DescriptionError{
					LineNum:     lineNum,
					Message:     message,
					LineContent: lineContent,
				}
			}
		}
	}

	return nil
}

func findMissingDeprecationReasons(doc *ast.Document, schemaString string) []DescriptionError {
	var errors []DescriptionError

	for _, enum := range doc.EnumTypeDefinitions {
		enumName := doc.Input.ByteSliceString(enum.Name)

		for _, valueRef := range enum.EnumValuesDefinition.Refs {
			valueDef := doc.EnumValueDefinitions[valueRef]

			valueName := doc.Input.ByteSliceString(valueDef.EnumValue)

			if err := checkEnumValueDeprecationReason(doc, enumName, valueName, valueDef, schemaString); err != nil {
				errors = append(errors, *err)
			}
		}
	}

	return errors
}

func findMissingQueryRootType(doc *ast.Document, schemaString string) []DescriptionError {
	for _, obj := range doc.ObjectTypeDefinitions {
		if doc.Input.ByteSliceString(obj.Name) == "Query" {
			return nil
		}
	}

	lineNum := 1
	lineContent := getLineContent(schemaString, lineNum)
	message := "invalid-graphql-schema: Query root type must be provided."

	return []DescriptionError{{
		LineNum:     lineNum,
		Message:     message,
		LineContent: lineContent,
	}}
}

func lintDescriptions(doc *ast.Document, schemaString string) ([]DescriptionError, []int, bool) {
	descriptionErrors := make([]DescriptionError, 0, defaultErrorCapacity)
	errorLines := make([]int, 0, defaultErrorCapacity)
	hasDeprecationReasonError := false

	helpers := []func(*ast.Document, string) []DescriptionError{
		findMissingQueryRootType,
		findUnusedTypes,
		findMissingTypeDescriptions,
		findMissingFieldDescriptions,
		findMissingArgumentDescriptions,
		findMissingEnumDescriptions,
		findMissingDeprecationReasons,
	}

	for _, helper := range helpers {
		errList := helper(doc, schemaString)

		for _, err := range errList {
			descriptionErrors = append(descriptionErrors, err)

			if err.LineNum > 0 {
				errorLines = append(errorLines, err.LineNum)
			}

			if strings.Contains(err.Message, "deprecations-have-a-reason") {
				hasDeprecationReasonError = true
			}
		}
	}

	return sortDescriptionErrors(descriptionErrors), errorLines, hasDeprecationReasonError
}
