package application

import (
	"fmt"
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
	"github.com/schubergphilis/mcvs-golang-project-root/pkg/projectroot"
	log "github.com/sirupsen/logrus"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/ast"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/astparser"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/operationreport"
)

const (
	linesAfterContext  = 3
	linesBeforeContext = 2
)

type Executor interface {
	Run() error
	Version()
	PrintReport(
		schemaFiles []string,
		totalErrors int,
		passedFiles int,
		allErrors []models.DescriptionError,
	)
}

type Debugger interface {
	ReadBuildInfo() (info *debug.BuildInfo, ok bool)
}

type Debug struct{}

type Execute struct {
	Debugger      Debugger
	TargetPath    string
	Verbose       bool
	VersionString string
}

func NewExecute(targetPath string, verbose bool, versionString string) (Execute, error) {
	execute := Execute{
		Debugger:      Debug{},
		TargetPath:    targetPath,
		Verbose:       verbose,
		VersionString: versionString,
	}

	return execute, nil
}

func (Debug) ReadBuildInfo() (*debug.BuildInfo, bool) {
	return debug.ReadBuildInfo()
}

func (e Execute) Run() error {
	dataStore, err := data.NewStore(e.TargetPath, e.Verbose)
	if err != nil {
		return fmt.Errorf("unable to load new store: %w", err)
	}

	linterConfig, err := dataStore.LoadConfig()
	if err != nil {
		return fmt.Errorf("unable to load config: %w", err)
	}

	log.Debugf("linter config: %v", linterConfig)
	dataStore.LinterConfig = linterConfig

	schemaFiles, err := e.FindAndLogGraphQLSchemaFiles()
	if err != nil {
		return fmt.Errorf("schema file discovery failed: %w", err)
	}

	for _, schemaFile := range schemaFiles {
		schemaString, ok := dataStore.ReadAndValidateSchemaFile(schemaFile)
		if !ok {
			return fmt.Errorf("failed to read schema file: %s", schemaFile)
		}

		filteredSchema := data.FilterSchemaComments(schemaString)
		if !federation.ValidateFederationSchema(filteredSchema) {
			return fmt.Errorf("federation validation failed for: %s", schemaFile)
		}
	}

	totalErrors, errorFilesCount, dataDescriptionError := e.lintSchemaFiles(
		linterConfig,
		schemaFiles,
	)

	report.Print(
		schemaFiles,
		totalErrors,
		len(schemaFiles)-errorFilesCount,
		dataDescriptionError,
	)

	return nil
}

func (e Execute) Version() string {
	if e.VersionString != "" {
		return e.VersionString
	}

	if info, ok := e.Debugger.ReadBuildInfo(); ok {
		return info.Main.Version
	}

	return "(unknown)"
}

func (e Execute) FindAndLogGraphQLSchemaFiles() ([]string, error) {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to determine project root: %w", err)
	}

	if e.TargetPath == "" {
		e.TargetPath = projectRoot
	}

	schemaFiles, err := findGraphQLFiles(e.TargetPath)
	if err != nil {
		return nil, fmt.Errorf("unable to find graphql files: %w", err)
	}

	if len(schemaFiles) == 0 {
		return nil, fmt.Errorf("no GraphQL schema files found in directory: %s", e.TargetPath)
	}

	if e.Verbose {
		log.Infof("found %d GraphQL schema files:", len(schemaFiles))

		for _, file := range schemaFiles {
			log.Infof("  - %s", file)
		}
	}

	return schemaFiles, nil
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

func lintDescriptions(
	doc *ast.Document,
	modelsLinterConfig *models.LinterConfig,
	schemaString string,
	schemaPath string,
) ([]models.DescriptionError, bool) {
	descriptionErrors := make([]models.DescriptionError, 0, pkg_rules.DefaultErrorCapacity)
	hasUnsuppressedDeprecationReasonError := false

	helpers := []func(*ast.Document, string) []models.DescriptionError{
		rules.FieldsAreCamelCased,
		rules.InputObjectFieldsSortedAlphabetically,
		rules.InputObjectValuesCamelCased,
		rules.MissingArgumentDescriptions,
		rules.MissingDeprecationReasons,
		rules.MissingEnumValueDescriptions,
		rules.MissingFieldDescriptions,
		rules.MissingInputObjectValueDescriptions,
		rules.MissingQueryRootType,
		rules.MissingTypeDescriptions,
		rules.RelayConnectionArgumentsSpec,
		rules.RelayConnectionTypesSpec,
		rules.RelayPageInfoSpec,
		rules.TypesAreCapitalized,
		data.UncapitalizedDescriptions,
		data.UnsortedInterfaceFields,
		data.UnsortedTypeFields,
		rules.UnusedTypes,
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

				if !pkg_rules.IsSuppressed(schemaPath, err.LineNum, modelsLinterConfig, rule, "") {
					hasUnsuppressedDeprecationReasonError = true
				}
			}
		}
	}

	enumSortErrors := rules.EnumValuesSortedAlphabetically(
		doc,
		modelsLinterConfig,
		schemaString,
		schemaPath,
	)

	descriptionErrors = append(descriptionErrors, enumSortErrors...)

	return sortDescriptionErrors(descriptionErrors), hasUnsuppressedDeprecationReasonError
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

		if !pkg_rules.IsSuppressed(schemaFile, err.LineNum, modelsLinterConfig, rule, "") {
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
	if e.Verbose {
		log.Infof("=== Linting %s ===", schemaFile)
	}

	dataStore, err := data.NewStore(e.TargetPath, e.Verbose)
	if err != nil {
		log.Errorf("unable to load new store: %v", err)
	}

	schemaString, ok := dataStore.ReadAndValidateSchemaFile(schemaFile)
	if !ok {
		return 1, 1, []models.DescriptionError{{
			FilePath:    schemaFile,
			LineNum:     0,
			Message:     "failed-to-read-schema-file: failed to read schema file",
			LineContent: "",
		}}
	}

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

	log.Errorf("Failed to parse schema - found %d errors:\n",
		len(parseReport.InternalErrors)+len(parseReport.ExternalErrors))

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
	descriptionErrors, hasUnsuppressedDeprecationReasonError := lintDescriptions(
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
	unsuppressedDirectiveOrFederationError := !federation_rules.ValidateDirectiveNames(doc)

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
