package application

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/application/report"
	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/data"
	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/data/base/models"
	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/data/base/rules"
	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/data/federation"
	federation_rules "github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/data/federation/rules"
	pkg_rules "github.com/schubergphilis/graphql-linter/internal/pkg/rules"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/ast"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/astparser"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/operationreport"
)

// ErrLintingFailed signals that at least one schema file contains lint errors.
// It travels up to main, which turns it into a non zero exit code.
var ErrLintingFailed = errors.New("linting failed")

const (
	linesAfterContext  = 3
	linesBeforeContext = 2
)

// readBuildInfo is a variable so tests can replace it.
var readBuildInfo = debug.ReadBuildInfo

type Execute struct {
	ConfigPath    string
	TargetPath    string
	VersionString string
}

func NewExecute(configPath, targetPath, versionString string) Execute {
	return Execute{
		ConfigPath:    configPath,
		TargetPath:    targetPath,
		VersionString: versionString,
	}
}

func (e Execute) Run() error {
	dataStore := data.NewStore(e.ConfigPath, e.TargetPath)

	linterConfig, err := dataStore.LoadConfig()
	if err != nil {
		return fmt.Errorf("unable to load config: %w", err)
	}

	slog.Debug(fmt.Sprintf("linter config: %v", linterConfig))
	dataStore.LinterConfig = linterConfig

	schemaFiles, err := e.FindAndLogGraphQLSchemaFiles()
	if err != nil {
		return fmt.Errorf("schema file discovery failed: %w", err)
	}

	if linterConfig.Settings.ValidateFederation {
		err = validateFederation(schemaFiles)
		if err != nil {
			return err
		}
	}

	totalErrors, errorFilesCount, dataDescriptionError := e.lintSchemaFiles(
		linterConfig,
		schemaFiles,
	)

	if report.Print(
		schemaFiles,
		totalErrors,
		len(schemaFiles)-errorFilesCount,
		dataDescriptionError,
	) {
		return ErrLintingFailed
	}

	return nil
}

func validateFederation(schemaFiles []string) error {
	for _, schemaFile := range schemaFiles {
		schemaBytes, err := os.ReadFile(schemaFile)
		if err != nil {
			return fmt.Errorf("failed to read schema file: %w", err)
		}

		filteredSchema := data.FilterSchemaComments(string(schemaBytes))
		if !federation.ValidateFederationSchema(filteredSchema) {
			return fmt.Errorf("federation validation failed for: %s", schemaFile)
		}
	}

	return nil
}

func (e Execute) Version() string {
	if e.VersionString != "" {
		return e.VersionString
	}

	if info, ok := readBuildInfo(); ok {
		return info.Main.Version
	}

	return "(unknown)"
}

func (e Execute) FindAndLogGraphQLSchemaFiles() ([]string, error) {
	if e.TargetPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to determine working directory: %w", err)
		}

		e.TargetPath = cwd
	}

	schemaFiles, err := findGraphQLFiles(e.TargetPath)
	if err != nil {
		return nil, fmt.Errorf("unable to find graphql files: %w", err)
	}

	if len(schemaFiles) == 0 {
		return nil, fmt.Errorf("no GraphQL schema files found in directory: %s", e.TargetPath)
	}

	slog.Debug(fmt.Sprintf("found %d GraphQL schema files:", len(schemaFiles)))

	for _, file := range schemaFiles {
		slog.Debug("  - " + file)
	}

	return schemaFiles, nil
}

