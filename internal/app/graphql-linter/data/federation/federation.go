package federation

import (
	"regexp"

	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/data/base/models"
	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/data/base/rules"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/ast"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/astparser"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/asttransform"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/astvalidation"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/operationreport"
)

// ValidateFederationSchema validates the merged schema of all target files as
// one subgraph: unique and known type names, populated type bodies, unique
// fields and enum values, and interface implementations.
//
// Two default definition rules are left out because they reject valid
// subgraphs: RequireDefinedTypesForExtensions (a subgraph extends entities
// that are owned by another subgraph) and DirectivesAreUniquePerLocation
// (@key and @tag are repeatable).
func ValidateFederationSchema(schemaString string) []models.DescriptionError {
	doc, report := astparser.ParseGraphqlDocumentString(schemaString)
	if report.HasErrors() {
		return nil // syntax errors are reported per file
	}

	err := asttransform.MergeDefinitionWithBaseSchema(&doc)
	if err != nil {
		return []models.DescriptionError{newFinding(schemaString, 1, err.Error())}
	}

	validator := astvalidation.NewDefinitionValidator(
		astvalidation.PopulatedTypeBodies(),
		astvalidation.UniqueOperationTypes(),
		astvalidation.UniqueTypeNames(),
		astvalidation.UniqueFieldDefinitionNames(),
		astvalidation.UniqueEnumValueNames(),
		astvalidation.UniqueUnionMemberTypes(),
		astvalidation.KnownTypeNames(),
		astvalidation.ImplementTransitiveInterfaces(),
		astvalidation.ImplementingTypesAreSupersets(),
	)

	var validationReport operationreport.Report

	validator.Validate(&doc, &validationReport)

	findings := make([]models.DescriptionError, 0, len(validationReport.ExternalErrors))
	seen := make(map[string]bool, len(validationReport.ExternalErrors))

	for _, externalErr := range validationReport.ExternalErrors {
		if seen[externalErr.Message] {
			continue
		}

		seen[externalErr.Message] = true

		findings = append(findings, newFinding(schemaString, locate(&doc, externalErr.Message), externalErr.Message))
	}

	return findings
}

// quotedName matches the first 'Type', 'Type.member' or "Type" in a message.
var quotedName = regexp.MustCompile(`['"]([_A-Za-z]\w*)(?:\.([_A-Za-z]\w*))?['"]`)

// locate returns the line for a definition validator message, as those carry
// no location.
//
// ponytail: guesses from the first quoted name in the message: the second
// definition (the duplicate) when there are several, else the definition or
// the first reference. Use real locations once graphql-go-tools reports them.
func locate(doc *ast.Document, message string) int {
	match := quotedName.FindStringSubmatch(message)
	if match == nil {
		return 1
	}

	lines := rules.DefinitionLines(doc, match[1], match[2])
	if len(lines) == 0 {
		lines = rules.ReferenceLines(doc, match[1])
	}

	switch {
	case len(lines) > 1:
		return lines[1]
	case len(lines) == 1:
		return lines[0]
	default:
		return 1
	}
}

func newFinding(schemaString string, lineNum int, message string) models.DescriptionError {
	return models.DescriptionError{
		LineNum:     lineNum,
		Message:     "invalid-federation-schema: " + message,
		LineContent: rules.GetLineContent(schemaString, lineNum),
	}
}
