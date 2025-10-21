//go:build component

package component

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/schubergphilis/mcvs-golang-project-root/pkg/projectroot"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	setup()

	code := m.Run()

	teardown()

	os.Exit(code)
}

func TestVersion(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		t.Fatalf("failed to find project root: %v", err)
	}

	binaryPath := filepath.Join(projectRoot, "graphql-linter")
	cmd := exec.CommandContext(ctx, binaryPath, "--version")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to run graphql-linter --version: %v", err)
	}

	if !strings.Contains(string(output), "v4.5.6") {
		t.Errorf("expected version output to contain v4.5.6, got: %s", output)
	}
}

//nolint:paralleltest //must not run in parallel as it conflicts with TestSuppressAllScenarios.
func TestOutput(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		t.Fatalf("failed to find project root: %v", err)
	}

	mainPath := filepath.Join(projectRoot, "cmd", "graphql-linter", "main.go")
	targetPath := filepath.Join(projectRoot, "test", "testdata", "graphql", "base", "invalid")
	cmd := exec.CommandContext(ctx, "go", "run", mainPath, "-targetPath", targetPath)
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err == nil {
		t.Errorf("expected non-zero exit status, got nil error")
	}

	sections := parseSections(outputStr)

	t.Run("Error type summary block", func(t *testing.T) {
		required := []string{
			"Error type summary:",
			"arguments-have-descriptions: 3",
			"defined-types-are-used: 10",
			"deprecations-have-a-reason: 1",
			"descriptions-are-capitalized: 2",
			"enum-values-have-descriptions: 5",
			"enum-values-sorted-alphabetically: 3",
			"fields-are-camel-cased: 1",
			"fields-have-descriptions: 6",
			"input-object-fields-sorted-alphabetically: 2",
			"input-object-values-are-camel-cased: 1",
			"input-object-values-have-descriptions: 1",
			"interface-fields-sorted-alphabetically: 1",
			"invalid-graphql-schema: 1",
			"relay-connection-arguments-spec: 2",
			"relay-connection-types-spec: 1",
			"relay-page-info-spec: 12",
			"suspicious-enum-value: 1",
			"type-fields-sorted-alphabetically: 9",
			"types-have-descriptions: 5",
		}
		checkRequiredSubstrings(t, "error type summary", sections["errorTypeSummary"], required)
	})

	t.Run("Summary block", func(t *testing.T) {
		required := []string{
			"linting summary",
			"passedFiles=0",
			"percentPassed=\"0.00%\"",
			"totalFiles=20",
			"files with at least one error",
			"filesWithAtLeastOneError=20",
			"percentage=\"100.00%\"",
			"totalErrors: 69",
			"exit status 1",
		}
		allLines := sections["all"]

		summaryBlock := extractSummaryBlock(allLines)
		if summaryBlock == nil {
			summaryBlock = append([]string{}, sections["summary"]...)
			summaryBlock = append(summaryBlock, sections["errors"]...)
		}

		checkRequiredSubstrings(t, "summary", summaryBlock, required)
	})

	t.Run("Error file lines format", func(t *testing.T) {
		lines := sections["errors"]
		for _, line := range lines {
			idx := strings.Index(line, ".graphql:")
			if idx > 0 {
				suffix := line[idx+len(".graphql:"):]

				if len(suffix) == 0 || suffix[0] < '0' || suffix[0] > '9' {
					t.Errorf("error line does not have valid file:line format: %q", line)
				}
			}
		}
	})
}

//nolint:paralleltest //must not run in parallel as it conflicts with TestOutput.
func TestSuppressAllScenarios(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		t.Fatalf("failed to find project root: %v", err)
	}

	targetPath := filepath.Join(projectRoot, "test", "testdata", "graphql", "base", "invalid")
	yamlPath := filepath.Join(projectRoot, ".graphql-linter.yml")

	suppressions := generateSuppressions()

	err = writeSuppressionsYAML(suppressions, yamlPath)
	if err != nil {
		t.Fatalf("failed to write .graphql-linter.yml: %v", err)
	}

	defer func() {
		err := os.Remove(yamlPath)
		require.NoError(t, err, "failed to remove .graphql-linter.yml")
	}()

	mainPath := filepath.Join(projectRoot, "cmd", "graphql-linter", "main.go")
	cmd := exec.CommandContext(ctx, "go", "run", mainPath, "-targetPath", targetPath)
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err != nil {
		t.Fatalf("expected zero exit status, got error: %v\nOutput:\n%s", err, outputStr)
	}

	if strings.Contains(outputStr, "Error type summary:") {
		t.Errorf("expected no error type summary, but found one. Output:\n%s", outputStr)
	}

	if strings.Contains(outputStr, "files with at least one error") {
		t.Errorf("expected no files with errors, but found some. Output:\n%s", outputStr)
	}

	if strings.Contains(outputStr, "totalErrors:") {
		t.Errorf("expected totalErrors to be zero, but found errors. Output:\n%s", outputStr)
	}

	if !strings.Contains(outputStr, "passedFiles=") ||
		!strings.Contains(outputStr, "percentPassed=\"100.00%\"") {
		t.Errorf("expected all files to pass, but did not. Output:\n%s", outputStr)
	}
}

//nolint:paralleltest //must not run in parallel as it conflicts with TestOutput.
func TestSuppressTwoScenarios(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		t.Fatalf("failed to find project root: %v", err)
	}

	targetPath := filepath.Join(projectRoot, "test", "testdata", "graphql", "base", "invalid")
	yamlPath := filepath.Join(projectRoot, ".graphql-linter.yml")

	suppressions := []SuppressionEntry{
		{
			File:   "test/testdata/graphql/base/invalid/01-arguments-have-descriptions.graphql",
			Line:   1,
			Rule:   "relay-page-info-spec",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/08-fields-are-camel-cased.graphql",
			Line:   9,
			Rule:   "type-fields-sorted-alphabetically",
			Value:  "",
			Reason: "suppress for test",
		},
	}

	err = writeSuppressionsYAML(suppressions, yamlPath)
	if err != nil {
		t.Fatalf("failed to write .graphql-linter.yml: %v", err)
	}

	defer func() {
		err := os.Remove(yamlPath)
		require.NoError(t, err, "failed to remove .graphql-linter.yml")
	}()

	mainPath := filepath.Join(projectRoot, "cmd", "graphql-linter", "main.go")
	cmd := exec.CommandContext(ctx, "go", "run", mainPath, "-targetPath", targetPath)
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err == nil {
		t.Errorf("expected non-zero exit status since not all errors are suppressed, got nil error")
	}

	if !strings.Contains(outputStr, "Error type summary:") {
		t.Errorf("expected error type summary since not all errors are suppressed. Output:\n%s", outputStr)
	}

	if !strings.Contains(outputStr, "totalErrors:") {
		t.Errorf("expected some totalErrors since not all errors are suppressed. Output:\n%s", outputStr)
	}

	sections := parseSections(outputStr)

	t.Run("Summary block", func(t *testing.T) {
		required := []string{
			"linting summary",
			"totalFiles=20",
			"files with at least one error",
			"totalErrors: 67",
			"exit status 1",
		}
		allLines := sections["all"]

		summaryBlock := extractSummaryBlock(allLines)
		if summaryBlock == nil {
			summaryBlock = append([]string{}, sections["summary"]...)
			summaryBlock = append(summaryBlock, sections["errors"]...)
		}

		checkRequiredSubstrings(t, "summary", summaryBlock, required)
	})
}
