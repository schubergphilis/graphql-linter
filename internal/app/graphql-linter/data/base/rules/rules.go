package rules

import (
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"unicode"

	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/data/base/models"
	pkg_rules "github.com/schubergphilis/graphql-linter/internal/pkg/rules"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/ast"
)

const (
	rootQueryType        = "Query"
	rootMutationType     = "Mutation"
	rootSubscriptionType = "Subscription"

	minEnumValuesForSortCheck = 2
	minFieldsForSortCheck     = 2
)

type Rule struct{}

func (r Rule) TypesAreCapitalized(doc *ast.Document, schemaString string) []models.DescriptionError {
	var errors []models.DescriptionError

	for _, def := range typeDefinitions(doc) {
		if (def.kind != kindObject && def.kind != kindInterface) || isRootType(def.name) {
			continue
		}

		if !unicode.IsUpper(rune(def.name[0])) {
			errors = append(errors, newFinding(schemaString, LineOf(doc, def.nameRef), def.name,
				"types-are-capitalized: The "+strings.ToLower(def.kind)+" '"+def.name+
					"' should start with a capital letter."))
		}
	}

	return errors
}

func (r Rule) EnumValuesAllCaps(doc *ast.Document, schemaString string) []models.DescriptionError {
	var errors []models.DescriptionError

	for _, enum := range doc.EnumTypeDefinitions {
		enumName := doc.Input.ByteSliceString(enum.Name)

		for _, valueRef := range enum.EnumValuesDefinition.Refs {
			valueDef := doc.EnumValueDefinitions[valueRef]

			valueName := doc.Input.ByteSliceString(valueDef.EnumValue)
			if valueName != strings.ToUpper(valueName) {
				errors = append(errors, newFinding(schemaString, LineOf(doc, valueDef.EnumValue), valueName,
					"enum-values-all-caps: The enum value `"+enumName+"."+valueName+"` should be uppercase."))
			}
		}
	}

	return errors
}

func (r Rule) EnumValuesSortedAlphabetically(
	doc *ast.Document,
	modelsLinterConfig *models.LinterConfig,
	schemaString string,
	schemaPath string,
) []models.DescriptionError {
	var errors []models.DescriptionError

	for _, enum := range doc.EnumTypeDefinitions {
		enumName := doc.Input.ByteSliceString(enum.Name)

		var valueNames []string

		for _, valueRef := range enum.EnumValuesDefinition.Refs {
			valueDef := doc.EnumValueDefinitions[valueRef]
			valueNames = append(valueNames, doc.Input.ByteSliceString(valueDef.EnumValue))
		}

		if err := checkSortedOrder(
			valueNames,
			minEnumValuesForSortCheck,
			schemaString,
			LineOf(doc, enum.Name),
			enumName,
			"enum-values-sorted-alphabetically",
		); err != nil {
			if !pkg_rules.IsSuppressed(
				schemaPath,
				err.LineNum,
				modelsLinterConfig,
				"enum-values-sorted-alphabetically",
				err.Value,
			) {
				errors = append(errors, *err)
			}
		}
	}

	return errors
}