func findGraphQLFiles(rootPath string) ([]string, error) {
	var files []string

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if path != rootPath && isIgnoredDir(info) {
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

func isIgnoredDir(info os.FileInfo) bool {
	if !info.IsDir() {
		return false
	}

	name := strings.ToLower(info.Name())

	return strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor"
}

func isGraphQLFile(info os.FileInfo) bool {
	if info.IsDir() {
		return false
	}

	ext := strings.ToLower(filepath.Ext(info.Name()))

	return ext == ".graphql" || ext == ".graphqls"
}

func (e Execute) lintDescriptions(
	doc *ast.Document,
	modelsLinterConfig *models.LinterConfig,
	schemaString string,
	schemaPath string,
) ([]models.DescriptionError, bool) {
	rule := rules.Rule{}
	dataStore := data.NewStore(e.ConfigPath, e.TargetPath)
	descriptionErrors := make([]models.DescriptionError, 0, pkg_rules.DefaultErrorCapacity)
	hasUnsuppressedDeprecationReasonError := false

	descriptionErrors, hasUnsuppressedDeprecationReasonError = e.collectDescriptionErrors(
		doc,
		&dataStore,
		modelsLinterConfig,
		schemaString,
		schemaPath,
		descriptionErrors,
		hasUnsuppressedDeprecationReasonError,
	)

	enumSortErrors := rule.EnumValuesSortedAlphabetically(
		doc,
		modelsLinterConfig,
		schemaString,
		schemaPath,
	)
	descriptionErrors = append(descriptionErrors, enumSortErrors...)

	return sortDescriptionErrors(descriptionErrors), hasUnsuppressedDeprecationReasonError
}

func (e Execute) collectDescriptionErrors(
	doc *ast.Document,
	dataStore *data.Store,
	modelsLinterConfig *models.LinterConfig,
	schemaString string,
	schemaPath string,
	descriptionErrors []models.DescriptionError,
	hasUnsuppressedDeprecationReasonError bool,
) ([]models.DescriptionError, bool) {
	rule := rules.Rule{}

	helpers := []func(*ast.Document, string) []models.DescriptionError{
		rule.FieldsAreCamelCased,
		rule.InputObjectFieldsSortedAlphabetically,
		rule.InputObjectValuesCamelCased,
		rule.MissingArgumentDescriptions,
		rule.MissingDeprecationReasons,
		rule.MissingEnumValueDescriptions,
		rule.MissingFieldDescriptions,
		rule.MissingInputObjectValueDescriptions,
		rule.MissingQueryRootType,
		rule.MissingTypeDescriptions,
		rule.RelayConnectionArgumentsSpec,
		rule.RelayConnectionTypesSpec,
		rule.RelayPageInfoSpec,
		rule.TypesAreCapitalized,
		dataStore.UncapitalizedDescriptions,
		dataStore.UnsortedInterfaceFields,
		dataStore.UnsortedTypeFields,
		rule.UnusedTypes,
	}
	for _, helper := range helpers {
		errList := helper(doc, schemaString)
		for _, err := range errList {
			descriptionErrors = append(descriptionErrors, err)
			if strings.Contains(err.Message, "deprecations-have-a-reason") {
				rule := err.Message
				if idx := strings.Index(rule, ":"); idx != -1 {
					rule = rule[:idx]
				}

				if !pkg_rules.IsSuppressed(schemaPath, err.LineNum, modelsLinterConfig, rule, err.Value) {
					hasUnsuppressedDeprecationReasonError = true
				}
			}
		}
	}

	return descriptionErrors, hasUnsuppressedDeprecationReasonError
}

func sortDescriptionErrors(errors []models.DescriptionError) []models.DescriptionError {
	return errors
}

func getUnsuppressedDescriptionErrors(
	descriptionErrors []models.DescriptionError,
	modelsLinterConfig *models.LinterConfig,
	schemaFile string,
) []models.DescriptionError {
	unsuppressed := make([]models.DescriptionError, 0, len(descriptionErrors))
	for _, err := range descriptionErrors {
		rule := err.Message
		if idx := strings.Index(rule, ":"); idx != -1 {
			rule = rule[:idx]
		}

		if !modelsLinterConfig.Settings.CheckDescriptions && strings.HasSuffix(rule, "-have-descriptions") {
			continue
		}

		if !pkg_rules.IsSuppressed(schemaFile, err.LineNum, modelsLinterConfig, rule, err.Value) {
			unsuppressed = append(unsuppressed, err)
		}
	}

	return unsuppressed
}

func (e Execute) lintSchemaFiles(
	modelsLinterConfig *models.LinterConfig,
	schemaFiles []string,
) (int, int, []models.DescriptionError) {
	totalErrors := 0
	errorFilesCount := 0

	var allErrors []models.DescriptionError

	for _, schemaFile := range schemaFiles {
		errCount, fileErrCount, fileErrors := e.lintSingleSchemaFile(modelsLinterConfig, schemaFile)
		totalErrors += errCount
		errorFilesCount += fileErrCount

		allErrors = append(allErrors, fileErrors...)
	}

	return totalErrors, errorFilesCount, allErrors
}

func (e Execute) lintSingleSchemaFile(
	modelsLinterConfig *models.LinterConfig,
	schemaFile string,
) (
	int,
	int,
	[]models.DescriptionError,
) {
	slog.Debug(fmt.Sprintf("=== Linting %s ===", schemaFile))

	dataStore := data.NewStore(e.ConfigPath, e.TargetPath)

	schemaBytes, err := os.ReadFile(schemaFile)
	if err != nil {
		slog.Error("failed to read schema file", "error", err)

		return 1, 1, []models.DescriptionError{{
			FilePath:    schemaFile,
			LineNum:     0,
			Message:     "failed-to-read-schema-file: failed to read schema file",
			LineContent: "",
		}}
	}

	schemaString := string(schemaBytes)
	_, doc, parseReport := dataStore.ParseAndFilterSchema(schemaString)
	LogSchemaParseErrors(schemaString, &parseReport)

	totalErrors, errorFilesCount, allErrors := e.collectLintErrors(
		&doc,
		modelsLinterConfig,
		schemaString,
		schemaFile,
		&dataStore,
	)

	return totalErrors, errorFilesCount, allErrors
}

func LogSchemaParseErrors(
	schemaString string,
	parseReport *operationreport.Report,
) {
	if !parseReport.HasErrors() {
		return
	}

	slog.Error(fmt.Sprintf(
		"Failed to parse schema - found %d errors:",
		len(parseReport.InternalErrors)+len(parseReport.ExternalErrors),
	))

	report.InternalErrors(parseReport)
	report.ExternalErrors(schemaString, parseReport, linesBeforeContext, linesAfterContext)
}

func parseGraphQLDocument(schemaContent string) *ast.Document {
	doc, _ := astparser.ParseGraphqlDocumentString(schemaContent)

	return &doc
}

func (e Execute) collectLintErrors(
	doc *ast.Document,
	modelsLinterConfig *models.LinterConfig,
	schemaString string,
	schemaFile string,
	dataStore *data.Store,
) (int, int, []models.DescriptionError) {
	descriptionErrors, hasUnsuppressedDeprecationReasonError := e.lintDescriptions(
		doc,
		modelsLinterConfig,
		schemaString,
		schemaFile,
	)
	unsuppressedDescriptionErrors := getUnsuppressedDescriptionErrors(
		descriptionErrors,
		modelsLinterConfig,
		schemaFile,
	)
	unsuppressedDataTypeErrors, dataTypeErrors := dataStore.CollectUnsuppressedDataTypeErrors(
		doc,
		modelsLinterConfig,
		schemaString,
		schemaFile,
	)
	allErrors := append([]models.DescriptionError{}, dataTypeErrors...)
	unsuppressedDirectiveOrFederationError := modelsLinterConfig.Settings.ValidateFederation &&
		!federation_rules.ValidateDirectiveNames(doc)

	totalErrors, errorFilesCount := report.SummarizeLintResults(
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
