//go:build component

package component

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/schubergphilis/mcvs-golang-project-root/pkg/projectroot"
	log "github.com/sirupsen/logrus"
)

func TestMain(m *testing.M) {
	setup()

	code := m.Run()

	teardown()

	os.Exit(code)
}

func setup() {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		log.WithError(err).Fatal("failed to find project root")
	}

	mainPath := filepath.Join(projectRoot, "cmd", "graphql-linter", "main.go")

	outputPath := filepath.Join(projectRoot, "graphql-linter")

	cmd := exec.Command(
		"go",
		"build",
		"-ldflags=-X 'main.Version=v4.5.6'",
		"-o",
		outputPath,
		mainPath,
	)
	cmd.Stdout = os.Stdout

	var stderrBuf strings.Builder

	cmd.Stderr = &stderrBuf

	err = cmd.Run()
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"command": cmd.String(),
			"stderr":  stderrBuf.String(),
			"stdout":  "os.Stdout",
		}).Fatal("failed to build graphql-linter")
	}
}

func teardown() {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		log.WithError(err).Fatal("failed to find project root during teardown")
	}

	binaryPath := filepath.Join(projectRoot, "graphql-linter")

	err = os.Remove(binaryPath)
	if err != nil && !os.IsNotExist(err) {
		log.WithError(err).
			WithField("binaryPath", binaryPath).
			Fatal("failed to remove built binary during teardown")
	}
}

func TestVersion(t *testing.T) {
	t.Parallel()

	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		t.Fatalf("failed to find project root: %v", err)
	}

	binaryPath := filepath.Join(projectRoot, "graphql-linter")

	cmd := exec.Command(binaryPath, "--version")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to run graphql-linter --version: %v", err)
	}

	if !strings.Contains(string(output), "v4.5.6") {
		t.Errorf("expected version output to contain v4.5.6, got: %s", output)
	}
}

func extractSummaryBlock(allLines []string) []string {
	startIdx := -1

	for i, l := range allLines {
		if startIdx == -1 && strings.Contains(l, "Error type summary:") {
			startIdx = i

			break
		}
	}

	if startIdx != -1 {
		return allLines[startIdx:]
	}

	return nil
}

func parseSections(outputStr string) map[string][]string {
	sections := map[string][]string{
		"all":              strings.Split(outputStr, "\n"),
		"summary":          {},
		"errorTypeSummary": {},
		"errors":           {},
	}

	inErrorTypeSummary := false

	for _, line := range sections["all"] {
		if strings.Contains(line, "Error type summary:") {
			inErrorTypeSummary = true

			sections["errorTypeSummary"] = append(sections["errorTypeSummary"], line)

			continue
		}

		if inErrorTypeSummary {
			if strings.TrimSpace(line) == "" {
				inErrorTypeSummary = false

				continue
			}

			sections["errorTypeSummary"] = append(sections["errorTypeSummary"], line)

			continue
		}

		if strings.Contains(line, "summary") ||
			strings.Contains(line, "filesWithAtLeastOneError") ||
			strings.Contains(line, "passedFiles=") {
			sections["summary"] = append(sections["summary"], line)
		}

		if strings.Contains(line, "level=error") || strings.Contains(line, "level=fatal") {
			sections["errors"] = append(sections["errors"], line)
		}
	}

	return sections
}

func checkRequiredSubstrings(t *testing.T, blockName string, block []string, required []string) {
	t.Helper()

	for _, substr := range required {
		found := false

		for _, line := range block {
			if strings.Contains(line, substr) {
				found = true

				break
			}
		}

		if !found {
			t.Errorf(
				"required %s substring not found: %q\nBlock:\n%s",
				blockName,
				substr,
				strings.Join(block, "\n"),
			)
		}
	}
}

//nolint:paralleltest //must not run in parallel as it conflicts with TestSuppressAllScenarios.
func TestOutput(t *testing.T) {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		t.Fatalf("failed to find project root: %v", err)
	}

	mainPath := filepath.Join(projectRoot, "cmd", "graphql-linter", "main.go")
	targetPath := filepath.Join(projectRoot, "test", "testdata", "graphql", "base", "invalid")
	cmd := exec.Command("go", "run", mainPath, "-targetPath", targetPath)
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
			"defined-types-are-used: 11",
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
			"totalErrors: 70",
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
			// Look for lines with a file path and line number: .../somefile.graphql:<number>:
			idx := strings.Index(line, ".graphql:")
			if idx > 0 {
				suffix := line[idx+len(".graphql:"):]
				// Check if it starts with a number
				if len(suffix) == 0 || suffix[0] < '0' || suffix[0] > '9' {
					t.Errorf("error line does not have valid file:line format: %q", line)
				}
			}
		}
	})
}

type SuppressionEntry struct {
	File   string
	Line   int
	Rule   string
	Value  string
	Reason string
}

