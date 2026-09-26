package rules

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/astparser"
)

func TestValidateDirectiveNames(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		schema string
		want   []string
	}{
		{
			name: "federation v2 and built-in directives",
			schema: `type Query @shareable { me: User @authenticated @requiresScopes(scopes: [["read"]]) }
type User @key(fields: "id") @key(fields: "email") @federation__tag(name: "a") {
  id: ID! @cost(weight: 1) name(first: Int @deprecated(reason: "x")): String @policy(policies: [["p"]])
}
enum Role @inaccessible { ADMIN @tag(name: "t") }
input Filter { id: ID @inaccessible }
scalar Date @specifiedBy(url: "https://example.com")
union Result @tag(name: "r") = User`,
		},
		{
			name: "directive defined in the schema or composed",
			schema: `extend schema @composeDirective(name: "@composed")
directive @custom on FIELD_DEFINITION
type Query { a: String @custom b: String @composed }`,
		},
		{
			name:   "unknown directive with suggestion",
			schema: "type Query @kye(fields: \"id\") { id: ID }",
			want: []string{
				"invalid-federation-directive: Invalid federation directive '@kye' on type 'Query'. Did you mean '@key'?",
			},
		},
		{
			name: "unknown directives on every kind",
			schema: `type Query { a(x: Int @foo): String @foo }
extend type User @foo { id: ID }
interface Node @foo { id: ID }
input In { a: Int @foo }
enum E { A @foo }
union U @foo = Query
scalar S @foo`,
			want: []string{
				"invalid-federation-directive: Invalid federation directive '@foo' on field 'Query.a'.",
				"invalid-federation-directive: Invalid federation directive '@foo' on argument 'Query.a.x'.",
				"invalid-federation-directive: Invalid federation directive '@foo' on type 'User'.",
				"invalid-federation-directive: Invalid federation directive '@foo' on interface 'Node'.",
				"invalid-federation-directive: Invalid federation directive '@foo' on input value 'In.a'.",
				"invalid-federation-directive: Invalid federation directive '@foo' on enum value 'E.A'.",
				"invalid-federation-directive: Invalid federation directive '@foo' on union 'U'.",
				"invalid-federation-directive: Invalid federation directive '@foo' on scalar 'S'.",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			doc, report := astparser.ParseGraphqlDocumentString(test.schema)
			assert.False(t, report.HasErrors(), report.Error())

			var got []string
			for _, finding := range ValidateDirectiveNames(&doc, test.schema) {
				got = append(got, finding.Message)
			}

			assert.ElementsMatch(t, test.want, got)
		})
	}
}

func TestValidateDirectiveNames_LineAndValue(t *testing.T) {
	t.Parallel()

	schema := "type Query {\n  id: ID @foo\n}"
	doc, _ := astparser.ParseGraphqlDocumentString(schema)

	got := ValidateDirectiveNames(&doc, schema)
	assert.Len(t, got, 1)
	assert.Equal(t, 2, got[0].LineNum)
	assert.Equal(t, "foo", got[0].Value)
	assert.Equal(t, "id: ID @foo", got[0].LineContent)
}
