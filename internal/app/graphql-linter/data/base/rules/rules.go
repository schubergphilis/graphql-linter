package rules

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/data/base/models"
	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/data/constants"
	pkg_rules "github.com/schubergphilis/graphql-linter/internal/pkg/rules"
	log "github.com/sirupsen/logrus"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/ast"
)

const (
	minEnumValuesForSortCheck = 2
	minFieldsForSortCheck     = 2
	splitNParts               = 2
)

//nolint:interfacebloat //TODO: generate one method per rule
type Ruler interface {
	EnumValuesSortedAlphabetically(
		doc *ast.Document,
		modelsLinterConfig *models.LinterConfig,
		schemaString string,
		schemaPath string,
	) []models.DescriptionError
	FieldsAreCamelCased(doc *ast.Document, schemaString string) []models.DescriptionError
	InputObjectFieldsSortedAlphabetically(doc *ast.Document, schemaString string) []models.DescriptionError
	InputObjectValuesCamelCased(doc *ast.Document, schemaString string) []models.DescriptionError
	MissingArgumentDescriptions(doc *ast.Document, schemaString string) []models.DescriptionError
	MissingDeprecationReasons(doc *ast.Document, schemaString string) []models.DescriptionError
	MissingEnumValueDescriptions(doc *ast.Document, schemaString string) []models.DescriptionError
	MissingFieldDescriptions(doc *ast.Document, schemaString string) []models.DescriptionError
	MissingInputObjectValueDescriptions(doc *ast.Document, schemaString string) []models.DescriptionError
	MissingQueryRootType(doc *ast.Document, schemaString string) []models.DescriptionError
	MissingTypeDescriptions(doc *ast.Document, schemaString string) []models.DescriptionError
	RelayConnectionArgumentsSpec(doc *ast.Document, schemaString string) []models.DescriptionError
	RelayConnectionTypesSpec(doc *ast.Document, schemaString string) []models.DescriptionError
	RelayPageInfoSpec(doc *ast.Document, schemaString string) []models.DescriptionError
	ReportUncapitalizedDescription(
		kind,
		parent,
		name,
		desc,
		schemaString string,
	) *models.DescriptionError
	TypesAreCapitalized(doc *ast.Document, schemaString string) []models.DescriptionError
	UnusedTypes(doc *ast.Document, schemaString string) []models.DescriptionError
	UnsortedFields(
		fieldDefs []int,
		getFieldName func(int) string,
		typeLabel,
		typeName,
		schemaString string,
	) []models.DescriptionError
	ValidateEnumTypes(
		doc *ast.Document,
		modelsLinterConfig *models.LinterConfig,
		schemaContent string,
		schemaPath string,
	) ([]string, []int, []models.DescriptionError)
	ValidateFieldTypes(
		doc *ast.Document,
		schemaContent string,
		builtInScalars, definedTypes map[string]bool,
	) ([]string, []int)
	ValidateInputFieldTypes(
		doc *ast.Document,
		schemaContent string,
		builtInScalars, definedTypes map[string]bool,
	) ([]string, []int)
}

type Rule struct{}

func NewRule() *Rule {
	return &Rule{}
}

func (r Rule) TypesAreCapitalized(doc *ast.Document, schemaString string) []models.DescriptionError {
	errors := make([]models.DescriptionError, 0)

	for _, obj := range doc.ObjectTypeDefinitions {
		typeName := doc.Input.ByteSliceString(obj.Name)
		if typeName == constants.RootQueryType ||
			typeName == constants.RootMutationType ||
			typeName == constants.RootSubscriptionType {
			continue
		}

		if len(typeName) == 0 || !unicode.IsUpper(rune(typeName[0])) {
			lineNum := findLineNumberByText(schemaString, "type "+typeName)
			lineContent := GetLineContent(schemaString, lineNum)
			message := "types-are-capitalized: The object type '" + typeName + "' should start with a capital letter."
			errors = append(errors, models.DescriptionError{
				LineNum:     lineNum,
				Message:     message,
				LineContent: lineContent,
			})
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
			"enum ",
			enumName,
			"enum-values-sorted-alphabetically",
		); err != nil {
			messageParts := strings.SplitN(err.Message, ": ", splitNParts)

			suppressionValue := ""
			if len(messageParts) > 1 {
				suppressionValue = messageParts[1]
			}

			if !pkg_rules.IsSuppressed(
				schemaPath,
				err.LineNum,
				modelsLinterConfig,
				"enum-values-sorted-alphabetically",
				suppressionValue,
			) {
				errors = append(errors, *err)
			}
		}
	}

	return errors
}