//nolint:maintidx // generateSuppressions is a data function and not logic-heavy.
func generateSuppressions() []SuppressionEntry {
	return []SuppressionEntry{
		{
			File:   "test/testdata/graphql/base/invalid/01-arguments-have-descriptions.graphql",
			Line:   1,
			Rule:   "relay-page-info-spec",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/01-arguments-have-descriptions.graphql",
			Line:   4,
			Rule:   "arguments-have-descriptions",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/02-defined-types-are-used.graphql",
			Line:   1,
			Rule:   "relay-page-info-spec",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/02-defined-types-are-used.graphql",
			Line:   2,
			Rule:   "defined-types-are-used",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/03-deprecations-have-a-reason.graphql",
			Line:   1,
			Rule:   "relay-page-info-spec",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/03-deprecations-have-a-reason.graphql",
			Line:   4,
			Rule:   "deprecations-have-a-reason",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/03-deprecations-have-a-reason.graphql",
			Line:   2,
			Rule:   "enum-values-sorted-alphabetically",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/04-descriptions-are-capitalized.graphql",
			Line:   1,
			Rule:   "relay-page-info-spec",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/04-descriptions-are-capitalized.graphql",
			Line:   2,
			Rule:   "descriptions-are-capitalized",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/04-descriptions-are-capitalized.graphql",
			Line:   4,
			Rule:   "descriptions-are-capitalized",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/05-enum-values-all-caps.graphql",
			Line:   9,
			Rule:   "type-fields-sorted-alphabetically",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/05-enum-values-all-caps.graphql",
			Line:   3,
			Rule:   "enum-values-have-descriptions",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/05-enum-values-all-caps.graphql",
			Line:   9,
			Rule:   "defined-types-are-used",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/06-enum-values-have-descriptions.graphql",
			Line:   1,
			Rule:   "relay-page-info-spec",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/06-enum-values-have-descriptions.graphql",
			Line:   3,
			Rule:   "enum-values-have-descriptions",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/06-enum-values-have-descriptions.graphql",
			Line:   2,
			Rule:   "enum-values-sorted-alphabetically",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/07-enum-values-sorted-alphabetically.graphql",
			Line:   12,
			Rule:   "type-fields-sorted-alphabetically",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/07-enum-values-sorted-alphabetically.graphql",
			Line:   12,
			Rule:   "defined-types-are-used",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/07-enum-values-sorted-alphabetically.graphql",
			Line:   2,
			Rule:   "enum-values-sorted-alphabetically",
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
		{
			File:   "test/testdata/graphql/base/invalid/08-fields-are-camel-cased.graphql",
			Line:   3,
			Rule:   "fields-are-camel-cased",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/08-fields-are-camel-cased.graphql",
			Line:   9,
			Rule:   "defined-types-are-used",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/08-fields-are-camel-cased.graphql",
			Line:   3,
			Rule:   "fields-have-descriptions",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/09-fields-have-descriptions.graphql",
			Line:   9,
			Rule:   "type-fields-sorted-alphabetically",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/09-fields-have-descriptions.graphql",
			Line:   9,
			Rule:   "defined-types-are-used",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/09-fields-have-descriptions.graphql",
			Line:   5,
			Rule:   "fields-have-descriptions",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/10-input-object-fields-sorted-alphabetically.graphql",
			Line:   1,
			Rule:   "relay-page-info-spec",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/10-input-object-fields-sorted-alphabetically.graphql",
			Line:   0,
			Rule:   "input-object-fields-sorted-alphabetically",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/10-input-object-fields-sorted-alphabetically.graphql",
			Line:   12,
			Rule:   "arguments-have-descriptions",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/11-input-object-values-are-camel-cased.graphql",
			Line:   1,
			Rule:   "relay-page-info-spec",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/11-input-object-values-are-camel-cased.graphql",
			Line:   4,
			Rule:   "input-object-values-are-camel-cased",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/11-input-object-values-are-camel-cased.graphql",
			Line:   2,
			Rule:   "defined-types-are-used",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/12-input-object-values-have-descriptions.graphql",
			Line:   9,
			Rule:   "type-fields-sorted-alphabetically",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/12-input-object-values-have-descriptions.graphql",
			Line:   0,
			Rule:   "input-object-fields-sorted-alphabetically",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/12-input-object-values-have-descriptions.graphql",
			Line:   9,
			Rule:   "defined-types-are-used",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/12-input-object-values-have-descriptions.graphql",
			Line:   2,
			Rule:   "defined-types-are-used",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/12-input-object-values-have-descriptions.graphql",
			Line:   5,
			Rule:   "input-object-values-have-descriptions",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/13-interface-fields-sorted-alphabetically.graphql",
			Line:   2,
			Rule:   "interface-fields-sorted-alphabetically",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/13-interface-fields-sorted-alphabetically.graphql",
			Line:   1,
			Rule:   "relay-page-info-spec",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/14-relay-connection-types-spec.graphql",
			Line:   1,
			Rule:   "relay-page-info-spec",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/14-relay-connection-types-spec.graphql",
			Line:   1,
			Rule:   "relay-connection-types-spec",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/14-relay-connection-types-spec.graphql",
			Line:   9,
			Rule:   "relay-connection-arguments-spec",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/14-relay-connection-types-spec.graphql",
			Line:   1,
			Rule:   "types-have-descriptions",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/15-relay-connection-arguments-spec.graphql",
			Line:   16,
			Rule:   "type-fields-sorted-alphabetically",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/15-relay-connection-arguments-spec.graphql",
			Line:   30,
			Rule:   "relay-connection-arguments-spec",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/16-type-fields-sorted-alphabetically.graphql",
			Line:   2,
			Rule:   "type-fields-sorted-alphabetically",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/16-type-fields-sorted-alphabetically.graphql",
			Line:   1,
			Rule:   "relay-page-info-spec",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/17-types-are-capitalized.graphql",
			Line:   1,
			Rule:   "types-are-capitalized",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/17-types-are-capitalized.graphql",
			Line:   9,
			Rule:   "type-fields-sorted-alphabetically",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/17-types-are-capitalized.graphql",
			Line:   9,
			Rule:   "defined-types-are-used",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/17-types-are-capitalized.graphql",
			Line:   1,
			Rule:   "types-have-descriptions",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/18-types-have-descriptions.graphql",
			Line:   1,
			Rule:   "types-are-capitalized",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/18-types-have-descriptions.graphql",
			Line:   9,
			Rule:   "type-fields-sorted-alphabetically",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/18-types-have-descriptions.graphql",
			Line:   9,
			Rule:   "defined-types-are-used",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/18-types-have-descriptions.graphql",
			Line:   1,
			Rule:   "types-have-descriptions",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/19-invalid-graphql-schema.graphql",
			Line:   1,
			Rule:   "invalid-graphql-schema",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/19-invalid-graphql-schema.graphql",
			Line:   1,
			Rule:   "relay-page-info-spec",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/20-suspicious-enum-value.graphql",
			Line:   13,
			Rule:   "suspicious-enum-value",
			Value:  "INACTIVE1",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/20-suspicious-enum-value.graphql",
			Line:   1,
			Rule:   "relay-page-info-spec",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/20-suspicious-enum-value.graphql",
			Line:   12,
			Rule:   "enum-values-have-descriptions",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/20-suspicious-enum-value.graphql",
			Line:   13,
			Rule:   "enum-values-have-descriptions",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/20-suspicious-enum-value.graphql",
			Line:   14,
			Rule:   "enum-values-have-descriptions",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/20-suspicious-enum-value.graphql",
			Line:   1,
			Rule:   "types-have-descriptions",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/20-suspicious-enum-value.graphql",
			Line:   6,
			Rule:   "types-have-descriptions",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/20-suspicious-enum-value.graphql",
			Line:   0,
			Rule:   "fields-have-descriptions",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/20-suspicious-enum-value.graphql",
			Line:   2,
			Rule:   "fields-have-descriptions",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/20-suspicious-enum-value.graphql",
			Line:   3,
			Rule:   "fields-have-descriptions",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/20-suspicious-enum-value.graphql",
			Line:   8,
			Rule:   "fields-have-descriptions",
			Value:  "",
			Reason: "suppress for test",
		},
		{
			File:   "test/testdata/graphql/base/invalid/20-suspicious-enum-value.graphql",
			Line:   3,
			Rule:   "arguments-have-descriptions",
			Value:  "",
			Reason: "suppress for test",
		},
	}
}

func writeSuppressionsYAML(suppressions []SuppressionEntry, yamlPath string) error {
	var suppressionsYAML strings.Builder
	suppressionsYAML.WriteString("suppressions:\n")

	for _, suppression := range suppressions {
		suppressionsYAML.WriteString("  - rule: " + suppression.Rule + "\n")

		if suppression.File != "" {
			suppressionsYAML.WriteString("    file: " + suppression.File + "\n")
		}

		if suppression.Line != 0 {
			suppressionsYAML.WriteString(fmt.Sprintf("    line: %d\n", suppression.Line))
		}

		if suppression.Value != "" {
			suppressionsYAML.WriteString("    value: " + suppression.Value + "\n")
		}

		suppressionsYAML.WriteString("    reason: " + suppression.Reason + "\n")
	}

	err := os.WriteFile(yamlPath, []byte(suppressionsYAML.String()), 0o600)
	if err != nil {
		return fmt.Errorf("failed to write suppressions YAML: %w", err)
	}

	return nil
}

//nolint:paralleltest //must not run in parallel as it conflicts with TestOutput.
func TestSuppressAllScenarios(t *testing.T) {
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

	defer os.Remove(yamlPath)

	mainPath := filepath.Join(projectRoot, "cmd", "graphql-linter", "main.go")
	cmd := exec.Command("go", "run", mainPath, "-targetPath", targetPath)
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
