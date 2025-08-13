package data

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/data/base/models"
	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/data/base/rules"
	pkg_rules "github.com/schubergphilis/graphql-linter/internal/pkg/rules"
	"github.com/schubergphilis/mcvs-golang-project-root/pkg/projectroot"
	log "github.com/sirupsen/logrus"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/ast"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/astparser"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/operationreport"
	"gopkg.in/yaml.v3"
)

const (
	descriptionErrorCapacity = 8
)

type Storer interface {
	FindAndLogGraphQLSchemaFiles() ([]string, error)
	LintSchemaFiles(schemaFiles []string) (int, int, []models.DescriptionError)
	LoadConfig() (*models.LinterConfig, error)
	UnsortedTypeFields(doc *ast.Document, schemaString string) []models.DescriptionError
}

type Store struct {
	ConfigPath   string
	LinterConfig *models.LinterConfig
	Ruler        rules.Ruler
	TargetPath   string
	Verbose      bool
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

type errorResult struct {
	errors     []string
	errorLines []int
}

func NewStore(
	configPath, targetPath string,
	ruler rules.Ruler,
	verbose bool,
) (Store, error) {
	store := Store{
		ConfigPath: configPath,
		Ruler:      ruler,
		TargetPath: targetPath,
		Verbose:    verbose,
	}

	return store, nil
}

func (s Store) LoadConfig() (*models.LinterConfig, error) {
	configPath := s.ConfigPath
	config := &models.LinterConfig{
		Settings: models.Settings{
			StrictMode:         true,
			ValidateFederation: true,
			CheckDescriptions:  true,
		},
	}

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

	if s.Verbose {
		log.Infof("loaded config with %d suppressions", len(config.Suppressions))
	}

	return config, nil
}

func readSchemaFile(schemaPath string) (string, bool) {
	schemaBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		log.WithError(err).Error("failed to read schema file")

		return "", false
	}

	return string(schemaBytes), true
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
		log.Error("Data type validation FAILED - schema contains invalid type references")

		return false, errorLines, enumDescErrors
	}

	log.Debug("Data type validation PASSED")

	return true, errorLines, enumDescErrors
}

func (s Store) UncapitalizedDescriptions(doc *ast.Document, schemaString string) []models.DescriptionError {
	errors := make([]models.DescriptionError, 0, pkg_rules.DefaultErrorCapacity)
	errors = append(errors, s.uncapitalizedTypeDescriptions(doc, schemaString)...)
	errors = append(errors, s.uncapitalizedFieldDescriptions(doc, schemaString)...)
	errors = append(errors, s.uncapitalizedEnumValueDescriptions(doc, schemaString)...)
	errors = append(errors, s.uncapitalizedArgumentDescriptions(doc, schemaString)...)

	return errors
}

func (s Store) UnsortedTypeFields(doc *ast.Document, schemaString string) []models.DescriptionError {
	var errors []models.DescriptionError

	for _, obj := range doc.ObjectTypeDefinitions {
		typeName := doc.Input.ByteSliceString(obj.Name)

		err := s.Ruler.UnsortedFields(
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

func (s Store) UnsortedInterfaceFields(doc *ast.Document, schemaString string) []models.DescriptionError {
	var errors []models.DescriptionError

	for _, iface := range doc.InterfaceTypeDefinitions {
		ifaceName := doc.Input.ByteSliceString(iface.Name)

		err := s.Ruler.UnsortedFields(
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

		if !pkg_rules.IsSuppressedNoValue(schemaFile, enumErr.LineNum, modelsLinterConfig, rule) {
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

func (s Store) ReadAndValidateSchemaFile(schemaFile string) (string, bool) {
	schemaString, ok := readSchemaFile(schemaFile)

	return schemaString, ok
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

	fieldTypeResultErrs, fieldTypeResultLines := s.Ruler.ValidateFieldTypes(
		doc,
		schemaContent,
		builtInScalars,
		definedTypes,
	)
	inputFieldTypeResultErrs, inputFieldTypeResultLines := s.Ruler.ValidateInputFieldTypes(
		doc,
		schemaContent,
		builtInScalars,
		definedTypes,
	)
	enumTypeResultErrs, enumTypeResultLines, descErrs := s.Ruler.ValidateEnumTypes(
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

func (s Store) uncapitalizedTypeDescriptions(
	doc *ast.Document,
	schemaString string,
) []models.DescriptionError {
	errors := make([]models.DescriptionError, 0, descriptionErrorCapacity)

	for _, obj := range doc.ObjectTypeDefinitions {
		if obj.Description.IsDefined {
			desc := doc.Input.ByteSliceString(obj.Description.Content)

			err := s.Ruler.ReportUncapitalizedDescription(
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

func (s Store) uncapitalizedFieldDescriptions(
	doc *ast.Document,
	schemaString string,
) []models.DescriptionError {
	errors := make([]models.DescriptionError, 0, descriptionErrorCapacity)

	for _, obj := range doc.ObjectTypeDefinitions {
		for _, fieldRef := range obj.FieldsDefinition.Refs {
			fieldDef := doc.FieldDefinitions[fieldRef]
			if fieldDef.Description.IsDefined {
				desc := doc.Input.ByteSliceString(fieldDef.Description.Content)

				err := s.Ruler.ReportUncapitalizedDescription(
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

func (s Store) uncapitalizedEnumValueDescriptions(
	doc *ast.Document,
	schemaString string,
) []models.DescriptionError {
	errors := make([]models.DescriptionError, 0, descriptionErrorCapacity)

	for _, enum := range doc.EnumTypeDefinitions {
		enumName := doc.Input.ByteSliceString(enum.Name)

		for _, valueRef := range enum.EnumValuesDefinition.Refs {
			valueDef := doc.EnumValueDefinitions[valueRef]
			if valueDef.Description.IsDefined {
				desc := doc.Input.ByteSliceString(valueDef.Description.Content)

				valueName := doc.Input.ByteSliceString(valueDef.EnumValue)

				err := s.Ruler.ReportUncapitalizedDescription(
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

func (s Store) uncapitalizedArgumentDescriptions(
	doc *ast.Document,
	schemaString string,
) []models.DescriptionError {
	errors := make([]models.DescriptionError, 0, descriptionErrorCapacity)

	for _, obj := range doc.ObjectTypeDefinitions {
		for _, fieldRef := range obj.FieldsDefinition.Refs {
			fieldDef := doc.FieldDefinitions[fieldRef]
			for _, argRef := range fieldDef.ArgumentsDefinition.Refs {
				argDef := doc.InputValueDefinitions[argRef]
				if argDef.Description.IsDefined {
					desc := doc.Input.ByteSliceString(argDef.Description.Content)
					argName := doc.Input.ByteSliceString(argDef.Name)

					fieldName := doc.Input.ByteSliceString(fieldDef.Name)

					err := s.Ruler.ReportUncapitalizedDescription(
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

func loadDefaultConfig(config *models.LinterConfig) (*models.LinterConfig, error) {
	log.Debug("No config path provided, using default project root search")

	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to determine project root: %w", err)
	}

	defaultConfigPath := filepath.Join(projectRoot, ".graphql-linter.yml")

	_, statErr := os.Stat(defaultConfigPath)
	if statErr == nil {
		data, readErr := os.ReadFile(defaultConfigPath)
		if readErr != nil {
			return nil, fmt.Errorf("failed to read config file: %w", readErr)
		}

		yamlErr := yaml.Unmarshal(data, config)
		if yamlErr != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", yamlErr)
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
