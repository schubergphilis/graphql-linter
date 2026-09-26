package rules

import (
	"fmt"
	"slices"
	"strings"

	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/data/base/models"
	base_rules "github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/data/base/rules"
	pkgRules "github.com/schubergphilis/graphql-linter/internal/pkg/rules"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/ast"
)

const (
	// federationPrefix is used when federation directives are imported with a
	// namespace, e.g. @federation__key.
	federationPrefix = "federation__"
	// maxSuggestionDistance is the largest Levenshtein distance for which a
	// "Did you mean" suggestion is offered.
	maxSuggestionDistance = 2
)

// knownDirectives are the Apollo Federation v2.x directives plus the
// directives built into GraphQL.
var knownDirectives = []string{
	"authenticated",
	"composeDirective",
	"context",
	"cost",
	"deprecated",
	"extends",
	"external",
	"fromContext",
	"inaccessible",
	"interfaceObject",
	"key",
	"link",
	"listSize",
	"oneOf",
	"override",
	"policy",
	"provides",
	"requires",
	"requiresScopes",
	"shareable",
	"specifiedBy",
	"tag",
}

// directiveSite is a schema element that can carry directives.
type directiveSite struct {
	kind       string
	name       string
	directives []int
}

// ValidateDirectiveNames reports every directive that is neither a known
// federation or built-in directive, nor defined in the schema, nor composed
// with @composeDirective. It is schema wide: run it on the merged schema.
func ValidateDirectiveNames(doc *ast.Document, schemaString string) []models.DescriptionError {
	allowed := make(map[string]bool, len(knownDirectives)+len(doc.DirectiveDefinitions))
	for _, name := range knownDirectives {
		allowed[name] = true
	}

	for _, def := range doc.DirectiveDefinitions {
		allowed[doc.Input.ByteSliceString(def.Name)] = true
	}

	for _, name := range composedDirectives(doc) {
		allowed[name] = true
	}

	var errors []models.DescriptionError

	for _, site := range directiveSites(doc) {
		for _, ref := range site.directives {
			directive := doc.Directives[ref]

			name := doc.Input.ByteSliceString(directive.Name)
			if allowed[name] || allowed[strings.TrimPrefix(name, federationPrefix)] {
				continue
			}

			message := fmt.Sprintf(
				"invalid-federation-directive: Invalid federation directive '@%s' on %s '%s'.",
				name,
				site.kind,
				site.name,
			)
			if suggestion := closestDirective(name); suggestion != "" {
				message += " Did you mean '@" + suggestion + "'?"
			}

			lineNum := int(directive.At.LineStart)
			errors = append(errors, models.DescriptionError{
				Value:       name,
				LineNum:     lineNum,
				Message:     message,
				LineContent: base_rules.GetLineContent(schemaString, lineNum),
			})
		}
	}

	return errors
}

// composedDirectives returns the names passed to @composeDirective(name: "@x").
func composedDirectives(doc *ast.Document) []string {
	var names []string

	for ref := range doc.Directives {
		if doc.DirectiveNameString(ref) != "composeDirective" {
			continue
		}

		value, ok := doc.DirectiveArgumentValueByName(ref, []byte("name"))
		if ok && value.Kind == ast.ValueKindString {
			names = append(names, strings.TrimPrefix(doc.StringValueContentString(value.Ref), "@"))
		}
	}

	return names
}

func closestDirective(name string) string {
	best, bestDistance := "", maxSuggestionDistance+1

	for _, known := range knownDirectives {
		if distance := pkgRules.LevenshteinDistance(name, known); distance < bestDistance {
			best, bestDistance = known, distance
		}
	}

	return best
}

func directiveSites(doc *ast.Document) []directiveSite {
	sites := fieldSites(doc)

	add := func(kind, name string, directives ast.DirectiveList) {
		sites = append(sites, directiveSite{kind: kind, name: name, directives: directives.Refs})
	}

	for _, def := range doc.InputObjectTypeDefinitions {
		name := doc.Input.ByteSliceString(def.Name)
		add("input", name, def.Directives)

		for _, ref := range def.InputFieldsDefinition.Refs {
			value := doc.InputValueDefinitions[ref]
			add("input value", name+"."+doc.Input.ByteSliceString(value.Name), value.Directives)
		}
	}

	for _, def := range doc.EnumTypeDefinitions {
		name := doc.Input.ByteSliceString(def.Name)
		add("enum", name, def.Directives)

		for _, ref := range def.EnumValuesDefinition.Refs {
			value := doc.EnumValueDefinitions[ref]
			add("enum value", name+"."+doc.Input.ByteSliceString(value.EnumValue), value.Directives)
		}
	}

	for _, def := range doc.UnionTypeDefinitions {
		add("union", doc.Input.ByteSliceString(def.Name), def.Directives)
	}

	for _, def := range doc.ScalarTypeDefinitions {
		add("scalar", doc.Input.ByteSliceString(def.Name), def.Directives)
	}

	return sites
}

// fieldSites returns the object and interface types, including extensions,
// with their fields and arguments.
func fieldSites(doc *ast.Document) []directiveSite {
	var sites []directiveSite

	add := func(kind, name string, directives ast.DirectiveList, fieldRefs []int) {
		sites = append(sites, directiveSite{kind: kind, name: name, directives: directives.Refs})

		for _, ref := range fieldRefs {
			field := doc.FieldDefinitions[ref]
			fieldName := name + "." + doc.Input.ByteSliceString(field.Name)
			sites = append(sites, directiveSite{kind: "field", name: fieldName, directives: field.Directives.Refs})

			for _, argRef := range field.ArgumentsDefinition.Refs {
				arg := doc.InputValueDefinitions[argRef]
				sites = append(sites, directiveSite{
					kind:       "argument",
					name:       fieldName + "." + doc.Input.ByteSliceString(arg.Name),
					directives: arg.Directives.Refs,
				})
			}
		}
	}

	objects := slices.Clone(doc.ObjectTypeDefinitions)
	for _, ext := range doc.ObjectTypeExtensions {
		objects = append(objects, ext.ObjectTypeDefinition)
	}

	for _, def := range objects {
		add("type", doc.Input.ByteSliceString(def.Name), def.Directives, def.FieldsDefinition.Refs)
	}

	interfaces := slices.Clone(doc.InterfaceTypeDefinitions)
	for _, ext := range doc.InterfaceTypeExtensions {
		interfaces = append(interfaces, ext.InterfaceTypeDefinition)
	}

	for _, def := range interfaces {
		add("interface", doc.Input.ByteSliceString(def.Name), def.Directives, def.FieldsDefinition.Refs)
	}

	return sites
}
