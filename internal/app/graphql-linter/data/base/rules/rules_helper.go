package rules

import (
	"maps"
	"slices"
	"strings"
	"unicode"

	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/data/base/models"
	pkgRules "github.com/schubergphilis/graphql-linter/internal/pkg/rules"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/ast"
)

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
	schemaString string,
	lineNum int,
	itemName,
	rulePrefix string,
) *models.DescriptionError {
	if len(names) < minLength {
		return nil
	}

	if !slices.IsSorted(names) {
		sorted := slices.Sorted(slices.Values(names))
		finding := newFinding(schemaString, lineNum, itemName, rulePrefix+": The "+itemName+
			" should be sorted in alphabetical order. Expected sorting: "+strings.Join(sorted, ", "))

		return &finding
	}

	return nil
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
	var result strings.Builder

	for _, char := range value {
		if char < '0' || char > '9' {
			result.WriteRune(char)
		}
	}

	return result.String()
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

func markUsedTypes(doc *ast.Document, definedTypes map[string]bool) {
	typeRefs := make([]int, 0, len(doc.FieldDefinitions)+len(doc.InputValueDefinitions))

	for _, fieldDef := range doc.FieldDefinitions {
		typeRefs = append(typeRefs, fieldDef.Type)
	}

	for _, inputValue := range doc.InputValueDefinitions {
		typeRefs = append(typeRefs, inputValue.Type)
	}

	for _, union := range doc.UnionTypeDefinitions {
		typeRefs = append(typeRefs, union.UnionMemberTypes.Refs...)
	}

	for _, obj := range doc.ObjectTypeDefinitions {
		typeRefs = append(typeRefs, obj.ImplementsInterfaces.Refs...)
	}

	for _, iface := range doc.InterfaceTypeDefinitions {
		typeRefs = append(typeRefs, iface.ImplementsInterfaces.Refs...)
	}

	for _, typeRef := range typeRefs {
		baseType := getBaseTypeName(doc, doc.Types[typeRef])
		if _, exists := definedTypes[baseType]; exists {
			definedTypes[baseType] = true
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
	return append(slices.Collect(maps.Keys(builtInScalars)), slices.Collect(maps.Keys(definedTypes))...)
}

func CollectDefinedTypes(doc *ast.Document) map[string]bool {
	definedTypes := make(map[string]bool)
	for _, def := range typeDefinitions(doc) {
		definedTypes[def.name] = true
	}

	return definedTypes
}