func (r Rule) MissingDeprecationReasons(doc *ast.Document, schemaString string) []models.DescriptionError {
	var errors []models.DescriptionError

	check := func(directiveRefs []int, kind, parent string, nameRef ast.ByteSliceReference) {
		name := doc.Input.ByteSliceString(nameRef)

		for _, dirRef := range directiveRefs {
			dir := doc.Directives[dirRef]
			if doc.Input.ByteSliceString(dir.Name) == "deprecated" && len(dir.Arguments.Refs) == 0 {
				errors = append(errors, newFinding(schemaString, LineOf(doc, nameRef), name,
					"deprecations-have-a-reason: Deprecated "+kind+" '"+parent+"."+name+"' is missing a reason."))
			}
		}
	}

	for _, def := range typeDefinitions(doc) {
		for _, ref := range def.enumValues {
			valueDef := doc.EnumValueDefinitions[ref]
			check(valueDef.Directives.Refs, "enum value", def.name, valueDef.EnumValue)
		}

		for _, ref := range def.inputValues {
			inputDef := doc.InputValueDefinitions[ref]
			check(inputDef.Directives.Refs, "input value", def.name, inputDef.Name)
		}

		for _, ref := range def.fields {
			fieldDef := doc.FieldDefinitions[ref]
			check(fieldDef.Directives.Refs, "field", def.name, fieldDef.Name)

			for _, argRef := range fieldDef.ArgumentsDefinition.Refs {
				argDef := doc.InputValueDefinitions[argRef]
				check(argDef.Directives.Refs, "argument", def.name+"."+doc.Input.ByteSliceString(fieldDef.Name), argDef.Name)
			}
		}
	}

	return errors
}

func (r Rule) MissingArgumentDescriptions(doc *ast.Document, schemaString string) []models.DescriptionError {
	var errors []models.DescriptionError

	for _, def := range typeDefinitions(doc) {
		for _, fieldRef := range def.fields {
			fieldDef := doc.FieldDefinitions[fieldRef]
			for _, argRef := range fieldDef.ArgumentsDefinition.Refs {
				argDef := doc.InputValueDefinitions[argRef]
				if !argDef.Description.IsDefined {
					argName := doc.Input.ByteSliceString(argDef.Name)
					fieldName := doc.Input.ByteSliceString(fieldDef.Name)
					errors = append(errors, newFinding(schemaString, LineOf(doc, argDef.Name), argName,
						"arguments-have-descriptions: The '"+argName+"' argument of '"+fieldName+
							"' is missing a description."))
				}
			}
		}
	}

	return errors
}

func (r Rule) UnsortedFields(
	fieldDefs []int,
	getFieldName func(int) string,
	typeLabel,
	typeName,
	schemaString string,
	lineNum int,
) []models.DescriptionError {
	fieldNames := make([]string, len(fieldDefs))
	for i, fieldRef := range fieldDefs {
		fieldNames[i] = getFieldName(fieldRef)
	}

	if len(fieldNames) < minFieldsForSortCheck || slices.IsSorted(fieldNames) {
		return nil
	}

	message := typeLabel + "-fields-sorted-alphabetically: The fields of " +
		typeLabel + " type `" + typeName + "` should be sorted in alphabetical order.\nExpected sorting: " +
		strings.Join(slices.Sorted(slices.Values(fieldNames)), ", ")

	return []models.DescriptionError{newFinding(schemaString, lineNum, typeName, message)}
}

func (r Rule) MissingInputObjectValueDescriptions(
	doc *ast.Document,
	schemaString string,
) []models.DescriptionError {
	var errors []models.DescriptionError

	for _, input := range doc.InputObjectTypeDefinitions {
		inputName := doc.Input.ByteSliceString(input.Name)
		for _, fieldRef := range input.InputFieldsDefinition.Refs {
			fieldDef := doc.InputValueDefinitions[fieldRef]
			if !fieldDef.Description.IsDefined {
				fieldName := doc.Input.ByteSliceString(fieldDef.Name)
				errors = append(errors, newFinding(schemaString, LineOf(doc, fieldDef.Name), fieldName, fmt.Sprintf(
					"input-object-values-have-descriptions: The input value `%s.%s` is missing a description.",
					inputName,
					fieldName,
				)))
			}
		}
	}

	return errors
}

