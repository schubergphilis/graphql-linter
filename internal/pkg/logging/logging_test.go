package logging

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewHandler(t *testing.T) {
	t.Parallel()

	var quiet bytes.Buffer

	logger := slog.New(newHandler(&quiet, false))
	logger.Debug("hidden")
	logger.Info("shown")
	assert.Equal(t, "level=INFO msg=shown\n", quiet.String())

	var verbose bytes.Buffer

	logger = slog.New(newHandler(&verbose, true))
	logger.Debug("shown")
	assert.Contains(t, verbose.String(), "time=")
	assert.Contains(t, verbose.String(), "level=DEBUG")
	assert.Contains(t, verbose.String(), "source=")
}
