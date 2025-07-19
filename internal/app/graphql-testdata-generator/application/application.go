package application

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/schubergphilis/graphql-linter/internal/app/graphql-testdata-generator/data"
	"github.com/schubergphilis/mcvs-golang-project-root/pkg/projectroot"
	"github.com/wundergraph/graphql-go-tools/v2/pkg/ast"
)

const (
	dirPerm     = 0o700
	filePerm    = 0o600
	unknownType = "Unknown"
)

type Executor interface {
	Run() error
}

type Execute struct{}

func NewExecute() Execute {
	execute := Execute{}

	return execute
}

func (e Execute) Run() error {
	if err := WriteTestSchemaToFile(); err != nil {
		return err
	}

	if err := WritePrioritySchemaToFile(); err != nil {
		return err
	}

	return nil
}

func GenerateTestSchema() *ast.Document {
	doc := data.NewDocument()

	data.AddEnum(doc, "Color", "Available colors.", []data.EnumValue{
		{Name: "blue", Description: ""},
		{Name: "red", Description: "Red color."},
	})

	pageInfoIdx := data.AddObject(doc, "PageInfo", "Relay-compliant PageInfo object.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasNextPage", "Boolean", "Has next page.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasPreviousPage", "Boolean", "Has previous page.")
	data.AddFieldToObject(doc, pageInfoIdx, "startCursor", "String", "Start cursor.")
	data.AddFieldToObject(doc, pageInfoIdx, "endCursor", "String", "End cursor.")

	queryIdx := data.AddObject(doc, "Query", "Query root.")
	data.AddFieldToObject(doc, queryIdx, "color", "Color", "Returns a color.")

	return doc
}

func WriteTestSchemaToFile() error {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	doc := GenerateTestSchema()
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(projectRoot, "test/testdata/graphql/invalid/01-enum-values-all-caps.graphql")
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func GeneratePrioritySchema() *ast.Document {
	doc := data.NewDocument()

	data.AddEnum(doc, "Priority", "Priority of the task.", []data.EnumValue{
		{Name: "HIGH", Description: "High priority."},
		{Name: "MEDIUM", Description: "Medium priority."}, // Should be after LOW for alphabetical order
		{Name: "LOW", Description: "Low priority."},
	})

	pageInfoIdx := data.AddObject(doc, "PageInfo", "Relay-compliant PageInfo object.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasNextPage", "Boolean", "Has next page.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasPreviousPage", "Boolean", "Has previous page.")
	data.AddFieldToObject(doc, pageInfoIdx, "startCursor", "String", "Start cursor.")
	data.AddFieldToObject(doc, pageInfoIdx, "endCursor", "String", "End cursor.")

	queryIdx := data.AddObject(doc, "Query", "Query root.")
	data.AddFieldToObject(doc, queryIdx, "priority", "Priority", "Returns a priority.")

	return doc
}

func WritePrioritySchemaToFile() error {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	doc := GeneratePrioritySchema()
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(projectRoot, "test/testdata/graphql/invalid/02-enum-values-sorted-alphabetically.graphql")
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
