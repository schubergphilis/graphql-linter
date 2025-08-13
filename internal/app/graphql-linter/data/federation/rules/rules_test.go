package rules

import (
	"testing"

	"github.com/wundergraph/graphql-go-tools/v2/pkg/ast"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/astparser"
)

func TestValidateDirectiveNames(t *testing.T) {
	t.Parallel()

	doc, _ := astparser.ParseGraphqlDocumentString("type Query { id: ID } directive @key on OBJECT")

	tests := []struct {
		name string
		doc  *ast.Document
		want bool
	}{
		{"valid directives", &doc, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := ValidateDirectiveNames(test.doc)
			if got != test.want {
				t.Errorf("got %v, want %v", got, test.want)
			}
		})
	}
}

func TestValidateDirectiveNames_Invalid(t *testing.T) {
	t.Parallel()

	doc, _ := astparser.ParseGraphqlDocumentString("type Query @invalid { id: ID }")

	got := ValidateDirectiveNames(&doc)
	if got {
		t.Errorf("expected false for invalid directive, got true")
	}
}

func TestValidateDirectives(t *testing.T) {
	t.Parallel()

	doc, _ := astparser.ParseGraphqlDocumentString("type Query { id: ID } directive @key on OBJECT")
	valid := map[string]bool{"key": true}

	tests := []struct {
		name       string
		directives []int
		want       bool
	}{
		{"valid", []int{}, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := validateDirectives(&doc, test.directives, valid, "Query", "type")
			if got != test.want {
				t.Errorf("got %v, want %v", got, test.want)
			}
		})
	}
}

func TestValidateDirectives_MoreCases(t *testing.T) {
	t.Parallel()

	doc, _ := astparser.ParseGraphqlDocumentString("type Query @invalid { id: ID }")
	valid := map[string]bool{"key": true}
	// Find the directive refs for the first object type
	var directiveRefs []int
	if len(doc.ObjectTypeDefinitions) > 0 {
		directiveRefs = doc.ObjectTypeDefinitions[0].Directives.Refs
	}

	got := validateDirectives(&doc, directiveRefs, valid, "Query", "type")
	if got {
		t.Errorf("expected false for invalid directive, got true")
	}

	invalid := map[string]bool{"invalid": false}

	got2 := validateDirectives(&doc, []int{}, invalid, "Query", "type")
	if !got2 {
		t.Errorf("expected true for no directives, got false")
	}
}

func TestReportDirectiveError(t *testing.T) {
	t.Parallel()
	// Just ensure it doesn't panic
	reportDirectiveError("invalid", "Query", "type")
}

func TestReportDirectiveError_AllKinds(t *testing.T) {
	t.Parallel()
	reportDirectiveError("invalid", "Query", "type")
	reportDirectiveError("invalid", "fieldName", "field")
}
