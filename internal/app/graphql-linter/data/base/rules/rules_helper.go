package rules

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/data/base/models"
	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/data/constants"
	pkgRules "github.com/schubergphilis/graphql-linter/internal/pkg/rules"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/ast"
)

func findLineNumberByText(schemaContent string, searchText string) int {
	lines := strings.Split(schemaContent, "\n")

	for i, line := range lines {
		if strings.Contains(line, searchText) {
			return i + 1
		}
	}

	return 0
}

func GetLineContent(schemaContent string, lineNum int) string {
	if lineNum <= 0 {
		return ""
	}

	lines := strings.Split(schemaContent, "\n")
	if lineNum > len(lines) {
		return ""
	}

	return strings.TrimSpace(lines[lineNum-1])
}

func checkSortedOrder(
	names []string,
	minLength int,
	schemaString,
	searchPrefix,
	itemName,
	rulePrefix string,
) *models.DescriptionError {
	fmt.Println("CP 44.0==========================> checkSortedOrder:", itemName, len(names))

	if len(names) < minLength {
		return nil
	}

	fmt.Println("CP 44.1==========================> checkSortedOrder:", itemName, len(names), "minLength:", minLength)

	sorted := make([]string, len(names))
	copy(sorted, names)
	sort.Strings(sorted)

	if !equalStringSlices(names, sorted) {
		lineNum := findLineNumberByText(schemaString, searchPrefix+itemName)
		lineContent := GetLineContent(schemaString, lineNum)
		message := rulePrefix + ": The " + itemName +
			" should be sorted in alphabetical order. Expected sorting: " + strings.Join(
			sorted,
			", ",
		)

		return &models.DescriptionError{
			LineNum:     lineNum,
			Message:     message,
			LineContent: lineContent,
		}
	}

	return nil
}

func equalStringSlices(sliceA, sliceB []string) bool {
	if len(sliceA) != len(sliceB) {
		return false
	}

	for i := range sliceA {
		if sliceA[i] != sliceB[i] {
			return false
		}
	}

	return true
}

func isCamelCase(str string) bool {
	if str == "" {
		return false
	}

	if strings.Contains(str, "_") {
		return false
	}

	return str[0] >= 'a' && str[0] <= 'z'
}

func isCapitalized(desc string) bool {
	desc = strings.TrimSpace(desc)
	if desc == "" {
		return true
	}

	r := rune(desc[0])

	return unicode.IsUpper(r)
}

func suggestCorrectEnumValue(value string) string {
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

	cleanValue := removeAllDigits(value)
	standardTypes := []string{"STRING", "BOOLEAN", "FLOAT", "INT", "INTEGER", "ID"}

	for _, standardType := range standardTypes {
		if cleanValue == standardType {
			return standardType
		}
	}

	for _, standardType := range standardTypes {
		if pkgRules.LevenshteinDistance(cleanValue, standardType) <= pkgRules.LevenshteinThreshold {
			return standardType
		}
	}

	return ""
}

func removeAllDigits(value string) string {
	result := ""

	for _, char := range value {
		if char < '0' || char > '9' {
			result += string(char)
		}
	}

	return result
}

func findFieldDefinitionLine(schemaContent string, fieldName string, typeName string) int {
	lines := strings.Split(schemaContent, "\n")
	for index, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		if strings.Contains(trimmedLine, fieldName+":") {
			if strings.Contains(trimmedLine, typeName+"!") ||
				strings.Contains(trimmedLine, typeName+"]") ||
				strings.Contains(trimmedLine, "["+typeName) ||
				strings.HasSuffix(trimmedLine, typeName) ||
				strings.Contains(trimmedLine, typeName+" ") {
				return index + 1
			}
		}
	}

	return 0
}

func getBaseTypeName(doc *ast.Document, typeRef ast.Type) string {
	switch typeRef.TypeKind {
	case ast.TypeKindNamed:
		return doc.Input.ByteSliceString(typeRef.Name)
	case ast.TypeKindList:
		return getBaseTypeName(doc, doc.Types[typeRef.OfType])
	case ast.TypeKindNonNull:
		return getBaseTypeName(doc, doc.Types[typeRef.OfType])
	case ast.TypeKindUnknown:
		return ""
	default:
		return ""
	}
}