func (r Rule) MissingDeprecationReasons(doc *ast.Document, schemaString string) []models.DescriptionError {
	var errors []models.DescriptionError

	for _, enum := range doc.EnumTypeDefinitions {
		enumName := doc.Input.ByteSliceString(enum.Name)
		for _, valueRef := range enum.EnumValuesDefinition.Refs {
			valueDef := doc.EnumValueDefinitions[valueRef]

			valueName := doc.Input.ByteSliceString(valueDef.EnumValue)
			for _, dirRef := range valueDef.Directives.Refs {
				dir := doc.Directives[dirRef]

				dirName := doc.Input.ByteSliceString(dir.Name)
				if dirName == "deprecated" && len(dir.Arguments.Refs) == 0 {
					lineNum := findLineNumberByText(schemaString, valueName)
					lineContent := GetLineContent(schemaString, lineNum)
					message := "deprecations-have-a-reason: Deprecated enum value '" + enumName + "." +
						valueName + "' is missing a reason."
					errors = append(errors, models.DescriptionError{
						LineNum:     lineNum,
						Message:     message,
						LineContent: lineContent,
					})
				}
			}
		}
	}

	return errors
}

func (r Rule) MissingArgumentDescriptions(doc *ast.Document, schemaString string) []models.DescriptionError {
	var errors []models.DescriptionError

	for _, obj := range doc.ObjectTypeDefinitions {
		for _, fieldRef := range obj.FieldsDefinition.Refs {
			fieldDef := doc.FieldDefinitions[fieldRef]
			for _, argRef := range fieldDef.ArgumentsDefinition.Refs {
				argDef := doc.InputValueDefinitions[argRef]
				if !argDef.Description.IsDefined {
					argName := doc.Input.ByteSliceString(argDef.Name)
					fieldName := doc.Input.ByteSliceString(fieldDef.Name)
					lineNum := findLineNumberByText(schemaString, argName+":")
					lineContent := GetLineContent(schemaString, lineNum)
					message := "arguments-have-descriptions: The '" + argName + "' argument of '" + fieldName +
						"' is missing a description."
					errors = append(errors, models.DescriptionError{
						LineNum:     lineNum,
						Message:     message,
						LineContent: lineContent,
					})
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
) []models.DescriptionError {
	fieldNames := make([]string, len(fieldDefs))
	for i, fieldRef := range fieldDefs {
		fieldNames[i] = getFieldName(fieldRef)
	}

	if len(fieldNames) < minFieldsForSortCheck {
		return nil
	}

	sorted := make([]string, len(fieldNames))
	copy(sorted, fieldNames)
	sort.Strings(sorted)

	for i := range fieldNames {
		if fieldNames[i] != sorted[i] {
			lineNum := findLineNumberByText(schemaString, typeLabel+" "+typeName)
			lineContent := GetLineContent(schemaString, lineNum)
			message := typeLabel + "-fields-sorted-alphabetically: The fields of " +
				typeLabel + " type `" + typeName + "` should be sorted in alphabetical order.\nExpected sorting: " +
				strings.Join(sorted, ", ")

			return []models.DescriptionError{{
				LineNum:     lineNum,
				Message:     message,
				LineContent: lineContent,
			}}
		}
	}

	return nil
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
				lineNum := findLineNumberByText(schemaString, fieldName+":")
				lineContent := GetLineContent(schemaString, lineNum)
				message := fmt.Sprintf(
					"input-object-values-have-descriptions: The input value `%s.%s` is missing a description.",
					inputName,
					fieldName,
				)
				errors = append(errors, models.DescriptionError{
					LineNum:     lineNum,
					Message:     message,
					LineContent: lineContent,
				})
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
			"input ",
			"fields of input type '"+inputName+"'",
			"input-object-fields-sorted-alphabetically",
		); err != nil {
			errors = append(errors, *err)
		}
	}

	return errors
}

func (r Rule) FieldsAreCamelCased(doc *ast.Document, schemaString string) []models.DescriptionError {
	var errors []models.DescriptionError

	for _, obj := range doc.ObjectTypeDefinitions {
		typeName := doc.Input.ByteSliceString(obj.Name)

		for _, fieldRef := range obj.FieldsDefinition.Refs {
			fieldDef := doc.FieldDefinitions[fieldRef]

			fieldName := doc.Input.ByteSliceString(fieldDef.Name)
			if !isCamelCase(fieldName) {
				lineNum := findFieldDefinitionLine(schemaString, fieldName, "")
				lineContent := GetLineContent(schemaString, lineNum)
				message := "fields-are-camel-cased: The field '" + typeName + "." + fieldName + "' is not camel cased."
				errors = append(errors, models.DescriptionError{
					LineNum:     lineNum,
					Message:     message,
					LineContent: lineContent,
				})
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
				lineNum := findLineNumberByText(schemaString, fieldName+":")
				lineContent := GetLineContent(schemaString, lineNum)
				message := "input-object-values-are-camel-cased: The input value `" +
					inputName + "." + fieldName + "` is not camel cased."
				errors = append(errors, models.DescriptionError{
					LineNum:     lineNum,
					Message:     message,
					LineContent: lineContent,
				})
			}
		}
	}

	return errors
}

func (r Rule) RelayPageInfoSpec(doc *ast.Document, schemaString string) []models.DescriptionError {
	for _, obj := range doc.ObjectTypeDefinitions {
		if doc.Input.ByteSliceString(obj.Name) == "PageInfo" {
			return nil
		}
	}

	lineNum := 1
	lineContent := GetLineContent(schemaString, lineNum)
	message := "relay-page-info-spec: A `PageInfo` object type is required as per the Relay spec."

	return []models.DescriptionError{{
		LineNum:     lineNum,
		Message:     message,
		LineContent: lineContent,
	}}
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
			lineNum := findFieldDefinitionLine(schemaString, fieldName, "")
			lineContent := GetLineContent(schemaString, lineNum)
			message := "relay-connection-arguments-spec: A field that returns a Connection Type must include forward" +
				"pagination arguments (`first` and `after`), backward pagination arguments (`last` and `before`), or both as" +
				"per the Relay spec."
			errors = append(errors, models.DescriptionError{
				LineNum:     lineNum,
				Message:     message,
				LineContent: lineContent,
			})
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

		lineNum := findLineNumberByText(schemaString, "type "+typeName)
		lineContent := GetLineContent(schemaString, lineNum)

		if !hasPageInfo {
			message := fmt.Sprintf(
				"relay-connection-types-spec: Connection `%s` is missing the following field: pageInfo.",
				typeName,
			)
			errors = append(errors, models.DescriptionError{
				LineNum:     lineNum,
				Message:     message,
				LineContent: lineContent,
			})
		}

		if !hasEdges {
			message := fmt.Sprintf(
				"relay-connection-types-spec: Connection `%s` is missing the following field: edges.",
				typeName,
			)
			errors = append(errors, models.DescriptionError{
				LineNum:     lineNum,
				Message:     message,
				LineContent: lineContent,
			})
		}
	}

	return errors
}

func (r Rule) MissingQueryRootType(doc *ast.Document, schemaString string) []models.DescriptionError {
	for _, obj := range doc.ObjectTypeDefinitions {
		if doc.Input.ByteSliceString(obj.Name) == "Query" {
			return nil
		}
	}

	lineNum := 1
	lineContent := GetLineContent(schemaString, lineNum)
	message := "invalid-graphql-schema: Query root type must be provided."

	return []models.DescriptionError{{
		LineNum:     lineNum,
		Message:     message,
		LineContent: lineContent,
	}}
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
				lineNum := findLineNumberByText(schemaString, valueName)
				lineContent := GetLineContent(schemaString, lineNum)
				message := "enum-values-have-descriptions: Enum value '" + enumName + "." + valueName +
					"' is missing a description."
				errors = append(errors, models.DescriptionError{
					LineNum:     lineNum,
					Message:     message,
					LineContent: lineContent,
				})
			}
		}
	}

	return errors
}

