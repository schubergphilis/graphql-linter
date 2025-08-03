package ast

import (
	"github.com/wundergraph/graphql-go-tools/v2/pkg/ast"
)

// getBaseTypeName recursively extracts the base type name from a type reference
func GetBaseTypeName(doc *ast.Document, typeRef ast.Type) string {
	switch typeRef.TypeKind {
	case ast.TypeKindNamed:
		return doc.Input.ByteSliceString(typeRef.Name)
	case ast.TypeKindList:
		return GetBaseTypeName(doc, doc.Types[typeRef.OfType])
	case ast.TypeKindNonNull:
		return GetBaseTypeName(doc, doc.Types[typeRef.OfType])
	case ast.TypeKindUnknown:
		return ""
	default:
		return ""
	}
}

// GetDefinedTypes collects all type names defined in the document
func GetDefinedTypes(doc *ast.Document) map[string]bool {
	definedTypes := make(map[string]bool)

	// Add object types
	for _, objType := range doc.ObjectTypeDefinitions {
		typeName := doc.Input.ByteSliceString(objType.Name)
		definedTypes[typeName] = true
	}

	// Add enum types
	for _, enumType := range doc.EnumTypeDefinitions {
		typeName := doc.Input.ByteSliceString(enumType.Name)
		definedTypes[typeName] = true
	}

	// Add input types
	for _, inputType := range doc.InputObjectTypeDefinitions {
		typeName := doc.Input.ByteSliceString(inputType.Name)
		definedTypes[typeName] = true
	}

	// Add interface types
	for _, interfaceType := range doc.InterfaceTypeDefinitions {
		typeName := doc.Input.ByteSliceString(interfaceType.Name)
		definedTypes[typeName] = true
	}

	// Add scalar types
	for _, scalarType := range doc.ScalarTypeDefinitions {
		typeName := doc.Input.ByteSliceString(scalarType.Name)
		definedTypes[typeName] = true
	}

	return definedTypes
}

// GetBuiltInTypes returns the standard GraphQL built-in scalar types
func GetBuiltInTypes() map[string]bool {
	return map[string]bool{
		"String":  true,
		"Int":     true,
		"Float":   true,
		"Boolean": true,
		"ID":      true,
	}
}
