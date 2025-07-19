package application

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/schubergphilis/graphql-linter/internal/app/graphql-testdata-generator/data"
	"github.com/schubergphilis/mcvs-golang-project-root/pkg/projectroot"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/ast"
)

const (
	dirPerm     = 0o700
	filePerm    = 0o600
	unknownType = "Unknown"
)

type Executor interface {
	Run() error
}

type Execute struct{}

func NewExecute() Execute {
	execute := Execute{}

	return execute
}

func (e Execute) Run() error {
	return WriteTestSchemaToFile()
}

func GenerateTestSchema() *ast.Document {
	doc := data.NewDocument()

	data.AddEnum(doc, "Color", "Available colors.", []data.EnumValue{
		{Name: "blue", Description: ""},
		{Name: "red", Description: "Red color."},
	})

	pageInfoIdx := data.AddObject(doc, "PageInfo", "Relay-compliant PageInfo object.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasNextPage", "Boolean", "Has next page.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasPreviousPage", "Boolean", "Has previous page.")
	data.AddFieldToObject(doc, pageInfoIdx, "startCursor", "String", "Start cursor.")
	data.AddFieldToObject(doc, pageInfoIdx, "endCursor", "String", "End cursor.")

	queryIdx := data.AddObject(doc, "Query", "Query root.")
	data.AddFieldToObject(doc, queryIdx, "color", "Color", "Returns a color.")

	return doc
}

func WriteTestSchemaToFile() error {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	doc := GenerateTestSchema()
	gql := generateGraphQLFromDocument(doc)

	outputPath := filepath.Join(projectRoot, "test/testdata/graphql/invalid/01-enum-values-all-caps.graphql")
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func generateGraphQLFromDocument(doc *ast.Document) string {
	var result string

	for _, enum := range doc.EnumTypeDefinitions {
		enumName := doc.Input.ByteSliceString(enum.Name)
		desc := getDescription(doc, enum.Description)
		result += fmt.Sprintf("%senum %s {\n", desc, enumName)

		for _, valueRef := range enum.EnumValuesDefinition.Refs {
			value := doc.EnumValueDefinitions[valueRef]
			valueName := doc.Input.ByteSliceString(value.EnumValue)
			valueDesc := getDescription(doc, value.Description)
			result += fmt.Sprintf("%s  %s\n", valueDesc, valueName)
		}

		result += "}\n\n"
	}

	for _, obj := range doc.ObjectTypeDefinitions {
		objName := doc.Input.ByteSliceString(obj.Name)
		desc := getDescription(doc, obj.Description)
		result += fmt.Sprintf("%stype %s {\n", desc, objName)

		for _, fieldRef := range obj.FieldsDefinition.Refs {
			field := doc.FieldDefinitions[fieldRef]
			fieldName := doc.Input.ByteSliceString(field.Name)
			fieldDesc := getDescription(doc, field.Description)
			fieldType := getFieldType(doc, field.Type)
			result += fmt.Sprintf("%s  %s: %s\n", fieldDesc, fieldName, fieldType)
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

	return fmt.Sprintf(`"""%s"""
`, content)
}

func getFieldType(doc *ast.Document, typeIndex int) string {
	if typeIndex >= len(doc.Types) {
		return unknownType
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
		return unknownType
	default:
		return unknownType
	}
}
