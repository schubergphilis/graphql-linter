package data

import (
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/ast"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/astparser"
)

func TestFindLineNumberByText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		schemaContent string
		searchText    string
		wantLine      int
	}{
		{
			name:          "text on first line",
			schemaContent: "enum Color {\nRED\nGREEN\n}",
			searchText:    "enum",
			wantLine:      1,
		},
		{
			name:          "text on second line",
			schemaContent: "enum Color {\nRED\nGREEN\n}",
			searchText:    "RED",
			wantLine:      2,
		},
		{
			name:          "text on third line",
			schemaContent: "enum Color {\nRED\nGREEN\n}",
			searchText:    "GREEN",
			wantLine:      3,
		},
		{
			name:          "text not found",
			schemaContent: "enum Color {\nRED\nGREEN\n}",
			searchText:    "BLUE",
			wantLine:      0,
		},
		{
			name:          "multiple matches, returns first",
			schemaContent: "A\nB\nA\nC",
			searchText:    "A",
			wantLine:      1,
		},
		{
			name:          "empty schemaContent",
			schemaContent: "",
			searchText:    "anything",
			wantLine:      0,
		},
		{
			name:          "empty searchText matches first line",
			schemaContent: "foo\nbar",
			searchText:    "",
			wantLine:      1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := findLineNumberByText(test.schemaContent, test.searchText)
			assert.Equal(t, test.wantLine, got)
		})
	}
}

func TestGetBaseTypeName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		types    []ast.Type
		typeRef  ast.Type
		expected string
	}{
		{
			name:  "Named type",
			input: "String",
			typeRef: ast.Type{
				TypeKind: ast.TypeKindNamed,
				Name:     ast.ByteSliceReference{Start: 0, End: 6}, // "String"
			},
			expected: "String",
		},
		{
			name:  "List of Named type",
			input: "Int",
			types: []ast.Type{
				{
					TypeKind: ast.TypeKindNamed,
					Name:     ast.ByteSliceReference{Start: 0, End: 3}, // "Int"
				},
			},
			typeRef: ast.Type{
				TypeKind: ast.TypeKindList,
				OfType:   0,
			},
			expected: "Int",
		},
		{
			name:  "NonNull of Named type",
			input: "Boolean",
			types: []ast.Type{
				{
					TypeKind: ast.TypeKindNamed,
					Name:     ast.ByteSliceReference{Start: 0, End: 7}, // "Boolean"
				},
			},
			typeRef: ast.Type{
				TypeKind: ast.TypeKindNonNull,
				OfType:   0,
			},
			expected: "Boolean",
		},
		{
			name:  "NonNull of List of Named type",
			input: "ID",
			types: []ast.Type{
				{
					TypeKind: ast.TypeKindList,
					OfType:   1,
				},
				{
					TypeKind: ast.TypeKindNamed,
					Name:     ast.ByteSliceReference{Start: 0, End: 2}, // "ID"
				},
			},
			typeRef: ast.Type{
				TypeKind: ast.TypeKindNonNull,
				OfType:   0,
			},
			expected: "ID",
		},
		{
			name:  "Unknown type kind",
			input: "",
			typeRef: ast.Type{
				TypeKind: ast.TypeKindUnknown,
			},
			expected: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			doc := &ast.Document{}
			doc.Input.ResetInputString(test.input)
			doc.Types = test.types
			got := getBaseTypeName(doc, test.typeRef)
			assert.Equal(t, test.expected, got)
		})
	}
}

func TestGetAvailableTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		builtInScalars map[string]bool
		definedTypes   map[string]bool
		expected       []string
	}{
		{
			name:           "both empty",
			builtInScalars: map[string]bool{},
			definedTypes:   map[string]bool{},
			expected:       []string{},
		},
		{
			name:           "only built-in scalars",
			builtInScalars: map[string]bool{"String": true, "Int": true},
			definedTypes:   map[string]bool{},
			expected:       []string{"Int", "String"},
		},
		{
			name:           "only defined types",
			builtInScalars: map[string]bool{},
			definedTypes:   map[string]bool{"User": true, "Post": true},
			expected:       []string{"Post", "User"},
		},
		{
			name:           "both, no overlap",
			builtInScalars: map[string]bool{"String": true},
			definedTypes:   map[string]bool{"User": true},
			expected:       []string{"String", "User"},
		},
		{
			name:           "both, with overlap",
			builtInScalars: map[string]bool{"String": true, "User": true},
			definedTypes:   map[string]bool{"User": true, "Post": true},
			expected:       []string{"Post", "String", "User", "User"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := getAvailableTypes(test.builtInScalars, test.definedTypes)
			sort.Strings(got)
			sort.Strings(test.expected)
			assert.Equal(t, test.expected, got)
		})
	}
}

func TestIsAlphaUnderOrDigit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    rune
		expected bool
	}{
		{"lowercase letter", 'a', true},
		{"uppercase letter", 'Z', true},
		{"digit", '5', true},
		{"underscore", '_', true},
		{"space", ' ', false},
		{"dash", '-', false},
		{"symbol", '$', false},
		{"unicode letter", 'ß', true},
		{"unicode digit", '٣', true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := isAlphaUnderOrDigit(test.input)
			assert.Equal(t, test.expected, got)
		})
	}
}

func TestIsValidEnumValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    string
		expected bool
	}{
		{"empty", "", false},
		{"starts with letter", "A", true},
		{"starts with underscore", "_A", true},
		{"starts with digit", "1A", false},
		{"contains dash", "A-B", false},
		{"contains space", "A B", false},
		{"all valid", "A1_B2", true},
		{"unicode letter", "Äpfel", false},
		{"unicode digit", "A٣", true},
		{"only underscore", "_", true},
		{"underscore and digits", "_123", true},
		{"invalid symbol", "A$", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := isValidEnumValue(test.value)
			assert.Equal(t, test.expected, got)
		})
	}
}

func TestLevenshteinDistance(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		source   string
		target   string
		expected int
	}{
		{"identical", "kitten", "kitten", 0},
		{"one substitution", "kitten", "sitten", 1},
		{"one insertion", "kitten", "kittens", 1},
		{"one deletion", "kitten", "kittn", 1},
		{"completely different", "abc", "xyz", 3},
		{"empty source", "", "abc", 3},
		{"empty target", "abc", "", 3},
		{"both empty", "", "", 0},
		{"case sensitive", "Kitten", "kitten", 1},
		{"unicode", "café", "coffee", 4},
		{"longer", "intention", "execution", 5},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := levenshteinDistance(test.source, test.target)
			assert.Equal(t, test.expected, got)
		})
	}
}

func TestHasSuspiciousEnumValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    string
		expected bool
	}{
		{"empty", "", false},
		{"ends with digit", "FOO1", true},
		{"ends with non-digit", "FOO", false},
		{"single digit", "1", true},
		{"single letter", "A", false},
		{"ends with zero", "BAR0", true},
		{"ends with nine", "BAR9", true},
		{"ends with symbol", "FOO$", false},
		{"unicode digit", "FOO٣", false}, // Arabic-Indic digit, not ASCII
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := hasSuspiciousEnumValue(test.value)
			assert.Equal(t, test.expected, got)
		})
	}
}

func TestHasEmbeddedDigits(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    string
		expected bool
	}{
		{"empty", "", false},
		{"no digits", "FOO", false},
		{"single digit", "1", true},
		{"digit at start", "1FOO", true},
		{"digit at end", "FOO1", true},
		{"digit in middle", "F1OO", true},
		{"multiple digits", "F12O3O", true},
		{"all digits", "12345", true},
		{"unicode digit", "FOO٣", false}, // Arabic-Indic digit, not ASCII
		{"symbol", "FOO$", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := hasEmbeddedDigits(test.value)
			assert.Equal(t, test.expected, got)
		})
	}
}

func TestSuggestCorrectEnumValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{"exact correction", "STRING2", "STRING"},
		{"another correction", "BOOLEAN2", "BOOLEAN"},
		{"typo with digit", "BOOLE3AN", "BOOLEAN"},
		{"typo with digit 2", "BOOL3AN", "BOOLEAN"},
		{"typo with digit 3", "BOOLEAN3", "BOOLEAN"},
		{"float typo", "FLOA2T", "FLOAT"},
		{"float typo 2", "FLO2AT", "FLOAT"},
		{"float with digit", "FLOAT2", "FLOAT"},
		{"int with digit", "INT2", "INT"},
		{"integer with digit", "INTEGER2", "INTEGER"},
		{"int typo", "I2NT", "INT"},
		{"integer typo", "INTE2GER", "INTEGER"},
		{"levenshtein match", "STRONG", "STRING"},
		{"levenshtein match 2", "BOOLEEN", "BOOLEAN"},
		{"levenshtein match 3", "INTEGRA", "INTEGER"},
		{"levenshtein match 4", "FLOT", "FLOAT"},
		{"levenshtein match 5", "ID", "ID"},
		{"no match", "CUSTOM", ""},
		{"empty", "", ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := suggestCorrectEnumValue(test.value)
			assert.Equal(t, test.expected, got)
		})
	}
}

func TestRemoveSuffixDigits(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{"no digits", "FOO", "FOO"},
		{"trailing digits", "FOO123", "FOO"},
		{"all digits", "123", ""},
		{"mixed", "FOO1BAR2", "FOO1BAR"},
		{"empty", "", ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := removeSuffixDigits(test.value)
			assert.Equal(t, test.expected, got)
		})
	}
}

func TestRemoveAllDigits(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{"no digits", "FOO", "FOO"},
		{"digits only", "123", ""},
		{"mixed", "F1O2O3", "FOO"},
		{"trailing digits", "FOO123", "FOO"},
		{"leading digits", "123FOO", "FOO"},
		{"empty", "", ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := removeAllDigits(test.value)
			assert.Equal(t, test.expected, got)
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

			doc, _ := astparser.ParseGraphqlDocumentString(test.schemaContent)
			descriptionErrors, _, hasDeprecationReasonError := lintDescriptions(&doc, test.schemaContent)

			found := false

			for _, err := range descriptionErrors {
				if test.errorSubstring == "" || (err.Message != "" && contains(err.Message, test.errorSubstring)) {
					found = true

					break
				}
			}

			if !found {
				t.Errorf("expected error containing '%s', but not found in errors: %v", test.errorSubstring, descriptionErrors)
			}

			if hasDeprecationReasonError != test.wantHasDeprecationReasonError {
				t.Errorf("got hasDeprecationReasonError=%v, want %v", hasDeprecationReasonError, test.wantHasDeprecationReasonError)
			}
		})
	}
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
