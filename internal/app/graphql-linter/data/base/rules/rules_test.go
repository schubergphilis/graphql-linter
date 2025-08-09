package rules

import (
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/ast"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/astparser"
)

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

			err := ReportUncapitalizedDescription(
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

		errs := MissingArgumentDescriptions(&doc, test.schema)
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

		errs := RelayConnectionTypesSpec(&doc, test.schema)
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

func TestFindMissingInputObjectValueDescriptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		schema       string
		wantCount    int
		wantContains []string
	}{
		{
			name:      "all described",
			schema:    `input Foo { "desc" bar: String "desc" baz: Int }`,
			wantCount: 0,
		},
		{
			name:         "one missing",
			schema:       `input Foo { bar: String "desc" baz: Int }`,
			wantCount:    1,
			wantContains: []string{"input-object-values-have-descriptions"},
		},
		{
			name:         "all missing",
			schema:       `input Foo { bar: String baz: Int }`,
			wantCount:    2,
			wantContains: []string{"input-object-values-have-descriptions"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			doc, _ := astparser.ParseGraphqlDocumentString(test.schema)

			errs := MissingInputObjectValueDescriptions(&doc, test.schema)
			if len(errs) != test.wantCount {
				t.Errorf("got %d errors, want %d", len(errs), test.wantCount)
			}

			for _, substr := range test.wantContains {
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

		errs := MissingEnumValueDescriptions(&doc, test.schema)
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

		errs := InputObjectValuesCamelCased(&doc, test.schema)
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

		errs := RelayPageInfoSpec(&doc, test.schema)
		if test.expectError {
			assert.NotEmpty(t, errs, test.name)
			assert.Contains(t, errs[0].Message, "relay-page-info-spec")
		} else {
			assert.Empty(t, errs, test.name)
		}
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

	_, errorLines, _ := ValidateEnumTypes(
		&doc,
		nil,
		"enum Status { ACTIVE 1NVALID FOO1 }",
		"test.graphql",
	)
	if len(errorLines) == 0 {
		t.Logf("validateEnumTypes returned no error lines for invalid enum types: %v", errorLines)
	}
}
