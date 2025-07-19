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

	if err := WriteUserSchemaToFile(); err != nil {
		return err
	}

	if err := WritePostSchemaToFile(); err != nil {
		return err
	}

	if err := WriteCreateUserInputSchemaToFile(); err != nil {
		return err
	}

	if err := WriteUpdateProfileInputSchemaToFile(); err != nil {
		return err
	}

	if err := WriteCreatePostInputSchemaToFile(); err != nil {
		return err
	}

	if err := WriteUpdateProfileSchemaToFile(); err != nil {
		return err
	}

	if err := WriteLowercaseUserSchemaToFile(); err != nil {
		return err
	}

	if err := WriteBlogPostSchemaToFile(); err != nil {
		return err
	}

	if err := WriteProductSchemaToFile(); err != nil {
		return err
	}

	if err := WriteBlogInputSchemaToFile(); err != nil {
		return err
	}

	if err := WriteNodeInterfaceSchemaToFile(); err != nil {
		return err
	}

	if err := WriteAnimalInterfaceSchemaToFile(); err != nil {
		return err
	}

	if err := WriteResourceInterfaceSchemaToFile(); err != nil {
		return err
	}

	if err := WriteUserInterfaceSchemaToFile(); err != nil {
		return err
	}

	if err := WriteMutationFieldArgsSchemaToFile(); err != nil {
		return err
	}

	if err := WriteMutationFieldArgsDescSchemaToFile(); err != nil {
		return err
	}

	if err := WriteMutationInputArgSchemaToFile(); err != nil {
		return err
	}

	if err := WriteMutationTypeNameSchemaToFile(); err != nil {
		return err
	}

	if err := WriteObjectFieldsCamelCasedSchemaToFile(); err != nil {
		return err
	}

	if err := WriteObjectFieldsDescSchemaToFile(); err != nil {
		return err
	}

	if err := WriteObjectTypeDescSchemaToFile(); err != nil {
		return err
	}

	if err := WriteQueryTypeNameSchemaToFile(); err != nil {
		return err
	}

	if err := WriteRelayConnectionSchemaToFile(); err != nil {
		return err
	}

	if err := WriteRelayEdgeSchemaToFile(); err != nil {
		return err
	}

	if err := WriteFieldsSortedSchemaToFile(); err != nil {
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

func GenerateUserSchema() *ast.Document {
	doc := data.NewDocument()

	userIdx := data.AddObject(doc, "User", "A user object.")
	data.AddFieldToObject(doc, userIdx, "first_name", "String", "") // triggers fields-are-camel-cased
	data.AddFieldToObject(doc, userIdx, "lastName", "String", "The user's last name.")

	pageInfoIdx := data.AddObject(doc, "PageInfo", "Relay-compliant PageInfo object.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasNextPage", "Boolean", "Has next page.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasPreviousPage", "Boolean", "Has previous page.")
	data.AddFieldToObject(doc, pageInfoIdx, "startCursor", "String", "Start cursor.")
	data.AddFieldToObject(doc, pageInfoIdx, "endCursor", "String", "End cursor.")

	queryIdx := data.AddObject(doc, "Query", "Query root.")
	data.AddFieldToObject(doc, queryIdx, "user", "User", "Returns a user.")

	return doc
}

func WriteUserSchemaToFile() error {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	doc := GenerateUserSchema()
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(projectRoot, "test/testdata/graphql/invalid/03-fields-are-camel-cased.graphql")
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func GeneratePostSchema() *ast.Document {
	doc := data.NewDocument()

	postIdx := data.AddObject(doc, "Post", "A post.")
	data.AddFieldToObject(doc, postIdx, "id", "ID!", "The post ID.")
	data.AddFieldToObject(doc, postIdx, "title", "String", "") // triggers fields-have-descriptions

	pageInfoIdx := data.AddObject(doc, "PageInfo", "Relay-compliant PageInfo object.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasNextPage", "Boolean", "Has next page.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasPreviousPage", "Boolean", "Has previous page.")
	data.AddFieldToObject(doc, pageInfoIdx, "startCursor", "String", "Start cursor.")
	data.AddFieldToObject(doc, pageInfoIdx, "endCursor", "String", "End cursor.")

	queryIdx := data.AddObject(doc, "Query", "Query root.")
	data.AddFieldToObject(doc, queryIdx, "post", "Post", "Returns a post.")

	return doc
}

func WritePostSchemaToFile() error {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	doc := GeneratePostSchema()
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(projectRoot, "test/testdata/graphql/invalid/04-fields-have-descriptions.graphql")
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func GenerateCreateUserInputSchema() *ast.Document {
	doc := data.NewDocument()

	data.AddInputObject(doc, "CreateUserInput", "Input for creating a user.", []data.InputField{
		{Name: "FirstName", Type: "String", Description: ""}, // triggers input-object-fields-are-camel-cased
		{Name: "lastName", Type: "String", Description: "The user's last name."},
	})

	pageInfoIdx := data.AddObject(doc, "PageInfo", "Relay-compliant PageInfo object.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasNextPage", "Boolean", "Has next page.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasPreviousPage", "Boolean", "Has previous page.")
	data.AddFieldToObject(doc, pageInfoIdx, "startCursor", "String", "Start cursor.")
	data.AddFieldToObject(doc, pageInfoIdx, "endCursor", "String", "End cursor.")

	queryIdx := data.AddObject(doc, "Query", "Query root.")
	data.AddFieldToObject(doc, queryIdx, "validateUser", "Boolean", "Validates input.")

	return doc
}

func WriteCreateUserInputSchemaToFile() error {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	doc := GenerateCreateUserInputSchema()
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(projectRoot, "test/testdata/graphql/invalid/05-input-object-fields-are-camel-cased.graphql")
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// Scenario 06: input-object-fields-have-descriptions
func GenerateUpdateProfileInputSchema() *ast.Document {
	doc := data.NewDocument()

	data.AddInputObject(doc, "UpdateProfileInput", "Input for updating a profile.", []data.InputField{
		{Name: "age", Type: "Int", Description: "The age of the profile owner."},
		{Name: "address", Type: "String", Description: ""}, // triggers input-object-fields-have-descriptions
	})

	pageInfoIdx := data.AddObject(doc, "PageInfo", "Relay-compliant PageInfo object.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasNextPage", "Boolean", "Has next page.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasPreviousPage", "Boolean", "Has previous page.")
	data.AddFieldToObject(doc, pageInfoIdx, "startCursor", "String", "Start cursor.")
	data.AddFieldToObject(doc, pageInfoIdx, "endCursor", "String", "End cursor.")

	queryIdx := data.AddObject(doc, "Query", "Query root.")
	data.AddFieldToObject(doc, queryIdx, "updateProfile", "Boolean", "Validates input.")

	return doc
}

func WriteUpdateProfileInputSchemaToFile() error {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	doc := GenerateUpdateProfileInputSchema()
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(projectRoot, "test/testdata/graphql/invalid/06-input-object-fields-have-descriptions.graphql")
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// Scenario 07: input-object-type-have-description
func GenerateCreatePostInputSchema() *ast.Document {
	doc := data.NewDocument()

	data.AddInputObject(doc, "CreatePostInput", "", []data.InputField{ // triggers input-object-type-have-description
		{Name: "title", Type: "String", Description: "The title for the post."},
	})

	pageInfoIdx := data.AddObject(doc, "PageInfo", "Relay-compliant PageInfo object.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasNextPage", "Boolean", "Has next page.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasPreviousPage", "Boolean", "Has previous page.")
	data.AddFieldToObject(doc, pageInfoIdx, "startCursor", "String", "Start cursor.")
	data.AddFieldToObject(doc, pageInfoIdx, "endCursor", "String", "End cursor.")

	queryIdx := data.AddObject(doc, "Query", "Query root.")
	data.AddFieldToObject(doc, queryIdx, "createPost", "Boolean", "Validates input.")

	return doc
}

func WriteCreatePostInputSchemaToFile() error {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	doc := GenerateCreatePostInputSchema()
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(projectRoot, "test/testdata/graphql/invalid/07-input-object-type-have-description.graphql")
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// Scenario 08: input-object-type-name-ends-with-input
func GenerateUpdateProfileSchema() *ast.Document {
	doc := data.NewDocument()

	data.AddInputObject(doc, "UpdateProfile", "Profile update input.", []data.InputField{ // triggers input-object-type-name-ends-with-input
		{Name: "age", Type: "Int", Description: "The age of the profile owner."},
	})

	pageInfoIdx := data.AddObject(doc, "PageInfo", "Relay-compliant PageInfo object.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasNextPage", "Boolean", "Has next page.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasPreviousPage", "Boolean", "Has previous page.")
	data.AddFieldToObject(doc, pageInfoIdx, "startCursor", "String", "Start cursor.")
	data.AddFieldToObject(doc, pageInfoIdx, "endCursor", "String", "End cursor.")

	queryIdx := data.AddObject(doc, "Query", "Query root.")
	data.AddFieldToObject(doc, queryIdx, "updateProfile", "Boolean", "Validates input.")

	return doc
}

func WriteUpdateProfileSchemaToFile() error {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	doc := GenerateUpdateProfileSchema()
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(projectRoot, "test/testdata/graphql/invalid/08-input-object-type-name-ends-with-input.graphql")
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func GenerateNodeInterfaceSchema() *ast.Document {
	doc := data.NewDocument()

	pageInfoIdx := data.AddObject(doc, "PageInfo", "Relay-compliant PageInfo object.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasNextPage", "Boolean", "Has next page.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasPreviousPage", "Boolean", "Has previous page.")
	data.AddFieldToObject(doc, pageInfoIdx, "startCursor", "String", "Start cursor.")
	data.AddFieldToObject(doc, pageInfoIdx, "endCursor", "String", "End cursor.")

	queryIdx := data.AddObject(doc, "Query", "Query root.")
	data.AddFieldToObject(doc, queryIdx, "dummy", "Boolean", "Returns true.")

	return doc
}

func WriteNodeInterfaceSchemaToFile() error {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	doc := GenerateNodeInterfaceSchema()
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(projectRoot, "test/testdata/graphql/invalid/09-interface-fields-are-camel-cased.graphql")
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func GenerateLowercaseUserSchema() *ast.Document {
	doc := data.NewDocument()

	userIdx := data.AddObject(doc, "user", "The user type.")
	data.AddFieldToObject(doc, userIdx, "id", "ID!", "ID.")

	pageInfoIdx := data.AddObject(doc, "PageInfo", "Relay-compliant PageInfo object.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasNextPage", "Boolean", "Has next page.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasPreviousPage", "Boolean", "Has previous page.")
	data.AddFieldToObject(doc, pageInfoIdx, "startCursor", "String", "Start cursor.")
	data.AddFieldToObject(doc, pageInfoIdx, "endCursor", "String", "End cursor.")

	queryIdx := data.AddObject(doc, "Query", "Query root.")
	data.AddFieldToObject(doc, queryIdx, "user", "user", "Returns a user.")

	return doc
}

func WriteLowercaseUserSchemaToFile() error {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	doc := GenerateLowercaseUserSchema()
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(projectRoot, "test/testdata/graphql/invalid/20-object-type-name.graphql")
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func GenerateBlogPostSchema() *ast.Document {
	doc := data.NewDocument()

	blogPostIdx := data.AddObject(doc, "blogPost", "A blog post.")
	data.AddFieldToObject(doc, blogPostIdx, "id", "ID!", "ID.")

	pageInfoIdx := data.AddObject(doc, "PageInfo", "Relay-compliant PageInfo object.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasNextPage", "Boolean", "Has next page.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasPreviousPage", "Boolean", "Has previous page.")
	data.AddFieldToObject(doc, pageInfoIdx, "startCursor", "String", "Start cursor.")
	data.AddFieldToObject(doc, pageInfoIdx, "endCursor", "String", "End cursor.")

	queryIdx := data.AddObject(doc, "Query", "Query root.")
	data.AddFieldToObject(doc, queryIdx, "blogPost", "blogPost", "Returns a blog post.")

	return doc
}

func WriteBlogPostSchemaToFile() error {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	doc := GenerateBlogPostSchema()
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(projectRoot, "test/testdata/graphql/invalid/25-type-name-shape.graphql")
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func GenerateProductSchema() *ast.Document {
	doc := data.NewDocument()

	productIdx := data.AddObject(doc, "Product", "")
	data.AddFieldToObject(doc, productIdx, "id", "ID!", "The product id.")
	data.AddFieldToObject(doc, productIdx, "name", "String", "Product name.")

	pageInfoIdx := data.AddObject(doc, "PageInfo", "Relay-compliant PageInfo object.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasNextPage", "Boolean", "Has next page.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasPreviousPage", "Boolean", "Has previous page.")
	data.AddFieldToObject(doc, pageInfoIdx, "startCursor", "String", "Start cursor.")
	data.AddFieldToObject(doc, pageInfoIdx, "endCursor", "String", "End cursor.")

	queryIdx := data.AddObject(doc, "Query", "Query root.")
	data.AddFieldToObject(doc, queryIdx, "product", "Product", "Returns a product.")

	return doc
}

func WriteProductSchemaToFile() error {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	doc := GenerateProductSchema()
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(projectRoot, "test/testdata/graphql/invalid/26-types-have-descriptions.graphql")
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func GenerateBlogInputSchema() *ast.Document {
	doc := data.NewDocument()

	blogInputIdx := data.AddObject(doc, "BlogInput", "Should be an input, but is an object.")
	data.AddFieldToObject(doc, blogInputIdx, "id", "ID!", "ID.")

	pageInfoIdx := data.AddObject(doc, "PageInfo", "Relay-compliant PageInfo object.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasNextPage", "Boolean", "Has next page.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasPreviousPage", "Boolean", "Has previous page.")
	data.AddFieldToObject(doc, pageInfoIdx, "startCursor", "String", "Start cursor.")
	data.AddFieldToObject(doc, pageInfoIdx, "endCursor", "String", "End cursor.")

	queryIdx := data.AddObject(doc, "Query", "Query root.")
	data.AddFieldToObject(doc, queryIdx, "blogInput", "BlogInput", "Returns a blog input.")

	return doc
}

func WriteBlogInputSchemaToFile() error {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	doc := GenerateBlogInputSchema()
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(projectRoot, "test/testdata/graphql/invalid/27-type-name-ends-with-input.graphql")
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func GenerateAnimalInterfaceSchema() *ast.Document {
	doc := data.NewDocument()

	pageInfoIdx := data.AddObject(doc, "PageInfo", "Relay-compliant PageInfo object.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasNextPage", "Boolean", "Has next page.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasPreviousPage", "Boolean", "Has previous page.")
	data.AddFieldToObject(doc, pageInfoIdx, "startCursor", "String", "Start cursor.")
	data.AddFieldToObject(doc, pageInfoIdx, "endCursor", "String", "End cursor.")

	queryIdx := data.AddObject(doc, "Query", "Query root.")
	data.AddFieldToObject(doc, queryIdx, "dummy", "Boolean", "Returns true.")

	return doc
}

func WriteAnimalInterfaceSchemaToFile() error {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	doc := GenerateAnimalInterfaceSchema()
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(projectRoot, "test/testdata/graphql/invalid/10-interface-fields-have-descriptions.graphql")
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func GenerateResourceInterfaceSchema() *ast.Document {
	doc := data.NewDocument()

	pageInfoIdx := data.AddObject(doc, "PageInfo", "Relay-compliant PageInfo object.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasNextPage", "Boolean", "Has next page.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasPreviousPage", "Boolean", "Has previous page.")
	data.AddFieldToObject(doc, pageInfoIdx, "startCursor", "String", "Start cursor.")
	data.AddFieldToObject(doc, pageInfoIdx, "endCursor", "String", "End cursor.")

	queryIdx := data.AddObject(doc, "Query", "Query root.")
	data.AddFieldToObject(doc, queryIdx, "dummy", "Boolean", "Returns true.")

	return doc
}

func WriteResourceInterfaceSchemaToFile() error {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	doc := GenerateResourceInterfaceSchema()
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(projectRoot, "test/testdata/graphql/invalid/11-interface-type-have-description.graphql")
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func GenerateUserInterfaceSchema() *ast.Document {
	doc := data.NewDocument()

	pageInfoIdx := data.AddObject(doc, "PageInfo", "Relay-compliant PageInfo object.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasNextPage", "Boolean", "Has next page.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasPreviousPage", "Boolean", "Has previous page.")
	data.AddFieldToObject(doc, pageInfoIdx, "startCursor", "String", "Start cursor.")
	data.AddFieldToObject(doc, pageInfoIdx, "endCursor", "String", "End cursor.")

	queryIdx := data.AddObject(doc, "Query", "Query root.")
	data.AddFieldToObject(doc, queryIdx, "dummy", "Boolean", "Returns true.")

	return doc
}

func WriteUserInterfaceSchemaToFile() error {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	doc := GenerateUserInterfaceSchema()
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(projectRoot, "test/testdata/graphql/invalid/12-interface-type-name.graphql")
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func GenerateMutationFieldArgsSchema() *ast.Document {
	doc := data.NewDocument()

	pageInfoIdx := data.AddObject(doc, "PageInfo", "Relay-compliant PageInfo object.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasNextPage", "Boolean", "Has next page.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasPreviousPage", "Boolean", "Has previous page.")
	data.AddFieldToObject(doc, pageInfoIdx, "startCursor", "String", "Start cursor.")
	data.AddFieldToObject(doc, pageInfoIdx, "endCursor", "String", "End cursor.")

	queryIdx := data.AddObject(doc, "Query", "Query root.")
	data.AddFieldToObject(doc, queryIdx, "dummy", "Boolean", "Returns true.")

	return doc
}

func WriteMutationFieldArgsSchemaToFile() error {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	doc := GenerateMutationFieldArgsSchema()
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(projectRoot, "test/testdata/graphql/invalid/13-mutation-field-arguments-are-camel-cased.graphql")
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func GenerateMutationFieldArgsDescSchema() *ast.Document {
	doc := data.NewDocument()

	pageInfoIdx := data.AddObject(doc, "PageInfo", "Relay-compliant PageInfo object.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasNextPage", "Boolean", "Has next page.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasPreviousPage", "Boolean", "Has previous page.")
	data.AddFieldToObject(doc, pageInfoIdx, "startCursor", "String", "Start cursor.")
	data.AddFieldToObject(doc, pageInfoIdx, "endCursor", "String", "End cursor.")

	queryIdx := data.AddObject(doc, "Query", "Query root.")
	data.AddFieldToObject(doc, queryIdx, "dummy", "Boolean", "Returns true.")

	return doc
}

func WriteMutationFieldArgsDescSchemaToFile() error {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	doc := GenerateMutationFieldArgsDescSchema()
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(projectRoot, "test/testdata/graphql/invalid/14-mutation-field-arguments-have-descriptions.graphql")
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func GenerateMutationInputArgSchema() *ast.Document {
	doc := data.NewDocument()

	pageInfoIdx := data.AddObject(doc, "PageInfo", "Relay-compliant PageInfo object.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasNextPage", "Boolean", "Has next page.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasPreviousPage", "Boolean", "Has previous page.")
	data.AddFieldToObject(doc, pageInfoIdx, "startCursor", "String", "Start cursor.")
	data.AddFieldToObject(doc, pageInfoIdx, "endCursor", "String", "End cursor.")

	queryIdx := data.AddObject(doc, "Query", "Query root.")
	data.AddFieldToObject(doc, queryIdx, "dummy", "Boolean", "Returns true.")

	return doc
}

func WriteMutationInputArgSchemaToFile() error {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	doc := GenerateMutationInputArgSchema()
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(projectRoot, "test/testdata/graphql/invalid/15-mutation-input-arg.graphql")
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func GenerateMutationTypeNameSchema() *ast.Document {
	doc := data.NewDocument()

	pageInfoIdx := data.AddObject(doc, "PageInfo", "Relay-compliant PageInfo object.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasNextPage", "Boolean", "Has next page.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasPreviousPage", "Boolean", "Has previous page.")
	data.AddFieldToObject(doc, pageInfoIdx, "startCursor", "String", "Start cursor.")
	data.AddFieldToObject(doc, pageInfoIdx, "endCursor", "String", "End cursor.")

	queryIdx := data.AddObject(doc, "Query", "Query root.")
	data.AddFieldToObject(doc, queryIdx, "dummy", "Boolean", "Returns true.")

	return doc
}

func WriteMutationTypeNameSchemaToFile() error {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	doc := GenerateMutationTypeNameSchema()
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(projectRoot, "test/testdata/graphql/invalid/16-mutation-type-name.graphql")
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func GenerateObjectFieldsCamelCasedSchema() *ast.Document {
	doc := data.NewDocument()

	blogIdx := data.AddObject(doc, "Blog", "A blog.")
	data.AddFieldToObject(doc, blogIdx, "Post_Title", "String", "")
	data.AddFieldToObject(doc, blogIdx, "author", "String", "The author of the blog.")

	pageInfoIdx := data.AddObject(doc, "PageInfo", "Relay-compliant PageInfo object.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasNextPage", "Boolean", "Has next page.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasPreviousPage", "Boolean", "Has previous page.")
	data.AddFieldToObject(doc, pageInfoIdx, "startCursor", "String", "Start cursor.")
	data.AddFieldToObject(doc, pageInfoIdx, "endCursor", "String", "End cursor.")

	queryIdx := data.AddObject(doc, "Query", "Query root.")
	data.AddFieldToObject(doc, queryIdx, "blog", "Blog", "Returns a blog.")

	return doc
}

func WriteObjectFieldsCamelCasedSchemaToFile() error {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	doc := GenerateObjectFieldsCamelCasedSchema()
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(projectRoot, "test/testdata/graphql/invalid/17-object-fields-are-camel-cased.graphql")
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func GenerateObjectFieldsDescSchema() *ast.Document {
	doc := data.NewDocument()

	employeeIdx := data.AddObject(doc, "Employee", "An employee.")
	data.AddFieldToObject(doc, employeeIdx, "name", "String", "The employee name.")
	data.AddFieldToObject(doc, employeeIdx, "department", "String", "")

	pageInfoIdx := data.AddObject(doc, "PageInfo", "Relay-compliant PageInfo object.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasNextPage", "Boolean", "Has next page.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasPreviousPage", "Boolean", "Has previous page.")
	data.AddFieldToObject(doc, pageInfoIdx, "startCursor", "String", "Start cursor.")
	data.AddFieldToObject(doc, pageInfoIdx, "endCursor", "String", "End cursor.")

	queryIdx := data.AddObject(doc, "Query", "Query root.")
	data.AddFieldToObject(doc, queryIdx, "employee", "Employee", "Returns an employee.")

	return doc
}

func WriteObjectFieldsDescSchemaToFile() error {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	doc := GenerateObjectFieldsDescSchema()
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(projectRoot, "test/testdata/graphql/invalid/18-object-fields-have-descriptions.graphql")
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func GenerateObjectTypeDescSchema() *ast.Document {
	doc := data.NewDocument()

	companyIdx := data.AddObject(doc, "Company", "")
	data.AddFieldToObject(doc, companyIdx, "id", "ID!", "The company ID.")

	pageInfoIdx := data.AddObject(doc, "PageInfo", "Relay-compliant PageInfo object.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasNextPage", "Boolean", "Has next page.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasPreviousPage", "Boolean", "Has previous page.")
	data.AddFieldToObject(doc, pageInfoIdx, "startCursor", "String", "Start cursor.")
	data.AddFieldToObject(doc, pageInfoIdx, "endCursor", "String", "End cursor.")

	queryIdx := data.AddObject(doc, "Query", "Query root.")
	data.AddFieldToObject(doc, queryIdx, "company", "Company", "Returns a company.")

	return doc
}

func WriteObjectTypeDescSchemaToFile() error {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	doc := GenerateObjectTypeDescSchema()
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(projectRoot, "test/testdata/graphql/invalid/19-object-type-have-description.graphql")
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func GenerateQueryTypeNameSchema() *ast.Document {
	doc := data.NewDocument()

	pageInfoIdx := data.AddObject(doc, "PageInfo", "Relay-compliant PageInfo object.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasNextPage", "Boolean", "Has next page.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasPreviousPage", "Boolean", "Has previous page.")
	data.AddFieldToObject(doc, pageInfoIdx, "startCursor", "String", "Start cursor.")
	data.AddFieldToObject(doc, pageInfoIdx, "endCursor", "String", "End cursor.")

	rootQueryIdx := data.AddObject(doc, "RootQuery", "Custom root query.")
	data.AddFieldToObject(doc, rootQueryIdx, "dummy", "Boolean", "Returns true.")

	return doc
}

func WriteQueryTypeNameSchemaToFile() error {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	doc := GenerateQueryTypeNameSchema()
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(projectRoot, "test/testdata/graphql/invalid/21-query-type-name.graphql")
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func GenerateRelayConnectionSchema() *ast.Document {
	doc := data.NewDocument()

	postIdx := data.AddObject(doc, "Post", "A post.")
	data.AddFieldToObject(doc, postIdx, "id", "ID!", "The post id.")

	postConnectionIdx := data.AddObject(doc, "PostConnection", "Post connection which is missing relay keys.")
	data.AddListFieldToObject(doc, postConnectionIdx, "items", "Post", "This should be 'edges' and 'pageInfo' per relay.")

	pageInfoIdx := data.AddObject(doc, "PageInfo", "Relay-compliant PageInfo object.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasNextPage", "Boolean", "Has next page.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasPreviousPage", "Boolean", "Has previous page.")
	data.AddFieldToObject(doc, pageInfoIdx, "startCursor", "String", "Start cursor.")
	data.AddFieldToObject(doc, pageInfoIdx, "endCursor", "String", "End cursor.")

	queryIdx := data.AddObject(doc, "Query", "Query root.")
	data.AddFieldToObject(doc, queryIdx, "posts", "PostConnection", "Returns a post connection.")

	return doc
}

func WriteRelayConnectionSchemaToFile() error {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	doc := GenerateRelayConnectionSchema()
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(projectRoot, "test/testdata/graphql/invalid/22-relay-connection-types-spec.graphql")
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func GenerateRelayEdgeSchema() *ast.Document {
	doc := data.NewDocument()

	postIdx := data.AddObject(doc, "Post", "A post.")
	data.AddFieldToObject(doc, postIdx, "id", "ID!", "The post id.")

	postEdgeIdx := data.AddObject(doc, "PostEdge", "Post edge, missing cursor field.")
	data.AddFieldToObject(doc, postEdgeIdx, "node", "Post", "Node ref.")

	pageInfoIdx := data.AddObject(doc, "PageInfo", "Relay-compliant PageInfo object.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasNextPage", "Boolean", "Has next page.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasPreviousPage", "Boolean", "Has previous page.")
	data.AddFieldToObject(doc, pageInfoIdx, "startCursor", "String", "Start cursor.")
	data.AddFieldToObject(doc, pageInfoIdx, "endCursor", "String", "End cursor.")

	queryIdx := data.AddObject(doc, "Query", "Query root.")
	data.AddFieldToObject(doc, queryIdx, "postEdge", "PostEdge", "Returns a post edge.")

	return doc
}

func WriteRelayEdgeSchemaToFile() error {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	doc := GenerateRelayEdgeSchema()
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(projectRoot, "test/testdata/graphql/invalid/23-relay-edge-types-spec.graphql")
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func GenerateFieldsSortedSchema() *ast.Document {
	doc := data.NewDocument()

	personIdx := data.AddObject(doc, "Person", "A person.")
	data.AddFieldToObject(doc, personIdx, "id", "ID!", "The ID.")
	data.AddFieldToObject(doc, personIdx, "name", "String", "The name.")
	data.AddFieldToObject(doc, personIdx, "age", "Int", "The age.")

	pageInfoIdx := data.AddObject(doc, "PageInfo", "Relay-compliant PageInfo object.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasNextPage", "Boolean", "Has next page.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasPreviousPage", "Boolean", "Has previous page.")
	data.AddFieldToObject(doc, pageInfoIdx, "startCursor", "String", "Start cursor.")
	data.AddFieldToObject(doc, pageInfoIdx, "endCursor", "String", "End cursor.")

	queryIdx := data.AddObject(doc, "Query", "Query root.")
	data.AddFieldToObject(doc, queryIdx, "person", "Person", "Returns a person.")

	return doc
}

func WriteFieldsSortedSchemaToFile() error {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	doc := GenerateFieldsSortedSchema()
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(projectRoot, "test/testdata/graphql/invalid/24-type-fields-sorted-alphabetically.graphql")
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
