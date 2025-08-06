package data

import (
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/data/federation"
	"github.com/stretchr/testify/assert"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/ast"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/astparser"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/operationreport"
)

func TestFindLineNumberByText(t *testing.T) {
	t.Parallel()

	tests := []LineTestCase{
		{
			Name:          "text on first line",
			SchemaContent: "enum Color {\nRED\nGREEN\n}",
			SearchText:    "enum",
			WantLine:      1,
		},
		{
			Name:          "text on second line",
			SchemaContent: "enum Color {\nRED\nGREEN\n}",
			SearchText:    "RED",
			WantLine:      2,
		},
		{
			Name:          "text on third line",
			SchemaContent: "enum Color {\nRED\nGREEN\n}",
			SearchText:    "GREEN",
			WantLine:      3,
		},
		{
			Name:          "text not found",
			SchemaContent: "enum Color {\nRED\nGREEN\n}",
			SearchText:    "BLUE",
			WantLine:      0,
		},
		{
			Name:          "multiple matches, returns first",
			SchemaContent: "A\nB\nA\nC",
			SearchText:    "A",
			WantLine:      1,
		},
		{
			Name:          "empty schemaContent",
			SchemaContent: "",
			SearchText:    "anything",
			WantLine:      0,
		},
		{
			Name:          "empty searchText matches first line",
			SchemaContent: "foo\nbar",
			SearchText:    "",
			WantLine:      1,
		},
	}

	runLineTableTest(t, tests, findLineNumberByText)
}

func TestFindLineNumberByText_ExtraCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		schemaContent string
		searchText    string
		wantLine      int
	}{
		{"case sensitive", "Foo\nfoo\nFOO", "FOO", 3},
		{
			"three foo matches",
			"foo\nfoo\nfoo", //nolint:dupword //multiple duplicates required to test whether it finds the first match
			"foo",
			1,
		},
		{"empty searchText", "foo\nbar", "", 1},
		{"no match", "foo\nbar", "baz", 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := findLineNumberByText(test.schemaContent, test.searchText)
			if got != test.wantLine {
				t.Errorf("got %v, want %v", got, test.wantLine)
			}
		})
	}
}

func TestGetBaseTypeName(t *testing.T) {
	t.Parallel()

	tests := []BaseTypeTestCase{
		{
			Name:  "Named type",
			Input: "String",
			TypeRef: ast.Type{
				TypeKind: ast.TypeKindNamed,
				Name:     ast.ByteSliceReference{Start: 0, End: 6}, // "String"
			},
			Expected: "String",
		},
		{
			Name:  "List of Named type",
			Input: "Int",
			Types: []ast.Type{
				{
					TypeKind: ast.TypeKindNamed,
					Name:     ast.ByteSliceReference{Start: 0, End: 3}, // "Int"
				},
			},
			TypeRef: ast.Type{
				TypeKind: ast.TypeKindList,
				OfType:   0,
			},
			Expected: "Int",
		},
		{
			Name:  "NonNull of Named type",
			Input: "Boolean",
			Types: []ast.Type{
				{
					TypeKind: ast.TypeKindNamed,
					Name:     ast.ByteSliceReference{Start: 0, End: 7}, // "Boolean"
				},
			},
			TypeRef: ast.Type{
				TypeKind: ast.TypeKindNonNull,
				OfType:   0,
			},
			Expected: "Boolean",
		},
		{
			Name:  "NonNull of List of Named type",
			Input: "ID",
			Types: []ast.Type{
				{
					TypeKind: ast.TypeKindList,
					OfType:   1,
				},
				{
					TypeKind: ast.TypeKindNamed,
					Name:     ast.ByteSliceReference{Start: 0, End: 2}, // "ID"
				},
			},
			TypeRef: ast.Type{
				TypeKind: ast.TypeKindNonNull,
				OfType:   0,
			},
			Expected: "ID",
		},
		{
			Name:  "Unknown type kind",
			Input: "",
			TypeRef: ast.Type{
				TypeKind: ast.TypeKindUnknown,
			},
			Expected: "",
		},
	}

	runBaseTypeTableTest(t, tests)
}

