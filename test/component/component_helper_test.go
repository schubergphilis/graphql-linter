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

type SuppressionEntry struct {
	File   string
	Line   int
	Rule   string
	Value  string
	Reason string
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

//nolint:maintidx //generateSuppressions is a data function and not logic-heavy.
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
