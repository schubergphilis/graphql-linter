package data

import (
	"os"
	"strings"
	"testing"

	"github.com/wundergraph/graphql-go-tools/v2/pkg/ast"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/astparser"
)

type ErrorWithMessage interface {
	GetMessage() string
}

type TestCase struct {
	Name          string
	SchemaContent string
	SearchText    string
	Input         string
	Expected      interface{}
	ExpectError   bool
	ExpectMsg     string
	WantLine      int
	WantValid     bool
	WantErrLines  int
}

type LineTestCase struct {
	Name          string
	SchemaContent string
	SearchText    string
	WantLine      int
}

type BaseTypeTestCase struct {
	Name     string
	Input    string
	Types    []ast.Type
	TypeRef  ast.Type
	Expected string
}

type BoolTestCase struct {
	Name     string
	Input    string
	Expected bool
}

type DistanceTestCase struct {
	Name     string
	Source   string
	Target   string
	Expected int
}

type ErrorTestCase struct {
	Name          string
	SchemaContent string
	ExpectError   bool
	ExpectMsg     string
}

func parseGraphQLDocument(schemaContent string) *ast.Document {
	doc, _ := astparser.ParseGraphqlDocumentString(schemaContent)

	return &doc
}

func createTestStore(verbose bool, config *LinterConfig) Store {
	store := Store{Verbose: verbose}
	if config != nil {
		store.LinterConfig = config
	}

	return store
}

func createTempSchemaFile(t *testing.T, content string) string {
	t.Helper()

	tempFile, err := os.CreateTemp(t.TempDir(), "schema*.graphql")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	_, err = tempFile.WriteString(content)
	if err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}

	tempFile.Close()

	return tempFile.Name()
}

func setupDocumentWithInput(input string, types []ast.Type) *ast.Document {
	doc := &ast.Document{}
	doc.Input.ResetInputString(input)
	doc.Types = types

	return doc
}

func createSuppressionConfig(suppressions []Suppression) *LinterConfig {
	return &LinterConfig{
		Suppressions: suppressions,
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

func runLineTableTest(t *testing.T, tests []LineTestCase, testFunc func(string, string) int) {
	t.Helper()

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()

			got := testFunc(test.SchemaContent, test.SearchText)
			if got != test.WantLine {
				t.Errorf("got %v, want %v", got, test.WantLine)
			}
		})
	}
}

func runBoolTableTest(t *testing.T, tests []BoolTestCase, testFunc func(string) bool) {
	t.Helper()

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()

			got := testFunc(test.Input)
			if got != test.Expected {
				t.Errorf("got %v, want %v", got, test.Expected)
			}
		})
	}
}

func runDistanceTableTest(
	t *testing.T,
	tests []DistanceTestCase,
	testFunc func(string, string) int,
) {
	t.Helper()

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()

			got := testFunc(test.Source, test.Target)
			if got != test.Expected {
				t.Errorf("got %v, want %v", got, test.Expected)
			}
		})
	}
}

func runBaseTypeTableTest(t *testing.T, tests []BaseTypeTestCase) {
	t.Helper()

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()

			doc := setupDocumentWithInput(test.Input, test.Types)

			got := getBaseTypeName(doc, test.TypeRef)
			if got != test.Expected {
				t.Errorf("got %v, want %v", got, test.Expected)
			}
		})
	}
}

func runLintDescriptionsTest(
	t *testing.T,
	testName, schemaContent, errorSubstring string,
	wantHasDeprecationReasonError bool,
) {
	t.Helper()

	doc := parseGraphQLDocument(schemaContent)
	s := Store{}
	descriptionErrors, hasDeprecationReasonError := s.lintDescriptions(
		doc,
		schemaContent,
		"test.graphql",
	)

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

func runValidateDataTypesTest(
	t *testing.T,
	testName, schemaContent string,
	wantValid bool,
	wantErrLines int,
) {
	t.Helper()

	doc := parseGraphQLDocument(schemaContent)
	s := Store{}

	valid, errorLines, _ := s.validateDataTypes(doc, schemaContent, "test.graphql")
	if valid != wantValid {
		t.Errorf("%s: got valid=%v, want %v", testName, valid, wantValid)
	}

	if len(errorLines) != wantErrLines {
		t.Errorf("%s: got %d errorLines, want %d", testName, len(errorLines), wantErrLines)
	}
}
