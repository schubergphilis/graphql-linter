package data

import (
	"fmt"

	"github.com/wundergraph/graphql-go-tools/v2/pkg/ast"
)

type EnumValue struct {
	Name        string
	Description string
}

type InputField struct {
	Name        string
	Type        string
	Description string
}

func NewDocument() *ast.Document {
	return ast.NewDocument()
}

func addDescription(doc *ast.Document, description string) ast.Description {
	if description == "" {
		return ast.Description{}
	}

	ref := doc.Input.AppendInputString(description)

	return ast.Description{
		IsDefined: true,
		Content:   ref,
	}
}

func AddEnum(doc *ast.Document, name, description string, values []EnumValue) int {
	nameRef := doc.Input.AppendInputString(name)
	enumDef := ast.EnumTypeDefinition{
		Name:        nameRef,
		Description: addDescription(doc, description),
	}
	idx := len(doc.EnumTypeDefinitions)
	doc.EnumTypeDefinitions = append(doc.EnumTypeDefinitions, enumDef)

	for _, v := range values {
		valRef := doc.Input.AppendInputString(v.Name)
		valueDef := ast.EnumValueDefinition{
			EnumValue:   valRef,
			Description: addDescription(doc, v.Description),
		}
		valIdx := len(doc.EnumValueDefinitions)
		doc.EnumValueDefinitions = append(doc.EnumValueDefinitions, valueDef)
		doc.EnumTypeDefinitions[idx].EnumValuesDefinition.Refs = append(
			doc.EnumTypeDefinitions[idx].EnumValuesDefinition.Refs, valIdx)
	}

	typeDef := ast.Type{
		TypeKind: ast.TypeKindNamed,
		Name:     nameRef,
	}
	doc.Types = append(doc.Types, typeDef)

	return idx
}

func AddObject(doc *ast.Document, name, description string) int {
	nameRef := doc.Input.AppendInputString(name)
	objDef := ast.ObjectTypeDefinition{
		Name:        nameRef,
		Description: addDescription(doc, description),
	}
	idx := len(doc.ObjectTypeDefinitions)
	doc.ObjectTypeDefinitions = append(doc.ObjectTypeDefinitions, objDef)

	typeDef := ast.Type{
		TypeKind: ast.TypeKindNamed,
		Name:     nameRef,
	}
	doc.Types = append(doc.Types, typeDef)

	return idx
}

func AddFieldToObject(doc *ast.Document, objIdx int, fieldName, fieldType, description string) int {
	nameRef := doc.Input.AppendInputString(fieldName)
	typeRef := doc.Input.AppendInputString(fieldType)

	typeDef := ast.Type{
		TypeKind: ast.TypeKindNamed,
		Name:     typeRef,
	}
	typeIdx := len(doc.Types)
	doc.Types = append(doc.Types, typeDef)

	fieldDef := ast.FieldDefinition{
		Name:        nameRef,
		Description: addDescription(doc, description),
		Type:        typeIdx,
	}
	fieldIdx := len(doc.FieldDefinitions)
	doc.FieldDefinitions = append(doc.FieldDefinitions, fieldDef)

	doc.ObjectTypeDefinitions[objIdx].FieldsDefinition.Refs = append(
		doc.ObjectTypeDefinitions[objIdx].FieldsDefinition.Refs, fieldIdx)

	return fieldIdx
}

func addComplexField(doc *ast.Document, objIdx int, fieldName, typeName, description string, kind ast.TypeKind) int {
	nameRef := doc.Input.AppendInputString(fieldName)
	typeRef := doc.Input.AppendInputString(typeName)

	innerTypeDef := ast.Type{
		TypeKind: ast.TypeKindNamed,
		Name:     typeRef,
	}
	innerTypeIdx := len(doc.Types)
	doc.Types = append(doc.Types, innerTypeDef)

	complexTypeDef := ast.Type{
		TypeKind: kind,
		OfType:   innerTypeIdx,
	}
	complexTypeIdx := len(doc.Types)
	doc.Types = append(doc.Types, complexTypeDef)

	fieldDef := ast.FieldDefinition{
		Name:        nameRef,
		Description: addDescription(doc, description),
		Type:        complexTypeIdx,
	}
	fieldIdx := len(doc.FieldDefinitions)
	doc.FieldDefinitions = append(doc.FieldDefinitions, fieldDef)
	doc.ObjectTypeDefinitions[objIdx].FieldsDefinition.Refs = append(
		doc.ObjectTypeDefinitions[objIdx].FieldsDefinition.Refs, fieldIdx)

	return fieldIdx
}

