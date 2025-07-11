package application

import (
	"fmt"
	"runtime/debug"

	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/data"
)

type Executor interface {
	Run() error
	Version()
}

type Execute struct {
	Verbose bool
}

func NewExecute(verbose bool) (Execute, error) {
	e := Execute{
		Verbose: verbose,
	}

	return e, nil
}

func (e Execute) Run() error {
	s, err := data.NewStore(e.Verbose)
	if err != nil {
		return fmt.Errorf("unable to load new store: %w", err)
	}

	if err := s.Run(); err != nil {
		return fmt.Errorf("unable to run store: %w", err)
	}

	return nil
}

func (e Execute) Version() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		return info.Main.Version
	}

	return "(unknown)"
}
