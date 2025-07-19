package application

import (
	"fmt"
	"path/filepath"

	"github.com/schubergphilis/graphql-linter/internal/app/graphql-testdata-generator/application/base"
	"github.com/schubergphilis/graphql-linter/internal/app/graphql-testdata-generator/application/federation"
	"github.com/schubergphilis/mcvs-golang-project-root/pkg/projectroot"
)

const (
	dirPerm     = 0o700
	filePerm    = 0o600
	unknownType = "Unknown"
)

type Executor interface {
	Run() error
}

type Execute struct {
	testdataBaseDir    string
	testdataInvalidDir string
}

func NewExecute() (Execute, error) {
	projectRoot, err := projectroot.FindProjectRoot()
	if err != nil {
		return Execute{}, fmt.Errorf("failed to determine project root: %w", err)
	}
	testdataBaseDir := filepath.Join(projectRoot, "test", "testdata", "graphql", "base")
	testdataInvalidDir := filepath.Join(testdataBaseDir, "invalid")
	return Execute{
		testdataBaseDir:    testdataBaseDir,
		testdataInvalidDir: testdataInvalidDir,
	}, nil
}

func (e Execute) Run() error {
	baseExec := base.NewExecute(e.testdataBaseDir, e.testdataInvalidDir)
	if err := baseExec.Run(); err != nil {
		return fmt.Errorf("failed to run base executor: %w", err)
	}

	federationExec := federation.NewExecute(e.testdataBaseDir, e.testdataInvalidDir)
	if err := federationExec.Run(); err != nil {
		return fmt.Errorf("failed to run federation executor: %w", err)
	}

	return nil
}
