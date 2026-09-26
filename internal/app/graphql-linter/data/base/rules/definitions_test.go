package rules

import (
	"testing"

	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/data/base/models"
	"github.com/stretchr/testify/assert"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/astparser"
)

// Every kind lacks descriptions; "id" appears in the text before its field so a
// text search would report the wrong line.
const allKindsSchema = `# id
interface Node {
  id: ID! @deprecated
}
enum Color {
  red
}
input Filter {
  id: ID @deprecated
}
union Result = Item
scalar Date
"""item."""
type Item implements Node {
  id: ID!
  items(first: Int @deprecated): [Item]
}
`

type finding struct {
	Line  int
	Value string
	Msg   string
}

func findings(errs []models.DescriptionError) []finding {
	out := make([]finding, 0, len(errs))
	for _, err := range errs {
		out = append(out, finding{err.LineNum, err.Value, err.Message})
	}

	return out
}

func TestRulesCoverAllKinds(t *testing.T) {
	t.Parallel()

	doc, report := astparser.ParseGraphqlDocumentString(allKindsSchema)
	assert.False(t, report.HasErrors(), report.Error())

	rule := Rule{}

	assert.Equal(t, []finding{
		{2, "Node", "types-have-descriptions: Interface 'Node' is missing a description"},
		{8, "Filter", "types-have-descriptions: Input object type 'Filter' is missing a description"},
		{5, "Color", "types-have-descriptions: Enum 'Color' is missing a description"},
		{11, "Result", "types-have-descriptions: Union 'Result' is missing a description"},
		{12, "Date", "types-have-descriptions: Scalar 'Date' is missing a description"},
	}, findings(rule.MissingTypeDescriptions(&doc, allKindsSchema)))

	assert.Equal(t, []finding{
		{15, "id", "fields-have-descriptions: Field 'Item.id' is missing a description."},
		{16, "items", "fields-have-descriptions: Field 'Item.items' is missing a description."},
		{3, "id", "fields-have-descriptions: Field 'Node.id' is missing a description."},
	}, findings(rule.MissingFieldDescriptions(&doc, allKindsSchema)))

	assert.Equal(t, []finding{
		{16, "first", "deprecations-have-a-reason: Deprecated argument 'Item.items.first' is missing a reason."},
		{3, "id", "deprecations-have-a-reason: Deprecated field 'Node.id' is missing a reason."},
		{9, "id", "deprecations-have-a-reason: Deprecated input value 'Filter.id' is missing a reason."},
	}, findings(rule.MissingDeprecationReasons(&doc, allKindsSchema)))

	assert.Equal(t, []finding{
		{6, "red", "enum-values-all-caps: The enum value `Color.red` should be uppercase."},
	}, findings(rule.EnumValuesAllCaps(&doc, allKindsSchema)))

	assert.Equal(t, []finding{
		{14, "Item", "descriptions-are-capitalized: The description for type `Item` should be capitalized."},
	}, findings(rule.UncapitalizedDescriptions(&doc, allKindsSchema)))

	// Node is used through "implements", Item through the union and a field.
	assert.Equal(t, []finding{
		{8, "Filter", "defined-types-are-used: Type 'Filter' is defined but not used"},
		{5, "Color", "defined-types-are-used: Type 'Color' is defined but not used"},
		{11, "Result", "defined-types-are-used: Type 'Result' is defined but not used"},
		{12, "Date", "defined-types-are-used: Type 'Date' is defined but not used"},
	}, findings(rule.UnusedTypes(&doc, allKindsSchema)))
}

func TestLineOf(t *testing.T) {
	t.Parallel()

	doc, _ := astparser.ParseGraphqlDocumentString("\n\ntype Query {\n  a: String\n}\n")

	assert.Equal(t, 3, LineOf(&doc, doc.ObjectTypeDefinitions[0].Name))
	assert.Equal(t, 4, LineOf(&doc, doc.FieldDefinitions[0].Name))
}