func (r Rule) MissingTypeDescriptions(doc *ast.Document, schemaString string) []models.DescriptionError {
	var errors []models.DescriptionError

	for _, obj := range doc.ObjectTypeDefinitions {
		if !obj.Description.IsDefined {
			name := doc.Input.ByteSliceString(obj.Name)
			lineNum := findLineNumberByText(schemaString, "type "+name)
			lineContent := GetLineContent(schemaString, lineNum)
			message := "types-have-descriptions: Object type '" + name + "' is missing a description"
			errors = append(errors, models.DescriptionError{
				LineNum:     lineNum,
				Message:     message,
				LineContent: lineContent,
			})
		}
	}

	return errors
}

func (r Rule) MissingFieldDescriptions(doc *ast.Document, schemaString string) []models.DescriptionError {
	var errors []models.DescriptionError

	for _, obj := range doc.ObjectTypeDefinitions {
		typeName := doc.Input.ByteSliceString(obj.Name)
		for _, fieldRef := range obj.FieldsDefinition.Refs {
			fieldDef := doc.FieldDefinitions[fieldRef]
			if !fieldDef.Description.IsDefined {
				fieldName := doc.Input.ByteSliceString(fieldDef.Name)
				lineNum := findFieldDefinitionLine(schemaString, fieldName, "")
				lineContent := GetLineContent(schemaString, lineNum)
				message := "fields-have-descriptions: Field '" + typeName + "." + fieldName + "' is missing a description."
				errors = append(errors, models.DescriptionError{
					LineNum:     lineNum,
					Message:     message,
					LineContent: lineContent,
				})
			}
		}
	}

	return errors
}