func collectDefinedTypeNames(doc *ast.Document) map[string]bool {
	definedTypes := make(map[string]bool)

	for _, obj := range doc.ObjectTypeDefinitions {
		name := doc.Input.ByteSliceString(obj.Name)
		if name != constants.RootQueryType &&
			name != constants.RootMutationType &&
			name != constants.RootSubscriptionType {
			definedTypes[name] = false
		}
	}

	for _, input := range doc.InputObjectTypeDefinitions {
		name := doc.Input.ByteSliceString(input.Name)
		definedTypes[name] = false
	}

	for _, enum := range doc.EnumTypeDefinitions {
		name := doc.Input.ByteSliceString(enum.Name)
		definedTypes[name] = false
	}

	for _, iface := range doc.InterfaceTypeDefinitions {
		name := doc.Input.ByteSliceString(iface.Name)
		definedTypes[name] = false
	}

	for _, union := range doc.UnionTypeDefinitions {
		name := doc.Input.ByteSliceString(union.Name)
		definedTypes[name] = false
	}

	for _, scalar := range doc.ScalarTypeDefinitions {
		name := doc.Input.ByteSliceString(scalar.Name)
		definedTypes[name] = false
	}

	return definedTypes
}

func markUsedTypes(doc *ast.Document, definedTypes map[string]bool) {
	for _, fieldDef := range doc.FieldDefinitions {
		baseType := getBaseTypeName(doc, doc.Types[fieldDef.Type])
		if _, exists := definedTypes[baseType]; exists {
			definedTypes[baseType] = true
		}
	}

	for _, inputValue := range doc.InputValueDefinitions {
		baseType := getBaseTypeName(doc, doc.Types[inputValue.Type])
		if _, exists := definedTypes[baseType]; exists {
			definedTypes[baseType] = true
		}
	}

	for _, union := range doc.UnionTypeDefinitions {
		for _, memberRef := range union.UnionMemberTypes.Refs {
			memberType := doc.Types[memberRef]

			baseType := getBaseTypeName(doc, memberType)
			if _, exists := definedTypes[baseType]; exists {
				definedTypes[baseType] = true
			}
		}
	}
}

func isValidEnumValue(value string) bool {
	if len(value) == 0 {
		return false
	}

	r := rune(value[0])
	if !unicode.IsLetter(r) && r != '_' {
		return false
	}

	for _, r := range value[1:] {
		if !isAlphaUnderOrDigit(r) {
			return false
		}
	}

	return true
}

func isAlphaUnderOrDigit(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

func findTypeLineNumber(typeName, schemaString string) int {
	searchTerms := []string{
		"type " + typeName,
		"input " + typeName,
		"enum " + typeName,
		"interface " + typeName,
		"union " + typeName,
		"scalar " + typeName,
	}

	for _, term := range searchTerms {
		if lineNum := findLineNumberByText(schemaString, term); lineNum > 0 {
			return lineNum
		}
	}

	return 0
}

func hasSuspiciousEnumValue(value string) bool {
	if len(value) == 0 {
		return false
	}

	lastChar := value[len(value)-1]

	return lastChar >= '0' && lastChar <= '9'
}

func hasEmbeddedDigits(value string) bool {
	for _, char := range value {
		if char >= '0' && char <= '9' {
			return true
		}
	}

	return false
}

func removeSuffixDigits(value string) string {
	result := value
	for len(result) > 0 {
		lastChar := result[len(result)-1]
		if lastChar >= '0' && lastChar <= '9' {
			result = result[:len(result)-1]
		} else {
			break
		}
	}

	return result
}

func indexSlice(n int) []int {
	indices := make([]int, n)
	for i := range indices {
		indices[i] = i
	}

	return indices
}

func getAvailableTypes(builtInScalars, definedTypes map[string]bool) []string {
	types := make([]string, 0, len(builtInScalars)+len(definedTypes))

	for t := range builtInScalars {
		types = append(types, t)
	}

	for t := range definedTypes {
		types = append(types, t)
	}

	return types
}

func CollectDefinedTypes(doc *ast.Document) map[string]bool {
	definedTypes := make(map[string]bool)

	for _, obj := range doc.ObjectTypeDefinitions {
		typeName := doc.Input.ByteSliceString(obj.Name)
		definedTypes[typeName] = true
	}

	for _, enum := range doc.EnumTypeDefinitions {
		typeName := doc.Input.ByteSliceString(enum.Name)
		definedTypes[typeName] = true
	}

	for _, input := range doc.InputObjectTypeDefinitions {
		typeName := doc.Input.ByteSliceString(input.Name)
		definedTypes[typeName] = true
	}

	for _, iface := range doc.InterfaceTypeDefinitions {
		typeName := doc.Input.ByteSliceString(iface.Name)
		definedTypes[typeName] = true
	}

	for _, union := range doc.UnionTypeDefinitions {
		typeName := doc.Input.ByteSliceString(union.Name)
		definedTypes[typeName] = true
	}

	for _, scalar := range doc.ScalarTypeDefinitions {
		typeName := doc.Input.ByteSliceString(scalar.Name)
		definedTypes[typeName] = true
	}

	return definedTypes
}
