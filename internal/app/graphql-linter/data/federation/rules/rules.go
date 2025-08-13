package rules

import (
	pkgRules "github.com/schubergphilis/graphql-linter/internal/pkg/rules"
	log "github.com/sirupsen/logrus"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/ast"
)

func reportDirectiveError(directiveName, parentName, parentKind string) {
	log.Errorf(
		"invalid-federation-directive: Invalid federation directive '@%s' on %s '%s'",
		directiveName,
		parentKind,
		parentName,
	)

	switch parentKind {
	case "type":
		log.Errorf(
			`  Federation only allows these directives: @key, @external, @requires, @provides, @extends,
          @shareable, @inaccessible, @override, @composeDirective, @interfaceObject, @tag, @deprecated, @specifiedBy,
          @oneOf`,
		)
		pkgRules.SuggestDirective(directiveName, "key")
		pkgRules.SuggestDirective(directiveName, "external")
	case "field":
		log.Errorf(
			`  Federation only allows these directives on fields: @external, @requires, @provides, @shareable,
          @inaccessible, @override, @tag, @deprecated`,
		)
	}
}

func validateDirectives(
	doc *ast.Document,
	directiveRefs []int,
	validDirectives map[string]bool,
	parentName, parentKind string,
) bool {
	hasErrors := false

	for _, directiveRef := range directiveRefs {
		directive := doc.Directives[directiveRef]

		directiveName := doc.Input.ByteSliceString(directive.Name)
		if !validDirectives[directiveName] {
			reportDirectiveError(directiveName, parentName, parentKind)

			hasErrors = true
		}
	}

	return !hasErrors
}

func ValidateDirectiveNames(doc *ast.Document) bool {
	validFederationDirectives := map[string]bool{
		"key":              true,
		"external":         true,
		"requires":         true,
		"provides":         true,
		"extends":          true,
		"shareable":        true,
		"inaccessible":     true,
		"override":         true,
		"composeDirective": true,
		"interfaceObject":  true,
		"tag":              true,
		"deprecated":       true, // Standard GraphQL directive
		"specifiedBy":      true, // Standard GraphQL directive
		"oneOf":            true, // Standard GraphQL directive
	}

	log.Debug("Validating federation directive names...")

	hasErrors := false

	for _, obj := range doc.ObjectTypeDefinitions {
		typeName := doc.Input.ByteSliceString(obj.Name)
		if !validateDirectives(
			doc,
			obj.Directives.Refs,
			validFederationDirectives,
			typeName,
			"type",
		) {
			hasErrors = true
		}
	}

	for _, fieldDef := range doc.FieldDefinitions {
		fieldName := doc.Input.ByteSliceString(fieldDef.Name)
		if !validateDirectives(
			doc,
			fieldDef.Directives.Refs,
			validFederationDirectives,
			fieldName,
			"field",
		) {
			hasErrors = true
		}
	}

	if hasErrors {
		log.Error("Federation directive validation FAILED - schema contains invalid directives")

		return false
	}

	log.Debug("federation directive validation PASSED")

	return true
}
