package rules

import (
	"testing"

	"github.com/wundergraph/graphql-go-tools/v2/pkg/ast"
)

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

type LineTestCase struct {
	Name          string
	SchemaContent string
	SearchText    string
	WantLine      int
}

func TestGetLineContent(t *testing.T) {
	t.Parallel()

	schema := "A\nB\nC"
	if GetLineContent(schema, 2) != "B" {
		t.Errorf("expected B for line 2")
	}

	if GetLineContent(schema, 0) != "" {
		t.Errorf("expected empty for line 0")
	}

	if GetLineContent("", 1) != "" {
		t.Errorf("expected empty for empty lines")
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

func setupDocumentWithInput(input string, types []ast.Type) *ast.Document {
	doc := &ast.Document{}
	doc.Input.ResetInputString(input)
	doc.Types = types

	return doc
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

func TestIsCamelCase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"empty", "", false},
		{"lowercase start", "fooBar", true},
		{"uppercase start", "FooBar", false},
		{"contains underscore", "foo_bar", false},
		{"single lowercase", "a", true},
		{"single uppercase", "A", false},
		{"all lowercase", "foobar", true},
		{"all uppercase", "FOOBAR", false},
	}
	for _, test := range tests {
		got := isCamelCase(test.input)
		if got != test.expected {
			t.Errorf("%s: got %v, want %v", test.name, got, test.expected)
		}
	}
}