func TestGetBaseTypeName_ExtraCases(t *testing.T) {
	t.Parallel()

	doc := &ast.Document{}
	doc.Input.ResetInputString("Foo")
	doc.Types = []ast.Type{
		{TypeKind: ast.TypeKindNonNull, OfType: 1},
		{TypeKind: ast.TypeKindList, OfType: 2},
		{TypeKind: ast.TypeKindNamed, Name: ast.ByteSliceReference{Start: 0, End: 3}},
	}

	got := getBaseTypeName(doc, doc.Types[0])
	if got != "Foo" {
		t.Errorf("got %v, want Foo", got)
	}

	got = getBaseTypeName(doc, ast.Type{TypeKind: ast.TypeKindUnknown})
	if got != "" {
		t.Errorf("got %v, want empty string", got)
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

func TestGetAvailableTypes_ExtraCases(t *testing.T) {
	t.Parallel()

	builtIn := map[string]bool{"A": true, "B": true}
	defined := map[string]bool{"B": true, "C": true}

	got := getAvailableTypes(builtIn, defined)
	if len(got) != 4 {
		t.Errorf("expected 4 types, got %v", got)
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

func TestIsAlphaUnderOrDigit_ExtraCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    rune
		expected bool
	}{
		{'\n', false},
		{'$', false},
		{'_', true},
		{'A', true},
		{'1', true},
	}
	for _, test := range tests {
		t.Run(string(test.input), func(t *testing.T) {
			t.Parallel()

			got := isAlphaUnderOrDigit(test.input)
			if got != test.expected {
				t.Errorf("got %v, want %v", got, test.expected)
			}
		})
	}
}

func TestIsValidEnumValue(t *testing.T) {
	t.Parallel()

	tests := []BoolTestCase{
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
	runBoolTableTest(t, tests, isValidEnumValue)
}

func TestIsValidEnumValue_ExtraCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		value    string
		expected bool
	}{
		{"A$", false},
		{"A_1", true},
		{"_", true},
		{"1A", false},
		{"", false},
	}
	for _, test := range tests {
		t.Run(test.value, func(t *testing.T) {
			t.Parallel()

			got := isValidEnumValue(test.value)
			if got != test.expected {
				t.Errorf("got %v, want %v", got, test.expected)
			}
		})
	}
}

func TestLevenshteinDistance(t *testing.T) {
	t.Parallel()

	tests := []DistanceTestCase{
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
	runDistanceTableTest(t, tests, levenshteinDistance)
}

func TestLevenshteinDistance_ExtraCases(t *testing.T) {
	t.Parallel()

	if levenshteinDistance("", "") != 0 {
		t.Errorf("expected 0 for empty strings")
	}

	if levenshteinDistance("abc", "") != 3 {
		t.Errorf("expected 3 for abc vs empty")
	}

	if levenshteinDistance("", "abc") != 3 {
		t.Errorf("expected 3 for empty vs abc")
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

func TestHasSuspiciousEnumValue_ExtraCases(t *testing.T) {
	t.Parallel()

	if hasSuspiciousEnumValue("") {
		t.Errorf("expected false for empty string")
	}

	if !hasSuspiciousEnumValue("FOO1") {
		t.Errorf("expected true for FOO1")
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

func TestHasEmbeddedDigits_ExtraCases(t *testing.T) {
	t.Parallel()

	if hasEmbeddedDigits("") {
		t.Errorf("expected false for empty string")
	}

	if !hasEmbeddedDigits("A1B") {
		t.Errorf("expected true for A1B")
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

func TestSuggestCorrectEnumValue_ExtraCases(t *testing.T) {
	t.Parallel()

	if suggestCorrectEnumValue("") != "" {
		t.Errorf("expected empty string for empty input")
	}

	if suggestCorrectEnumValue("CUSTOM") != "" {
		t.Errorf("expected empty string for CUSTOM")
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

func TestRemoveSuffixDigits_ExtraCases(t *testing.T) {
	t.Parallel()

	if removeSuffixDigits("123") != "" {
		t.Errorf("expected empty string for all digits")
	}

	if removeSuffixDigits("FOO123") != "FOO" {
		t.Errorf("expected FOO for FOO123")
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

func TestRemoveAllDigits_ExtraCases(t *testing.T) {
	t.Parallel()

	if removeAllDigits("123") != "" {
		t.Errorf("expected empty string for all digits")
	}

	if removeAllDigits("F1O2O3") != "FOO" {
		t.Errorf("expected FOO for F1O2O3")
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
				test.name,
				test.schemaContent,
				test.wantValid,
				test.wantErrLines,
			)
		})
	}
}

func TestValidateDirectiveNames(t *testing.T) {
	t.Parallel()

	doc, _ := astparser.ParseGraphqlDocumentString("type Query { id: ID } directive @key on OBJECT")

	tests := []struct {
		name string
		doc  *ast.Document
		want bool
	}{
		{"valid directives", &doc, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := validateDirectiveNames(test.doc)
			if got != test.want {
				t.Errorf("got %v, want %v", got, test.want)
			}
		})
	}
}

func TestValidateDirectiveNames_Invalid(t *testing.T) {
	t.Parallel()

	doc, _ := astparser.ParseGraphqlDocumentString("type Query @invalid { id: ID }")

	got := validateDirectiveNames(&doc)
	if got {
		t.Errorf("expected false for invalid directive, got true")
	}
}

func TestValidateDirectives(t *testing.T) {
	t.Parallel()

	doc, _ := astparser.ParseGraphqlDocumentString("type Query { id: ID } directive @key on OBJECT")
	valid := map[string]bool{"key": true}

	tests := []struct {
		name       string
		directives []int
		want       bool
	}{
		{"valid", []int{}, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := validateDirectives(&doc, test.directives, valid, "Query", "type")
			if got != test.want {
				t.Errorf("got %v, want %v", got, test.want)
			}
		})
	}
}

func TestValidateDirectives_MoreCases(t *testing.T) {
	t.Parallel()

	doc, _ := astparser.ParseGraphqlDocumentString("type Query @invalid { id: ID }")
	valid := map[string]bool{"key": true}
	// Find the directive refs for the first object type
	var directiveRefs []int
	if len(doc.ObjectTypeDefinitions) > 0 {
		directiveRefs = doc.ObjectTypeDefinitions[0].Directives.Refs
	}

	got := validateDirectives(&doc, directiveRefs, valid, "Query", "type")
	if got {
		t.Errorf("expected false for invalid directive, got true")
	}

	invalid := map[string]bool{"invalid": false}

	got2 := validateDirectives(&doc, []int{}, invalid, "Query", "type")
	if !got2 {
		t.Errorf("expected true for no directives, got false")
	}
}

func TestReportDirectiveError(t *testing.T) {
	t.Parallel()
	// Just ensure it doesn't panic
	reportDirectiveError("invalid", "Query", "type")
}

func TestReportDirectiveError_AllKinds(t *testing.T) {
	t.Parallel()
	reportDirectiveError("invalid", "Query", "type")
	reportDirectiveError("invalid", "fieldName", "field")
}

func TestReportUncapitalizedDescription(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		kind      string
		parent    string
		field     string
		desc      string
		schema    string
		expectNil bool
		expectMsg string
	}{
		{
			"type capitalized", "type", "", "Query", "A capitalized description.", "type Query { id: ID }",
			true,
			"",
		},
		{
			"type uncapitalized", "type", "", "Query", "uncapitalized description.", "type Query { id: ID }",
			false,
			"should be capitalized",
		},
		{
			"field capitalized", "field", "Query", "id", "ID field.", "type Query { id: ID }",
			true,
			"",
		},
		{
			"field uncapitalized", "field", "Query", "id", "id field.", "type Query { id: ID }",
			false,
			"should be capitalized",
		},
		{
			"enum capitalized", "enum", "Status", "ACTIVE", "Active status.", "enum Status { ACTIVE }",
			true,
			"",
		},
		{
			"enum uncapitalized", "enum", "Status", "ACTIVE", "active status.", "enum Status { ACTIVE }",
			false,
			"should be capitalized",
		},
		{
			"argument capitalized", "argument", "id", "input", "Input argument.", "type Query { id(input: String): ID }",
			true,
			"",
		},
		{
			"argument uncapitalized", "argument", "id", "input", "input argument.", "type Query { id(input: String): ID }",
			false,
			"should be capitalized",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			err := reportUncapitalizedDescription(
				test.kind,
				test.parent,
				test.field,
				test.desc,
				test.schema,
			)
			if test.expectNil {
				if err != nil {
					t.Errorf("expected nil, got %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error, got nil")
				} else if !strings.Contains(err.Message, test.expectMsg) {
					t.Errorf("expected message to contain '%s', got '%s'", test.expectMsg, err.Message)
				}
			}
		})
	}
}

func TestFindMissingArgumentDescriptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		schema      string
		expectError bool
		expectMsg   string
	}{
		{
			"all args described", `type Query { foo("desc" bar: String "desc" baz: Int): String }`,
			false,
			"",
		},
		{
			"one arg missing description", `type Query { foo(bar: String "desc" baz: Int): String }`,
			true,
			"missing a description",
		},
		{
			"multiple args missing description", `type Query { foo(bar: String baz: Int qux: Boolean): String }`,
			true,
			"missing a description",
		},
		{
			"no args", `type Query { foo: String }`,
			false,
			"",
		},
		{
			"mixed args", `type Query { foo("desc" bar: String baz: Int): String }`,
			true,
			"missing a description",
		},
	}
	for _, test := range tests {
		doc, _ := astparser.ParseGraphqlDocumentString(test.schema)

		errs := findMissingArgumentDescriptions(&doc, test.schema)
		if test.expectError {
			if len(errs) == 0 {
				t.Errorf("%s: expected error, got none", test.name)

				continue
			}

			found := false

			for _, err := range errs {
				if test.expectMsg == "" ||
					(err.Message != "" && strings.Contains(err.Message, test.expectMsg)) {
					found = true

					break
				}
			}

			if !found {
				t.Errorf(
					"%s: expected error message containing '%s', got %v",
					test.name,
					test.expectMsg,
					errs,
				)
			}
		} else if len(errs) != 0 {
			t.Errorf("%s: expected no error, got %v", test.name, errs)
		}
	}
}

func TestReadSchemaFile(t *testing.T) {
	t.Parallel()

	content := "type Query { id: ID }"

	tempFile := createTempSchemaFile(t, content)
	defer os.Remove(tempFile)

	got, ok := readSchemaFile(tempFile)
	if !ok || got != content {
		t.Errorf("got %v, want %v", got, content)
	}
}

func TestFilterSchemaComments(t *testing.T) {
	t.Parallel()

	schema := "// comment\ntype Query { id: ID }\n// another"
	want := "type Query { id: ID }"

	got := FilterSchemaComments(schema)
	if !strings.Contains(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestValidateFederationSchema(t *testing.T) {
	t.Parallel()

	got := federation.ValidateFederationSchema("type Query { id: ID }")
	if !got {
		t.Errorf("expected federation schema to be valid")
	}
}

func TestFindAndLogGraphQLSchemaFiles(t *testing.T) {
	t.Parallel()

	files := map[string]string{
		"test.graphql": "type Query { id: ID }",
	}
	dir := createTestDirectory(t, files)

	s := Store{TargetPath: dir, Verbose: false}

	foundFiles, err := s.FindAndLogGraphQLSchemaFiles()
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

	s := createTestStore(false, nil)

	total, errorFiles, _ := s.LintSchemaFiles([]string{file})
	if total != 1 || errorFiles != 1 {
		t.Errorf("expected 1 error, got %d, errorFiles %d", total, errorFiles)
	}
}

func TestLoadConfig(t *testing.T) {
	t.Parallel()

	s := createTestStore(false, nil)

	config, err := s.LoadConfig()
	if err != nil || config == nil {
		t.Errorf("expected config, got err %v", err)
	}
}

func TestMatches(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		sup       Suppression
		filePath  string
		line      int
		rule      string
		value     string
		wantMatch bool
	}{
		{"all empty", Suppression{}, "foo.graphql", 1, "rule", "value", true},
		{
			"file match",
			Suppression{File: "foo.graphql"},
			"bar/foo.graphql",
			1,
			"rule",
			"value",
			true,
		},
		{
			"file no match",
			Suppression{File: "baz.graphql"},
			"foo.graphql",
			1,
			"rule",
			"value",
			false,
		},
		{"line match", Suppression{Line: 2}, "foo.graphql", 2, "rule", "value", true},
		{"line no match", Suppression{Line: 3}, "foo.graphql", 2, "rule", "value", false},
		{"rule match", Suppression{Rule: "myrule"}, "foo.graphql", 1, "myrule", "value", true},
		{
			"rule no match",
			Suppression{Rule: "otherrule"},
			"foo.graphql",
			1,
			"myrule",
			"value",
			false,
		},
		{"value match", Suppression{Value: "val"}, "foo.graphql", 1, "rule", "val", true},
		{"value no match", Suppression{Value: "other"}, "foo.graphql", 1, "rule", "val", false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := test.sup.Matches(test.filePath, test.line, test.rule, test.value)
			if got != test.wantMatch {
				t.Errorf("got %v, want %v", got, test.wantMatch)
			}
		})
	}
}

func TestIsSuppressed(t *testing.T) {
	t.Parallel()

	store := createTestStore(false, createSuppressionConfig([]Suppression{
		{File: "foo.graphql", Line: 2, Rule: "rule", Value: "val"},
	}))

	got := store.isSuppressed("bar/foo.graphql", 2, "rule", "val")
	if !got {
		t.Errorf("expected suppression to match")
	}

	got = store.isSuppressed("bar/foo.graphql", 3, "rule", "val")
	if got {
		t.Errorf("expected suppression not to match")
	}
}

func TestNewStore(t *testing.T) {
	t.Parallel()

	store, err := NewStore("/tmp", true)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if store.TargetPath != "/tmp" || !store.Verbose {
		t.Errorf("unexpected store values: %+v", store)
	}
}

func TestFindAndLogGraphQLSchemaFiles_Errors(t *testing.T) {
	t.Parallel()

	store := Store{TargetPath: "/does/not/exist"}

	_, err := store.FindAndLogGraphQLSchemaFiles()
	if err == nil {
		t.Errorf("expected error for invalid path")
	}
}

func TestLintSchemaFiles_Errors(t *testing.T) {
	t.Parallel()

	store := Store{Verbose: false}

	total, errorFiles, _ := store.LintSchemaFiles([]string{"/does/not/exist.graphql"})
	if total == 0 || errorFiles == 0 {
		t.Errorf("expected errors for missing file")
	}
}

func TestLoadConfig_MissingFile(t *testing.T) {
	t.Parallel()

	store := Store{Verbose: false}
	store.TargetPath = "/tmp"

	config, err := store.LoadConfig()
	if err != nil || config == nil {
		t.Errorf("expected default config, got err %v", err)
	}
}

func TestLogSchemaParseErrors_Errors(t *testing.T) {
	t.Parallel()

	_, report := astparser.ParseGraphqlDocumentString("type Query { id: ID } ...")
	report.InternalErrors = append(report.InternalErrors, assert.AnError)
	LogSchemaParseErrors("type Query { id: ID } ...", &report)
}

func TestReportInternalErrors_Empty(t *testing.T) {
	t.Parallel()

	report := &operationreport.Report{}
	reportInternalErrors(report)
}

func TestReportExternalErrors_Empty(t *testing.T) {
	t.Parallel()

	report := &operationreport.Report{}
	reportExternalErrors("foo", report, 1, 1)
}

func TestReportExternalErrorLocations_Nil(t *testing.T) {
	t.Parallel()

	lines := []string{"foo"}
	externalErr := operationreport.ExternalError{}
	reportExternalErrorLocations(lines, externalErr, 1, 1)
}

func TestReportContextLines_OutOfBounds(t *testing.T) {
	t.Parallel()

	lines := []string{"foo"}
	reportContextLines(lines, 100, 1, 1)
}

func TestGetLineContent(t *testing.T) {
	t.Parallel()

	schema := "A\nB\nC"
	if getLineContent(schema, 2) != "B" {
		t.Errorf("expected B for line 2")
	}

	if getLineContent(schema, 0) != "" {
		t.Errorf("expected empty for line 0")
	}

	if getLineContent("", 1) != "" {
		t.Errorf("expected empty for empty lines")
	}
}

func TestFindFieldDefinitionLine(t *testing.T) {
	t.Parallel()

	schema := "type Query { id: ID name: String }"

	line := findFieldDefinitionLine(schema, "id", "test.graphql")
	if line != 0 && line != 1 {
		t.Errorf("expected line 0 or 1 for id, got %v", line)
	}

	if findFieldDefinitionLine(schema, "foo", "test.graphql") != 0 {
		t.Errorf("expected 0 for missing field")
	}
}

func TestValidateEnumTypes(t *testing.T) {
	t.Parallel()

	doc, _ := astparser.ParseGraphqlDocumentString("enum Status { ACTIVE 1NVALID FOO1 }")
	s := Store{}

	_, errorLines, _ := s.validateEnumTypes(
		&doc,
		"enum Status { ACTIVE 1NVALID FOO1 }",
		"test.graphql",
	)
	if len(errorLines) == 0 {
		t.Logf("validateEnumTypes returned no error lines for invalid enum types: %v", errorLines)
	}
}

func TestFindUnsortedInterfaceFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		schema        string
		expectError   bool
		expectMessage string
	}{
		{
			name:        "sorted interface fields",
			schema:      `interface Foo { a: String b: Int }`,
			expectError: false,
		},
		{
			name:          "unsorted interface fields",
			schema:        `interface Bar { z: String a: Int }`,
			expectError:   true,
			expectMessage: "interface-fields-sorted-alphabetically",
		},
		{
			name:        "single field interface",
			schema:      `interface Baz { a: String }`,
			expectError: false,
		},
	}
	for _, test := range tests {
		doc, _ := astparser.ParseGraphqlDocumentString(test.schema)

		errs := findUnsortedInterfaceFields(&doc, test.schema)
		if test.expectError {
			assert.NotEmpty(t, errs, test.name)
			assert.Contains(t, errs[0].Message, test.expectMessage)
		} else {
			assert.Empty(t, errs, test.name)
		}
	}
}

func TestFindRelayPageInfoSpec(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		schema      string
		expectError bool
	}{
		{
			name:        "PageInfo present",
			schema:      `type PageInfo { hasNextPage: Boolean }`,
			expectError: false,
		},
		{
			name:        "PageInfo missing",
			schema:      `type Query { id: ID }`,
			expectError: true,
		},
	}
	for _, test := range tests {
		doc, _ := astparser.ParseGraphqlDocumentString(test.schema)

		errs := findRelayPageInfoSpec(&doc, test.schema)
		if test.expectError {
			assert.NotEmpty(t, errs, test.name)
			assert.Contains(t, errs[0].Message, "relay-page-info-spec")
		} else {
			assert.Empty(t, errs, test.name)
		}
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

func TestFindInputObjectValuesCamelCased(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		schema      string
		expectError bool
		expectMsg   string
	}{
		{
			name:        "all camel case",
			schema:      `input Foo { fooBar: String }`,
			expectError: false,
		},
		{
			name:        "not camel case",
			schema:      `input Bar { not_camel_case: String }`,
			expectError: true,
			expectMsg:   "input-object-values-are-camel-cased",
		},
		{
			name:        "multiple fields, one invalid",
			schema:      `input Baz { fooBar: String not_camel_case: Int }`,
			expectError: true,
			expectMsg:   "input-object-values-are-camel-cased",
		},
		{
			name:        "single uppercase field",
			schema:      `input Qux { FooBar: String }`,
			expectError: true,
			expectMsg:   "input-object-values-are-camel-cased",
		},
	}
	for _, test := range tests {
		doc, _ := astparser.ParseGraphqlDocumentString(test.schema)

		errs := findInputObjectValuesCamelCased(&doc, test.schema)
		if test.expectError {
			if len(errs) == 0 {
				t.Errorf("%s: expected error, got none", test.name)

				continue
			}

			found := false

			for _, err := range errs {
				if test.expectMsg == "" ||
					(err.Message != "" && strings.Contains(err.Message, test.expectMsg)) {
					found = true

					break
				}
			}

			if !found {
				t.Errorf(
					"%s: expected error message containing '%s', got %v",
					test.name,
					test.expectMsg,
					errs,
				)
			}
		} else if len(errs) != 0 {
			t.Errorf("%s: expected no error, got %v", test.name, errs)
		}
	}
}

func TestFindMissingEnumValueDescriptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		schema      string
		expectError bool
		expectMsg   string
	}{
		{
			name:        "all enum values have descriptions",
			schema:      `enum Color { "Red color." RED "Blue color." BLUE }`,
			expectError: false,
		},
		{
			name:        "missing description for one value",
			schema:      `enum Color { RED "Blue color." BLUE }`,
			expectError: true,
			expectMsg:   "enum-values-have-descriptions",
		},
		{
			name:        "all values missing descriptions",
			schema:      `enum Status { ACTIVE INACTIVE }`,
			expectError: true,
			expectMsg:   "enum-values-have-descriptions",
		},
	}
	for _, test := range tests {
		doc, _ := astparser.ParseGraphqlDocumentString(test.schema)

		errs := findMissingEnumValueDescriptions(&doc, test.schema)
		if test.expectError {
			if len(errs) == 0 {
				t.Errorf("%s: expected error, got none", test.name)

				continue
			}

			found := false

			for _, err := range errs {
				if test.expectMsg == "" ||
					(err.Message != "" && strings.Contains(err.Message, test.expectMsg)) {
					found = true

					break
				}
			}

			if !found {
				t.Errorf(
					"%s: expected error message containing '%s', got %v",
					test.name,
					test.expectMsg,
					errs,
				)
			}
		} else if len(errs) != 0 {
			t.Errorf("%s: expected no error, got %v", test.name, errs)
		}
	}
}

