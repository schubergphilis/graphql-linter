//go:build integration

package data

import (
	"os"
	"path/filepath"
	"testing"
)

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
		wantVerbose  bool
	}{
		{
			name:         "no config file",
			configYAML:   "",
			wantStrict:   true,
			wantSuppress: 0,
		},
		{
			name:         "with verbose logging",
			configYAML:   "",
			wantStrict:   true,
			wantSuppress: 0,
			wantVerbose:  true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()

			store := Store{TargetPath: dir}
			if test.wantVerbose {
				store.Verbose = true
			}

			configPath := filepath.Join(dir, ".graphql-linter.yml")
			if test.configYAML != "" {
				err := os.WriteFile(configPath, []byte(test.configYAML), 0o600)
				if err != nil {
					t.Fatalf("failed to write config: %v", err)
				}

				store.LinterConfigPath = configPath
			}

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
