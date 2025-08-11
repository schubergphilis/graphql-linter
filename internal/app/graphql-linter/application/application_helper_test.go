package application

import (
	"os"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/application/mocks"
	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/data"
	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/data/base/models"
	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/data/base/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func runValidateDataTypesTest(
	t *testing.T,
	modelsLinterConfig *models.LinterConfig,
	testName, schemaContent string,
	wantValid bool,
	wantErrLines int,
) {
	t.Helper()

	doc := parseGraphQLDocument(schemaContent)

	dataStore, err := data.NewStore("", "", rules.Rule{}, true)
	require.NoError(t, err, "Failed to create data store")

	valid, errorLines, _ := dataStore.ValidateDataTypes(
		doc,
		modelsLinterConfig,
		schemaContent,
		"test.graphql",
	)
	if valid != wantValid {
		t.Errorf("%s: got valid=%v, want %v", testName, valid, wantValid)
	}

	if len(errorLines) != wantErrLines {
		t.Errorf("%s: got %d errorLines, want %d", testName, len(errorLines), wantErrLines)
	}
}

func runLintDescriptionsTest(
	t *testing.T,
	modelsLinterConfig *models.LinterConfig,
	testName, schemaContent, errorSubstring string,
	wantHasDeprecationReasonError bool,
	numberOfDescriptionErrors int,
) {
	t.Helper()

	mocksDebugger := &mocks.Debugger{}
	mocksDebugger.EXPECT().ReadBuildInfo().Return(&debug.BuildInfo{Main: debug.Module{Version: "4.3.2"}}, true).Times(1)

	execute, err := NewExecute(mocksDebugger, "", "", "", false)
	require.NoError(t, err, "failed to create execute instance")

	version := execute.Version()
	assert.Equal(t, "4.3.2", version, "expected version to be 4.3.2, got %s", version)

	doc := parseGraphQLDocument(schemaContent)
	descriptionErrors, hasDeprecationReasonError := execute.lintDescriptions(
		doc,
		modelsLinterConfig,
		schemaContent,
		"test.graphql",
	)

	assert.Len(t, descriptionErrors, numberOfDescriptionErrors)

	found := false

	for _, err := range descriptionErrors {
		if errorSubstring == "" ||
			(err.Message != "" && strings.Contains(err.Message, errorSubstring)) {
			found = true

			break
		}
	}

	if !found {
		t.Errorf(
			"%s: expected error containing '%s', but not found in errors: %v",
			testName,
			errorSubstring,
			descriptionErrors,
		)
	}

	if hasDeprecationReasonError != wantHasDeprecationReasonError {
		t.Errorf(
			"%s: got hasDeprecationReasonError=%v, want %v",
			testName,
			hasDeprecationReasonError,
			wantHasDeprecationReasonError,
		)
	}
}

func createTestDirectory(t *testing.T, files map[string]string) string {
	t.Helper()

	dir := t.TempDir()

	for filename, content := range files {
		filepath := dir + "/" + filename

		err := os.WriteFile(filepath, []byte(content), 0o600)
		if err != nil {
			t.Fatalf("failed to write test file %s: %v", filename, err)
		}
	}

	return dir
}

func createTestExecute(verbose bool) Execute {
	execute := Execute{Verbose: verbose}

	return execute
}
