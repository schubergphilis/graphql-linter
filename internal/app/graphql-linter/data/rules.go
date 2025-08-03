// Package rules provides linting rules for GraphQL schemas
package data

import (
	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/data/stringutils"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/ast"
)

// LintDescriptions checks for missing descriptions and other description-related issues
func (s Store) LintDescriptions(doc *ast.Document, schemaString string, schemaPath string) ([]DescriptionError, bool) {
	var errors []DescriptionError

	hasDeprecationReasonError := false

	// Check if Query root type exists
	hasQuery := false

	for _, objType := range doc.ObjectTypeDefinitions {
		typeName := doc.Input.ByteSliceString(objType.Name)
		if typeName == "Query" {
			hasQuery = true
		}

		// Check if object type has description
		if !objType.Description.IsDefined {
			errors = append(errors, DescriptionError{
				FilePath:    schemaPath,
				LineNum:     1,
				Message:     "Object type '" + typeName + "' is missing a description",
				LineContent: "",
			})
		}
	}

	if !hasQuery {
		errors = append(errors, DescriptionError{
			FilePath:    schemaPath,
			LineNum:     1,
			Message:     "invalid-graphql-schema: Query root type is missing",
			LineContent: "",
		})
	}

	// Check for deprecated enum values without reason
	for _, enumType := range doc.EnumTypeDefinitions {
		for _, valueRef := range enumType.EnumValuesDefinition.Refs {
			value := doc.EnumValueDefinitions[valueRef]

			// Check if deprecated directive is present
			for _, directiveRef := range value.Directives.Refs {
				directive := doc.Directives[directiveRef]
				directiveName := doc.Input.ByteSliceString(directive.Name)

				if directiveName == "deprecated" {
					// Check if reason argument is provided
					hasReason := false

					for _, argRef := range directive.Arguments.Refs {
						arg := doc.Arguments[argRef]

						argName := doc.Input.ByteSliceString(arg.Name)
						if argName == "reason" {
							hasReason = true
							break
						}
					}

					if !hasReason {
						hasDeprecationReasonError = true

						errors = append(errors, DescriptionError{
							FilePath:    schemaPath,
							LineNum:     1,
							Message:     "deprecations-have-a-reason: Deprecated enum value is missing reason",
							LineContent: "",
						})
					}
				}
			}
		}
	}

	return errors, hasDeprecationReasonError
}

// ReportUncapitalizedDescription checks if descriptions start with a capital letter
func ReportUncapitalizedDescription(kind, parent, name, desc, schemaString string) *DescriptionError {
	if desc == "" {
		return nil
	}

	// Check if description starts with a capital letter
	if len(desc) > 0 && desc[0] >= 'a' && desc[0] <= 'z' {
		return &DescriptionError{
			FilePath:    "schema",
			LineNum:     1,
			Message:     kind + " '" + name + "' description should be capitalized",
			LineContent: "",
		}
	}

	return nil
}

// FindMissingArgumentDescriptions finds arguments without descriptions
func FindMissingArgumentDescriptions(doc *ast.Document, schemaString string) []DescriptionError {
	var errors []DescriptionError

	for _, objType := range doc.ObjectTypeDefinitions {
		for _, fieldRef := range objType.FieldsDefinition.Refs {
			field := doc.FieldDefinitions[fieldRef]
			for _, argRef := range field.ArgumentsDefinition.Refs {
				arg := doc.InputValueDefinitions[argRef]
				argName := doc.Input.ByteSliceString(arg.Name)

				if arg.Description.IsDefined == false {
					errors = append(errors, DescriptionError{
						FilePath:    "schema",
						LineNum:     1,
						Message:     "Argument '" + argName + "' is missing a description",
						LineContent: "",
					})
				}
			}
		}
	}

	return errors
}

// FindUnsortedInterfaceFields checks if interface fields are sorted alphabetically
func FindUnsortedInterfaceFields(doc *ast.Document, schemaString string) []DescriptionError {
	var errors []DescriptionError

	for _, interfaceType := range doc.InterfaceTypeDefinitions {
		var fieldNames []string

		// Collect field names
		for _, fieldRef := range interfaceType.FieldsDefinition.Refs {
			field := doc.FieldDefinitions[fieldRef]
			fieldName := doc.Input.ByteSliceString(field.Name)
			fieldNames = append(fieldNames, fieldName)
		}

		// Check if fields are sorted alphabetically (only if there are 2 or more fields)
		if len(fieldNames) >= minFieldsForSortCheck {
			for i := 1; i < len(fieldNames); i++ {
				if fieldNames[i-1] > fieldNames[i] {
					interfaceName := doc.Input.ByteSliceString(interfaceType.Name)
					errors = append(errors, DescriptionError{
						FilePath:    "schema",
						LineNum:     1,
						Message:     "interface-fields-sorted-alphabetically: Interface '" + interfaceName + "' fields are not sorted alphabetically",
						LineContent: "",
					})

					break
				}
			}
		}
	}

	return errors
}