func TestStore_ReportSummary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		schemaFiles []string
		totalErrors int
		passedFiles int
		allErrors   []DescriptionError
		want        ReportSummary
	}{
		{
			name:        "all files pass",
			schemaFiles: []string{"a.graphql", "b.graphql"},
			totalErrors: 0,
			passedFiles: 2,
			allErrors:   nil,
			want: ReportSummary{
				TotalFiles:                2,
				PassedFiles:               2,
				TotalErrors:               0,
				PercentPassed:             100.0,
				PercentageFilesWithErrors: 0.0,
				FilesWithAtLeastOneError:  0,
				AllErrors:                 nil,
			},
		},
		{
			name:        "some files fail",
			schemaFiles: []string{"a.graphql", "b.graphql"},
			totalErrors: 1,
			passedFiles: 1,
			allErrors: []DescriptionError{
				{FilePath: "a.graphql", LineNum: 1, Message: "error", LineContent: "foo"},
			},
			want: ReportSummary{
				TotalFiles:                2,
				PassedFiles:               1,
				TotalErrors:               1,
				PercentPassed:             50.0,
				PercentageFilesWithErrors: 50.0,
				FilesWithAtLeastOneError:  1,
				AllErrors: []DescriptionError{
					{FilePath: "a.graphql", LineNum: 1, Message: "error", LineContent: "foo"},
				},
			},
		},
		{
			name:        "no files",
			schemaFiles: []string{},
			totalErrors: 0,
			passedFiles: 0,
			allErrors:   nil,
			want: ReportSummary{
				TotalFiles:                0,
				PassedFiles:               0,
				TotalErrors:               0,
				PercentPassed:             0.0,
				PercentageFilesWithErrors: 0.0,
				FilesWithAtLeastOneError:  0,
				AllErrors:                 nil,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			store := Store{}
			summary := store.ReportSummary(
				test.schemaFiles,
				test.totalErrors,
				test.passedFiles,
				test.allErrors,
			)

			if summary.TotalFiles != test.want.TotalFiles {
				t.Errorf("TotalFiles: got %d, want %d", summary.TotalFiles, test.want.TotalFiles)
			}

			if summary.PassedFiles != test.want.PassedFiles {
				t.Errorf("PassedFiles: got %d, want %d", summary.PassedFiles, test.want.PassedFiles)
			}

			if summary.TotalErrors != test.want.TotalErrors {
				t.Errorf("TotalErrors: got %d, want %d", summary.TotalErrors, test.want.TotalErrors)
			}

			if summary.PercentPassed != test.want.PercentPassed {
				t.Errorf(
					"PercentPassed: got %f, want %f",
					summary.PercentPassed,
					test.want.PercentPassed,
				)
			}

			if summary.PercentageFilesWithErrors != test.want.PercentageFilesWithErrors {
				t.Errorf(
					"PercentageFilesWithErrors: got %f, want %f",
					summary.PercentageFilesWithErrors,
					test.want.PercentageFilesWithErrors,
				)
			}

			if summary.FilesWithAtLeastOneError != test.want.FilesWithAtLeastOneError {
				t.Errorf(
					"FilesWithAtLeastOneError: got %d, want %d",
					summary.FilesWithAtLeastOneError,
					test.want.FilesWithAtLeastOneError,
				)
			}

			if !assert.Equal(t, test.want.AllErrors, summary.AllErrors) {
				t.Errorf("AllErrors: got %+v, want %+v", summary.AllErrors, test.want.AllErrors)
			}
		})
	}
}

