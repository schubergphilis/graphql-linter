//go:build integration

package application

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/schubergphilis/mcvs-golang-project-root/pkg/projectroot"
	"github.com/stretchr/testify/require"
)

func validateErrorName(t *testing.T, fileName string) string {
	t.Helper()

	parts := strings.SplitN(fileName, "-", 2)
	if len(parts) < 2 {
		require.FailNow(t, "Filename does not contain error name.", fileName)
	}

	errorName := strings.TrimSuffix(parts[1], ".graphql")
	for _, r := range errorName {
		if r >= '0' && r <= '9' {
			require.FailNow(t, "Error name contains digits (possible typo)", fileName+": extracted error name: "+errorName)
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

func TestInvalidSchemas(t *testing.T) {
	t.Parallel()

	projectRoot, err := projectroot.FindProjectRoot()
	require.NoError(t, err, "failed to determine project root")

	baseDir := filepath.Join(projectRoot, "test/testdata/graphql/base/invalid")
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
			require.FailNow(t, "Expected error for file, but linter passed.", file.Name())

			continue
		}

		errorName := validateErrorName(t, file.Name())
		rules := parseLinterOutput(t, output, file.Name())
		fmt.Println("------------", rules, errorName)
		found := false
		for _, rule := range rules {
			if rule == errorName {
				found = true
				break
			}
		}

		if !found {
			require.FailNow(
				t,
				"File: expected error rule not found in output",
				file.Name()+": expected error rule '"+errorName+"' in output, got rules: "+strings.Join(rules, ", "),
			)
		}
	}
}
