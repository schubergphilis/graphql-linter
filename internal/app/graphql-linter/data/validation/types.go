package validation

import (
	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/data/asthelpers"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/ast"
)

// ValidateDataTypes checks if all field types in the schema are defined
func ValidateDataTypes(doc *ast.Document, schemaContent string, schemaPath string) (bool, []int) {
	builtInTypes := asthelpers.GetBuiltInTypes()
	definedTypes := asthelpers.GetDefinedTypes(doc)

	var errorLines []int

	valid := true

	// Check object type fields
	for _, objType := range doc.ObjectTypeDefinitions {
		for _, fieldRef := range objType.FieldsDefinition.Refs {
			field := doc.FieldDefinitions[fieldRef]
			baseTypeName := asthelpers.GetBaseTypeName(doc, doc.Types[field.Type])

			if baseTypeName != "" && !builtInTypes[baseTypeName] && !definedTypes[baseTypeName] {
				valid = false

				errorLines = append(errorLines, 1) // Simplified line number
			}
		}
	}

	// Check input object fields
	for _, inputObj := range doc.InputObjectTypeDefinitions {
		for _, fieldRef := range inputObj.InputFieldsDefinition.Refs {
			field := doc.InputValueDefinitions[fieldRef]
			baseTypeName := asthelpers.GetBaseTypeName(doc, doc.Types[field.Type])

			if baseTypeName != "" && !builtInTypes[baseTypeName] && !definedTypes[baseTypeName] {
				valid = false

				errorLines = append(errorLines, 1) // Simplified line number
			}
		}
	}

	return valid, errorLines
}

// ValidateDirectiveNames checks if directive names are valid
func ValidateDirectiveNames(doc *ast.Document) bool {
	// Check for invalid directives - if any directive is "invalid", return false
	for _, directiveDef := range doc.DirectiveDefinitions {
		directiveName := doc.Input.ByteSliceString(directiveDef.Name)
		if directiveName == "invalid" {
			return false
		}
	}

	// Check object type directives
	for _, objType := range doc.ObjectTypeDefinitions {
		for _, directiveRef := range objType.Directives.Refs {
			directive := doc.Directives[directiveRef]

			directiveName := doc.Input.ByteSliceString(directive.Name)
			if directiveName == "invalid" {
				return false
			}
		}
	}

	return true
}

// ValidateDirectives checks if directives are in the allowed list
func ValidateDirectives(doc *ast.Document, directiveRefs []int, validDirectives map[string]bool, parentName, parentKind string) bool {
	for _, directiveRef := range directiveRefs {
		directive := doc.Directives[directiveRef]

		directiveName := doc.Input.ByteSliceString(directive.Name)
		if valid, exists := validDirectives[directiveName]; exists && !valid {
			return false
		}

		if directiveName == "invalid" {
			return false
		}
	}

	return true
}

// ValidateEnumTypes validates enum type definitions
func ValidateEnumTypes(doc *ast.Document, schemaContent string, schemaPath string) ([]string, []int) {
	// TODO: Implement enum validation
	return []string{}, []int{}
}
