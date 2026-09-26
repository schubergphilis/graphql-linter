package data

import (
	"fmt"
	"strings"
	"testing"

	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/data/base/models"
	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/data/federation"
	"github.com/stretchr/testify/assert"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/astparser"
)

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

	assert.Empty(t, federation.ValidateFederationSchema("type Query { id: ID }"))

	// A subgraph may extend an entity owned by another subgraph and repeat @key.
	assert.Empty(t, federation.ValidateFederationSchema(
		`type Query { me: User } extend type User @key(fields: "id") @key(fields: "email") { id: ID! email: String }`,
	))

	got := federation.ValidateFederationSchema("type Query { a: String }\n\ntype Query {\n  b: Unknown\n}")
	assert.Equal(t, []string{
		"3 invalid-federation-schema: there can be only one type named 'Query'",
		"4 invalid-federation-schema: Unknown type \"Unknown\".",
	}, []string{
		fmt.Sprintf("%d %s", got[0].LineNum, got[0].Message),
		fmt.Sprintf("%d %s", got[1].LineNum, got[1].Message),
	})
}

func TestNewStore(t *testing.T) {
	t.Parallel()

	store := NewStore("", "/tmp")
	if store.TargetPath != "/tmp" {
		t.Errorf("unexpected store values: %+v", store)
	}
}

func TestFindUnsortedInterfaceFields(t *testing.T) {
	t.Parallel()

	store := NewStore("", "/tmp")

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
			name:      "valid enum types",
			schema:    `enum Color { RED GREEN BLUE } type Query { color: Color }`,
			config:    &models.LinterConfig{},
			wantCount: 0,
		},
		{
			name:      "valid types without enum errors",
			schema:    `type Query { foo: String }`,
			config:    &models.LinterConfig{},
			wantCount: 0,
		},
		{
			name:      "schema with no enum description errors",
			schema:    `enum Status { ACTIVE INACTIVE } type Query { status: Status }`,
			config:    &models.LinterConfig{},
			wantCount: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			store := Store{LinterConfig: test.config}
			doc, _ := astparser.ParseGraphqlDocumentString(test.schema)

			count, errs := store.CollectUnsuppressedDataTypeErrors(
				&doc,
				test.config,
				test.schema,
				"test.graphql",
			)

			assert.Equal(t, test.wantCount, count, "error count mismatch for %s", test.name)
			assert.Len(t, errs, test.wantCount, "errors slice length mismatch for %s", test.name)

			for _, substr := range test.wantContains {
				found := false

				for _, err := range errs {
					if strings.Contains(err.Message, substr) {
						found = true

						break
					}
				}

				assert.True(t, found, "expected error message containing '%s' in test %s, got %v", substr, test.name, errs)
			}
		})
	}
}