func (r Rule) InputObjectFieldsSortedAlphabetically(
	doc *ast.Document,
	schemaString string,
) []models.DescriptionError {
	var errors []models.DescriptionError

	for _, input := range doc.InputObjectTypeDefinitions {
		inputName := doc.Input.ByteSliceString(input.Name)

		var fieldNames []string

		for _, fieldRef := range input.InputFieldsDefinition.Refs {
			fieldDef := doc.InputValueDefinitions[fieldRef]
			fieldNames = append(fieldNames, doc.Input.ByteSliceString(fieldDef.Name))
		}

		if err := checkSortedOrder(
			fieldNames,
			minFieldsForSortCheck,
			schemaString,
			LineOf(doc, input.Name),
			"fields of input type '"+inputName+"'",
			"input-object-fields-sorted-alphabetically",
		); err != nil {
			err.Value = inputName
			errors = append(errors, *err)
		}
	}

	return errors
}

func (r Rule) FieldsAreCamelCased(doc *ast.Document, schemaString string) []models.DescriptionError {
	var errors []models.DescriptionError

	for _, def := range typeDefinitions(doc) {
		for _, fieldRef := range def.fields {
			fieldDef := doc.FieldDefinitions[fieldRef]

			fieldName := doc.Input.ByteSliceString(fieldDef.Name)
			if !isCamelCase(fieldName) {
				errors = append(errors, newFinding(schemaString, LineOf(doc, fieldDef.Name), fieldName,
					"fields-are-camel-cased: The field '"+def.name+"."+fieldName+"' is not camel cased."))
			}
		}
	}

	return errors
}

func (r Rule) InputObjectValuesCamelCased(doc *ast.Document, schemaString string) []models.DescriptionError {
	var errors []models.DescriptionError

	for _, input := range doc.InputObjectTypeDefinitions {
		inputName := doc.Input.ByteSliceString(input.Name)

		for _, fieldRef := range input.InputFieldsDefinition.Refs {
			fieldDef := doc.InputValueDefinitions[fieldRef]

			fieldName := doc.Input.ByteSliceString(fieldDef.Name)
			if !isCamelCase(fieldName) {
				errors = append(errors, newFinding(schemaString, LineOf(doc, fieldDef.Name), fieldName,
					"input-object-values-are-camel-cased: The input value `"+inputName+"."+fieldName+
						"` is not camel cased."))
			}
		}
	}

	return errors
}

// RelayPageInfoSpec is schema wide: run it on the merged schema of all files.
func (r Rule) RelayPageInfoSpec(doc *ast.Document, schemaString string) []models.DescriptionError {
	for _, obj := range doc.ObjectTypeDefinitions {
		if doc.Input.ByteSliceString(obj.Name) == "PageInfo" {
			return nil
		}
	}

	return []models.DescriptionError{newFinding(schemaString, 1, "PageInfo",
		"relay-page-info-spec: A `PageInfo` object type is required as per the Relay spec.")}
}

func (r Rule) RelayConnectionArgumentsSpec(
	doc *ast.Document,
	schemaString string,
) []models.DescriptionError {
	var errors []models.DescriptionError

	for _, fieldDef := range doc.FieldDefinitions {
		fieldType := doc.Types[fieldDef.Type]
		baseType := getBaseTypeName(doc, fieldType)

		if !strings.HasSuffix(baseType, "Connection") {
			continue
		}

		hasForwardArgs := false
		hasBackwardArgs := false

		for _, argRef := range fieldDef.ArgumentsDefinition.Refs {
			argDef := doc.InputValueDefinitions[argRef]
			argName := doc.Input.ByteSliceString(argDef.Name)

			switch argName {
			case "first", "after":
				hasForwardArgs = true
			case "last", "before":
				hasBackwardArgs = true
			}
		}

		if !hasForwardArgs && !hasBackwardArgs {
			fieldName := doc.Input.ByteSliceString(fieldDef.Name)
			errors = append(errors, newFinding(schemaString, LineOf(doc, fieldDef.Name), fieldName,
				"relay-connection-arguments-spec: A field that returns a Connection Type must include forward"+
					"pagination arguments (`first` and `after`), backward pagination arguments (`last` and `before`), or both as"+
					"per the Relay spec."))
		}
	}

	return errors
}

