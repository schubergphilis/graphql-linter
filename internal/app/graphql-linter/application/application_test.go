package application

import (
	"os"
	"reflect"
	"runtime/debug"
	"testing"

	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/application/mocks"
	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/application/report"
	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/data/base/models"
	"github.com/stretchr/testify/assert"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/astparser"
)

func TestExecute_Version(t *testing.T) {
	t.Parallel()

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
			t.Parallel()

			mockDebugger := new(mocks.Debugger)
			if test.versionString == "" {
				mockDebugger.On("ReadBuildInfo").Return(test.buildInfo, test.buildInfoOK)
			}

			e := Execute{
				Debugger:      mockDebugger,
				VersionString: test.versionString,
			}
			got := e.Version()
			assert.Equal(t, test.expected, got)
			mockDebugger.AssertExpectations(t)
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
		schemaContent                 string
		errorSubstring                string
		wantHasDeprecationReasonError bool
	}{
		{
			name:                          "missing Query root type",
			schemaContent:                 "type User { id: ID }",
			errorSubstring:                "invalid-graphql-schema",
			wantHasDeprecationReasonError: false,
		},
		{
			name:                          "all valid, no errors",
			schemaContent:                 "type Query { id: ID }",
			errorSubstring:                "Object type 'Query' is missing a description",
			wantHasDeprecationReasonError: false,
		},
		{
			name:                          "missing deprecation reason",
			schemaContent:                 `enum Status {\n  ACTIVE\n  INACTIVE @deprecated\n}`,
			errorSubstring:                "deprecations-have-a-reason",
			wantHasDeprecationReasonError: true,
		},
		{
			name:                          "missing type description",
			schemaContent:                 "type Query { id: ID }\ntype Foo { bar: String }",
			errorSubstring:                "Object type 'Foo' is missing a description",
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

	e := Execute{TargetPath: dir, Verbose: false}

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

	e := createTestExecute(false, nil)

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

	execute := Execute{Verbose: false}

	total, errorFiles, _ := execute.lintSchemaFiles(nil, []string{"/does/not/exist.graphql"})
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
