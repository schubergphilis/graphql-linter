package presentation

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/application"
	"github.com/schubergphilis/graphql-linter/internal/pkg/logging"
)

type Presenter interface {
	Run() error
}

type Flagger interface {
	BoolVar(p *bool, name string, value bool, usage string)
	StringVar(p *string, name string, value string, usage string)
	Parse()
}

type Flag struct{}

type CLI struct {
	configPathFlag string
	targetPathFlag string
	version        string
	versionFlag    bool
	verboseFlag    bool
}

func NewCLI(flagger Flagger, version string) CLI {
	cli := CLI{
		version: version,
	}
	flagger.StringVar(
		&cli.configPathFlag,
		"configPath",
		"",
		"The path to the configuration file (optional, defaults to .graphql-linter.yaml in the current directory)",
	)
	flagger.StringVar(
		&cli.targetPathFlag,
		"targetPath",
		"",
		"The directory with GraphQL files that should be checked",
	)
	flagger.BoolVar(&cli.versionFlag, "version", false, "Show version")
	flagger.BoolVar(&cli.verboseFlag, "verbose", false, "Enable verbose output")
	flagger.Parse()

	return cli
}

func NewFlag() Flag {
	return Flag{}
}

func (c CLI) Run() error {
	logging.Setup(c.verboseFlag)

	applicationExecute, err := application.NewExecute(
		application.NewDebug(),
		c.configPathFlag,
		c.targetPathFlag,
		c.version,
		c.verboseFlag,
	)
	if err != nil {
		return fmt.Errorf("unable to load new execute: %w", err)
	}

	if c.versionFlag {
		_, err = fmt.Fprintln(os.Stdout, applicationExecute.Version())
		if err != nil {
			return fmt.Errorf("unable to print version: %w", err)
		}

		return nil
	}

	slog.Debug("Verbose output enabled")

	err = applicationExecute.Run()
	if err != nil {
		return fmt.Errorf("unable to run execute: %w", err)
	}

	return nil
}

func (f Flag) BoolVar(p *bool, name string, value bool, usage string) {
	flag.BoolVar(p, name, value, usage)
}

func (f Flag) StringVar(p *string, name string, value string, usage string) {
	flag.StringVar(p, name, value, usage)
}

func (f Flag) Parse() {
	flag.Parse()
}