func (r Rule) RelayConnectionTypesSpec(doc *ast.Document, schemaString string) []models.DescriptionError {
	var errors []models.DescriptionError

	for _, obj := range doc.ObjectTypeDefinitions {
		typeName := doc.Input.ByteSliceString(obj.Name)
		if !strings.HasSuffix(typeName, "Connection") {
			continue
		}

		hasPageInfo := false
		hasEdges := false

		for _, fieldRef := range obj.FieldsDefinition.Refs {
			fieldDef := doc.FieldDefinitions[fieldRef]

			fieldName := doc.Input.ByteSliceString(fieldDef.Name)
			if fieldName == "pageInfo" {
				hasPageInfo = true
			}

			if fieldName == "edges" {
				hasEdges = true
			}
		}

		lineNum := LineOf(doc, obj.Name)

		if !hasPageInfo {
			errors = append(errors, newFinding(schemaString, lineNum, typeName, fmt.Sprintf(
				"relay-connection-types-spec: Connection `%s` is missing the following field: pageInfo.",
				typeName,
			)))
		}

		if !hasEdges {
			errors = append(errors, newFinding(schemaString, lineNum, typeName, fmt.Sprintf(
				"relay-connection-types-spec: Connection `%s` is missing the following field: edges.",
				typeName,
			)))
		}
	}

	return errors
}

// MissingQueryRootType is schema wide: run it on the merged schema of all files.
func (r Rule) MissingQueryRootType(doc *ast.Document, schemaString string) []models.DescriptionError {
	for _, obj := range doc.ObjectTypeDefinitions {
		if doc.Input.ByteSliceString(obj.Name) == rootQueryType {
			return nil
		}
	}

	return []models.DescriptionError{newFinding(schemaString, 1, rootQueryType,
		"invalid-graphql-schema: Query root type must be provided.")}
}

func (r Rule) MissingEnumValueDescriptions(
	doc *ast.Document,
	schemaString string,
) []models.DescriptionError {
	var errors []models.DescriptionError

	for _, enum := range doc.EnumTypeDefinitions {
		enumName := doc.Input.ByteSliceString(enum.Name)
		for _, valueRef := range enum.EnumValuesDefinition.Refs {
			valueDef := doc.EnumValueDefinitions[valueRef]
			if !valueDef.Description.IsDefined {
				valueName := doc.Input.ByteSliceString(valueDef.EnumValue)
				errors = append(errors, newFinding(schemaString, LineOf(doc, valueDef.EnumValue), valueName,
					"enum-values-have-descriptions: Enum value '"+enumName+"."+valueName+"' is missing a description."))
			}
		}
	}

	return errors
}

func (r Rule) MissingTypeDescriptions(doc *ast.Document, schemaString string) []models.DescriptionError {
	var errors []models.DescriptionError

	for _, def := range typeDefinitions(doc) {
		if !def.description.IsDefined {
			errors = append(errors, newFinding(schemaString, LineOf(doc, def.nameRef), def.name,
				"types-have-descriptions: "+def.kind+" '"+def.name+"' is missing a description"))
		}
	}

	return errors
}

func (r Rule) MissingFieldDescriptions(doc *ast.Document, schemaString string) []models.DescriptionError {
	var errors []models.DescriptionError

	for _, def := range typeDefinitions(doc) {
		for _, fieldRef := range def.fields {
			fieldDef := doc.FieldDefinitions[fieldRef]
			if !fieldDef.Description.IsDefined {
				fieldName := doc.Input.ByteSliceString(fieldDef.Name)
				errors = append(errors, newFinding(schemaString, LineOf(doc, fieldDef.Name), fieldName,
					"fields-have-descriptions: Field '"+def.name+"."+fieldName+"' is missing a description."))
			}
		}
	}

	return errors
}

