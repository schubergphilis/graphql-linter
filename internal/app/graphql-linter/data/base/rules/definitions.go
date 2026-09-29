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

// DefinitionLines returns, in document order, the lines on which typeName is
// defined, or with memberName set, its fields, input values or enum values.
func DefinitionLines(doc *ast.Document, typeName, memberName string) []int {
	var lines []int

	for _, def := range typeDefinitions(doc) {
		if def.name != typeName {
			continue
		}

		if memberName == "" {
			lines = append(lines, LineOf(doc, def.nameRef))

			continue
		}

		var memberRefs []ast.ByteSliceReference
		for _, ref := range def.fields {
			memberRefs = append(memberRefs, doc.FieldDefinitions[ref].Name)
		}

		for _, ref := range def.inputValues {
			memberRefs = append(memberRefs, doc.InputValueDefinitions[ref].Name)
		}

		for _, ref := range def.enumValues {
			memberRefs = append(memberRefs, doc.EnumValueDefinitions[ref].EnumValue)
		}

		for _, nameRef := range memberRefs {
			if doc.Input.ByteSliceString(nameRef) == memberName {
				lines = append(lines, LineOf(doc, nameRef))
			}
		}
	}

	return lines
}

// ReferenceLines returns the lines of the fields and input values whose type
// is typeName, in document order.
func ReferenceLines(doc *ast.Document, typeName string) []int {
	var lines []int

	for _, fieldDef := range doc.FieldDefinitions {
		if getBaseTypeName(doc, doc.Types[fieldDef.Type]) == typeName {
			lines = append(lines, LineOf(doc, fieldDef.Name))
		}
	}

	for _, inputValue := range doc.InputValueDefinitions {
		if getBaseTypeName(doc, doc.Types[inputValue.Type]) == typeName {
			lines = append(lines, LineOf(doc, inputValue.Name))
		}
	}

	return lines
}
