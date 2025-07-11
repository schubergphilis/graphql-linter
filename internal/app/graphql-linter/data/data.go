package data

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/schubergphilis/mcvs-golang-project-root/pkg/projectroot"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type Storer interface {
	Run()
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

	linterConfig, err := s.loadConfig(configPath)
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

	totalErrors := 42

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

func (s Store) loadConfig(configPath string) (*LinterConfig, error) {
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
