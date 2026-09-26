package application

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/application/report"
	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/data"
	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/data/base/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/astparser"
)

//nolint:paralleltest //swaps the package level readBuildInfo
func TestExecute_Version(t *testing.T) {
	tests := []struct {
		name          string
		versionString string
		buildInfo     *debug.BuildInfo
		buildInfoOK   bool
		expected      string
	}{
		{
			name:          "VersionString set",
			versionString: "v1.2.3",
			buildInfo:     &debug.BuildInfo{Main: debug.Module{Version: "v0.0.0"}},
			buildInfoOK:   true,
			expected:      "v1.2.3",
		},
		{
			name:          "BuildInfo available",
			versionString: "",
			buildInfo:     &debug.BuildInfo{Main: debug.Module{Version: "v9.8.7"}},
			buildInfoOK:   true,
			expected:      "v9.8.7",
		},
		{
			name:          "BuildInfo not available",
			versionString: "",
			buildInfo:     nil,
			buildInfoOK:   false,
			expected:      "(unknown)",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			readBuildInfo = func() (*debug.BuildInfo, bool) { return test.buildInfo, test.buildInfoOK }

			t.Cleanup(func() { readBuildInfo = debug.ReadBuildInfo })

			got := Execute{VersionString: test.versionString}.Version()
			assert.Equal(t, test.expected, got)
		})
	}
}

func TestErrorTypeCounts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input []models.DescriptionError
		want  map[string]int
	}{
		{
			name:  "empty slice",
			input: []models.DescriptionError{},
			want:  map[string]int{},
		},
		{
			name:  "single error with colon",
			input: []models.DescriptionError{{Message: "type-error: something went wrong"}},
			want:  map[string]int{"type-error": 1},
		},
		{
			name:  "single error with space",
			input: []models.DescriptionError{{Message: "field error something went wrong"}},
			want:  map[string]int{"field": 1},
		},
		{
			name: "multiple errors, mixed",
			input: []models.DescriptionError{
				{Message: "type-error: foo"},
				{Message: "type-error: bar"},
				{Message: "field error baz"},
				{Message: "other"},
			},
			want: map[string]int{
				"type-error": 2,
				"field":      1,
				"other":      1,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := report.ErrorTypeCounts(test.input)
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("errorTypeCounts() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestLintDescriptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                          string
		errorSubstring                string
		schemaContent                 string
		wantNumberOfDescriptionErrors int
		wantHasDeprecationReasonError bool
	}{
		{
			name:                          "schema wide rules are not run per file",
			errorSubstring:                "Object type 'User' is missing a description",
			schemaContent:                 "type User { id: ID }",
			wantNumberOfDescriptionErrors: 2,
			wantHasDeprecationReasonError: false,
		},
		{
			name:                          "all valid, no description reason errors",
			errorSubstring:                "Object type 'Query' is missing a description",
			schemaContent:                 "type Query { id: ID }",
			wantNumberOfDescriptionErrors: 2,
			wantHasDeprecationReasonError: false,
		},
		{
			name:                          "missing deprecation reason",
			errorSubstring:                "deprecations-have-a-reason",
			schemaContent:                 "enum Status {\n  ACTIVE\n  INACTIVE @deprecated\n}",
			wantNumberOfDescriptionErrors: 4,
			wantHasDeprecationReasonError: true,
		},
		{
			name:                          "missing type description",
			errorSubstring:                "Object type 'Foo' is missing a description",
			schemaContent:                 "type Query { id: ID }\ntype Foo { bar: String }",
			wantNumberOfDescriptionErrors: 4,
			wantHasDeprecationReasonError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			runLintDescriptionsTest(
				t,
				&models.LinterConfig{},
				test.name,
				test.schemaContent,
				test.errorSubstring,
				test.wantHasDeprecationReasonError,
				test.wantNumberOfDescriptionErrors,
			)
		})
	}
}

func TestValidateDataTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		schemaContent string
		wantValid     bool
		wantErrLines  int
	}{
		{
			name:          "valid types",
			schemaContent: "type Query { id: ID name: String }",
			wantValid:     true,
			wantErrLines:  0,
		},
		{
			name:          "undefined type",
			schemaContent: "type Query { foo: Bar }",
			wantValid:     false,
			wantErrLines:  1,
		},
		{
			name:          "valid enum",
			schemaContent: "enum Status { ACTIVE INACTIVE } type Query { status: Status }",
			wantValid:     true,
			wantErrLines:  0,
		},
		{
			name:          "input with undefined type",
			schemaContent: "input FooInput { bar: Baz } type Query { foo(input: FooInput): String }",
			wantValid:     false,
			wantErrLines:  1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			runValidateDataTypesTest(
				t,
				nil,
				test.name,
				test.schemaContent,
				test.wantValid,
				test.wantErrLines,
			)
		})
	}
}