func AddListFieldToObject(doc *ast.Document, objIdx int, fieldName, elementType, description string) int {
	return addComplexField(doc, objIdx, fieldName, elementType, description, ast.TypeKindList)
}

func AddNonNullFieldToObject(doc *ast.Document, objIdx int, fieldName, fieldType, description string) int {
	return addComplexField(doc, objIdx, fieldName, fieldType, description, ast.TypeKindNonNull)
}

func AddInputObject(doc *ast.Document, name, description string, fields []InputField) int {
	nameRef := doc.Input.AppendInputString(name)
	inputDef := ast.InputObjectTypeDefinition{
		Name:        nameRef,
		Description: addDescription(doc, description),
	}
	idx := len(doc.InputObjectTypeDefinitions)
	doc.InputObjectTypeDefinitions = append(doc.InputObjectTypeDefinitions, inputDef)

	for _, field := range fields {
		fieldNameRef := doc.Input.AppendInputString(field.Name)
		typeRef := doc.Input.AppendInputString(field.Type)

		typeDef := ast.Type{
			TypeKind: ast.TypeKindNamed,
			Name:     typeRef,
		}
		typeIdx := len(doc.Types)
		doc.Types = append(doc.Types, typeDef)

		inputValueDef := ast.InputValueDefinition{
			Name:        fieldNameRef,
			Description: addDescription(doc, field.Description),
			Type:        typeIdx,
		}
		fieldIdx := len(doc.InputValueDefinitions)
		doc.InputValueDefinitions = append(doc.InputValueDefinitions, inputValueDef)
		doc.InputObjectTypeDefinitions[idx].InputFieldsDefinition.Refs = append(
			doc.InputObjectTypeDefinitions[idx].InputFieldsDefinition.Refs, fieldIdx)
	}

	return idx
}

const UnknownType = "Unknown"

func GenerateGraphQLFromDocument(doc *ast.Document) string {
	var result string

	for _, enum := range doc.EnumTypeDefinitions {
		enumName := doc.Input.ByteSliceString(enum.Name)
		desc := getDescription(doc, enum.Description)
		if desc != "" {
			result += desc + "\n"
		}
		result += fmt.Sprintf("enum %s {\n", enumName)
		for _, valueRef := range enum.EnumValuesDefinition.Refs {
			value := doc.EnumValueDefinitions[valueRef]
			valueName := doc.Input.ByteSliceString(value.EnumValue)
			valueDesc := getDescription(doc, value.Description)
			if valueDesc != "" {
				result += fmt.Sprintf("  %s\n", valueDesc)
			}
			result += fmt.Sprintf("  %s\n", valueName)
		}
		result += "}\n\n"
	}

	for _, obj := range doc.ObjectTypeDefinitions {
		objName := doc.Input.ByteSliceString(obj.Name)
		desc := getDescription(doc, obj.Description)
		if desc != "" {
			result += desc + "\n"
		}
		result += fmt.Sprintf("type %s {\n", objName)
		for _, fieldRef := range obj.FieldsDefinition.Refs {
			field := doc.FieldDefinitions[fieldRef]
			fieldName := doc.Input.ByteSliceString(field.Name)
			fieldDesc := getDescription(doc, field.Description)
			fieldType := getFieldType(doc, field.Type)
			if fieldDesc != "" {
				result += fmt.Sprintf("  %s\n", fieldDesc)
			}
			result += fmt.Sprintf("  %s: %s\n", fieldName, fieldType)
		}
		result += "}\n\n"
	}

	return result
}

func getDescription(doc *ast.Document, desc ast.Description) string {
	if !desc.IsDefined {
		return ""
	}

	content := doc.Input.ByteSliceString(desc.Content)

	return fmt.Sprintf(`"""%s"""`, content)
}

func getFieldType(doc *ast.Document, typeIndex int) string {
	if typeIndex >= len(doc.Types) {
		return UnknownType
	}

	typeInfo := doc.Types[typeIndex]
	switch typeInfo.TypeKind {
	case ast.TypeKindNamed:
		return doc.Input.ByteSliceString(typeInfo.Name)
	case ast.TypeKindNonNull:
		innerType := getFieldType(doc, typeInfo.OfType)

		return innerType + "!"
	case ast.TypeKindList:
		innerType := getFieldType(doc, typeInfo.OfType)

		return "[" + innerType + "]"
	case ast.TypeKindUnknown:
		return UnknownType
	default:
		return UnknownType
	}
}
