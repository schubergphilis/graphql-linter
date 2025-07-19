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
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	testdataBaseDir := filepath.Join(projectRoot, "test", "testdata", "graphql", "base")
	testdataInvalidDir := filepath.Join(testdataBaseDir, "invalid")

	if err := os.RemoveAll(testdataInvalidDir); err != nil {
		return fmt.Errorf("unable to remove directory: '%s'. Error: %w", testdataInvalidDir, err)
	}

	writers := []func() error{
		WriteTestSchemaToFile,
		WritePrioritySchemaToFile,
		WriteUserSchemaToFile,
		WritePostSchemaToFile,
		WriteUpdateProfileInputSchemaToFile,
		WriteCreatePostInputSchemaToFile,
		WriteUpdateProfileSchemaToFile,
		WriteLowercaseUserSchemaToFile,
		WriteBlogPostSchemaToFile,
		WriteProductSchemaToFile,
		WriteBlogInputSchemaToFile,
		WriteNodeInterfaceSchemaToFile,
		WriteAnimalInterfaceSchemaToFile,
		WriteResourceInterfaceSchemaToFile,
		WriteUserInterfaceSchemaToFile,
		WriteMutationFieldArgsSchemaToFile,
		WriteMutationFieldArgsDescSchemaToFile,
		WriteMutationInputArgSchemaToFile,
		WriteMutationTypeNameSchemaToFile,
		WriteObjectFieldsCamelCasedSchemaToFile,
		WriteObjectFieldsDescSchemaToFile,
		WriteObjectTypeDescSchemaToFile,
		WriteQueryTypeNameSchemaToFile,
		WriteRelayConnectionSchemaToFile,
		WriteRelayEdgeSchemaToFile,
		WriteFieldsSortedSchemaToFile,
		WriteArgumentsHaveDescriptionsSchemaToFile,
		WriteDefinedTypesAreUsedSchemaToFile,
		WriteDeprecationsHaveAReasonSchemaToFile,
		WriteDescriptionsAreCapitalizedSchemaToFile,
		WriteEnumValuesHaveDescriptionsSchemaToFile,
		WriteInputObjectFieldsSortedAlphabeticallySchemaToFile,
		WriteInputObjectValuesAreCamelCasedSchemaToFile,
		WriteInputObjectValuesHaveDescriptionsSchemaToFile,
		WriteInterfaceFieldsSortedAlphabeticallySchemaToFile,
		WriteRelayConnectionArgumentsSpecSchemaToFile,
		WriteQueryRootMustBeProvidedSchemaToFile,
	}

	for _, writer := range writers {
		if err := writer(); err != nil {
			return err
		}
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
	data.AddNonNullFieldToObject(
		doc,
		pageInfoIdx,
		"hasPreviousPage",
		"Boolean",
		"Has previous page.",
	)
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

	outputPath := filepath.Join(
		projectRoot,
		"test/testdata/graphql/base/invalid/01-enum-values-all-caps.graphql",
	)
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
		{
			Name:        "MEDIUM",
			Description: "Medium priority.",
		}, // Should be after LOW for alphabetical order
		{Name: "LOW", Description: "Low priority."},
	})

	pageInfoIdx := data.AddObject(doc, "PageInfo", "Relay-compliant PageInfo object.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasNextPage", "Boolean", "Has next page.")
	data.AddNonNullFieldToObject(
		doc,
		pageInfoIdx,
		"hasPreviousPage",
		"Boolean",
		"Has previous page.",
	)
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

	outputPath := filepath.Join(
		projectRoot,
		"test/testdata/graphql/base/invalid/02-enum-values-sorted-alphabetically.graphql",
	)
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
	data.AddFieldToObject(
		doc,
		userIdx,
		"first_name",
		"String",
		"",
	) // triggers fields-are-camel-cased
	data.AddFieldToObject(doc, userIdx, "lastName", "String", "The user's last name.")

	pageInfoIdx := data.AddObject(doc, "PageInfo", "Relay-compliant PageInfo object.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasNextPage", "Boolean", "Has next page.")
	data.AddNonNullFieldToObject(
		doc,
		pageInfoIdx,
		"hasPreviousPage",
		"Boolean",
		"Has previous page.",
	)
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

	outputPath := filepath.Join(
		projectRoot,
		"test/testdata/graphql/base/invalid/03-fields-are-camel-cased.graphql",
	)
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
	data.AddNonNullFieldToObject(
		doc,
		pageInfoIdx,
		"hasPreviousPage",
		"Boolean",
		"Has previous page.",
	)
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

	outputPath := filepath.Join(
		projectRoot,
		"test/testdata/graphql/base/invalid/04-fields-have-descriptions.graphql",
	)
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
		{
			Name:        "FirstName",
			Type:        "String",
			Description: "",
		}, // triggers input-object-fields-are-camel-cased
		{Name: "lastName", Type: "String", Description: "The user's last name."},
	})

	pageInfoIdx := data.AddObject(doc, "PageInfo", "Relay-compliant PageInfo object.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasNextPage", "Boolean", "Has next page.")
	data.AddNonNullFieldToObject(
		doc,
		pageInfoIdx,
		"hasPreviousPage",
		"Boolean",
		"Has previous page.",
	)
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

	outputPath := filepath.Join(
		projectRoot,
		"test/testdata/graphql/base/invalid/05-input-object-fields-are-camel-cased.graphql",
	)
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func GenerateUpdateProfileInputSchema() *ast.Document {
	doc := data.NewDocument()

	data.AddInputObject(
		doc,
		"UpdateProfileInput",
		"Input for updating a profile.",
		[]data.InputField{
			{Name: "age", Type: "Int", Description: "The age of the profile owner."},
			{
				Name:        "address",
				Type:        "String",
				Description: "",
			}, // triggers input-object-fields-have-descriptions
		},
	)

	pageInfoIdx := data.AddObject(doc, "PageInfo", "Relay-compliant PageInfo object.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasNextPage", "Boolean", "Has next page.")
	data.AddNonNullFieldToObject(
		doc,
		pageInfoIdx,
		"hasPreviousPage",
		"Boolean",
		"Has previous page.",
	)
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

	outputPath := filepath.Join(
		projectRoot,
		"test/testdata/graphql/base/invalid/06-input-object-fields-have-descriptions.graphql",
	)
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func GenerateCreatePostInputSchema() *ast.Document {
	doc := data.NewDocument()

	data.AddInputObject(
		doc,
		"CreatePostInput",
		"",
		[]data.InputField{ // triggers input-object-type-have-description
			{Name: "title", Type: "String", Description: "The title for the post."},
		},
	)

	pageInfoIdx := data.AddObject(doc, "PageInfo", "Relay-compliant PageInfo object.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasNextPage", "Boolean", "Has next page.")
	data.AddNonNullFieldToObject(
		doc,
		pageInfoIdx,
		"hasPreviousPage",
		"Boolean",
		"Has previous page.",
	)
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

	outputPath := filepath.Join(
		projectRoot,
		"test/testdata/graphql/base/invalid/07-input-object-type-have-description.graphql",
	)
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func GenerateUpdateProfileSchema() *ast.Document {
	doc := data.NewDocument()

	data.AddInputObject(
		doc,
		"UpdateProfile",
		"Profile update input.",
		[]data.InputField{ // triggers input-object-type-name-ends-with-input
			{Name: "age", Type: "Int", Description: "The age of the profile owner."},
		},
	)

	pageInfoIdx := data.AddObject(doc, "PageInfo", "Relay-compliant PageInfo object.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasNextPage", "Boolean", "Has next page.")
	data.AddNonNullFieldToObject(
		doc,
		pageInfoIdx,
		"hasPreviousPage",
		"Boolean",
		"Has previous page.",
	)
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

	outputPath := filepath.Join(
		projectRoot,
		"test/testdata/graphql/base/invalid/08-input-object-type-name-ends-with-input.graphql",
	)
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
	data.AddNonNullFieldToObject(
		doc,
		pageInfoIdx,
		"hasPreviousPage",
		"Boolean",
		"Has previous page.",
	)
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

	outputPath := filepath.Join(
		projectRoot,
		"test/testdata/graphql/base/invalid/09-interface-fields-are-camel-cased.graphql",
	)
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
	data.AddNonNullFieldToObject(
		doc,
		pageInfoIdx,
		"hasPreviousPage",
		"Boolean",
		"Has previous page.",
	)
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

	outputPath := filepath.Join(
		projectRoot,
		"test/testdata/graphql/base/invalid/20-object-type-name.graphql",
	)
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
	data.AddNonNullFieldToObject(
		doc,
		pageInfoIdx,
		"hasPreviousPage",
		"Boolean",
		"Has previous page.",
	)
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

	outputPath := filepath.Join(
		projectRoot,
		"test/testdata/graphql/base/invalid/25-type-name-shape.graphql",
	)
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
	data.AddNonNullFieldToObject(
		doc,
		pageInfoIdx,
		"hasPreviousPage",
		"Boolean",
		"Has previous page.",
	)
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

	outputPath := filepath.Join(
		projectRoot,
		"test/testdata/graphql/base/invalid/26-types-have-descriptions.graphql",
	)
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

	outputPath := filepath.Join(
		projectRoot,
		"test/testdata/graphql/base/invalid/27-type-name-ends-with-input.graphql",
	)
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func GenerateArgumentsHaveDescriptionsSchema() *ast.Document {
	doc := data.NewDocument()
	objIdx := data.AddObject(doc, "Mutation", "Mutation root.")
	// Argument missing description
	data.AddFieldWithArgsToObject(
		doc,
		objIdx,
		"doSomething",
		"Boolean",
		"Does something.",
		[]data.Argument{{Name: "input", Type: "String", Description: ""}},
	)

	return doc
}

func WriteArgumentsHaveDescriptionsSchemaToFile() error {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	doc := GenerateArgumentsHaveDescriptionsSchema()
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(
		projectRoot,
		"test/testdata/graphql/base/invalid/28-arguments-have-descriptions.graphql",
	)
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func GenerateDefinedTypesAreUsedSchema() *ast.Document {
	doc := data.NewDocument()
	_ = data.AddObject(doc, "UnusedType", "This type is not used.")
	queryIdx := data.AddObject(doc, "Query", "Query root.")
	data.AddFieldToObject(doc, queryIdx, "dummy", "Boolean", "Dummy field.")

	return doc
}

func WriteDefinedTypesAreUsedSchemaToFile() error {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	doc := GenerateDefinedTypesAreUsedSchema()
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(
		projectRoot,
		"test/testdata/graphql/base/invalid/29-defined-types-are-used.graphql",
	)
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func GenerateDeprecationsHaveAReasonSchema() *ast.Document {
	doc := data.NewDocument()
	_ = data.AddEnum(doc, "Status", "Status enum.", []data.EnumValue{
		{Name: "OLD", Description: "Old status."},
	})
	queryIdx := data.AddObject(doc, "Query", "Query root.")
	data.AddFieldToObject(doc, queryIdx, "status", "Status", "Returns status.")

	return doc
}

func WriteDeprecationsHaveAReasonSchemaToFile() error {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	doc := GenerateDeprecationsHaveAReasonSchema()
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(
		projectRoot,
		"test/testdata/graphql/base/invalid/30-deprecations-have-a-reason.graphql",
	)
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func GenerateDescriptionsAreCapitalizedSchema() *ast.Document {
	doc := data.NewDocument()
	objIdx := data.AddObject(doc, "Query", "query root.")                  // not capitalized
	data.AddFieldToObject(doc, objIdx, "dummy", "Boolean", "dummy field.") // not capitalized

	return doc
}

func WriteDescriptionsAreCapitalizedSchemaToFile() error {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	doc := GenerateDescriptionsAreCapitalizedSchema()
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(
		projectRoot,
		"test/testdata/graphql/base/invalid/31-descriptions-are-capitalized.graphql",
	)
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func GenerateEnumValuesHaveDescriptionsSchema() *ast.Document {
	doc := data.NewDocument()
	_ = data.AddEnum(
		doc,
		"Color",
		"Colors.",
		[]data.EnumValue{
			{Name: "RED", Description: ""},
			{Name: "BLUE", Description: "Blue color."},
		},
	)
	queryIdx := data.AddObject(doc, "Query", "Query root.")
	data.AddFieldToObject(doc, queryIdx, "color", "Color", "Returns a color.")

	return doc
}

func WriteEnumValuesHaveDescriptionsSchemaToFile() error {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	doc := GenerateEnumValuesHaveDescriptionsSchema()
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(
		projectRoot,
		"test/testdata/graphql/base/invalid/32-enum-values-have-descriptions.graphql",
	)
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func GenerateInputObjectFieldsSortedAlphabeticallySchema() *ast.Document {
	doc := data.NewDocument()
	_ = data.AddInputObject(
		doc,
		"InputType",
		"Input type.",
		[]data.InputField{
			{Name: "zeta", Type: "String", Description: "Zeta field."},
			{Name: "alpha", Type: "String", Description: "Alpha field."},
		},
	)
	queryIdx := data.AddObject(doc, "Query", "Query root.")
	data.AddFieldToObject(doc, queryIdx, "input", "InputType", "Returns input.")

	return doc
}

func WriteInputObjectFieldsSortedAlphabeticallySchemaToFile() error {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	doc := GenerateInputObjectFieldsSortedAlphabeticallySchema()
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(
		projectRoot,
		"test/testdata/graphql/base/invalid/33-input-object-fields-sorted-alphabetically.graphql",
	)
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func GenerateInputObjectValuesAreCamelCasedSchema() *ast.Document {
	doc := data.NewDocument()

	_ = data.AddInputObject(
		doc,
		"InputType",
		"Input type.",
		[]data.InputField{{Name: "not_camel_case", Type: "String", Description: "Not camel case."}},
	)
	outputIdx := data.AddObject(doc, "DummyOutput", "Dummy output type.")
	data.AddFieldToObject(doc, outputIdx, "dummy", "String", "Dummy field.")
	queryIdx := data.AddObject(doc, "Query", "Query root.")
	data.AddFieldToObject(doc, queryIdx, "dummy", "DummyOutput", "Returns dummy output.")

	return doc
}

func WriteInputObjectValuesAreCamelCasedSchemaToFile() error {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	doc := GenerateInputObjectValuesAreCamelCasedSchema()
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(
		projectRoot,
		"test/testdata/graphql/base/invalid/05-input-object-values-are-camel-cased.graphql",
	)
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func GenerateInputObjectValuesHaveDescriptionsSchema() *ast.Document {
	doc := data.NewDocument()
	_ = data.AddInputObject(
		doc,
		"InputType",
		"Input type.",
		[]data.InputField{{Name: "value", Type: "String", Description: ""}},
	)
	queryIdx := data.AddObject(doc, "Query", "Query root.")
	data.AddFieldToObject(doc, queryIdx, "input", "InputType", "Returns input.")

	return doc
}

func WriteInputObjectValuesHaveDescriptionsSchemaToFile() error {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	doc := GenerateInputObjectValuesHaveDescriptionsSchema()
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(
		projectRoot,
		"test/testdata/graphql/base/invalid/35-input-object-values-have-descriptions.graphql",
	)
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func GenerateInterfaceFieldsSortedAlphabeticallySchema() *ast.Document {
	doc := data.NewDocument()
	ifaceIdx := data.AddInterface(doc, "TestInterface", "Test interface.")
	data.AddFieldToInterface(doc, ifaceIdx, "zeta", "String", "Zeta field.")
	data.AddFieldToInterface(doc, ifaceIdx, "alpha", "String", "Alpha field.")
	objIdx := data.AddObject(doc, "Query", "Query root.")
	data.AddFieldToObject(doc, objIdx, "iface", "TestInterface", "Returns interface.")

	return doc
}

func WriteInterfaceFieldsSortedAlphabeticallySchemaToFile() error {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	doc := GenerateInterfaceFieldsSortedAlphabeticallySchema()
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(
		projectRoot,
		"test/testdata/graphql/base/invalid/36-interface-fields-sorted-alphabetically.graphql",
	)
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func GenerateRelayConnectionArgumentsSpecSchema() *ast.Document {
	doc := data.NewDocument()
	postIdx := data.AddObject(doc, "Post", "A post.")
	data.AddFieldToObject(doc, postIdx, "id", "ID!", "The post id.")
	postConnectionIdx := data.AddObject(doc, "PostConnection", "Post connection.")
	// Missing required pagination arguments
	data.AddFieldToObject(doc, postConnectionIdx, "edges", "[Post]", "Edges.")
	data.AddFieldToObject(doc, postConnectionIdx, "pageInfo", "PageInfo!", "Page info.")
	pageInfoIdx := data.AddObject(doc, "PageInfo", "Relay-compliant PageInfo object.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasNextPage", "Boolean", "Has next page.")
	data.AddNonNullFieldToObject(
		doc,
		pageInfoIdx,
		"hasPreviousPage",
		"Boolean",
		"Has previous page.",
	)
	data.AddFieldToObject(doc, pageInfoIdx, "startCursor", "String", "Start cursor.")
	data.AddFieldToObject(doc, pageInfoIdx, "endCursor", "String", "End cursor.")
	queryIdx := data.AddObject(doc, "Query", "Query root.")
	data.AddFieldToObject(doc, queryIdx, "posts", "PostConnection", "Returns a post connection.")

	return doc
}

func WriteRelayConnectionArgumentsSpecSchemaToFile() error {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	doc := GenerateRelayConnectionArgumentsSpecSchema()
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(
		projectRoot,
		"test/testdata/graphql/base/invalid/37-relay-connection-arguments-spec.graphql",
	)
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func WriteRelayConnectionArgumentsSpec2SchemaToFile() error {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	doc := data.NewDocument()
	postIdx := data.AddObject(doc, "Post", "A post.")
	data.AddFieldToObject(doc, postIdx, "id", "ID!", "The post id.")
	postConnectionIdx := data.AddObject(doc, "PostConnection", "Post connection.")
	data.AddFieldToObject(doc, postConnectionIdx, "edges", "[Post]", "Edges.")
	data.AddFieldToObject(doc, postConnectionIdx, "pageInfo", "PageInfo!", "Page info.")
	pageInfoIdx := data.AddObject(doc, "PageInfo", "Relay-compliant PageInfo object.")
	data.AddNonNullFieldToObject(doc, pageInfoIdx, "hasNextPage", "Boolean", "Has next page.")
	data.AddNonNullFieldToObject(
		doc,
		pageInfoIdx,
		"hasPreviousPage",
		"Boolean",
		"Has previous page.",
	)
	data.AddFieldToObject(doc, pageInfoIdx, "startCursor", "String", "Start cursor.")
	data.AddFieldToObject(doc, pageInfoIdx, "endCursor", "String", "End cursor.")
	queryIdx := data.AddObject(doc, "Query", "Query root.")
	data.AddFieldToObject(doc, queryIdx, "posts", "PostConnection", "Returns a post connection.")
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(
		projectRoot,
		"test/testdata/graphql/base/invalid/38-relay-connection-arguments-spec.graphql",
	)
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func WriteAnimalInterfaceSchemaToFile() error {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	doc := data.NewDocument()
	ifaceIdx := data.AddInterface(doc, "Animal", "")              // missing description
	data.AddFieldToInterface(doc, ifaceIdx, "name", "String", "") // missing field description
	objIdx := data.AddObject(doc, "Query", "Query root.")
	data.AddFieldToObject(doc, objIdx, "animal", "Animal", "Returns animal.")
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(
		projectRoot,
		"test/testdata/graphql/base/invalid/10-interface-fields-have-descriptions.graphql",
	)
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func WriteResourceInterfaceSchemaToFile() error {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	doc := data.NewDocument()
	ifaceIdx := data.AddInterface(doc, "Resource", "") // missing description
	data.AddFieldToInterface(doc, ifaceIdx, "id", "ID", "Resource id.")
	objIdx := data.AddObject(doc, "Query", "Query root.")
	data.AddFieldToObject(doc, objIdx, "resource", "Resource", "Returns resource.")
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(
		projectRoot,
		"test/testdata/graphql/base/invalid/11-interface-type-have-description.graphql",
	)
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func WriteUserInterfaceSchemaToFile() error {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	doc := data.NewDocument()
	ifaceIdx := data.AddInterface(doc, "user", "A user interface.") // lowercase name
	data.AddFieldToInterface(doc, ifaceIdx, "id", "ID", "User id.")
	objIdx := data.AddObject(doc, "Query", "Query root.")
	data.AddFieldToObject(doc, objIdx, "user", "user", "Returns user.")
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(
		projectRoot,
		"test/testdata/graphql/base/invalid/12-interface-type-name.graphql",
	)
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func WriteMutationFieldArgsSchemaToFile() error {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	doc := data.NewDocument()
	objIdx := data.AddObject(doc, "Mutation", "Mutation root.")
	data.AddFieldWithArgsToObject(
		doc,
		objIdx,
		"doSomething",
		"Boolean",
		"Does something.",
		[]data.Argument{{Name: "not_camel_case", Type: "String", Description: "Argument."}},
	)

	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(
		projectRoot,
		"test/testdata/graphql/base/invalid/13-mutation-field-arguments-are-camel-cased.graphql",
	)
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func WriteMutationFieldArgsDescSchemaToFile() error {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	doc := data.NewDocument()
	objIdx := data.AddObject(doc, "Mutation", "Mutation root.")
	data.AddFieldWithArgsToObject(
		doc,
		objIdx,
		"doSomething",
		"Boolean",
		"Does something.",
		[]data.Argument{{Name: "input", Type: "String", Description: ""}},
	)

	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(
		projectRoot,
		"test/testdata/graphql/base/invalid/14-mutation-field-arguments-have-descriptions.graphql",
	)
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func WriteMutationInputArgSchemaToFile() error {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	doc := data.NewDocument()
	objIdx := data.AddObject(doc, "Mutation", "Mutation root.")
	data.AddFieldWithArgsToObject(
		doc,
		objIdx,
		"doSomething",
		"Boolean",
		"Does something.",
		[]data.Argument{{Name: "input", Type: "InputType", Description: "Input argument."}},
	)

	_ = data.AddInputObject(
		doc,
		"InputType",
		"Input type.",
		[]data.InputField{{Name: "field", Type: "String", Description: "Field."}},
	)

	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(
		projectRoot,
		"test/testdata/graphql/base/invalid/15-mutation-input-arg.graphql",
	)
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func WriteMutationTypeNameSchemaToFile() error {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	doc := data.NewDocument()
	objIdx := data.AddObject(doc, "mutation", "Mutation root.") // lowercase name
	data.AddFieldToObject(doc, objIdx, "dummy", "Boolean", "Dummy field.")
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(
		projectRoot,
		"test/testdata/graphql/base/invalid/16-mutation-type-name.graphql",
	)
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func WriteObjectFieldsCamelCasedSchemaToFile() error {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	doc := data.NewDocument()
	objIdx := data.AddObject(doc, "TestType", "Test type.")
	data.AddFieldToObject(doc, objIdx, "not_camel_case", "String", "Not camel case.")
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(
		projectRoot,
		"test/testdata/graphql/base/invalid/17-object-fields-are-camel-cased.graphql",
	)
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func WriteObjectFieldsDescSchemaToFile() error {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	doc := data.NewDocument()
	objIdx := data.AddObject(doc, "TestType", "Test type.")
	data.AddFieldToObject(doc, objIdx, "field", "String", "") // missing description
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(
		projectRoot,
		"test/testdata/graphql/base/invalid/18-object-fields-have-descriptions.graphql",
	)
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func WriteObjectTypeDescSchemaToFile() error {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	doc := data.NewDocument()
	_ = data.AddObject(doc, "TestType", "") // missing description
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(
		projectRoot,
		"test/testdata/graphql/base/invalid/19-object-type-have-description.graphql",
	)
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func WriteQueryTypeNameSchemaToFile() error {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	doc := data.NewDocument()
	_ = data.AddObject(doc, "query", "Query root.") // lowercase name
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(
		projectRoot,
		"test/testdata/graphql/base/invalid/21-query-type-name.graphql",
	)
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func WriteRelayConnectionSchemaToFile() error {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	doc := data.NewDocument()
	_ = data.AddObject(doc, "PostConnection", "") // missing description
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(
		projectRoot,
		"test/testdata/graphql/base/invalid/22-relay-connection-types-spec.graphql",
	)
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func WriteRelayEdgeSchemaToFile() error {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	doc := data.NewDocument()
	_ = data.AddObject(doc, "PostEdge", "") // missing description
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(
		projectRoot,
		"test/testdata/graphql/base/invalid/23-relay-edge-types-spec.graphql",
	)
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func WriteFieldsSortedSchemaToFile() error {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	doc := data.NewDocument()
	objIdx := data.AddObject(doc, "TestType", "Test type.")
	data.AddFieldToObject(doc, objIdx, "zeta", "String", "Zeta field.")
	data.AddFieldToObject(doc, objIdx, "alpha", "String", "Alpha field.")
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(
		projectRoot,
		"test/testdata/graphql/base/invalid/24-type-fields-sorted-alphabetically.graphql",
	)
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func WriteQueryRootMustBeProvidedSchemaToFile() error {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return fmt.Errorf("failed to determine project root: %w", err)
	}

	doc := data.NewDocument()
	mutationIdx := data.AddObject(doc, "Mutation", "Mutation root.")
	data.AddFieldToObject(doc, mutationIdx, "dummy", "String", "Dummy field.")
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(
		projectRoot,
		"test/testdata/graphql/base/invalid/38-query-root-must-be-provided.graphql",
	)
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
