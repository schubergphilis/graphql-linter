//go:build integration

package data

import (
	"cmp"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIntegrationLoadConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		configYAML   string
		wantDefaults bool
		wantSuppress int
	}{
		{
			name:         "no config file",
			configYAML:   "",
			wantDefaults: true,
			wantSuppress: 0,
		},
		{
			name:         "settings disabled",
			configYAML:   "settings:\n  validateFederation: false\n  checkDescriptions: false\n",
			wantDefaults: false,
			wantSuppress: 0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()

			store := Store{TargetPath: dir}

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

			if config.Settings.ValidateFederation != test.wantDefaults ||
				config.Settings.CheckDescriptions != test.wantDefaults {
				t.Errorf("Settings got %+v, want defaults %v", config.Settings, test.wantDefaults)
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

//nolint:cyclop,paralleltest //t.Chdir cannot run in parallel
func TestLoadConfig(t *testing.T) {
	projectRoot := t.TempDir()

	tests := []struct {
		name           string
		configPath     string
		configContent  string
		defaultFile    string
		chdir          string
		prepareFile    bool
		removeFile     bool
		expectError    bool
		expectSuppress int
	}{
		{
			name:           "no configPath, no .graphql-linter.yml",
			configPath:     "",
			chdir:          ".",
			prepareFile:    false,
			removeFile:     false,
			expectError:    false,
			expectSuppress: 0,
		},
		{
			name:           "no configPath, .graphql-linter.yml exists",
			configPath:     "",
			chdir:          projectRoot,
			prepareFile:    true,
			removeFile:     true,
			expectError:    false,
			expectSuppress: 1,
			configContent: "suppressions:\n  - rule: test-rule\n    file: test.graphql\n    line: 1\n" +
				"    value: test value\n    reason: test reason\n",
		},
		{
			name:           "no configPath, .graphql-linter.yaml exists",
			chdir:          projectRoot,
			defaultFile:    ".graphql-linter.yaml",
			prepareFile:    true,
			removeFile:     true,
			expectSuppress: 1,
			configContent:  "suppressions:\n  - rule: test-rule\n",
		},
		{
			name:           "configPath provided, file exists",
			prepareFile:    true,
			removeFile:     true,
			expectError:    false,
			expectSuppress: 1,
			configContent: "suppressions:\n  - rule: test-rule\n    file: test.graphql\n    line: 1\n" +
				"    value: test value\n    reason: test reason\n",
		},
		{
			name:           "configPath provided, file does not exist",
			prepareFile:    false,
			removeFile:     false,
			expectError:    true,
			expectSuppress: 0,
			configPath:     "some-config-file-that-does-not-exist",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			oldWd, _ := os.Getwd()

			if testCase.chdir != "" {
				defer func() { t.Chdir(oldWd) }()

				t.Chdir(testCase.chdir)
			}

			var configPath string
			switch {
			case testCase.name == "configPath provided, file exists":
				dir := t.TempDir()
				configPath = dir + "/custom.yml"
			case testCase.configPath != "":
				configPath = testCase.configPath
			default:
				configPath = ""
			}

			if testCase.prepareFile {
				file := configPath
				if file == "" {
					file = cmp.Or(testCase.defaultFile, ".graphql-linter.yml")
				}

				err := os.WriteFile(file, []byte(testCase.configContent), 0o600)
				require.NoError(t, err, "failed to write config file")

				if testCase.removeFile {
					defer func() {
						err := os.Remove(file)
						require.NoError(t, err, "failed to remove config file")
					}()
				}
			}

			store := Store{ConfigPath: configPath}

			config, err := store.LoadConfig()

			if testCase.expectError {
				require.Error(t, err, "expected error when loading config")
				require.Nil(t, config, "expected nil config when error occurs")

				return
			}

			require.NoError(t, err, "expected no error")
			require.NotNil(t, config, "expected config, got nil")

			if len(config.Suppressions) != testCase.expectSuppress {
				t.Fatalf("expected %d suppressions, got %+v", testCase.expectSuppress, config)
			}
		})
	}
}
