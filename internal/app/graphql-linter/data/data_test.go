package data

import (
	"os"
	"strings"
	"testing"

	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/data/base/models"
	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/data/base/rules"
	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/data/federation"
	"github.com/stretchr/testify/assert"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/astparser"
)

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

func TestNewStore(t *testing.T) {
	t.Parallel()

	store, err := NewStore("", "/tmp", rules.Rule{}, true)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if store.TargetPath != "/tmp" || !store.Verbose {
		t.Errorf("unexpected store values: %+v", store)
	}
}

func TestFindUnsortedInterfaceFields(t *testing.T) {
	t.Parallel()

	store, err := NewStore("", "/tmp", rules.Rule{}, true)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

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

		errs := store.UnsortedInterfaceFields(&doc, test.schema)
		if test.expectError {
			assert.NotEmpty(t, errs, test.name)
			assert.Contains(t, errs[0].Message, test.expectMessage)
		} else {
			assert.Empty(t, errs, test.name)
		}
	}
}

func TestCollectUnsuppressedDataTypeErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		schema       string
		config       *models.LinterConfig
		wantCount    int
		wantContains []string
	}{
		{
			name:      "valid types",
			schema:    `type Query { id: ID name: String }`,
			config:    &models.LinterConfig{},
			wantCount: 0,
		},
		{
			name:         "undefined type",
			schema:       `type Query { foo: Bar }`,
			config:       &models.LinterConfig{},
			wantCount:    1,
			wantContains: []string{"defined-types-are-used"},
		},
		{
			name:   "suppressed error",
			schema: `type Query { foo: Bar }`,
			config: &models.LinterConfig{
				Suppressions: []models.Suppression{{Line: 1, Rule: "defined-types-are-used"}},
			},
			wantCount: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			store := Store{LinterConfig: test.config, Ruler: rules.Rule{}}
			doc, _ := astparser.ParseGraphqlDocumentString(test.schema)

			count, errs := store.CollectUnsuppressedDataTypeErrors(
				&doc,
				test.config,
				test.schema,
				"test.graphql",
			)
			if count != test.wantCount {
				t.Errorf("got count %d, want %d", count, test.wantCount)
			}

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
