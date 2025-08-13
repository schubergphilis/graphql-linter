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

type Argument struct {
	Name        string
	Type        string
	Description string
}

type InterfaceField struct {
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

func addComplexField(
	doc *ast.Document,
	objIdx int,
	fieldName, typeName, description string,
	kind ast.TypeKind,
) int {
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

func AddListFieldToObject(
	doc *ast.Document,
	objIdx int,
	fieldName, elementType, description string,
) int {
	return addComplexField(doc, objIdx, fieldName, elementType, description, ast.TypeKindList)
}

func AddNonNullFieldToObject(
	doc *ast.Document,
	objIdx int,
	fieldName, fieldType, description string,
) int {
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

func AddInterface(doc *ast.Document, name, description string) int {
	nameRef := doc.Input.AppendInputString(name)
	interfaceDef := ast.InterfaceTypeDefinition{
		Name:        nameRef,
		Description: addDescription(doc, description),
	}
	idx := len(doc.InterfaceTypeDefinitions)
	doc.InterfaceTypeDefinitions = append(doc.InterfaceTypeDefinitions, interfaceDef)

	typeDef := ast.Type{
		TypeKind: ast.TypeKindNamed,
		Name:     nameRef,
	}
	doc.Types = append(doc.Types, typeDef)

	return idx
}

func AddFieldToInterface(
	doc *ast.Document,
	interfaceIdx int,
	fieldName, fieldType, description string,
) int {
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

	doc.InterfaceTypeDefinitions[interfaceIdx].FieldsDefinition.Refs = append(
		doc.InterfaceTypeDefinitions[interfaceIdx].FieldsDefinition.Refs, fieldIdx)

	return fieldIdx
}

func AddImplementsInterface(doc *ast.Document, objIdx int, interfaceName string) {
	interfaceRef := doc.Input.AppendInputString(interfaceName)
	typeDef := ast.Type{
		TypeKind: ast.TypeKindNamed,
		Name:     interfaceRef,
	}
	typeIdx := len(doc.Types)
	doc.Types = append(doc.Types, typeDef)

	doc.ObjectTypeDefinitions[objIdx].ImplementsInterfaces.Refs = append(
		doc.ObjectTypeDefinitions[objIdx].ImplementsInterfaces.Refs, typeIdx)
}

func AddMutation(doc *ast.Document, name, description string) int {
	nameRef := doc.Input.AppendInputString(name)
	mutationDef := ast.ObjectTypeDefinition{
		Name:        nameRef,
		Description: addDescription(doc, description),
	}
	idx := len(doc.ObjectTypeDefinitions)
	doc.ObjectTypeDefinitions = append(doc.ObjectTypeDefinitions, mutationDef)

	typeDef := ast.Type{
		TypeKind: ast.TypeKindNamed,
		Name:     nameRef,
	}
	doc.Types = append(doc.Types, typeDef)

	return idx
}

func AddFieldWithArgsToObject(
	doc *ast.Document,
	objIdx int,
	fieldName, fieldType, description string,
	args []Argument,
) int {
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

	for _, arg := range args {
		argNameRef := doc.Input.AppendInputString(arg.Name)
		argTypeRef := doc.Input.AppendInputString(arg.Type)

		argTypeDef := ast.Type{
			TypeKind: ast.TypeKindNamed,
			Name:     argTypeRef,
		}
		argTypeIdx := len(doc.Types)
		doc.Types = append(doc.Types, argTypeDef)

		inputValueDef := ast.InputValueDefinition{
			Name:        argNameRef,
			Description: addDescription(doc, arg.Description),
			Type:        argTypeIdx,
		}
		argIdx := len(doc.InputValueDefinitions)
		doc.InputValueDefinitions = append(doc.InputValueDefinitions, inputValueDef)
		fieldDef.ArgumentsDefinition.Refs = append(fieldDef.ArgumentsDefinition.Refs, argIdx)
	}

	fieldIdx := len(doc.FieldDefinitions)
	doc.FieldDefinitions = append(doc.FieldDefinitions, fieldDef)

	doc.ObjectTypeDefinitions[objIdx].FieldsDefinition.Refs = append(
		doc.ObjectTypeDefinitions[objIdx].FieldsDefinition.Refs, fieldIdx)

	return fieldIdx
}

const (
	UnknownType = "Unknown"
	blockClose  = "}\n\n"
)

func GenerateGraphQLFromDocument(doc *ast.Document) string {
	var result string

	result += generateEnums(doc)
	result += generateInterfaces(doc)
	result += generateInputObjects(doc)
	result += generateObjects(doc)

	return result
}

func generateEnums(doc *ast.Document) string {
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

		result += blockClose
	}

	return result
}

func generateInterfaces(doc *ast.Document) string {
	var result string

	for _, iface := range doc.InterfaceTypeDefinitions {
		ifaceName := doc.Input.ByteSliceString(iface.Name)

		desc := getDescription(doc, iface.Description)
		if desc != "" {
			result += desc + "\n"
		}

		result += fmt.Sprintf("interface %s {\n", ifaceName)

		for _, fieldRef := range iface.FieldsDefinition.Refs {
			field := doc.FieldDefinitions[fieldRef]
			fieldName := doc.Input.ByteSliceString(field.Name)
			fieldDesc := getDescription(doc, field.Description)
			fieldType := getFieldType(doc, field.Type)

			if fieldDesc != "" {
				result += fmt.Sprintf("  %s\n", fieldDesc)
			}

			result += fmt.Sprintf("  %s: %s\n", fieldName, fieldType)
		}

		result += blockClose
	}

	return result
}

func generateInputObjects(doc *ast.Document) string {
	var result string

	for _, input := range doc.InputObjectTypeDefinitions {
		inputName := doc.Input.ByteSliceString(input.Name)

		desc := getDescription(doc, input.Description)
		if desc != "" {
			result += desc + "\n"
		}

		result += fmt.Sprintf("input %s {\n", inputName)

		for _, fieldRef := range input.InputFieldsDefinition.Refs {
			field := doc.InputValueDefinitions[fieldRef]
			fieldName := doc.Input.ByteSliceString(field.Name)
			fieldDesc := getDescription(doc, field.Description)
			fieldType := getFieldType(doc, field.Type)

			if fieldDesc != "" {
				result += fmt.Sprintf("  %s\n", fieldDesc)
			}

			result += fmt.Sprintf("  %s: %s\n", fieldName, fieldType)
		}

		result += blockClose
	}

	return result
}

func renderObjectField(doc *ast.Document, field ast.FieldDefinition) string {
	fieldName := doc.Input.ByteSliceString(field.Name)
	fieldDesc := getDescription(doc, field.Description)
	fieldType := getFieldType(doc, field.Type)

	argsClause := ""
	if len(field.ArgumentsDefinition.Refs) > 0 {
		argsClause = "("

		for i, argRef := range field.ArgumentsDefinition.Refs {
			if i > 0 {
				argsClause += ", "
			}

			arg := doc.InputValueDefinitions[argRef]
			argName := doc.Input.ByteSliceString(arg.Name)
			argType := getFieldType(doc, arg.Type)
			argsClause += fmt.Sprintf("%s: %s", argName, argType)
		}

		argsClause += ")"
	}

	var result string
	if fieldDesc != "" {
		result += fmt.Sprintf("  %s\n", fieldDesc)
	}

	result += fmt.Sprintf("  %s%s: %s\n", fieldName, argsClause, fieldType)

	return result
}

func generateObjects(doc *ast.Document) string {
	var result string

	for _, obj := range doc.ObjectTypeDefinitions {
		objName := doc.Input.ByteSliceString(obj.Name)

		desc := getDescription(doc, obj.Description)
		if desc != "" {
			result += desc + "\n"
		}

		implementsClause := ""
		if len(obj.ImplementsInterfaces.Refs) > 0 {
			implementsClause = " implements "

			for i, interfaceRef := range obj.ImplementsInterfaces.Refs {
				if i > 0 {
					implementsClause += " & "
				}

				implementsClause += getFieldType(doc, interfaceRef)
			}
		}

		result += fmt.Sprintf("type %s%s {\n", objName, implementsClause)

		for _, fieldRef := range obj.FieldsDefinition.Refs {
			field := doc.FieldDefinitions[fieldRef]
			result += renderObjectField(doc, field)
		}

		result += blockClose
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