func (r Rule) UncapitalizedDescriptions(doc *ast.Document, schemaString string) []models.DescriptionError {
	var errors []models.DescriptionError

	check := func(description ast.Description, kind, parent string, nameRef ast.ByteSliceReference) {
		if !description.IsDefined || isCapitalized(doc.Input.ByteSliceString(description.Content)) {
			return
		}

		name := doc.Input.ByteSliceString(nameRef)

		qualified := name
		if parent != "" {
			qualified = parent + "." + name
		}

		errors = append(errors, newFinding(schemaString, LineOf(doc, nameRef), name,
			"descriptions-are-capitalized: The description for "+kind+" `"+qualified+"` should be capitalized."))
	}

	for _, def := range typeDefinitions(doc) {
		check(def.description, "type", "", def.nameRef)

		for _, ref := range def.fields {
			fieldDef := doc.FieldDefinitions[ref]
			check(fieldDef.Description, "field", def.name, fieldDef.Name)

			for _, argRef := range fieldDef.ArgumentsDefinition.Refs {
				argDef := doc.InputValueDefinitions[argRef]
				check(argDef.Description, "argument", doc.Input.ByteSliceString(fieldDef.Name), argDef.Name)
			}
		}

		for _, ref := range def.inputValues {
			inputDef := doc.InputValueDefinitions[ref]
			check(inputDef.Description, "input value", def.name, inputDef.Name)
		}

		for _, ref := range def.enumValues {
			valueDef := doc.EnumValueDefinitions[ref]
			check(valueDef.Description, "enum value", def.name, valueDef.EnumValue)
		}
	}

	return errors
}

// UnusedTypes is schema wide: run it on the merged schema of all files.
func (r Rule) UnusedTypes(doc *ast.Document, schemaString string) []models.DescriptionError {
	defs := typeDefinitions(doc)

	usedTypes := make(map[string]bool, len(defs))
	for _, def := range defs {
		usedTypes[def.name] = false
	}

	markUsedTypes(doc, usedTypes)

	var unusedTypeErrors []models.DescriptionError

	for _, def := range defs {
		if usedTypes[def.name] || isRootType(def.name) {
			continue
		}

		unusedTypeErrors = append(unusedTypeErrors, newFinding(schemaString, LineOf(doc, def.nameRef), def.name,
			fmt.Sprintf("defined-types-are-used: Type '%s' is defined but not used", def.name)))
	}

	return unusedTypeErrors
}

func (r Rule) ValidateEnumTypes(
	doc *ast.Document,
	modelsLinterConfig *models.LinterConfig,
	schemaContent string,
	schemaPath string,
) ([]string, []int, []models.DescriptionError) {
	var (
		errors     []string
		errorLines []int
		descErrors []models.DescriptionError
	)

	for _, enumDef := range doc.EnumTypeDefinitions {
		enumName := doc.Input.ByteSliceString(enumDef.Name)

		for _, valueRef := range enumDef.EnumValuesDefinition.Refs {
			valueDef := doc.EnumValueDefinitions[valueRef]
			valueName := doc.Input.ByteSliceString(valueDef.EnumValue)

			valueLine := LineOf(doc, valueDef.EnumValue)

			if errValue, line := checkInvalidEnumValue(enumName, valueName, valueLine); errValue != "" {
				errors = append(errors, errValue)
				if line > 0 {
					errorLines = append(errorLines, line)
				}
			}

			if errValue, line := checkSuspiciousEnumValue(
				enumName,
				valueName,
				valueLine,
				schemaPath,
				modelsLinterConfig,
			); errValue != "" {
				errors = append(errors, errValue)
				if line > 0 {
					errorLines = append(errorLines, line)
					descErrors = append(descErrors, models.DescriptionError{
						Value:    valueName,
						FilePath: schemaPath,
						LineNum:  line,
						Message: fmt.Sprintf(
							"suspicious-enum-value: Enum '%s' has suspicious value '%s'",
							enumName,
							errValue,
						),
						LineContent: GetLineContent(schemaContent, line),
					})
				}
			}
		}
	}

	return errors, errorLines, descErrors
}

