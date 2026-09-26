# Development

This project follows a Clean Architecture layout (presentation → application →
data) and uses [Task](https://taskfile.dev) for common workflows. The tasks are
pulled in from
[mcvs-golang-action](https://github.com/schubergphilis/mcvs-golang-action) via
the `remote:` include in the [Taskfile](../Taskfile.yml).

## Build from source

```zsh
git clone https://github.com/schubergphilis/graphql-linter.git
cd graphql-linter
go build -o graphql-linter ./cmd/graphql-linter
```

## Tasks

```zsh
# Run the unit tests
task remote:test

# Run integration and component tests
task remote:test-integration
task remote:test-component

# Lint and format
task remote:lint
task remote:format
task remote:fix-linting-issues

# Check code coverage
task remote:coverage
```

Run the linter from a checkout without building it:

```zsh
go run ./cmd/graphql-linter -targetPath test/testdata/graphql/base/invalid
```

## Test fixtures

The fixtures in `test/testdata/graphql/` are hand-written `.graphql` files.
Edit or add them directly:

| Directory              | Contents                                         |
| ---------------------- | ------------------------------------------------ |
| `valid/`               | Schemas that must lint without findings.         |
| `base/invalid/`        | One schema per schema rule violation.            |
| `federation/valid/`    | Valid Apollo Federation subgraph schemas.        |
| `federation/invalid/`  | Invalid Apollo Federation directives or schemas. |

## Project layout

```text
cmd/
  graphql-linter/             CLI entry point
internal/
  app/graphql-linter/
    presentation/             CLI parsing and I/O
    application/              Linting orchestration
      report/                 Error reporting and output
    data/                     Config, schema parsing, rule execution
      base/models/            Config, error and suppression models
      base/rules/             Schema rules
      federation/             Apollo Federation schema validation
      federation/rules/       Apollo Federation rules
  pkg/
    logging/                  slog handler setup
    rules/                    Helpers shared by base and federation rules
test/
  component/                  End-to-end tests
  testdata/graphql/           GraphQL fixtures
```

See [CLAUDE.md](../CLAUDE.md) for a more detailed architecture overview.

## Contributing

Contributions are welcome! To propose a change:

1. Fork the repository and create a feature branch.
2. Add or update tests for your change.
3. Ensure `task remote:test` and `task remote:lint` pass.
4. Open a pull request describing the motivation and behaviour change.

Please keep pull requests focused and include test coverage for new rules or
fixes.
