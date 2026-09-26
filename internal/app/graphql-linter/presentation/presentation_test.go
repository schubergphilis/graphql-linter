package presentation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCLI(t *testing.T) {
	t.Parallel()

	cli := NewCLI([]string{"-targetPath", "schemas", "-configPath", "c.yml", "-verbose"}, "1.0.0")

	assert.Equal(t, "1.0.0", cli.version)
	assert.Equal(t, "schemas", cli.targetPathFlag)
	assert.Equal(t, "c.yml", cli.configPathFlag)
	assert.False(t, cli.versionFlag)
	assert.True(t, cli.verboseFlag)
}
