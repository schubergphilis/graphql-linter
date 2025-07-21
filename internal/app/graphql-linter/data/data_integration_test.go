package data

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestFile(t *testing.T, rootDir, relPath string) string {
	t.Helper()

	rootDirPath := filepath.Join(rootDir, relPath)

	err := os.MkdirAll(filepath.Dir(rootDirPath), 0o755)
	if err != nil {
		t.Fatalf("mkdir error: %v", err)
	}

	f, err := os.Create(rootDirPath)
	if err != nil {
		t.Fatalf("file create error: %v", err)
	}

	err = f.Close()
	require.NoError(t, err, "unable to close file")

	return rootDirPath
}

func TestFindGraphQLFiles(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	files := []string{
		"a.graphql",
		"b.graphqls",
		"notgraphql.txt",
		".hidden/test.graphql",
		"normaldir/test2.graphql",
		"node_modules/lib/test3.graphql",
		"vendor/conf/test4.graphql",
		".git/test5.graphql",
	}
	for _, rel := range files {
		createTestFile(t, tmpDir, rel)
	}

	tests := []struct {
		name        string
		expectFiles []string
	}{
		{
			name: "Find all matching GraphQL files, skipping hidden/vendor/node_modules/.git",
			expectFiles: []string{
				filepath.Join(tmpDir, "a.graphql"),
				filepath.Join(tmpDir, "b.graphqls"),
				filepath.Join(tmpDir, "normaldir/test2.graphql"),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := findGraphQLFiles(tmpDir)
			require.NoError(t, err)

			assert.ElementsMatch(t, test.expectFiles, got)
		})
	}
}
