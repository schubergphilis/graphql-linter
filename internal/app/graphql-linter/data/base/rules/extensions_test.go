package rules

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/astparser"
)

func TestDescriptionOnExtensions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		schema    string
		wantLines []int
		wantValue []string
		wantText  string
	}{
		{
			name:      "object type",
			schema:    "type Animal { a: String }\n\n\"What.\"\nextend type Animal { b: String }",
			wantLines: []int{3},
			wantValue: []string{"Animal"},
			wantText: "extend-has-no-description: Description not allowed on 'extend type Animal'; " +
				"the description belongs on the definition that is extended",
		},
		{
			name:      "block string and comment in between",
			schema:    "type Animal { a: String }\n\"\"\"\nWhat.\n\"\"\"\n# note\nextend type Animal { b: String }",
			wantLines: []int{2},
			wantValue: []string{"Animal"},
		},
		{
			name: "every extension kind",
			schema: `"I" extend interface Node { id: ID }
"E" extend enum Color { RED }
"In" extend input Filter { id: ID }
"U" extend union Result = Node
"S" extend scalar Date @specifiedBy(url: "https://example.com")
"Sc" extend schema @link(url: "https://specs.apollo.dev/federation/v2.9")`,
			wantLines: []int{1, 2, 3, 4, 5, 6},
			wantValue: []string{"Node", "Color", "Filter", "Result", "Date", "schema"},
		},
		{
			name:   "extension without description",
			schema: "\"Animal.\"\ntype Animal { a: String }\n\nextend type Animal { b: String }",
		},
		{
			name: "string before extend that is not a description",
			schema: `scalar Date @specifiedBy(url: "https://example.com")
extend type Animal @key(fields: "id") { b: String }
extend type Animal { c(arg: String = "x"): String }
extend schema @link(url: "https://specs.apollo.dev/federation/v2.9")`,
		},
		{
			name:   "field named extend with a description",
			schema: "type Animal {\n  \"The extend field.\"\n  extend: String\n}",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			doc, report := astparser.ParseGraphqlDocumentString(test.schema)
			require.False(t, report.HasErrors(), report.Error())

			findings := Rule{}.DescriptionOnExtensions(&doc, test.schema)
			require.Len(t, findings, len(test.wantLines), "%v", findings)

			for i, finding := range findings {
				assert.Equal(t, test.wantLines[i], finding.LineNum)
				assert.Equal(t, test.wantValue[i], finding.Value)
				assert.Contains(t, finding.Message, "extend-has-no-description: ")
			}

			if test.wantText != "" {
				assert.Equal(t, test.wantText, findings[0].Message)
			}
		})
	}
}
