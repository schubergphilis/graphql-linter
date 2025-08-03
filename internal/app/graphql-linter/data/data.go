package data

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/data/fileutil"
	"github.com/schubergphilis/mcvs-golang-project-root/pkg/projectroot"
	log "github.com/sirupsen/logrus"
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

	schemaFiles, err := fileutil.FindGraphQLFiles(s.TargetPath)
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

	// TODO: Move to report package
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
	passedFiles := 0

	var allErrors []DescriptionError

	// Load config if not already loaded
	if s.LinterConfig == nil {
		config, err := s.LoadConfig()
		if err != nil {
			log.Debugf("Failed to load config: %v", err)
		} else {
			s.LinterConfig = config
		}
	}

	for _, schemaFile := range schemaFiles {
		// Check if file exists
		if _, err := os.Stat(schemaFile); os.IsNotExist(err) {
			totalErrors++

			allErrors = append(allErrors, DescriptionError{
				FilePath:    schemaFile,
				LineNum:     1,
				Message:     "File does not exist",
				LineContent: "",
			})

			continue
		}

		// Read and validate schema file
		schemaContent, ok := s.ReadAndValidateSchemaFile(schemaFile)
		if !ok {
			totalErrors++

			allErrors = append(allErrors, DescriptionError{
				FilePath:    schemaFile,
				LineNum:     1,
				Message:     "Failed to read schema file",
				LineContent: "",
			})

			continue
		}

		// Skip empty files
		if strings.TrimSpace(schemaContent) == "" {
			passedFiles++
			continue
		}

		log.Debugf("Linting schema file: %s", schemaFile)

		// Basic validation - check if content looks like GraphQL
		fileErrors := s.validateBasicGraphQLStructure(schemaFile, schemaContent)

		// Filter out suppressed errors
		var unsuppressedErrors []DescriptionError

		for _, err := range fileErrors {
			if !s.IsSuppressed(err.FilePath, err.LineNum, "invalid-graphql-schema", "") {
				unsuppressedErrors = append(unsuppressedErrors, err)
			}
		}

		if len(unsuppressedErrors) > 0 {
			totalErrors += len(unsuppressedErrors)
			allErrors = append(allErrors, unsuppressedErrors...)
		} else {
			passedFiles++
		}
	}

	// Calculate error files
	errorFilesCount := len(schemaFiles) - passedFiles

	return totalErrors, errorFilesCount, allErrors
}

// validateBasicGraphQLStructure performs basic validation on GraphQL schema content
func (s Store) validateBasicGraphQLStructure(filePath, content string) []DescriptionError {
	var errors []DescriptionError

	// Basic check: if it doesn't contain common GraphQL keywords, it might be invalid
	hasGraphQLKeywords := strings.Contains(content, "type") ||
		strings.Contains(content, "enum") ||
		strings.Contains(content, "input") ||
		strings.Contains(content, "interface") ||
		strings.Contains(content, "union") ||
		strings.Contains(content, "scalar")

	if !hasGraphQLKeywords {
		errors = append(errors, DescriptionError{
			FilePath:    filePath,
			LineNum:     1,
			Message:     "File does not appear to contain valid GraphQL schema definitions",
			LineContent: "",
		})

		return errors
	}

	// Basic linting rule: Check if Query type exists (most basic GraphQL requirement)
	if !strings.Contains(content, "type Query") {
		errors = append(errors, DescriptionError{
			FilePath:    filePath,
			LineNum:     1,
			Message:     "invalid-graphql-schema: Query root type is missing",
			LineContent: "",
		})
	}

	return errors
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

// ReadAndValidateSchemaFile reads and validates a schema file
func (s Store) ReadAndValidateSchemaFile(schemaFile string) (string, bool) {
	return fileutil.ReadSchemaFile(schemaFile)
}