func (r Rule) ReportUncapitalizedDescription(
	kind,
	parent,
	name,
	desc,
	schemaString string,
) *models.DescriptionError {
	if isCapitalized(desc) {
		return nil
	}

	var (
		lineNum     int
		lineContent string
		message     string
	)

	switch kind {
	case "type":
		lineNum = findLineNumberByText(schemaString, "type "+name)
		lineContent = GetLineContent(schemaString, lineNum)
		message = "descriptions-are-capitalized: The description for type `" + name + "` should be capitalized."
	case "field":
		lineNum = findFieldDefinitionLine(schemaString, name, "")
		lineContent = GetLineContent(schemaString, lineNum)
		message = "descriptions-are-capitalized: The description for field `" + parent + "." + name +
			"` should be capitalized."
	case "enum":
		lineNum = findLineNumberByText(schemaString, name)
		lineContent = GetLineContent(schemaString, lineNum)
		message = "descriptions-are-capitalized: The description for enum value `" + parent + "." + name +
			"` should be capitalized."
	case "argument":
		lineNum = findLineNumberByText(schemaString, name+":")
		lineContent = GetLineContent(schemaString, lineNum)
		message = "descriptions-are-capitalized: The description for argument `" + parent + "." + name +
			"` should be capitalized."
	}

	return &models.DescriptionError{
		LineNum:     lineNum,
		Message:     message,
		LineContent: lineContent,
	}
}

