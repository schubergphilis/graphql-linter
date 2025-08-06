//go:build integration

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

func TestStore_ReadAndValidateSchemaFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		content  string
		file     string
		exists   bool
		wantStr  string
		wantBool bool
	}{
		{
			name:     "file exists",
			content:  "type Query { id: ID }",
			file:     "test_exists.graphql",
			exists:   true,
			wantStr:  "type Query { id: ID }",
			wantBool: true,
		},
		{
			name:     "file does not exist",
			content:  "",
			file:     "test_missing.graphql",
			exists:   false,
			wantStr:  "",
			wantBool: false,
		},
	}

	dir := t.TempDir()
	store := Store{}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			path := dir + "/" + test.file
			if test.exists {
				err := os.WriteFile(path, []byte(test.content), 0o600)
				if err != nil {
					t.Fatalf("failed to write file: %v", err)
				}
			}

			gotStr, gotBool := store.ReadAndValidateSchemaFile(path)
			if gotStr != test.wantStr {
				t.Errorf("gotStr = %q, want %q", gotStr, test.wantStr)
			}

			if gotBool != test.wantBool {
				t.Errorf("gotBool = %v, want %v", gotBool, test.wantBool)
			}
		})
	}
}

func TestIntegrationLoadConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		configYAML   string
		wantStrict   bool
		wantSuppress int
	}{
		{
			name:         "no config file",
			configYAML:   "",
			wantStrict:   true,
			wantSuppress: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()

			configPath := filepath.Join(dir, ".graphql-linter.yml")
			if test.configYAML != "" {
				err := os.WriteFile(configPath, []byte(test.configYAML), 0o600)
				if err != nil {
					t.Fatalf("failed to write config: %v", err)
				}
			}

			store := Store{TargetPath: dir}

			config, err := store.LoadConfig()
			if err != nil {
				t.Fatalf("LoadConfig error: %v", err)
			}

			if config.Settings.StrictMode != test.wantStrict {
				t.Errorf("StrictMode got %v, want %v", config.Settings.StrictMode, test.wantStrict)
			}

			if len(config.Suppressions) != test.wantSuppress {
				t.Errorf("Suppressions got %d, want %d", len(config.Suppressions), test.wantSuppress)
			}
		})
	}
}