func (r Rule) ValidateFieldTypes(
	doc *ast.Document,
	builtInScalars, definedTypes map[string]bool,
) ([]string, []int) {
	return validateTypeReferences(
		doc,
		builtInScalars,
		definedTypes,
		indexSlice(len(doc.FieldDefinitions)),
		func(i int) ast.ByteSliceReference { return doc.FieldDefinitions[i].Name },
		func(i int) ast.Type { return doc.Types[doc.FieldDefinitions[i].Type] },
		"invalid-field-types: Field",
	)
}

func (r Rule) ValidateInputFieldTypes(
	doc *ast.Document,
	builtInScalars, definedTypes map[string]bool,
) ([]string, []int) {
	return validateTypeReferences(
		doc,
		builtInScalars,
		definedTypes,
		indexSlice(len(doc.InputValueDefinitions)),
		func(i int) ast.ByteSliceReference { return doc.InputValueDefinitions[i].Name },
		func(i int) ast.Type { return doc.Types[doc.InputValueDefinitions[i].Type] },
		"invalid-input-field-types: Input field",
	)
}

func checkInvalidEnumValue(enumName, valueName string, lineNum int) (string, int) {
	if isValidEnumValue(valueName) {
		return "", 0
	}

	slog.Info(fmt.Sprintf(
		"invalid-enum-value: Enum '%s' has invalid value '%s' (line %d)",
		enumName,
		valueName,
		lineNum,
	))
	slog.Info(
		"  Enum values should be valid GraphQL identifiers (letters, digits, underscores, no leading digits)",
	)

	return valueName, lineNum
}

func checkSuspiciousEnumValue(
	enumName,
	valueName string,
	lineNum int,
	schemaPath string,
	modelsLinterConfig *models.LinterConfig,
) (string, int) {
	if !hasSuspiciousEnumValue(valueName) &&
		!hasEmbeddedDigits(valueName) {
		return "", 0
	}

	if pkg_rules.IsSuppressed(
		schemaPath,
		lineNum,
		modelsLinterConfig,
		"suspicious-enum-value",
		valueName,
	) {
		return "", 0
	}

	slog.Error(fmt.Sprintf(
		"suspicious-enum-value: Enum '%s' has suspicious value '%s' (line %d)",
		enumName,
		valueName,
		lineNum,
	))

	if suggestion := suggestCorrectEnumValue(valueName); suggestion != "" {
		slog.Error(fmt.Sprintf("  Did you mean '%s'?", suggestion))
	} else {
		suggestedValue := removeSuffixDigits(valueName)
		slog.Error(fmt.Sprintf("  Did you mean '%s'? Enum values typically don't contain numbers.", suggestedValue))
	}

	return valueName, lineNum
}

func validateTypeReferences(
	doc *ast.Document,
	builtInScalars, definedTypes map[string]bool,
	typeRefs []int,
	getNameRef func(int) ast.ByteSliceReference,
	getType func(int) ast.Type,
	errorPrefix string,
) ([]string, []int) {
	var (
		errors     []string
		errorLines []int
	)

	for _, ref := range typeRefs {
		nameRef := getNameRef(ref)
		fieldName := doc.Input.ByteSliceString(nameRef)
		fieldType := getType(ref)

		baseType := getBaseTypeName(doc, fieldType)
		if !builtInScalars[baseType] && !definedTypes[baseType] {
			lineNum := LineOf(doc, nameRef)

			slog.Error(fmt.Sprintf(
				"%s '%s' references undefined type '%s' (line %d)",
				errorPrefix,
				fieldName,
				baseType,
				lineNum,
			))
			slog.Error(fmt.Sprintf("  Available types: %v", getAvailableTypes(builtInScalars, definedTypes)))

			if lineNum > 0 {
				errorLines = append(errorLines, lineNum)
			}

			errors = append(errors, fieldName)
		}
	}

	return errors, errorLines
}