func TestFindAndLogGraphQLSchemaFiles(t *testing.T) {
	t.Parallel()

	files := map[string]string{
		"test.graphql": "type Query { id: ID }",
	}
	dir := createTestDirectory(t, files)

	e := Execute{TargetPath: dir}

	foundFiles, err := e.FindAndLogGraphQLSchemaFiles()
	if err != nil || len(foundFiles) != 1 {
		t.Errorf("expected 1 graphql file, got %v, err %v", foundFiles, err)
	}
}

func TestLintSchemaFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	file := dir + "/test.graphql"

	schema := `"""Query root"""
type Query { """ID field""" id: ID }`

	err := os.WriteFile(file, []byte(schema), 0o600)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	e := Execute{}

	total, errorFiles, _ := e.lintSchemaFiles(&models.LinterConfig{}, []string{file})
	if total != 1 || errorFiles != 1 {
		t.Errorf("expected 1 error, got %d, errorFiles %d", total, errorFiles)
	}
}

func TestFindAndLogGraphQLSchemaFiles_Errors(t *testing.T) {
	t.Parallel()

	execute := Execute{TargetPath: "/does/not/exist"}

	_, err := execute.FindAndLogGraphQLSchemaFiles()
	if err == nil {
		t.Errorf("expected error for invalid path")
	}
}

func TestLintSchemaFiles_Errors(t *testing.T) {
	t.Parallel()

	execute := Execute{}

	total, errorFiles, _ := execute.lintSchemaFiles(models.NewLinterConfig(), []string{"/does/not/exist.graphql"})
	if total == 0 || errorFiles == 0 {
		t.Errorf("expected errors for missing file")
	}
}

func TestLogSchemaParseErrors_Errors(t *testing.T) {
	t.Parallel()

	_, report := astparser.ParseGraphqlDocumentString("type Query { id: ID } ...")
	report.InternalErrors = append(report.InternalErrors, assert.AnError)
	LogSchemaParseErrors("type Query { id: ID } ...", &report)
}

//nolint:paralleltest //t.Chdir cannot run in parallel
func TestRun_OutsideGoModule(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	err := os.WriteFile("s.graphql", []byte("type Query { a: String }\n"), 0o600)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	for _, targetPath := range []string{"", dir} {
		err := Execute{TargetPath: targetPath}.Run()
		if err != nil && !errors.Is(err, ErrLintingFailed) {
			t.Errorf("targetPath %q: expected lint result, got %v", targetPath, err)
		}
	}
}

func TestGetUnsuppressedDescriptionErrors_CheckDescriptions(t *testing.T) {
	t.Parallel()

	errs := []models.DescriptionError{
		{Message: "types-have-descriptions: Object type 'Query' is missing a description"},
		{Message: "types-are-capitalized: The object type 'foo' should start with a capital letter."},
	}

	config := models.NewLinterConfig()
	assert.Len(t, getUnsuppressedDescriptionErrors(errs, config, "s.graphql"), 2)

	config.Settings.CheckDescriptions = false
	got := getUnsuppressedDescriptionErrors(errs, config, "s.graphql")
	assert.Equal(t, errs[1:], got)
}

//nolint:paralleltest //t.Chdir cannot run in parallel
func TestRun_ValidateFederationDisabled(t *testing.T) {
	t.Chdir(t.TempDir())

	// An unknown directive is rejected by the federation checks only.
	schema := `"""Q"""
type Query @foo {
  """A"""
  a: PageInfo
}
"""P"""
type PageInfo {
  """E"""
  endCursor: String
  """N"""
  hasNextPage: Boolean!
  """P"""
  hasPreviousPage: Boolean!
  """S"""
  startCursor: String
}
`
	require.NoError(t, os.WriteFile("s.graphql", []byte(schema), 0o600))

	require.Error(t, Execute{}.Run(), "federation checks should reject @foo by default")

	require.NoError(t, os.WriteFile(".graphql-linter.yml", []byte("settings:\n  validateFederation: false\n"), 0o600))
	require.NoError(t, Execute{}.Run())
}

