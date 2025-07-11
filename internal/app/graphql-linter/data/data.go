package data

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/schubergphilis/mcvs-golang-project-root/pkg/projectroot"
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
	StrictMode         bool `yaml:"strict_mode"`
	ValidateFederation bool `yaml:"validate_federation"`
	CheckDescriptions  bool `yaml:"check_descriptions"`
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
	fmt.Println(projectRoot)

	configPath := filepath.Join(projectRoot, ".graphql-linter.yml")
	var configErr error
	linterConfig, configErr := s.loadConfig(configPath)
	if configErr != nil {
		fmt.Printf("Warning: %v\n", configErr)
	}
	fmt.Println(linterConfig)

	targetPath := projectRoot
	schemaFiles := findGraphQLFiles(targetPath)
	if len(schemaFiles) == 0 {
		fmt.Printf("no GraphQL schema files found in directory: %s", targetPath)
		os.Exit(1)
	}
	if s.Verbose {
		fmt.Printf("found %d GraphQL schema files:\n", len(schemaFiles))
		for _, file := range schemaFiles {
			fmt.Printf("  - %s\n", file)
		}
		fmt.Println()
	}

	// totalErrors := 0
	for _, schemaFile := range schemaFiles {
		if s.Verbose {
			fmt.Printf("=== Linting %s ===\n", schemaFile)
		}
		// if !lintSchemaFile(schemaFile) {
		// 	totalErrors++
		// }
		if s.Verbose {
			fmt.Println()
		}
	}

	return nil
}

func (s Store) loadConfig(configPath string) (*LinterConfig, error) {
	config := &LinterConfig{
		Settings: Settings{
			StrictMode:         false,
			ValidateFederation: true,
			CheckDescriptions:  true,
		},
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Printf("No config file found at %s, using defaults\n", configPath)
		return config, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	err = yaml.Unmarshal(data, config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	if s.Verbose {
		fmt.Printf("Loaded config with %d suppressions\n", len(config.Suppressions))
	}

	return config, nil
}

func findGraphQLFiles(rootPath string) []string {
	var files []string

	filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if strings.HasPrefix(info.Name(), ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() {
			dirName := strings.ToLower(info.Name())
			if dirName == "node_modules" || dirName == "vendor" || dirName == ".git" {
				return filepath.SkipDir
			}
		}

		if !info.IsDir() {
			ext := strings.ToLower(filepath.Ext(path))
			if ext == ".graphql" || ext == ".graphqls" {
				files = append(files, path)
			}
		}

		return nil
	})

	return files
}