// FindRelayPageInfoSpec checks for Relay PageInfo spec compliance
func FindRelayPageInfoSpec(doc *ast.Document, schemaString string) []DescriptionError {
	// Check if PageInfo type exists
	hasPageInfo := false

	for _, objType := range doc.ObjectTypeDefinitions {
		typeName := doc.Input.ByteSliceString(objType.Name)
		if typeName == "PageInfo" {
			hasPageInfo = true
			break
		}
	}

	if !hasPageInfo {
		return []DescriptionError{
			{
				FilePath:    "schema",
				LineNum:     1,
				Message:     "relay-page-info-spec: PageInfo type is missing",
				LineContent: "",
			},
		}
	}

	return []DescriptionError{}
}

// FindInputObjectValuesCamelCased checks if input object field names are camelCase
func FindInputObjectValuesCamelCased(doc *ast.Document, schemaString string) []DescriptionError {
	var errors []DescriptionError

	for _, inputObj := range doc.InputObjectTypeDefinitions {
		for _, fieldRef := range inputObj.InputFieldsDefinition.Refs {
			field := doc.InputValueDefinitions[fieldRef]
			fieldName := doc.Input.ByteSliceString(field.Name)

			if !stringutils.IsCamelCase(fieldName) {
				errors = append(errors, DescriptionError{
					FilePath:    "schema",
					LineNum:     1,
					Message:     "input-object-values-are-camel-cased: Field '" + fieldName + "' should be camelCase",
					LineContent: "",
				})
			}
		}
	}

	return errors
}

// FindMissingEnumValueDescriptions checks for enum values without descriptions
func FindMissingEnumValueDescriptions(doc *ast.Document, schemaString string) []DescriptionError {
	var errors []DescriptionError

	for _, enumType := range doc.EnumTypeDefinitions {
		for _, valueRef := range enumType.EnumValuesDefinition.Refs {
			value := doc.EnumValueDefinitions[valueRef]
			valueName := doc.Input.ByteSliceString(value.EnumValue)

			// Check if the enum value has a description
			if value.Description.IsDefined == false {
				errors = append(errors, DescriptionError{
					FilePath:    "schema",
					LineNum:     1,
					Message:     "enum-values-have-descriptions: Enum value '" + valueName + "' is missing a description",
					LineContent: "",
				})
			}
		}
	}

	return errors
}

// SuggestCorrectEnumValue suggests corrections for common enum value typos
func SuggestCorrectEnumValue(value string) string {
	if len(value) == 0 {
		return ""
	}

	corrections := map[string]string{
		"STRING2":  "STRING",
		"BOOLEAN2": "BOOLEAN",
		"BOOLE3AN": "BOOLEAN",
		"BOOL3AN":  "BOOLEAN",
		"BOOLEAN3": "BOOLEAN",
		"FLOA2T":   "FLOAT",
		"FLO2AT":   "FLOAT",
		"FLOAT2":   "FLOAT",
		"INT2":     "INT",
		"INTEGER2": "INTEGER",
		"I2NT":     "INT",
		"INTE2GER": "INTEGER",
	}

	if correction, exists := corrections[value]; exists {
		return correction
	}

	cleanValue := stringutils.RemoveAllDigits(value)
	standardTypes := []string{"STRING", "BOOLEAN", "FLOAT", "INT", "INTEGER", "ID"}

	for _, standardType := range standardTypes {
		if cleanValue == standardType {
			return standardType
		}
	}

	for _, standardType := range standardTypes {
		if stringutils.LevenshteinDistance(cleanValue, standardType) <= levenshteinThreshold {
			return standardType
		}
	}

	return ""
}

// GetAvailableTypes returns a list of all available types (built-in + defined)
func GetAvailableTypes(builtInScalars, definedTypes map[string]bool) []string {
	types := make([]string, 0, len(builtInScalars)+len(definedTypes))
	for t := range builtInScalars {
		types = append(types, t)
	}

	for t := range definedTypes {
		types = append(types, t)
	}

	return types
}

// FindFieldDefinitionLine finds the line number where a field is defined
func FindFieldDefinitionLine(schemaContent string, fieldName string, typeName string) int {
	// TODO: Implement field definition line finding
	return 0
}
