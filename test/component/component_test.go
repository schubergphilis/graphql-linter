//go:build component

package component

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	setup()

	code := m.Run()

	teardown()

	os.Exit(code)
}

func setup() {
	// Setup code here, if needed
}

func teardown() {
	// Teardown code here, if needed
}
