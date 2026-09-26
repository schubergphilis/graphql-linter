package data

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/data/base/models"
	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/data/base/rules"
	pkg_rules "github.com/schubergphilis/graphql-linter/internal/pkg/rules"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/ast"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/astparser"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/operationreport"
	"gopkg.in/yaml.v3"
)

type Store struct {
	ConfigPath   string
	LinterConfig *models.LinterConfig
	TargetPath   string
}

type errorResult struct {
	errors     []string
	errorLines []int
}

func NewStore(configPath, targetPath string) Store {
	return Store{
		ConfigPath: configPath,
		TargetPath: targetPath,
	}
}

func (s Store) LoadConfig() (*models.LinterConfig, error) {
	configPath := s.ConfigPath
	config := models.NewLinterConfig()

	if configPath == "" {
		cfg, err := loadDefaultConfig(config)
		if err != nil {
			return nil, err
		}

		config = cfg
	} else {
		cfg, err := loadCustomConfig(configPath, config)
		if err != nil {
			return nil, err
		}

		config = cfg
	}

	slog.Debug(fmt.Sprintf("loaded config with %d suppressions", len(config.Suppressions)))

	return config, nil
}

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

func (s Store) ValidateDataTypes(
	doc *ast.Document,
	modelsLinterConfig *models.LinterConfig,
	schemaContent string,
	schemaPath string,
) (bool, []int, []models.DescriptionError) {
	builtInScalars := map[string]bool{
		"String":  true,
		"Int":     true,
		"Float":   true,
		"Boolean": true,
		"ID":      true,
	}
	definedTypesSet := rules.CollectDefinedTypes(doc)

	definedTypes := make(map[string]bool)
	for k := range definedTypesSet {
		definedTypes[k] = true
	}

	hasErrors, errorLines, enumDescErrors := s.collectDataTypeErrors(
		doc,
		modelsLinterConfig,
		schemaContent,
		schemaPath,
		builtInScalars,
		definedTypes,
	)

	if hasErrors {
		slog.Error("Data type validation FAILED - schema contains invalid type references")

		return false, errorLines, enumDescErrors
	}

	slog.Debug("Data type validation PASSED")

	return true, errorLines, enumDescErrors
}

func (s Store) UnsortedTypeFields(doc *ast.Document, schemaString string) []models.DescriptionError {
	var errors []models.DescriptionError

	for _, obj := range doc.ObjectTypeDefinitions {
		typeName := doc.Input.ByteSliceString(obj.Name)

		err := rules.Rule{}.UnsortedFields(
			obj.FieldsDefinition.Refs,
			func(fieldRef int) string { return doc.Input.ByteSliceString(doc.FieldDefinitions[fieldRef].Name) },
			"type",
			typeName,
			schemaString,
			rules.LineOf(doc, obj.Name),
		)
		if err != nil {
			errors = append(errors, err...)
		}
	}

	return errors
}

func (s Store) UnsortedInterfaceFields(doc *ast.Document, schemaString string) []models.DescriptionError {
	var errors []models.DescriptionError

	for _, iface := range doc.InterfaceTypeDefinitions {
		ifaceName := doc.Input.ByteSliceString(iface.Name)

		err := rules.Rule{}.UnsortedFields(
			iface.FieldsDefinition.Refs,
			func(fieldRef int) string { return doc.Input.ByteSliceString(doc.FieldDefinitions[fieldRef].Name) },
			"interface",
			ifaceName,
			schemaString,
			rules.LineOf(doc, iface.Name),
		)
		if err != nil {
			errors = append(errors, err...)
		}
	}

	return errors
}

func (s Store) CollectUnsuppressedDataTypeErrors(
	doc *ast.Document,
	modelsLinterConfig *models.LinterConfig,
	schemaString, schemaFile string,
) (int, []models.DescriptionError) {
	unsuppressedDataTypeErrors := 0

	var allErrors []models.DescriptionError

	_, _, enumDescErrors := s.ValidateDataTypes(
		doc,
		modelsLinterConfig,
		schemaString,
		schemaFile,
	)

	for _, enumErr := range enumDescErrors {
		rule := enumErr.Message
		if idx := strings.Index(rule, ":"); idx != -1 {
			rule = rule[:idx]
		}

		if !pkg_rules.IsSuppressed(schemaFile, enumErr.LineNum, modelsLinterConfig, rule, enumErr.Value) {
			allErrors = append(allErrors, enumErr)
			unsuppressedDataTypeErrors++
		}
	}

	return unsuppressedDataTypeErrors, allErrors
}

func (s Store) ParseAndFilterSchema(
	schemaString string,
) (string, ast.Document, operationreport.Report) {
	filteredSchema := FilterSchemaComments(schemaString)
	doc, parseReport := astparser.ParseGraphqlDocumentString(schemaString)

	return filteredSchema, doc, parseReport
}

func (s Store) collectDataTypeErrors(
	doc *ast.Document,
	modelsLinterConfig *models.LinterConfig,
	schemaContent string,
	schemaPath string,
	builtInScalars map[string]bool,
	definedTypes map[string]bool,
) (bool, []int, []models.DescriptionError) {
	hasErrors := false

	var (
		errorLines     []int
		enumDescErrors []models.DescriptionError
	)

	fieldTypeResultErrs, fieldTypeResultLines := rules.Rule{}.ValidateFieldTypes(
		doc,
		builtInScalars,
		definedTypes,
	)
	inputFieldTypeResultErrs, inputFieldTypeResultLines := rules.Rule{}.ValidateInputFieldTypes(
		doc,
		builtInScalars,
		definedTypes,
	)
	enumTypeResultErrs, enumTypeResultLines, descErrs := rules.Rule{}.ValidateEnumTypes(
		doc,
		modelsLinterConfig,
		schemaContent,
		schemaPath,
	)
	enumDescErrors = descErrs

	errorResults := []errorResult{
		{fieldTypeResultErrs, fieldTypeResultLines},
		{inputFieldTypeResultErrs, inputFieldTypeResultLines},
		{enumTypeResultErrs, enumTypeResultLines},
	}

	for _, res := range errorResults {
		if len(res.errors) > 0 {
			hasErrors = true

			errorLines = append(errorLines, res.errorLines...)
		}
	}

	return hasErrors, errorLines, enumDescErrors
}

// defaultConfigFiles are looked up in the working directory, in this order.
var defaultConfigFiles = []string{".graphql-linter.yml", ".graphql-linter.yaml"}

func loadDefaultConfig(config *models.LinterConfig) (*models.LinterConfig, error) {
	for _, defaultConfigPath := range defaultConfigFiles {
		_, err := os.Stat(defaultConfigPath)
		if err == nil {
			slog.Debug("No config path provided, using " + defaultConfigPath)

			return loadCustomConfig(defaultConfigPath, config)
		}
	}

	return config, nil
}

func loadCustomConfig(
	configPath string,
	config *models.LinterConfig,
) (*models.LinterConfig, error) {
	_, statErr := os.Stat(configPath)
	if os.IsNotExist(statErr) {
		return nil, fmt.Errorf("config file does not exist at path: %s", configPath)
	}

	data, readErr := os.ReadFile(configPath)
	if readErr != nil {
		return nil, fmt.Errorf("failed to read config file: %w", readErr)
	}

	yamlErr := yaml.Unmarshal(data, config)
	if yamlErr != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", yamlErr)
	}

	return config, nil
}
