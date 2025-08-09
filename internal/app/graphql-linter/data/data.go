package data

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/data/base/models"
	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/data/base/rules"
	pkgrules "github.com/schubergphilis/graphql-linter/internal/pkg/rules"
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
}

type Store struct {
	LinterConfig     *models.LinterConfig
	LinterConfigPath string
	TargetPath       string
	Verbose          bool
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

func NewStore(targetPath string, verbose bool) (Store, error) {
	s := Store{
		TargetPath: targetPath,
		Verbose:    verbose,
	}

	return s, nil
}

func (s Store) LoadConfig() (*models.LinterConfig, error) {
	configPath := s.LinterConfigPath

	if configPath == "" {
		log.Debug("No config path provided, using default project root search")

		projectRoot, err := projectroot.FindProjectRoot()
		if err != nil {
			return nil, fmt.Errorf("failed to determine project root: %w", err)
		}

		configPath = filepath.Join(projectRoot, ".graphql-linter.yml")
	}

	config := &models.LinterConfig{
		Settings: models.Settings{
			StrictMode:         true,
			ValidateFederation: true,
			CheckDescriptions:  true,
		},
	}

	_, err := os.Stat(configPath)
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

func (s Store) ReadAndValidateSchemaFile(schemaFile string) (string, bool) {
	return s.readAndValidateSchemaFile(schemaFile)
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
	definedTypes := rules.CollectDefinedTypes(doc)
	hasErrors := false

	var (
		errorLines     []int
		enumDescErrors []models.DescriptionError
	)

	errorResults := []errorResult{
		func() errorResult {
			errs, lines := rules.ValidateFieldTypes(doc, schemaContent, builtInScalars, definedTypes)

			return errorResult{errs, lines}
		}(),
		func() errorResult {
			errs, lines := rules.ValidateInputFieldTypes(doc, schemaContent, builtInScalars, definedTypes)

			return errorResult{errs, lines}
		}(),
		func() errorResult {
			errs, lines, descErrs := rules.ValidateEnumTypes(doc, modelsLinterConfig, schemaContent, schemaPath)
			enumDescErrors = descErrs

			return errorResult{errs, lines}
		}(),
	}

	for _, res := range errorResults {
		if len(res.errors) > 0 {
			hasErrors = true

			errorLines = append(errorLines, res.errorLines...)
		}
	}

	if hasErrors {
		log.Error("Data type validation FAILED - schema contains invalid type references")

		return false, errorLines, enumDescErrors
	}

	log.Debug("Data type validation PASSED")

	return true, errorLines, enumDescErrors
}

func uncapitalizedTypeDescriptions(doc *ast.Document, schemaString string) []models.DescriptionError {
	errors := make([]models.DescriptionError, 0, descriptionErrorCapacity)

	for _, obj := range doc.ObjectTypeDefinitions {
		if obj.Description.IsDefined {
			desc := doc.Input.ByteSliceString(obj.Description.Content)

			err := rules.ReportUncapitalizedDescription(
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

func uncapitalizedFieldDescriptions(doc *ast.Document, schemaString string) []models.DescriptionError {
	errors := make([]models.DescriptionError, 0, descriptionErrorCapacity)

	for _, obj := range doc.ObjectTypeDefinitions {
		for _, fieldRef := range obj.FieldsDefinition.Refs {
			fieldDef := doc.FieldDefinitions[fieldRef]
			if fieldDef.Description.IsDefined {
				desc := doc.Input.ByteSliceString(fieldDef.Description.Content)

				err := rules.ReportUncapitalizedDescription(
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

func uncapitalizedEnumValueDescriptions(doc *ast.Document, schemaString string) []models.DescriptionError {
	errors := make([]models.DescriptionError, 0, descriptionErrorCapacity)

	for _, enum := range doc.EnumTypeDefinitions {
		enumName := doc.Input.ByteSliceString(enum.Name)

		for _, valueRef := range enum.EnumValuesDefinition.Refs {
			valueDef := doc.EnumValueDefinitions[valueRef]
			if valueDef.Description.IsDefined {
				desc := doc.Input.ByteSliceString(valueDef.Description.Content)

				valueName := doc.Input.ByteSliceString(valueDef.EnumValue)

				err := rules.ReportUncapitalizedDescription(
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

func uncapitalizedArgumentDescriptions(doc *ast.Document, schemaString string) []models.DescriptionError {
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

					err := rules.ReportUncapitalizedDescription(
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

func UncapitalizedDescriptions(doc *ast.Document, schemaString string) []models.DescriptionError {
	errors := make([]models.DescriptionError, 0, pkgrules.DefaultErrorCapacity)
	errors = append(errors, uncapitalizedTypeDescriptions(doc, schemaString)...)
	errors = append(errors, uncapitalizedFieldDescriptions(doc, schemaString)...)
	errors = append(errors, uncapitalizedEnumValueDescriptions(doc, schemaString)...)
	errors = append(errors, uncapitalizedArgumentDescriptions(doc, schemaString)...)

	return errors
}

func UnsortedTypeFields(doc *ast.Document, schemaString string) []models.DescriptionError {
	var errors []models.DescriptionError

	for _, obj := range doc.ObjectTypeDefinitions {
		typeName := doc.Input.ByteSliceString(obj.Name)

		err := rules.UnsortedFields(
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

func UnsortedInterfaceFields(doc *ast.Document, schemaString string) []models.DescriptionError {
	var errors []models.DescriptionError

	for _, iface := range doc.InterfaceTypeDefinitions {
		ifaceName := doc.Input.ByteSliceString(iface.Name)

		err := rules.UnsortedFields(
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

	dataTypesValid, dataTypeErrorLines, enumDescErrors := s.ValidateDataTypes(
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

		if !pkgrules.IsSuppressed(schemaFile, enumErr.LineNum, modelsLinterConfig, rule, "") {
			allErrors = append(allErrors, enumErr)
			unsuppressedDataTypeErrors++
		}
	}

	if !dataTypesValid {
		for _, lineNum := range dataTypeErrorLines {
			lineContent := rules.GetLineContent(schemaString, lineNum)

			var message string
			if strings.Contains(lineContent, "enum") {
				message = "suspicious-enum-value: Type validation failed"
			} else {
				message = "defined-types-are-used: Type is defined but not used"
			}

			if !pkgrules.IsSuppressed(schemaFile, lineNum, modelsLinterConfig, strings.Split(message, ":")[0], "") {
				allErrors = append(allErrors, models.DescriptionError{
					FilePath:    schemaFile,
					LineNum:     lineNum,
					Message:     message,
					LineContent: lineContent,
				})
				unsuppressedDataTypeErrors++
			}
		}
	}

	return unsuppressedDataTypeErrors, allErrors
}

func (s Store) readAndValidateSchemaFile(schemaFile string) (string, bool) {
	schemaString, ok := readSchemaFile(schemaFile)

	return schemaString, ok
}

func (s Store) ParseAndFilterSchema(
	schemaString string,
) (string, ast.Document, operationreport.Report) {
	filteredSchema := FilterSchemaComments(schemaString)
	doc, parseReport := astparser.ParseGraphqlDocumentString(schemaString)

	return filteredSchema, doc, parseReport
}
