package presentation

import (
	"testing"

	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/presentation/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewCLI(t *testing.T) {
	t.Parallel()

	mocksFlagger := &mocks.Flagger{}

	mocksFlagger.EXPECT().StringVar(
		mock.Anything,
		"configPath",
		"",
		"The path to the configuration file (optional, defaults to .graphql-linter.yaml in the current directory)",
	).Times(1)

	mocksFlagger.EXPECT().StringVar(
		mock.Anything,
		"targetPath",
		"",
		"The directory with GraphQL files that should be checked",
	).Times(1)

	mocksFlagger.EXPECT().BoolVar(mock.Anything, "version", false, "Show version").Times(1)
	mocksFlagger.EXPECT().BoolVar(mock.Anything, "verbose", false, "Enable verbose output").Times(1)
	mocksFlagger.EXPECT().Parse().Times(1)

	cli := NewCLI(mocksFlagger, "1.0.0")
	require.NotNil(t, cli)

	assert.Equal(t, "1.0.0", cli.version)
	assert.False(t, cli.versionFlag)
	assert.False(t, cli.verboseFlag)

	mocksFlagger.AssertExpectations(t)
}
