package base

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/wundergraph/graphql-go-tools/v2/pkg/ast"

	"github.com/schubergphilis/graphql-linter/internal/app/graphql-testdata-generator/data"
)

const (
	dirPerm  = 0o700
	filePerm = 0o600
)

type Execute struct {
	testdataBaseDir    string
	testdataInvalidDir string
}

func NewExecute(testdataBaseDir, testdataInvalidDir string) Execute {
	return Execute{
		testdataBaseDir:    testdataBaseDir,
		testdataInvalidDir: testdataInvalidDir,
	}
}

func (e Execute) Run() error {
	if err := os.RemoveAll(e.testdataInvalidDir); err != nil {
		return fmt.Errorf("unable to remove directory: '%s'. Error: %w", e.testdataInvalidDir, err)
	}
	writers := []func() error{
		func() error { return WriteArgumentsHaveDescriptionsSchemaToFile(e.testdataInvalidDir) },
		func() error { return WriteDefinedTypesAreUsedSchemaToFile(e.testdataInvalidDir) },
		func() error { return WriteDeprecationsHaveAReasonSchemaToFile(e.testdataInvalidDir) },
		func() error { return WriteDescriptionsAreCapitalizedSchemaToFile(e.testdataInvalidDir) },
		func() error { return WriteTestSchemaToFile(e.testdataInvalidDir) },
		func() error { return WriteEnumValuesHaveDescriptionsSchemaToFile(e.testdataInvalidDir) },
		func() error { return WritePrioritySchemaToFile(e.testdataInvalidDir) },
		func() error { return WriteUserSchemaToFile(e.testdataInvalidDir) },
		func() error { return WritePostSchemaToFile(e.testdataInvalidDir) },
		func() error { return WriteInputObjectFieldsSortedAlphabeticallySchemaToFile(e.testdataInvalidDir) },
		func() error { return WriteInputObjectValuesAreCamelCasedSchemaToFile(e.testdataInvalidDir) },
		func() error { return WriteInputObjectValuesHaveDescriptionsSchemaToFile(e.testdataInvalidDir) },
		func() error { return WriteInterfaceFieldsSortedAlphabeticallySchemaToFile(e.testdataInvalidDir) },
		func() error { return WriteRelayConnectionSchemaToFile(e.testdataInvalidDir) },
		func() error { return WriteRelayConnectionArgumentsSpecSchemaToFile(e.testdataInvalidDir) },
		func() error { return WriteFieldsSortedSchemaToFile(e.testdataInvalidDir) },
		func() error { return WriteBlogPostSchemaToFile(e.testdataInvalidDir) },
		func() error { return WriteProductSchemaToFile(e.testdataInvalidDir) },
		func() error { return WriteQueryRootMustBeProvidedSchemaToFile(e.testdataInvalidDir) },
		func() error { return WriteRelayConnectionArgumentsSpec2SchemaToFile(e.testdataInvalidDir) },
		func() error { return WriteUpdateProfileInputSchemaToFile(e.testdataInvalidDir) },
		func() error { return WriteCreatePostInputSchemaToFile(e.testdataInvalidDir) },
	}
	for i, writer := range writers {
		if err := writer(); err != nil {
			return fmt.Errorf("error in writer %d: %w", i+1, err)
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

func WriteTestSchemaToFile(outputDir string) error {
	doc := GenerateTestSchema()
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(
		outputDir,
		"05-enum-values-all-caps.graphql",
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
		},
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

func WritePrioritySchemaToFile(outputDir string) error {
	doc := GeneratePrioritySchema()
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(
		outputDir,
		"07-enum-values-sorted-alphabetically.graphql",
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
	)
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

func WriteUserSchemaToFile(outputDir string) error {
	doc := GenerateUserSchema()
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(
		outputDir,
		"08-fields-are-camel-cased.graphql",
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
	data.AddFieldToObject(doc, postIdx, "title", "String", "")

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

func WritePostSchemaToFile(outputDir string) error {
	doc := GeneratePostSchema()
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(
		outputDir,
		"09-fields-have-descriptions.graphql",
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
		},
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
			},
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

func WriteUpdateProfileInputSchemaToFile(outputDir string) error {
	doc := GenerateUpdateProfileInputSchema()
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(
		outputDir,
		"12-input-object-values-have-descriptions.graphql",
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
		[]data.InputField{
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

func WriteCreatePostInputSchemaToFile(outputDir string) error {
	doc := GenerateCreatePostInputSchema()
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(
		outputDir,
		"20-types-have-descriptions.graphql",
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
		[]data.InputField{{Name: "age", Type: "Int", Description: "The age of the profile owner."}},
	)

	queryIdx := data.AddObject(doc, "Query", "Query root.")
	data.AddFieldToObject(doc, queryIdx, "updateProfile", "UpdateProfile", "Validates input.")

	return doc
}

func GenerateNodeInterfaceSchema() *ast.Document {
	doc := data.NewDocument()
	ifaceIdx := data.AddInterface(doc, "Node", "A node interface.")
	data.AddFieldToInterface(doc, ifaceIdx, "not_camel_case", "ID", "Not camel case.")
	queryIdx := data.AddObject(doc, "Query", "Query root.")
	data.AddFieldToObject(doc, queryIdx, "node", "Node", "Returns node.")

	return doc
}

func GenerateAnimalInterfaceSchema() *ast.Document {
	doc := data.NewDocument()
	ifaceIdx := data.AddInterface(doc, "Animal", "Animal interface.")
	data.AddFieldToInterface(doc, ifaceIdx, "name", "String", "")
	queryIdx := data.AddObject(doc, "Query", "Query root.")
	data.AddFieldToObject(doc, queryIdx, "animal", "Animal", "Returns animal.")

	return doc
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

func WriteBlogPostSchemaToFile(outputDir string) error {
	doc := GenerateBlogPostSchema()
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(
		outputDir,
		"17-type-name-shape.graphql",
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

func WriteProductSchemaToFile(outputDir string) error {
	doc := GenerateProductSchema()
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(
		outputDir,
		"18-types-have-descriptions.graphql",
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

func GenerateArgumentsHaveDescriptionsSchema() *ast.Document {
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

	return doc
}

func WriteArgumentsHaveDescriptionsSchemaToFile(outputDir string) error {
	doc := GenerateArgumentsHaveDescriptionsSchema()
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(
		outputDir,
		"01-arguments-have-descriptions.graphql",
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

func WriteDefinedTypesAreUsedSchemaToFile(outputDir string) error {
	doc := GenerateDefinedTypesAreUsedSchema()
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(
		outputDir,
		"02-defined-types-are-used.graphql",
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

func WriteDeprecationsHaveAReasonSchemaToFile(outputDir string) error {
	doc := GenerateDeprecationsHaveAReasonSchema()
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(
		outputDir,
		"03-deprecations-have-a-reason.graphql",
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
	objIdx := data.AddObject(doc, "Query", "query root.")
	data.AddFieldToObject(doc, objIdx, "dummy", "Boolean", "dummy field.")

	return doc
}

func WriteDescriptionsAreCapitalizedSchemaToFile(outputDir string) error {
	doc := GenerateDescriptionsAreCapitalizedSchema()
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(
		outputDir,
		"04-descriptions-are-capitalized.graphql",
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

func WriteEnumValuesHaveDescriptionsSchemaToFile(outputDir string) error {
	doc := GenerateEnumValuesHaveDescriptionsSchema()
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(
		outputDir,
		"06-enum-values-have-descriptions.graphql",
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

func WriteInputObjectFieldsSortedAlphabeticallySchemaToFile(outputDir string) error {
	doc := GenerateInputObjectFieldsSortedAlphabeticallySchema()
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(
		outputDir,
		"10-input-object-fields-sorted-alphabetically.graphql",
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

func WriteInputObjectValuesAreCamelCasedSchemaToFile(outputDir string) error {
	doc := GenerateInputObjectValuesAreCamelCasedSchema()
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(
		outputDir,
		"11-input-object-values-are-camel-cased.graphql",
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

func WriteInputObjectValuesHaveDescriptionsSchemaToFile(outputDir string) error {
	doc := GenerateInputObjectValuesHaveDescriptionsSchema()
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(
		outputDir,
		"12-input-object-values-have-descriptions.graphql",
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

func WriteInterfaceFieldsSortedAlphabeticallySchemaToFile(outputDir string) error {
	doc := GenerateInterfaceFieldsSortedAlphabeticallySchema()
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(
		outputDir,
		"13-interface-fields-sorted-alphabetically.graphql",
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

func WriteRelayConnectionArgumentsSpecSchemaToFile(outputDir string) error {
	doc := GenerateRelayConnectionArgumentsSpecSchema()
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(
		outputDir,
		"15-relay-connection-arguments-spec.graphql",
	)
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func WriteRelayConnectionArgumentsSpec2SchemaToFile(outputDir string) error {
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
		outputDir,
		"16-relay-connection-arguments-spec-2.graphql",
	)
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func WriteRelayConnectionSchemaToFile(outputDir string) error {
	doc := data.NewDocument()
	_ = data.AddObject(doc, "PostConnection", "")
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(outputDir, "14-relay-connection-types-spec.graphql")
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func WriteFieldsSortedSchemaToFile(outputDir string) error {
	doc := data.NewDocument()
	objIdx := data.AddObject(doc, "TestType", "Test type.")
	data.AddFieldToObject(doc, objIdx, "zeta", "String", "Zeta field.")
	data.AddFieldToObject(doc, objIdx, "alpha", "String", "Alpha field.")
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(outputDir, "21-type-fields-sorted-alphabetically.graphql")
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

func GenerateQueryRootMustBeProvidedSchema() *ast.Document {
	doc := data.NewDocument()
	mutationIdx := data.AddObject(doc, "Mutation", "Mutation root.")
	data.AddFieldToObject(doc, mutationIdx, "dummy", "String", "Dummy field.")

	return doc
}

func WriteQueryRootMustBeProvidedSchemaToFile(outputDir string) error {
	doc := GenerateQueryRootMustBeProvidedSchema()
	gql := data.GenerateGraphQLFromDocument(doc)

	outputPath := filepath.Join(outputDir, "19-query-root-must-be-provided.graphql")
	if err := os.MkdirAll(filepath.Dir(outputPath), dirPerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(gql), filePerm); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
