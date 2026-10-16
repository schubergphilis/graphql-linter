package rules

import (
	"fmt"

	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/data/base/models"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/ast"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/lexer"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/lexer/keyword"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/lexer/token"
)

const extendKeyword = "extend"

// nesting tells how a token changes the depth, so strings inside a type,
// argument list or list value are not taken for top level descriptions.
var nesting = map[keyword.Keyword]int{
	keyword.LBRACE: 1, keyword.LPAREN: 1, keyword.LBRACK: 1,
	keyword.RBRACE: -1, keyword.RPAREN: -1, keyword.RBRACK: -1,
}

// DescriptionOnExtensions reports a description on an extension such as
// `extend type`. The GraphQL grammar has no description on extensions, the
// description belongs on the definition that is extended. The parser accepts
// and drops it, so the rule scans the tokens: at the top level of a document a
// string is always a description, so a string directly before `extend` is one.
func (r Rule) DescriptionOnExtensions(doc *ast.Document, schemaString string) []models.DescriptionError {
	var errors []models.DescriptionError

	input := &ast.Input{}
	input.ResetInputBytes(doc.Input.RawBytes)

	lex := &lexer.Lexer{}
	lex.SetInput(input)

	depth := 0
	previous := token.Token{Keyword: keyword.UNDEFINED}

	for tok := lex.Read(); tok.Keyword != keyword.EOF; tok = lex.Read() {
		if tok.Keyword == keyword.COMMENT {
			continue
		}

		depth += nesting[tok.Keyword]

		if depth == 0 && tok.Keyword == keyword.IDENT && isDescription(previous) &&
			input.ByteSliceString(tok.Literal) == extendKeyword {
			errors = append(errors, descriptionOnExtension(lex, input, schemaString, previous))
		}

		previous = tok
	}

	return errors
}

func isDescription(tok token.Token) bool {
	return tok.Keyword == keyword.STRING || tok.Keyword == keyword.BLOCKSTRING
}

func descriptionOnExtension(
	lex *lexer.Lexer,
	input *ast.Input,
	schemaString string,
	description token.Token,
) models.DescriptionError {
	kind := input.ByteSliceString(lex.Read().Literal)
	extension := extendKeyword + " " + kind
	value := kind

	// Every extension but `extend schema` is followed by the type name.
	if kind != "schema" {
		value = input.ByteSliceString(lex.Read().Literal)
		extension += " " + value
	}

	return newFinding(schemaString, int(description.TextPosition.LineStart), value, fmt.Sprintf(
		"extend-has-no-description: Description not allowed on '%s'; "+
			"the description belongs on the definition that is extended",
		extension,
	))
}
