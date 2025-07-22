//go:build component

package component

import (
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
	cmd := exec.Command("go", "build", "-ldflags=-X 'main.Version=v4.5.6'", "-o", outputPath, mainPath)
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
		log.WithError(err).WithField("binaryPath", binaryPath).Fatal("failed to remove built binary during teardown")
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
			t.Errorf("required %s substring not found: %q\nBlock:\n%s", blockName, substr, strings.Join(block, "\n"))
		}
	}
}

func TestOutput(t *testing.T) {
	t.Parallel()

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
		t.Parallel()
		required := []string{
			"Error type summary:",
			"arguments-have-descriptions: 2",
			"defined-types-are-used: 10",
			"deprecations-have-a-reason: 1",
			"descriptions-are-capitalized: 2",
			"enum-values-have-descriptions: 2",
			"enum-values-sorted-alphabetically: 3",
			"fields-are-camel-cased: 1",
			"fields-have-descriptions: 2",
			"input-object-fields-sorted-alphabetically: 2",
			"input-object-values-are-camel-cased: 1",
			"input-object-values-have-descriptions: 1",
			"interface-fields-sorted-alphabetically: 1",
			"invalid-graphql-schema: 1",
			"relay-connection-arguments-spec: 2",
			"relay-connection-types-spec: 1",
			"relay-page-info-spec: 11",
			"type-fields-sorted-alphabetically: 9",
			"types-have-descriptions: 3",
		}
		checkRequiredSubstrings(t, "error type summary", sections["errorTypeSummary"], required)
	})

	t.Run("Summary block", func(t *testing.T) {
		t.Parallel()
		required := []string{
			"linting summary",
			"passedFiles=0",
			"percentPassed=\"0.00%\"",
			"totalFiles=19",
			"files with at least one error",
			"filesWithAtLeastOneError=19",
			"percentage=\"100.00%\"",
			"totalErrors: 57",
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
		t.Parallel()
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