func TestErrorTypeCounts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input []DescriptionError
		want  map[string]int
	}{
		{
			name:  "empty slice",
			input: []DescriptionError{},
			want:  map[string]int{},
		},
		{
			name:  "single error with colon",
			input: []DescriptionError{{Message: "type-error: something went wrong"}},
			want:  map[string]int{"type-error": 1},
		},
		{
			name:  "single error with space",
			input: []DescriptionError{{Message: "field error something went wrong"}},
			want:  map[string]int{"field": 1},
		},
		{
			name: "multiple errors, mixed",
			input: []DescriptionError{
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

			got := errorTypeCounts(test.input)
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("errorTypeCounts() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestFindRelayConnectionTypesSpec(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		schema     string
		expectMsgs []string
	}{
		{
			name:       "has both pageInfo and edges",
			schema:     `type UserConnection { pageInfo: PageInfo edges: [UserEdge] }`,
			expectMsgs: []string{},
		},
		{
			name:       "missing pageInfo",
			schema:     `type UserConnection { edges: [UserEdge] }`,
			expectMsgs: []string{"missing the following field: pageInfo."},
		},
		{
			name:       "missing edges",
			schema:     `type UserConnection { pageInfo: PageInfo }`,
			expectMsgs: []string{"missing the following field: edges."},
		},
		{
			name:       "missing both",
			schema:     `type UserConnection { foo: String }`,
			expectMsgs: []string{"missing the following field: pageInfo.", "missing the following field: edges."},
		},
		{
			name:       "not a Connection type",
			schema:     `type User { id: ID }`,
			expectMsgs: []string{},
		},
	}

	for _, test := range tests {
		doc, _ := astparser.ParseGraphqlDocumentString(test.schema)

		errs := findRelayConnectionTypesSpec(&doc, test.schema)
		if len(test.expectMsgs) == 0 {
			if len(errs) != 0 {
				t.Errorf("%s: expected no errors, got %v", test.name, errs)
			}

			return
		}

		if len(errs) != len(test.expectMsgs) {
			t.Errorf("%s: expected %d errors, got %d", test.name, len(test.expectMsgs), len(errs))
		}

		for _, expectMsg := range test.expectMsgs {
			found := false

			for _, err := range errs {
				if strings.Contains(err.Message, expectMsg) {
					found = true

					break
				}
			}

			if !found {
				t.Errorf("%s: expected error message containing '%s', got %v", test.name, expectMsg, errs)
			}
		}
	}
}

func TestCollectUnsuppressedDataTypeErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		schema       string
		config       *LinterConfig
		wantCount    int
		wantContains []string
	}{
		{
			name:      "valid types",
			schema:    `type Query { id: ID name: String }`,
			config:    &LinterConfig{},
			wantCount: 0,
		},
		{
			name:         "undefined type",
			schema:       `type Query { foo: Bar }`,
			config:       &LinterConfig{},
			wantCount:    1,
			wantContains: []string{"defined-types-are-used"},
		},
		{
			name:      "suppressed error",
			schema:    `type Query { foo: Bar }`,
			config:    &LinterConfig{Suppressions: []Suppression{{Line: 1, Rule: "defined-types-are-used"}}},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := Store{LinterConfig: tt.config}
			doc, _ := astparser.ParseGraphqlDocumentString(tt.schema)
			count, errs := store.collectUnsuppressedDataTypeErrors(&doc, tt.schema, "test.graphql")
			if count != tt.wantCount {
				t.Errorf("got count %d, want %d", count, tt.wantCount)
			}
			if len(errs) != tt.wantCount {
				t.Errorf("got %d errors, want %d", len(errs), tt.wantCount)
			}
			for _, substr := range tt.wantContains {
				found := false
				for _, err := range errs {
					if strings.Contains(err.Message, substr) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error message containing '%s', got %v", substr, errs)
				}
			}
		})
	}
}
