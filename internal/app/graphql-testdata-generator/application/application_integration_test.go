//go:build integration

package application

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/schubergphilis/graphql-linter/internal/pkg/constants"
	"github.com/schubergphilis/mcvs-golang-project-root/pkg/projectroot"
	"github.com/stretchr/testify/require"
)

const graphqlSchemaLinterVersion = "3.0.1"

func validateErrorName(t *testing.T, fileName string) string {
	t.Helper()

	parts := strings.SplitN(fileName, "-", 2)
	if len(parts) < 2 {
		t.Errorf("Filename does not contain error name: %s", fileName)
	}

	errorName := strings.TrimSuffix(parts[1], ".graphql")

	for _, r := range errorName {
		if r >= '0' && r <= '9' {
			t.Errorf(
				"Error name contains digits (possible typo): %s: extracted error name: %s",
				fileName,
				errorName,
			)
		}
	}

	return errorName
}

func parseLinterOutput(t *testing.T, output []byte, fileName string) []string {
	t.Helper()

	var result struct {
		Errors []struct {
			Rule string `json:"rule"`
			File string `json:"file,omitempty"`
		} `json:"errors"`
	}

	err := json.Unmarshal(output, &result)
	require.NoError(t, err, "Failed to parse linter JSON output for "+fileName)

	rules := make([]string, 0, len(result.Errors))
	for _, e := range result.Errors {
		rules = append(rules, e.Rule)
	}

	return rules
}

func checkLinterVersion(t *testing.T) {
	t.Helper()

	cmd := exec.Command("graphql-schema-linter", "--version")
	output, err := cmd.CombinedOutput()
	version := strings.TrimSpace(string(output))

	if err != nil || version != graphqlSchemaLinterVersion {
		installCmd := exec.Command(
			"npm",
			"install",
			"-g",
			"graphql-schema-linter@"+graphqlSchemaLinterVersion,
			"graphql",
		)

		installOut, installErr := installCmd.CombinedOutput()
		if installErr != nil {
			t.Fatalf(
				"Failed to install graphql-schema-linter@%s: %v\n%s",
				graphqlSchemaLinterVersion,
				installErr,
				string(installOut),
			)
		}

		cmd = exec.Command("graphql-schema-linter", "--version")
		output, err = cmd.CombinedOutput()
		version = strings.TrimSpace(string(output))

		if err != nil || version != graphqlSchemaLinterVersion {
			t.Fatalf(
				"graphql-schema-linter version is %s, but %s is required after install.\n%s",
				version,
				graphqlSchemaLinterVersion,
				string(output),
			)
		}
	}
}

func TestInvalidSchemas(t *testing.T) {
	t.Parallel()
	checkLinterVersion(t)

	projectRoot, err := projectroot.FindProjectRoot()
	require.NoError(t, err, "failed to determine project root")

	baseDir := filepath.Join(projectRoot, constants.TestdataGraphqlDir, "base/invalid")
	files, err := os.ReadDir(baseDir)
	require.NoError(t, err, "failed to read directory")

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".graphql") {
			continue
		}

		filePath := filepath.Join(baseDir, file.Name())
		cmd := exec.Command("graphql-schema-linter", "-f", "json", filePath)

		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Errorf("Expected error for file, but linter passed: %s", file.Name())

			continue
		}

		errorName := validateErrorName(t, file.Name())

		errorsToBeSkippedAsNotReportedByGraphqlSchemaLinter := []string{
			"suspicious-enum-value",
		}

		if slices.Contains(errorsToBeSkippedAsNotReportedByGraphqlSchemaLinter, errorName) {
			return
		}

		rules := parseLinterOutput(t, output, file.Name())

		found := slices.Contains(rules, errorName)

		if !found {
			t.Fatalf(
				"File: expected error rule not found in output: %s: expected error rule '%s' in output, got rules: %s",
				file.Name(),
				errorName,
				strings.Join(rules, ", "),
			)
		}
	}
}
