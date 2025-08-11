//go:build integration

package data

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/schubergphilis/mcvs-golang-project-root/pkg/projectroot"
	"github.com/stretchr/testify/require"
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

				store.ConfigPath = configPath
			}

			config, err := store.LoadConfig()
			if err != nil {
				t.Fatalf("LoadConfig error: %v", err)
			}

			if config.Settings.StrictMode != test.wantStrict {
				t.Errorf("StrictMode got %v, want %v", config.Settings.StrictMode, test.wantStrict)
			}

			if len(config.Suppressions) != test.wantSuppress {
				t.Errorf(
					"Suppressions got %d, want %d",
					len(config.Suppressions),
					test.wantSuppress,
				)
			}
		})
	}
}

//nolint:paralleltest,tparallel //t.Chdir cannot run in parallel
func TestLoadConfig(t *testing.T) {
	projectRoot, err := projectroot.FindProjectRoot()
	require.NoError(t, err, "failed to determine project root")

	t.Run("no configPath, no .graphql-linter.yml", testNoConfigPathNoConfigFile)
	t.Run("no configPath, .graphql-linter.yml exists", func(t *testing.T) {
		oldWd, _ := os.Getwd()

		defer func() {
			t.Chdir(oldWd)
		}()

		t.Chdir(projectRoot)

		configContent := "suppressions:\n  - rule: test-rule\n    file: test.graphql\n    line: 1\n" +
			"    value: test value\n    reason: test reason\n"

		err = os.WriteFile(".graphql-linter.yml", []byte(configContent), 0o600)
		require.NoError(t, err, "failed to write config file")

		defer func() {
			err := os.Remove(".graphql-linter.yml")
			require.NoError(t, err, "failed to remove config file")
		}()

		store := Store{ConfigPath: "", Verbose: false}

		config, err := store.LoadConfig()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if config == nil || len(config.Suppressions) != 1 {
			t.Fatalf("expected config with suppressions, got %+v", config)
		}
	})

	t.Run("configPath provided, file exists", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		configPath := dir + "/custom.yml"
		configContent := "suppressions:\n  - rule: test-rule\n    file: test.graphql\n    line: 1\n" +
			"    value: test value\n    reason: test reason\n"
		err := os.WriteFile(configPath, []byte(configContent), 0o600)
		require.NoError(t, err, "failed to write config file")

		defer func() {
			err := os.Remove(configPath)
			require.NoError(t, err, "failed to remove config file")
		}()

		store := Store{ConfigPath: configPath, Verbose: false}

		config, err := store.LoadConfig()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if config == nil || len(config.Suppressions) != 1 {
			t.Fatalf("expected config with suppressions, got %+v", config)
		}
	})

	t.Run("configPath provided, file does not exist", func(t *testing.T) {
		t.Parallel()

		store := Store{ConfigPath: "some-config-file-that-does-not-exist", Verbose: false}
		config, err := store.LoadConfig()
		require.Error(t, err, "expected error when loading config")
		require.Nil(t, config, "expected nil config when error occurs")
	})
}

func testNoConfigPathNoConfigFile(t *testing.T) {
	oldWd, _ := os.Getwd()

	defer func() {
		t.Chdir(oldWd)
	}()

	t.Chdir(".")

	store := Store{ConfigPath: "", Verbose: false}

	config, err := store.LoadConfig()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if config == nil {
		t.Fatalf("expected config, got nil")
	}
}
