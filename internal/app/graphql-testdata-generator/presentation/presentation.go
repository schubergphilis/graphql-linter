package presentation

import (
	"fmt"

	"github.com/schubergphilis/graphql-linter/internal/app/graphql-testdata-generator/application"
)

type Presenter interface {
	Run() error
}

type CLI struct{}

func NewCLI() (CLI, error) {
	cli := CLI{}

	return cli, nil
}

func (c CLI) Run() error {
	applicationExecute, err := application.NewExecute()
	if err != nil {
		return fmt.Errorf("unable to create application execute: %w", err)
	}

	if err := applicationExecute.Run(); err != nil {
		return fmt.Errorf("unable to run execute: %w", err)
	}

	return nil
}
