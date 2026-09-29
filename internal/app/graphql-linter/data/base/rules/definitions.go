package rules

import (
	"bytes"

	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/data/base/models"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/ast"
)

const (
	kindObject    = "Object type"
	kindInterface = "Interface"
)

// typeDefinition is the common view of every named type definition, so rules
// can cover all kinds with one loop.
type typeDefinition struct {
	kind        string
	name        string
	nameRef     ast.ByteSliceReference
	description ast.Description
	fields      []int // doc.FieldDefinitions refs (object and interface types)
	inputValues []int // doc.InputValueDefinitions refs (input object types)
	enumValues  []int // doc.EnumValueDefinitions refs (enum types)
}

func typeDefinitions(doc *ast.Document) []typeDefinition {
	defs := make([]typeDefinition, 0, len(doc.RootNodes))

	add := func(kind string, nameRef ast.ByteSliceReference, description ast.Description) *typeDefinition {
		defs = append(defs, typeDefinition{
			kind:        kind,
			name:        doc.Input.ByteSliceString(nameRef),
			nameRef:     nameRef,
			description: description,
		})

		return &defs[len(defs)-1]
	}

	for _, def := range doc.ObjectTypeDefinitions {
		add(kindObject, def.Name, def.Description).fields = def.FieldsDefinition.Refs
	}

	for _, def := range doc.InterfaceTypeDefinitions {
		add(kindInterface, def.Name, def.Description).fields = def.FieldsDefinition.Refs
	}

	for _, def := range doc.InputObjectTypeDefinitions {
		add("Input object type", def.Name, def.Description).inputValues = def.InputFieldsDefinition.Refs
	}

	for _, def := range doc.EnumTypeDefinitions {
		add("Enum", def.Name, def.Description).enumValues = def.EnumValuesDefinition.Refs
	}

	for _, def := range doc.UnionTypeDefinitions {
		add("Union", def.Name, def.Description)
	}

	for _, def := range doc.ScalarTypeDefinitions {
		add("Scalar", def.Name, def.Description)
	}

	return defs
}

// LineOf returns the 1-based line on which ref starts in the document input.
func LineOf(doc *ast.Document, ref ast.ByteSliceReference) int {
	return bytes.Count(doc.Input.RawBytes[:ref.Start], []byte("\n")) + 1
}

func newFinding(schemaString string, lineNum int, value, message string) models.DescriptionError {
	return models.DescriptionError{
		Value:       value,
		LineNum:     lineNum,
		Message:     message,
		LineContent: GetLineContent(schemaString, lineNum),
	}
}

func isRootType(name string) bool {
	return name == rootQueryType || name == rootMutationType || name == rootSubscriptionType
}