func (r Rule) UnusedTypes(doc *ast.Document, schemaString string) []models.DescriptionError {
	definedTypes := collectDefinedTypeNames(doc)

	unusedTypeErrors := make([]models.DescriptionError, 0, len(definedTypes))

	markUsedTypes(doc, definedTypes)

	for typeName, isUsed := range definedTypes {
		if isUsed {
			continue
		}

		lineNum := findTypeLineNumber(typeName, schemaString)
		lineContent := GetLineContent(schemaString, lineNum)
		message := fmt.Sprintf(
			"defined-types-are-used: Type '%s' is defined but not used",
			typeName,
		)
		unusedTypeErrors = append(unusedTypeErrors, models.DescriptionError{
			LineNum:     lineNum,
			Message:     message,
			LineContent: lineContent,
		})
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

			if errValue, line := checkInvalidEnumValue(enumName, valueName, schemaContent); errValue != "" {
				errors = append(errors, errValue)
				if line > 0 {
					errorLines = append(errorLines, line)
				}
			}

			if errValue, line := checkSuspiciousEnumValue(
				enumName,
				valueName,
				schemaContent,
				schemaPath,
				modelsLinterConfig,
			); errValue != "" {
				errors = append(errors, errValue)
				if line > 0 {
					errorLines = append(errorLines, line)
					descErrors = append(descErrors, models.DescriptionError{
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
	schemaContent string,
	builtInScalars, definedTypes map[string]bool,
) ([]string, []int) {
	return validateTypeReferences(
		doc,
		schemaContent,
		builtInScalars,
		definedTypes,
		indexSlice(len(doc.FieldDefinitions)),
		func(i int) string { return doc.Input.ByteSliceString(doc.FieldDefinitions[i].Name) },
		func(i int) ast.Type { return doc.Types[doc.FieldDefinitions[i].Type] },
		"invalid-field-types: Field",
	)
}

func (r Rule) ValidateInputFieldTypes(
	doc *ast.Document,
	schemaContent string,
	builtInScalars, definedTypes map[string]bool,
) ([]string, []int) {
	return validateTypeReferences(
		doc,
		schemaContent,
		builtInScalars,
		definedTypes,
		indexSlice(len(doc.InputValueDefinitions)),
		func(i int) string { return doc.Input.ByteSliceString(doc.InputValueDefinitions[i].Name) },
		func(i int) ast.Type { return doc.Types[doc.InputValueDefinitions[i].Type] },
		"invalid-input-field-types: Input field",
	)
}

func checkInvalidEnumValue(enumName, valueName, schemaContent string) (string, int) {
	if isValidEnumValue(valueName) {
		return "", 0
	}

	lineNum := findLineNumberByText(schemaContent, valueName)
	log.Infof(
		"invalid-enum-value: Enum '%s' has invalid value '%s' (line %d)\n",
		enumName,
		valueName,
		lineNum,
	)
	log.Infof(
		"  Enum values should be valid GraphQL identifiers (letters, digits, underscores, no leading digits)\n",
	)

	return valueName, lineNum
}

func checkSuspiciousEnumValue(
	enumName,
	valueName,
	schemaContent,
	schemaPath string,
	modelsLinterConfig *models.LinterConfig,
) (string, int) {
	if !hasSuspiciousEnumValue(valueName) &&
		!hasEmbeddedDigits(valueName) {
		return "", 0
	}

	lineNum := findLineNumberByText(schemaContent, valueName)
	if pkg_rules.IsSuppressed(
		schemaPath,
		lineNum,
		modelsLinterConfig,
		"suspicious-enum-value",
		valueName,
	) {
		return "", 0
	}

	log.Errorf(
		"suspicious-enum-value: Enum '%s' has suspicious value '%s' (line %d)\n",
		enumName,
		valueName,
		lineNum,
	)

	if suggestion := suggestCorrectEnumValue(valueName); suggestion != "" {
		log.Errorf("  Did you mean '%s'?\n", suggestion)
	} else {
		suggestedValue := removeSuffixDigits(valueName)
		log.Errorf("  Did you mean '%s'? Enum values typically don't contain numbers.\n", suggestedValue)
	}

	return valueName, lineNum
}

func validateTypeReferences(
	doc *ast.Document,
	schemaContent string,
	builtInScalars, definedTypes map[string]bool,
	typeRefs []int,
	getName func(int) string,
	getType func(int) ast.Type,
	errorPrefix string,
) ([]string, []int) {
	var (
		errors     []string
		errorLines []int
	)

	for _, ref := range typeRefs {
		fieldName := getName(ref)
		fieldType := getType(ref)

		baseType := getBaseTypeName(doc, fieldType)
		if !builtInScalars[baseType] && !definedTypes[baseType] {
			lineNum := findFieldDefinitionLine(schemaContent, fieldName, baseType)
			if lineNum == 0 {
				lineNum = findLineNumberByText(schemaContent, fieldName+": "+baseType)
			}

			if lineNum == 0 {
				lineNum = findLineNumberByText(schemaContent, fieldName+":")
			}

			log.Errorf(
				"%s '%s' references undefined type '%s' (line %d)\n",
				errorPrefix,
				fieldName,
				baseType,
				lineNum,
			)
			log.Errorf("  Available types: %v\n", getAvailableTypes(builtInScalars, definedTypes))

			if lineNum > 0 {
				errorLines = append(errorLines, lineNum)
			}

			errors = append(errors, fieldName)
		}
	}

	return errors, errorLines
}
