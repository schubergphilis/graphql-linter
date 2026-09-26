package presentation

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/schubergphilis/graphql-linter/internal/app/graphql-linter/application"
	"github.com/schubergphilis/graphql-linter/internal/pkg/logging"
)

type CLI struct {
	configPathFlag string
	targetPathFlag string
	version        string
	versionFlag    bool
	verboseFlag    bool
}

func NewCLI(args []string, version string) CLI {
	cli := CLI{
		version: version,
	}

	flags := flag.NewFlagSet("graphql-linter", flag.ExitOnError)
	flags.StringVar(
		&cli.configPathFlag,
		"configPath",
		"",
		"The path to the configuration file (optional, defaults to .graphql-linter.yml or "+
			".graphql-linter.yaml in the current directory)",
	)
	flags.StringVar(
		&cli.targetPathFlag,
		"targetPath",
		"",
		"The directory with GraphQL files that should be checked",
	)
	flags.BoolVar(&cli.versionFlag, "version", false, "Show version")
	flags.BoolVar(&cli.verboseFlag, "verbose", false, "Enable verbose output")
	_ = flags.Parse(args) // ExitOnError: Parse exits instead of returning an error

	return cli
}

func (c CLI) Run() error {
	logging.Setup(c.verboseFlag)

	applicationExecute := application.NewExecute(
		c.configPathFlag,
		c.targetPathFlag,
		c.version,
	)

	if c.versionFlag {
		_, err := fmt.Fprintln(os.Stdout, applicationExecute.Version())
		if err != nil {
			return fmt.Errorf("unable to print version: %w", err)
		}

		return nil
	}

	slog.Debug("Verbose output enabled")

	err := applicationExecute.Run()
	if err != nil {
		return fmt.Errorf("unable to run execute: %w", err)
	}

	return nil
}
