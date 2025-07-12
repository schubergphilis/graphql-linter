package data

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/schubergphilis/mcvs-golang-project-root/pkg/projectroot"
	log "github.com/sirupsen/logrus"
	"github.com/wundergraph/graphql-go-tools/pkg/lexer/position"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/ast"
	"gopkg.in/yaml.v3"
)

const levenshteinThreshold = 2

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
		} // if !lintSchemaFile(schemaFile) { totalErrors++ }
	}

	printReport(schemaFiles, totalErrors)

	return nil
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

func levenshteinDistance(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}

	if len(b) == 0 {
		return len(a)
	}

	matrix := make([][]int, len(a)+1)
	for row := range matrix {
		matrix[row] = make([]int, len(b)+1)
	}

	for row := 0; row <= len(a); row++ {
		matrix[row][0] = row
	}

	for col := 0; col <= len(b); col++ {
		matrix[0][col] = col
	}

	for row := 1; row <= len(a); row++ {
		for col := 1; col <= len(b); col++ {
			cost := 0
			if a[row-1] != b[col-1] {
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

	return matrix[len(a)][len(b)]
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
