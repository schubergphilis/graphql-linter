package data

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/data/federation"
	"github.com/schubergphilis/mcvs-golang-project-root/pkg/projectroot"
	log "github.com/sirupsen/logrus"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/ast"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/astparser"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/operationreport"
	"gopkg.in/yaml.v3"
)

const (
	levenshteinThreshold      = 2
	linesAfterContext         = 3
	linesBeforeContext        = 2
	defaultErrorCapacity      = 32
	minFieldsForSortCheck     = 2
	percentMultiplier         = 100
	descriptionErrorCapacity  = 8
	minEnumValuesForSortCheck = 2

	RootQueryType        = "Query"
	RootMutationType     = "Mutation"
	RootSubscriptionType = "Subscription"

	splitNParts = 2
)

type Storer interface {
	FindAndLogGraphQLSchemaFiles() ([]string, error)
	LintSchemaFiles(schemaFiles []string) (int, int, []DescriptionError)
	LoadConfig() (*LinterConfig, error)
	PrintReport(
		schemaFiles []string,
		totalErrors int,
		passedFiles int,
		allErrors []DescriptionError,
	)
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
	FilePath    string
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

func (s Store) PrintReport(
	schemaFiles []string,
	totalErrors int,
	passedFiles int,
	allErrors []DescriptionError,
) {
	percentPassed := 0.0
	if len(schemaFiles) > 0 {
		percentPassed = float64(passedFiles) / float64(len(schemaFiles)) * percentMultiplier
	}

	percentageFilesWithAtLeastOneError := 0.0
	if len(schemaFiles) > 0 {
		percentageFilesWithAtLeastOneError = float64(len(schemaFiles)-passedFiles) /
			float64(len(schemaFiles)) * percentMultiplier
	}

	printDetailedErrors(allErrors)
	printErrorTypeSummary(allErrors)

	log.WithFields(log.Fields{
		"passedFiles":   passedFiles,
		"totalFiles":    len(schemaFiles),
		"percentPassed": fmt.Sprintf("%.2f%%", percentPassed),
	}).Info("linting summary")

	if totalErrors > 0 {
		log.WithFields(log.Fields{
			"filesWithAtLeastOneError": len(schemaFiles) - passedFiles,
			"percentage":               fmt.Sprintf("%.2f%%", percentageFilesWithAtLeastOneError),
		}).Error("files with at least one error")

		log.Fatalf("totalErrors: %d", totalErrors)

		return
	}

	log.Infof("All %d schema file(s) passed linting successfully!", len(schemaFiles))
}

func (s Store) LintSchemaFiles(schemaFiles []string) (int, int, []DescriptionError) {
	totalErrors := 0
	errorFilesCount := 0

	var allErrors []DescriptionError

	for _, schemaFile := range schemaFiles {
		errCount, fileErrCount, fileErrors := s.lintSingleSchemaFile(schemaFile)
		totalErrors += errCount
		errorFilesCount += fileErrCount

		allErrors = append(allErrors, fileErrors...)
	}

	return totalErrors, errorFilesCount, allErrors
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

	_, err = os.Stat(configPath)
	if os.IsNotExist(err) {
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

func printDetailedErrors(errors []DescriptionError) {
	if len(errors) == 0 {
		return
	}

	for _, err := range errors {
		log.Errorf("%s:%d: %s\n  %s", err.FilePath, err.LineNum, err.Message, err.LineContent)
	}
}

func printErrorTypeSummary(errors []DescriptionError) {
	errorTypeCounts := make(map[string]int)

	for _, err := range errors {
		msg := err.Message

		typeKey := msg
		if idx := strings.Index(msg, ":"); idx != -1 {
			typeKey = msg[:idx]
		} else if idx := strings.Index(msg, " "); idx != -1 {
			typeKey = msg[:idx]
		}

		errorTypeCounts[typeKey]++
	}

	if len(errorTypeCounts) == 0 {
		return
	}

	log.Error("Error type summary:")

	keys := make([]string, 0, len(errorTypeCounts))
	for k := range errorTypeCounts {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, k := range keys {
		log.Errorf("  %s: %d", k, errorTypeCounts[k])
	}
}

func findGraphQLFiles(rootPath string) ([]string, error) {
	var files []string

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
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
	})
	if err != nil {
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
	normalizedFilePath := strings.ReplaceAll(filePath, "\\", "/")

	fileMatches := sup.File == "" ||
		strings.HasSuffix(normalizedFilePath, normalizedSuppressionFile)
	lineMatches := sup.Line == 0 || sup.Line == line
	ruleMatches := sup.Rule == "" || sup.Rule == rule

	valueMatches := true
	if sup.Value != "" {
		valueMatches = sup.Value == value
	}

	return fileMatches && lineMatches && ruleMatches && valueMatches
}

func (s Store) isSuppressed(filePath string, line int, rule string, value string) bool {
	if s.LinterConfig == nil || len(s.LinterConfig.Suppressions) == 0 {
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
		"invalid-federation-directive: Invalid federation directive '@%s' on %s '%s'",
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
		"invalid-field-types: Field",
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
		"invalid-input-field-types: Input field",
	)
}

func checkInvalidEnumValue(enumName, valueName, schemaContent string) (string, int) {
	if isValidEnumValue(valueName) {
		return "", 0
	}

	lineNum := findLineNumberByText(schemaContent, valueName)
	log.Infof(
		"invalid-enum-value: Enum '%s' has invalid value '%s' (line %d)\n",
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
		"suspicious-enum-value: Enum '%s' has suspicious value '%s' (line %d)\n",
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

func collectDefinedTypeNames(doc *ast.Document) map[string]bool {
	definedTypes := make(map[string]bool)

	for _, obj := range doc.ObjectTypeDefinitions {
		name := doc.Input.ByteSliceString(obj.Name)
		if name != RootQueryType && name != RootMutationType && name != RootSubscriptionType {
			definedTypes[name] = false // false means unused
		}
	}

	for _, input := range doc.InputObjectTypeDefinitions {
		name := doc.Input.ByteSliceString(input.Name)
		definedTypes[name] = false
	}

	for _, enum := range doc.EnumTypeDefinitions {
		name := doc.Input.ByteSliceString(enum.Name)
		definedTypes[name] = false
	}

	for _, iface := range doc.InterfaceTypeDefinitions {
		name := doc.Input.ByteSliceString(iface.Name)
		definedTypes[name] = false
	}

	for _, union := range doc.UnionTypeDefinitions {
		name := doc.Input.ByteSliceString(union.Name)
		definedTypes[name] = false
	}

	for _, scalar := range doc.ScalarTypeDefinitions {
		name := doc.Input.ByteSliceString(scalar.Name)
		definedTypes[name] = false
	}

	return definedTypes
}

func markUsedTypes(doc *ast.Document, definedTypes map[string]bool) {
	// Check field types
	for _, fieldDef := range doc.FieldDefinitions {
		baseType := getBaseTypeName(doc, doc.Types[fieldDef.Type])
		if _, exists := definedTypes[baseType]; exists {
			definedTypes[baseType] = true
		}
	}

	// Check input value types (arguments and input object fields)
	for _, inputValue := range doc.InputValueDefinitions {
		baseType := getBaseTypeName(doc, doc.Types[inputValue.Type])
		if _, exists := definedTypes[baseType]; exists {
			definedTypes[baseType] = true
		}
	}

	// Check union member types
	for _, union := range doc.UnionTypeDefinitions {
		for _, memberRef := range union.UnionMemberTypes.Refs {
			memberType := doc.Types[memberRef]

			baseType := getBaseTypeName(doc, memberType)
			if _, exists := definedTypes[baseType]; exists {
				definedTypes[baseType] = true
			}
		}
	}
}

func findTypeLineNumber(typeName, schemaString string) int {
	searchTerms := []string{
		"type " + typeName,
		"input " + typeName,
		"enum " + typeName,
		"interface " + typeName,
		"union " + typeName,
		"scalar " + typeName,
	}

	for _, term := range searchTerms {
		if lineNum := findLineNumberByText(schemaString, term); lineNum > 0 {
			return lineNum
		}
	}

	return 0
}

func findUnusedTypes(doc *ast.Document, schemaString string) []DescriptionError {
	// Collect all defined type names
	definedTypes := collectDefinedTypeNames(doc)

	// Pre-allocate with estimated capacity
	unusedTypeErrors := make([]DescriptionError, 0, len(definedTypes))

	// Mark types as used
	markUsedTypes(doc, definedTypes) // Report unused types

	for typeName, isUsed := range definedTypes {
		if isUsed {
			continue
		}

		lineNum := findTypeLineNumber(typeName, schemaString)
		lineContent := getLineContent(schemaString, lineNum)
		message := fmt.Sprintf(
			"defined-types-are-used: Type '%s' is defined but not used",
			typeName,
		)
		unusedTypeErrors = append(unusedTypeErrors, DescriptionError{
			LineNum:     lineNum,
			Message:     message,
			LineContent: lineContent,
		})
	}

	return unusedTypeErrors
}

func isCapitalized(desc string) bool {
	desc = strings.TrimSpace(desc)
	if desc == "" {
		return true
	}

	r := rune(desc[0])

	return unicode.IsUpper(r)
}

func reportUncapitalizedDescription(
	kind, parent, name, desc, schemaString string,
) *DescriptionError {
	if isCapitalized(desc) {
		return nil
	}

	var (
		lineNum     int
		lineContent string
		message     string
	)

	switch kind {
	case "type":
		lineNum = findLineNumberByText(schemaString, "type "+name)
		lineContent = getLineContent(schemaString, lineNum)
		message = "descriptions-are-capitalized: The description for type `" + name + "` should be capitalized."
	case "field":
		lineNum = findFieldDefinitionLine(schemaString, name, "")
		lineContent = getLineContent(schemaString, lineNum)
		message = "descriptions-are-capitalized: The description for field `" + parent + "." + name +
			"` should be capitalized."
	case "enum":
		lineNum = findLineNumberByText(schemaString, name)
		lineContent = getLineContent(schemaString, lineNum)
		message = "descriptions-are-capitalized: The description for enum value `" + parent + "." + name +
			"` should be capitalized."
	case "argument":
		lineNum = findLineNumberByText(schemaString, name+":")
		lineContent = getLineContent(schemaString, lineNum)
		message = "descriptions-are-capitalized: The description for argument `" + parent + "." + name +
			"` should be capitalized."
	}

	return &DescriptionError{
		LineNum:     lineNum,
		Message:     message,
		LineContent: lineContent,
	}
}

func uncapitalizedTypeDescriptions(doc *ast.Document, schemaString string) []DescriptionError {
	errors := make([]DescriptionError, 0, descriptionErrorCapacity)

	for _, obj := range doc.ObjectTypeDefinitions {
		if obj.Description.IsDefined {
			desc := doc.Input.ByteSliceString(obj.Description.Content)

			err := reportUncapitalizedDescription(
				"type",
				"",
				doc.Input.ByteSliceString(obj.Name), desc, schemaString)
			if err != nil {
				errors = append(errors, *err)
			}
		}
	}

	return errors
}

func uncapitalizedFieldDescriptions(doc *ast.Document, schemaString string) []DescriptionError {
	errors := make([]DescriptionError, 0, descriptionErrorCapacity)

	for _, obj := range doc.ObjectTypeDefinitions {
		for _, fieldRef := range obj.FieldsDefinition.Refs {
			fieldDef := doc.FieldDefinitions[fieldRef]
			if fieldDef.Description.IsDefined {
				desc := doc.Input.ByteSliceString(fieldDef.Description.Content)

				err := reportUncapitalizedDescription(
					"field",
					doc.Input.ByteSliceString(obj.Name),
					doc.Input.ByteSliceString(fieldDef.Name),
					desc, schemaString)
				if err != nil {
					errors = append(errors, *err)
				}
			}
		}
	}

	return errors
}

func uncapitalizedEnumValueDescriptions(doc *ast.Document, schemaString string) []DescriptionError {
	errors := make([]DescriptionError, 0, descriptionErrorCapacity)

	for _, enum := range doc.EnumTypeDefinitions {
		enumName := doc.Input.ByteSliceString(enum.Name)

		for _, valueRef := range enum.EnumValuesDefinition.Refs {
			valueDef := doc.EnumValueDefinitions[valueRef]
			if valueDef.Description.IsDefined {
				desc := doc.Input.ByteSliceString(valueDef.Description.Content)

				valueName := doc.Input.ByteSliceString(valueDef.EnumValue)

				err := reportUncapitalizedDescription(
					"enum",
					enumName,
					valueName,
					desc,
					schemaString,
				)
				if err != nil {
					errors = append(errors, *err)
				}
			}
		}
	}

	return errors
}

func uncapitalizedArgumentDescriptions(doc *ast.Document, schemaString string) []DescriptionError {
	errors := make([]DescriptionError, 0, descriptionErrorCapacity)

	for _, obj := range doc.ObjectTypeDefinitions {
		for _, fieldRef := range obj.FieldsDefinition.Refs {
			fieldDef := doc.FieldDefinitions[fieldRef]
			for _, argRef := range fieldDef.ArgumentsDefinition.Refs {
				argDef := doc.InputValueDefinitions[argRef]
				if argDef.Description.IsDefined {
					desc := doc.Input.ByteSliceString(argDef.Description.Content)
					argName := doc.Input.ByteSliceString(argDef.Name)

					fieldName := doc.Input.ByteSliceString(fieldDef.Name)

					err := reportUncapitalizedDescription(
						"argument",
						fieldName,
						argName,
						desc,
						schemaString,
					)
					if err != nil {
						errors = append(errors, *err)
					}
				}
			}
		}
	}

	return errors
}

func findUncapitalizedDescriptions(doc *ast.Document, schemaString string) []DescriptionError {
	errors := make([]DescriptionError, 0, defaultErrorCapacity)
	errors = append(errors, uncapitalizedTypeDescriptions(doc, schemaString)...)
	errors = append(errors, uncapitalizedFieldDescriptions(doc, schemaString)...)
	errors = append(errors, uncapitalizedEnumValueDescriptions(doc, schemaString)...)
	errors = append(errors, uncapitalizedArgumentDescriptions(doc, schemaString)...)

	return errors
}

func findMissingEnumValueDescriptions(doc *ast.Document, schemaString string) []DescriptionError {
	var errors []DescriptionError

	for _, enum := range doc.EnumTypeDefinitions {
		enumName := doc.Input.ByteSliceString(enum.Name)
		for _, valueRef := range enum.EnumValuesDefinition.Refs {
			valueDef := doc.EnumValueDefinitions[valueRef]
			if !valueDef.Description.IsDefined {
				valueName := doc.Input.ByteSliceString(valueDef.EnumValue)
				lineNum := findLineNumberByText(schemaString, valueName)
				lineContent := getLineContent(schemaString, lineNum)
				message := "enum-values-have-descriptions: Enum value '" + enumName + "." + valueName +
					"' is missing a description."
				errors = append(errors, DescriptionError{
					LineNum:     lineNum,
					Message:     message,
					LineContent: lineContent,
				})
			}
		}
	}

	return errors
}

func findMissingTypeDescriptions(doc *ast.Document, schemaString string) []DescriptionError {
	var errors []DescriptionError

	for _, obj := range doc.ObjectTypeDefinitions {
		if !obj.Description.IsDefined {
			name := doc.Input.ByteSliceString(obj.Name)
			lineNum := findLineNumberByText(schemaString, "type "+name)
			lineContent := getLineContent(schemaString, lineNum)
			message := "types-have-descriptions: Object type '" + name + "' is missing a description"
			errors = append(errors, DescriptionError{
				LineNum:     lineNum,
				Message:     message,
				LineContent: lineContent,
			})
		}
	}

	return errors
}

func findMissingFieldDescriptions(doc *ast.Document, schemaString string) []DescriptionError {
	var errors []DescriptionError

	for _, obj := range doc.ObjectTypeDefinitions {
		typeName := doc.Input.ByteSliceString(obj.Name)
		for _, fieldRef := range obj.FieldsDefinition.Refs {
			fieldDef := doc.FieldDefinitions[fieldRef]
			if !fieldDef.Description.IsDefined {
				fieldName := doc.Input.ByteSliceString(fieldDef.Name)
				lineNum := findFieldDefinitionLine(schemaString, fieldName, "")
				lineContent := getLineContent(schemaString, lineNum)
				message := "fields-have-descriptions: Field '" + typeName + "." + fieldName + "' is missing a description."
				errors = append(errors, DescriptionError{
					LineNum:     lineNum,
					Message:     message,
					LineContent: lineContent,
				})
			}
		}
	}

	return errors
}

func findTypesAreCapitalized(doc *ast.Document, schemaString string) []DescriptionError {
	errors := make([]DescriptionError, 0)

	for _, obj := range doc.ObjectTypeDefinitions {
		typeName := doc.Input.ByteSliceString(obj.Name)
		if typeName == RootQueryType || typeName == RootMutationType ||
			typeName == RootSubscriptionType {
			continue
		}

		if len(typeName) == 0 || !unicode.IsUpper(rune(typeName[0])) {
			lineNum := findLineNumberByText(schemaString, "type "+typeName)
			lineContent := getLineContent(schemaString, lineNum)
			message := "types-are-capitalized: The object type '" + typeName + "' should start with a capital letter."
			errors = append(errors, DescriptionError{
				LineNum:     lineNum,
				Message:     message,
				LineContent: lineContent,
			})
		}
	}

	return errors
}

func (s Store) lintDescriptions(
	doc *ast.Document,
	schemaString string,
	schemaPath string,
) ([]DescriptionError, bool) {
	descriptionErrors := make([]DescriptionError, 0, defaultErrorCapacity)
	hasUnsuppressedDeprecationReasonError := false

	helpers := []func(*ast.Document, string) []DescriptionError{
		findTypesAreCapitalized,
		findMissingQueryRootType,
		findUnsortedTypeFields,
		findUnsortedInterfaceFields,
		findRelayPageInfoSpec,
		findRelayConnectionTypesSpec,
		findRelayConnectionArgumentsSpec,
		findInputObjectValuesCamelCased,
		findFieldsAreCamelCased,
		findInputObjectFieldsSortedAlphabetically,
		findMissingEnumValueDescriptions,
		findUncapitalizedDescriptions,
		findUnusedTypes,
		findMissingTypeDescriptions,
		findMissingFieldDescriptions,
		findMissingInputObjectValueDescriptions,
		findMissingArgumentDescriptions,
		findMissingDeprecationReasons,
	}

	for _, helper := range helpers {
		errList := helper(doc, schemaString)
		for _, err := range errList {
			descriptionErrors = append(descriptionErrors, err)

			if strings.Contains(err.Message, "deprecations-have-a-reason") {
				// Check if this specific deprecation error is suppressed
				rule := err.Message
				if idx := strings.Index(rule, ":"); idx != -1 {
					rule = rule[:idx]
				}

				if !s.isSuppressed(schemaPath, err.LineNum, rule, "") {
					hasUnsuppressedDeprecationReasonError = true
				}
			}
		}
	}

	// Handle enum values sorted alphabetically separately since it requires Store receiver
	enumSortErrors := s.findEnumValuesSortedAlphabetically(doc, schemaString, schemaPath)
	descriptionErrors = append(descriptionErrors, enumSortErrors...)

	return sortDescriptionErrors(descriptionErrors), hasUnsuppressedDeprecationReasonError
}

func sortDescriptionErrors(errors []DescriptionError) []DescriptionError {
	// No-op: return as-is
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

func findUnsortedFields(
	fieldDefs []int,
	getFieldName func(int) string,
	typeLabel,
	typeName,
	schemaString string,
) []DescriptionError {
	fieldNames := make([]string, len(fieldDefs))
	for i, fieldRef := range fieldDefs {
		fieldNames[i] = getFieldName(fieldRef)
	}

	if len(fieldNames) < minFieldsForSortCheck {
		return nil
	}

	sorted := make([]string, len(fieldNames))
	copy(sorted, fieldNames)
	sort.Strings(sorted)

	for i := range fieldNames {
		if fieldNames[i] != sorted[i] {
			lineNum := findLineNumberByText(schemaString, typeLabel+" "+typeName)
			lineContent := getLineContent(schemaString, lineNum)
			message := typeLabel + "-fields-sorted-alphabetically: The fields of " +
				typeLabel + " type `" + typeName + "` should be sorted in alphabetical order.\nExpected sorting: " +
				strings.Join(sorted, ", ")

			return []DescriptionError{{
				LineNum:     lineNum,
				Message:     message,
				LineContent: lineContent,
			}}
		}
	}

	return nil
}

func findUnsortedTypeFields(doc *ast.Document, schemaString string) []DescriptionError {
	var errors []DescriptionError

	for _, obj := range doc.ObjectTypeDefinitions {
		typeName := doc.Input.ByteSliceString(obj.Name)

		err := findUnsortedFields(
			obj.FieldsDefinition.Refs,
			func(fieldRef int) string { return doc.Input.ByteSliceString(doc.FieldDefinitions[fieldRef].Name) },
			"type",
			typeName,
			schemaString,
		)
		if err != nil {
			errors = append(errors, err...)
		}
	}

	return errors
}

func findUnsortedInterfaceFields(doc *ast.Document, schemaString string) []DescriptionError {
	var errors []DescriptionError

	for _, iface := range doc.InterfaceTypeDefinitions {
		ifaceName := doc.Input.ByteSliceString(iface.Name)

		err := findUnsortedFields(
			iface.FieldsDefinition.Refs,
			func(fieldRef int) string { return doc.Input.ByteSliceString(doc.FieldDefinitions[fieldRef].Name) },
			"interface",
			ifaceName,
			schemaString,
		)
		if err != nil {
			errors = append(errors, err...)
		}
	}

	return errors
}

func findRelayConnectionTypesSpec(doc *ast.Document, schemaString string) []DescriptionError {
	var errors []DescriptionError

	for _, obj := range doc.ObjectTypeDefinitions {
		typeName := doc.Input.ByteSliceString(obj.Name)
		if !strings.HasSuffix(typeName, "Connection") {
			continue
		}

		hasPageInfo := false

		for _, fieldRef := range obj.FieldsDefinition.Refs {
			fieldDef := doc.FieldDefinitions[fieldRef]

			fieldName := doc.Input.ByteSliceString(fieldDef.Name)
			if fieldName == "pageInfo" {
				hasPageInfo = true

				break
			}
		}

		if !hasPageInfo {
			lineNum := findLineNumberByText(schemaString, "type "+typeName)
			lineContent := getLineContent(schemaString, lineNum)
			message := fmt.Sprintf(
				"relay-connection-types-spec: Connection `%s` is missing the following field: pageInfo.",
				typeName,
			)
			errors = append(errors, DescriptionError{
				LineNum:     lineNum,
				Message:     message,
				LineContent: lineContent,
			})
		}
	}

	return errors
}

func findRelayConnectionArgumentsSpec(doc *ast.Document, schemaString string) []DescriptionError {
	var errors []DescriptionError

	for _, fieldDef := range doc.FieldDefinitions {
		fieldType := doc.Types[fieldDef.Type]
		baseType := getBaseTypeName(doc, fieldType)

		if !strings.HasSuffix(baseType, "Connection") {
			continue
		}

		hasForwardArgs := false
		hasBackwardArgs := false

		for _, argRef := range fieldDef.ArgumentsDefinition.Refs {
			argDef := doc.InputValueDefinitions[argRef]
			argName := doc.Input.ByteSliceString(argDef.Name)

			switch argName {
			case "first", "after":
				hasForwardArgs = true
			case "last", "before":
				hasBackwardArgs = true
			}
		}

		if !hasForwardArgs && !hasBackwardArgs {
			fieldName := doc.Input.ByteSliceString(fieldDef.Name)
			lineNum := findFieldDefinitionLine(schemaString, fieldName, "")
			lineContent := getLineContent(schemaString, lineNum)
			message := "relay-connection-arguments-spec: A field that returns a Connection Type must include forward" +
				"pagination arguments (`first` and `after`), backward pagination arguments (`last` and `before`), or both as" +
				"per the Relay spec."
			errors = append(errors, DescriptionError{
				LineNum:     lineNum,
				Message:     message,
				LineContent: lineContent,
			})
		}
	}

	return errors
}

func findRelayPageInfoSpec(doc *ast.Document, schemaString string) []DescriptionError {
	for _, obj := range doc.ObjectTypeDefinitions {
		if doc.Input.ByteSliceString(obj.Name) == "PageInfo" {
			return nil
		}
	}

	lineNum := 1
	lineContent := getLineContent(schemaString, lineNum)
	message := "relay-page-info-spec: A `PageInfo` object type is required as per the Relay spec."

	return []DescriptionError{{
		LineNum:     lineNum,
		Message:     message,
		LineContent: lineContent,
	}}
}

func findInputObjectValuesCamelCased(doc *ast.Document, schemaString string) []DescriptionError {
	var errors []DescriptionError

	for _, input := range doc.InputObjectTypeDefinitions {
		inputName := doc.Input.ByteSliceString(input.Name)

		for _, fieldRef := range input.InputFieldsDefinition.Refs {
			fieldDef := doc.InputValueDefinitions[fieldRef]

			fieldName := doc.Input.ByteSliceString(fieldDef.Name)
			if !isCamelCase(fieldName) {
				lineNum := findLineNumberByText(schemaString, fieldName+":")
				lineContent := getLineContent(schemaString, lineNum)
				message := "input-object-values-are-camel-cased: The input value `" +
					inputName + "." + fieldName + "` is not camel cased."
				errors = append(errors, DescriptionError{
					LineNum:     lineNum,
					Message:     message,
					LineContent: lineContent,
				})
			}
		}
	}

	return errors
}

func isCamelCase(str string) bool {
	if str == "" {
		return false
	}

	if strings.Contains(str, "_") {
		return false
	}

	return str[0] >= 'a' && str[0] <= 'z'
}

func findFieldsAreCamelCased(doc *ast.Document, schemaString string) []DescriptionError {
	var errors []DescriptionError

	for _, obj := range doc.ObjectTypeDefinitions {
		typeName := doc.Input.ByteSliceString(obj.Name)

		for _, fieldRef := range obj.FieldsDefinition.Refs {
			fieldDef := doc.FieldDefinitions[fieldRef]

			fieldName := doc.Input.ByteSliceString(fieldDef.Name)
			if !isCamelCase(fieldName) {
				lineNum := findFieldDefinitionLine(schemaString, fieldName, "")
				lineContent := getLineContent(schemaString, lineNum)
				message := "fields-are-camel-cased: The field '" + typeName + "." + fieldName + "' is not camel cased."
				errors = append(errors, DescriptionError{
					LineNum:     lineNum,
					Message:     message,
					LineContent: lineContent,
				})
			}
		}
	}

	return errors
}

func checkSortedOrder(names []string,
	minLength int,
	schemaString,
	searchPrefix,
	itemName,
	rulePrefix string,
) *DescriptionError {
	if len(names) < minLength {
		return nil
	}

	sorted := make([]string, len(names))
	copy(sorted, names)
	sort.Strings(sorted)

	if !equalStringSlices(names, sorted) {
		lineNum := findLineNumberByText(schemaString, searchPrefix+itemName)
		lineContent := getLineContent(schemaString, lineNum)
		message := rulePrefix + ": The " + itemName +
			" should be sorted in alphabetical order. Expected sorting: " + strings.Join(
			sorted,
			", ",
		)

		return &DescriptionError{
			LineNum:     lineNum,
			Message:     message,
			LineContent: lineContent,
		}
	}

	return nil
}

func findInputObjectFieldsSortedAlphabetically(
	doc *ast.Document,
	schemaString string,
) []DescriptionError {
	var errors []DescriptionError

	for _, input := range doc.InputObjectTypeDefinitions {
		inputName := doc.Input.ByteSliceString(input.Name)

		var fieldNames []string

		for _, fieldRef := range input.InputFieldsDefinition.Refs {
			fieldDef := doc.InputValueDefinitions[fieldRef]
			fieldNames = append(fieldNames, doc.Input.ByteSliceString(fieldDef.Name))
		}

		if err := checkSortedOrder(
			fieldNames,
			minFieldsForSortCheck,
			schemaString,
			"input ",
			"fields of input type '"+inputName+"'",
			"input-object-fields-sorted-alphabetically",
		); err != nil {
			errors = append(errors, *err)
		}
	}

	return errors
}

func findMissingInputObjectValueDescriptions(
	doc *ast.Document,
	schemaString string,
) []DescriptionError {
	var errors []DescriptionError

	for _, input := range doc.InputObjectTypeDefinitions {
		inputName := doc.Input.ByteSliceString(input.Name)
		for _, fieldRef := range input.InputFieldsDefinition.Refs {
			fieldDef := doc.InputValueDefinitions[fieldRef]
			if !fieldDef.Description.IsDefined {
				fieldName := doc.Input.ByteSliceString(fieldDef.Name)
				lineNum := findLineNumberByText(schemaString, fieldName+":")
				lineContent := getLineContent(schemaString, lineNum)
				message := fmt.Sprintf(
					"input-object-values-have-descriptions: The input value `%s.%s` is missing a description.",
					inputName,
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
					message := "arguments-have-descriptions: The '" + argName + "' argument of '" + fieldName +
						"' is missing a description."
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

func findMissingDeprecationReasons(doc *ast.Document, schemaString string) []DescriptionError {
	var errors []DescriptionError

	for _, enum := range doc.EnumTypeDefinitions {
		enumName := doc.Input.ByteSliceString(enum.Name)
		for _, valueRef := range enum.EnumValuesDefinition.Refs {
			valueDef := doc.EnumValueDefinitions[valueRef]

			valueName := doc.Input.ByteSliceString(valueDef.EnumValue)
			for _, dirRef := range valueDef.Directives.Refs {
				dir := doc.Directives[dirRef]

				dirName := doc.Input.ByteSliceString(dir.Name)
				if dirName == "deprecated" && len(dir.Arguments.Refs) == 0 {
					lineNum := findLineNumberByText(schemaString, valueName)
					lineContent := getLineContent(schemaString, lineNum)
					message := "deprecations-have-a-reason: Deprecated enum value '" + enumName + "." +
						valueName + "' is missing a reason."
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

func (s Store) findEnumValuesSortedAlphabetically(
	doc *ast.Document,
	schemaString string,
	schemaPath string,
) []DescriptionError {
	var errors []DescriptionError

	for _, enum := range doc.EnumTypeDefinitions {
		enumName := doc.Input.ByteSliceString(enum.Name)

		var valueNames []string

		for _, valueRef := range enum.EnumValuesDefinition.Refs {
			valueDef := doc.EnumValueDefinitions[valueRef]
			valueNames = append(valueNames, doc.Input.ByteSliceString(valueDef.EnumValue))
		}

		if err := checkSortedOrder(
			valueNames,
			minEnumValuesForSortCheck,
			schemaString,
			"enum ",
			enumName,
			"enum-values-sorted-alphabetically",
		); err != nil {
			// Extract the error message part after the colon for suppression value matching
			messageParts := strings.SplitN(err.Message, ": ", splitNParts)

			suppressionValue := ""
			if len(messageParts) > 1 {
				suppressionValue = messageParts[1]
			}

			if !s.isSuppressed(
				schemaPath,
				err.LineNum,
				"enum-values-sorted-alphabetically",
				suppressionValue,
			) {
				errors = append(errors, *err)
			}
		}
	}

	return errors
}

func equalStringSlices(sliceA, sliceB []string) bool {
	if len(sliceA) != len(sliceB) {
		return false
	}

	for i := range sliceA {
		if sliceA[i] != sliceB[i] {
			return false
		}
	}

	return true
}

func (s Store) collectUnsuppressedDataTypeErrors(
	doc *ast.Document,
	schemaString, schemaFile string,
) (int, []DescriptionError) {
	unsuppressedDataTypeErrors := 0

	var allErrors []DescriptionError

	dataTypesValid, dataTypeErrorLines := s.validateDataTypes(doc, schemaString, schemaFile)
	if !dataTypesValid {
		for _, lineNum := range dataTypeErrorLines {
			if !s.isSuppressed(schemaFile, lineNum, "defined-types-are-used", "") {
				allErrors = append(allErrors, DescriptionError{
					FilePath:    schemaFile,
					LineNum:     lineNum,
					Message:     "defined-types-are-used: Type is defined but not used",
					LineContent: getLineContent(schemaString, lineNum),
				})
				unsuppressedDataTypeErrors++
			}
		}
	}

	return unsuppressedDataTypeErrors, allErrors
}

func (s Store) getUnsuppressedDescriptionErrors(
	descriptionErrors []DescriptionError,
	schemaFile string,
) []DescriptionError {
	unsuppressed := make([]DescriptionError, 0, len(descriptionErrors))
	for _, err := range descriptionErrors {
		rule := err.Message
		if idx := strings.Index(rule, ":"); idx != -1 {
			rule = rule[:idx]
		}

		if !s.isSuppressed(schemaFile, err.LineNum, rule, "") {
			unsuppressed = append(unsuppressed, err)
		}
	}

	return unsuppressed
}

func (s Store) lintSingleSchemaFile(schemaFile string) (int, int, []DescriptionError) {
	if s.Verbose {
		log.Infof("=== Linting %s ===", schemaFile)
	}

	schemaString, ok := s.readAndValidateSchemaFile(schemaFile)
	if !ok {
		return 1, 1, []DescriptionError{{
			FilePath:    schemaFile,
			LineNum:     0,
			Message:     "failed-to-read-schema-file: failed to read schema file",
			LineContent: "",
		}}
	}

	filteredSchema, doc, parseReport := s.parseAndFilterSchema(schemaString)
	LogSchemaParseErrors(schemaString, &parseReport)
	descriptionErrors, hasUnsuppressedDeprecationReasonError := s.lintDescriptions(
		&doc,
		schemaString,
		schemaFile,
	)
	unsuppressedDescriptionErrors := s.getUnsuppressedDescriptionErrors(
		descriptionErrors,
		schemaFile,
	)
	unsuppressedDataTypeErrors, dataTypeErrors := s.collectUnsuppressedDataTypeErrors(
		&doc,
		schemaString,
		schemaFile,
	)
	allErrors := append([]DescriptionError{}, dataTypeErrors...)
	unsuppressedDirectiveOrFederationError := !validateDirectiveNames(&doc) ||
		!federation.ValidateFederationSchema(filteredSchema)

	totalErrors, errorFilesCount := s.summarizeLintResults(
		len(unsuppressedDescriptionErrors),
		hasUnsuppressedDeprecationReasonError,
		unsuppressedDataTypeErrors,
		unsuppressedDirectiveOrFederationError,
	)
	if totalErrors > 0 {
		for i := range unsuppressedDescriptionErrors {
			unsuppressedDescriptionErrors[i].FilePath = schemaFile
		}

		allErrors = append(allErrors, unsuppressedDescriptionErrors...)
	}

	return totalErrors, errorFilesCount, allErrors
}

func (s Store) readAndValidateSchemaFile(schemaFile string) (string, bool) {
	schemaString, ok := readSchemaFile(schemaFile)

	return schemaString, ok
}

func (s Store) parseAndFilterSchema(
	schemaString string,
) (string, ast.Document, operationreport.Report) {
	filteredSchema := filterSchemaComments(schemaString)
	doc, parseReport := astparser.ParseGraphqlDocumentString(schemaString)

	return filteredSchema, doc, parseReport
}

func (s Store) summarizeLintResults(
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
