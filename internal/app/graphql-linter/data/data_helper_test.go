package data

import (
	"os"
	"testing"
)

type ErrorWithMessage interface {
	GetMessage() string
}

type TestCase struct {
	Name          string
	SchemaContent string
	SearchText    string
	Input         string
	Expected      interface{}
	ExpectError   bool
	ExpectMsg     string
	WantLine      int
	WantValid     bool
	WantErrLines  int
}

type ErrorTestCase struct {
	Name          string
	SchemaContent string
	ExpectError   bool
	ExpectMsg     string
}

func createTempSchemaFile(t *testing.T, content string) string {
	t.Helper()

	tempFile, err := os.CreateTemp(t.TempDir(), "schema*.graphql")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	_, err = tempFile.WriteString(content)
	if err != nil {
		t.Fatalf("failed to write to temp file: %v", err)
	}

	tempFile.Close()

	return tempFile.Name()
}