func TestSuppressionValueMatchesEveryRule(t *testing.T) {
	t.Parallel()

	files, err := filepath.Glob("../../../../test/testdata/graphql/base/invalid/*.graphql")
	require.NoError(t, err)
	require.NotEmpty(t, files)

	seenRules := map[string]bool{}

	for _, file := range files {
		schemaBytes, err := os.ReadFile(file)
		require.NoError(t, err)

		schema := string(schemaBytes)
		doc := parseGraphQLDocument(schema)
		config := models.NewLinterConfig()

		findings, _ := Execute{}.lintDescriptions(doc, config, schema, file)
		_, dataTypeErrors := data.NewStore("", "").CollectUnsuppressedDataTypeErrors(doc, config, schema, file)
		findings = append(findings, dataTypeErrors...)

		for _, finding := range findings {
			rule, _, _ := strings.Cut(finding.Message, ":")
			seenRules[rule] = true

			require.NotEmpty(t, finding.Value, "%s: %s has no value", file, finding.Message)

			matching := models.NewLinterConfig()
			matching.Suppressions = []models.Suppression{{Rule: rule, Value: finding.Value}}
			assert.Empty(t, getUnsuppressedDescriptionErrors([]models.DescriptionError{finding}, matching, file),
				"%s: value %q should suppress %s", file, finding.Value, finding.Message)

			other := models.NewLinterConfig()
			other.Suppressions = []models.Suppression{{Rule: rule, Value: "no-such-value"}}
			assert.Len(t, getUnsuppressedDescriptionErrors([]models.DescriptionError{finding}, other, file), 1,
				"%s: other value should not suppress %s", file, finding.Message)
		}
	}

	assert.GreaterOrEqual(t, len(seenRules), 15, "rules covered: %v", seenRules)
}

func TestLintMergedSchema_SplitFiles(t *testing.T) {
	t.Parallel()

	dir := createTestDirectory(t, map[string]string{
		"a.graphql": "\"\"\"Q.\"\"\"\ntype Query {\n  \"\"\"U.\"\"\"\n  user: User\n}\n",
		"b.graphql": "\"\"\"U.\"\"\"\ntype User {\n  \"\"\"P.\"\"\"\n  page: PageInfo\n}\n\n" +
			"\"\"\"Unused.\"\"\"\nscalar Unused\n",
		"c.graphql": "\"\"\"P.\"\"\"\ntype PageInfo {\n  \"\"\"E.\"\"\"\n  endCursor: String\n}\n",
	})
	files := []string{
		filepath.Join(dir, "a.graphql"),
		filepath.Join(dir, "b.graphql"),
		filepath.Join(dir, "c.graphql"),
	}

	// Query, User and PageInfo are defined and used across files: only Unused
	// is reported, on its own line in b.graphql.
	got := lintMergedSchema(models.NewLinterConfig(), files)
	require.Len(t, got, 1)
	assert.Equal(t, files[1], got[0].FilePath)
	assert.Equal(t, 8, got[0].LineNum)
	assert.Equal(t, "scalar Unused", got[0].LineContent)
	assert.Contains(t, got[0].Message, "defined-types-are-used: Type 'Unused'")
}

func TestLintMergedSchema_DirectiveFindingsAreCountedAndSuppressible(t *testing.T) {
	t.Parallel()

	dir := createTestDirectory(t, map[string]string{
		"s.graphql": "\"\"\"Q.\"\"\"\ntype Query @foo {\n  \"\"\"P.\"\"\"\n  page: PageInfo @foo\n}\n" +
			"\"\"\"P.\"\"\"\ntype PageInfo {\n  \"\"\"E.\"\"\"\n  endCursor: String\n}\n",
	})
	files := []string{filepath.Join(dir, "s.graphql")}

	got := lintMergedSchema(models.NewLinterConfig(), files)
	require.Len(t, got, 2)
	assert.Equal(t, []int{2, 4}, []int{got[0].LineNum, got[1].LineNum})

	config := models.NewLinterConfig()
	config.Suppressions = []models.Suppression{{Rule: "invalid-federation-directive", Value: "foo", Line: 4}}
	got = lintMergedSchema(config, files)
	require.Len(t, got, 1)
	assert.Equal(t, 2, got[0].LineNum)

	config.Settings.ValidateFederation = false
	assert.Empty(t, lintMergedSchema(config, files))
}
