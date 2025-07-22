//go:build component

package component

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/schubergphilis/mcvs-golang-project-root/pkg/projectroot"
	log "github.com/sirupsen/logrus"
)

func TestMain(m *testing.M) {
	setup()

	code := m.Run()

	teardown()

	os.Exit(code)
}

func setup() {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		log.WithError(err).Fatal("failed to find project root")
	}
	mainPath := filepath.Join(projectRoot, "cmd", "graphql-linter", "main.go")
	outputPath := filepath.Join(projectRoot, "graphql-linter")
	cmd := exec.Command("go", "build", "-ldflags=-X 'main.Version=v4.5.6'", "-o", outputPath, mainPath)
	cmd.Stdout = os.Stdout
	var stderrBuf strings.Builder
	cmd.Stderr = &stderrBuf
	err = cmd.Run()
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"command": cmd.String(),
			"stderr":  stderrBuf.String(),
			"stdout":  "os.Stdout",
		}).Fatal("failed to build graphql-linter")
	}
}

func teardown() {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		log.WithError(err).Fatal("failed to find project root during teardown")
	}
	binaryPath := filepath.Join(projectRoot, "graphql-linter")
	err = os.Remove(binaryPath)
	if err != nil && !os.IsNotExist(err) {
		log.WithError(err).WithField("binaryPath", binaryPath).Fatal("failed to remove built binary during teardown")
	}
}

func TestVersion(t *testing.T) {
	t.Parallel()

	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		t.Fatalf("failed to find project root: %v", err)
	}
	binaryPath := filepath.Join(projectRoot, "graphql-linter")
	cmd := exec.Command(binaryPath, "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to run graphql-linter --version: %v", err)
	}
	if !strings.Contains(string(output), "v4.5.6") {
		t.Errorf("expected version output to contain v4.5.6, got: %s", output)
	}
}
