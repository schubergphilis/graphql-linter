package fileutil

import (
	"fmt"
	"os"
	"path/filepath"
)

func FindGraphQLFiles(rootPath string) ([]string, error) {
	var files []string

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if ShouldSkip(info) {
			if info.IsDir() {
				return filepath.SkipDir
			}

			return nil
		}

		if IsIgnoredDir(info) {
			return filepath.SkipDir
		}

		if IsGraphQLFile(info) {
			files = append(files, path)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("unable to walk dir for graphql files: %w", err)
	}

	return files, nil
}

func ShouldSkip(info os.FileInfo) bool {
	return len(info.Name()) > 0 && info.Name()[0] == '.'
}

func IsIgnoredDir(info os.FileInfo) bool {
	if !info.IsDir() {
		return false
	}

	switch info.Name() {
	case "node_modules", "vendor", ".git":
		return true
	default:
		return false
	}
}

func IsGraphQLFile(info os.FileInfo) bool {
	if info.IsDir() {
		return false
	}

	ext := filepath.Ext(info.Name())

	return ext == ".graphql" || ext == ".graphqls"
}

func ReadSchemaFile(schemaPath string) (string, bool) {
	schemaBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		return "", false
	}

	return string(schemaBytes), true
}
