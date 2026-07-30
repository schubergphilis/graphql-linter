# GraphQL Linter

A fast, opinionated linter for GraphQL SDL (Schema Definition Language) with
first-class **Apollo Federation** support.

`graphql-linter` validates your `.graphql` and `.graphqls` files for syntax
errors, schema design best practices, and correct usage of Apollo Federation
directives — all from a single, dependency-free binary.

## Table of contents

- [Why GraphQL Linter?](#why-graphql-linter)
- [Features](#features)
- [Installation](#installation)
- [Quick start](#quick-start)
- [Usage](#usage)
- [Configuration](#configuration)
- [Rules](#rules)
- [Suppressing findings](#suppressing-findings)
- [Pre-commit hook](#pre-commit-hook)
- [Development](#development)
- [Contributing](#contributing)
- [License](#license)

## Why GraphQL Linter?

The widely used [`graphql-schema-linter`](https://github.com/cjoudrey/graphql-schema-linter)
does not support Apollo Federation, and the
[request to add it](https://github.com/cjoudrey/graphql-schema-linter/issues/210)
has been open since 2020.

`graphql-linter` fills that gap. It honours the `graphql-schema-linter` rule set
that teams already rely on and adds validation for Apollo Federation directives
and composition on top. Because it ships as a single static Go binary, there is
no Node.js toolchain to install and it drops cleanly into CI pipelines and
pre-commit hooks.

## Features

- **Drop-in rule parity** — implements the rules from `graphql-schema-linter`.
- **Apollo Federation aware** — recognizes and validates federation directives
  (`@key`, `@external`, `@requires`, `@provides`, `@shareable`, `@override`,
  `@inaccessible`, `@tag`, and more) and flags invalid directives or typos.
- **Schema hygiene checks** — enforces descriptions, naming conventions,
  alphabetical sorting, deprecation reasons, and Relay connection specs.
- **Clear diagnostics** — reports the rule, file, line number, and context for
  every finding.
- **Flexible suppressions** — silence specific findings per file, line, and rule
  through a config file.
- **Single binary** — no runtime dependencies; runs anywhere Go binaries run.

## Installation

### Pre-built binary

```zsh
ARCH=$(uname -m | awk '{if ($1=="x86_64") print "amd64"; else if ($1=="arm64" || $1=="aarch64") print "arm64"; else { print "Unsupported architecture: " $1 > "/dev/stderr"; exit 1 }}')
OS=$(uname | tr '[:upper:]' '[:lower:]')
VERSION=v0.1.0
curl --fail -L "https://github.com/schubergphilis/graphql-linter/releases/download/${VERSION}/graphql-linter-${VERSION}-${OS}-${ARCH}" \
  -o graphql-linter && \
  chmod +x graphql-linter && \
  ./graphql-linter --version | grep "${VERSION}"
```

Pre-built binaries are published for `linux/amd64`, `linux/arm64`, and
`darwin/arm64`. See the [releases page](https://github.com/schubergphilis/graphql-linter/releases)
for all available builds.

### Go

```zsh
go install github.com/schubergphilis/graphql-linter/cmd/graphql-linter@v0.1.0 && \
  graphql-linter --version
```

### From source

```zsh
git clone https://github.com/schubergphilis/graphql-linter.git
cd graphql-linter
go build -o graphql-linter ./cmd/graphql-linter
```

## Quick start

Lint every GraphQL file in a directory:

```zsh
graphql-linter -targetPath ./schema
```

Lint a single file:

```zsh
graphql-linter -targetPath ./schema/user.graphqls
```

The linter walks the target path recursively, skipping `node_modules`,
`vendor`, `.git`, and any dot-directory. It exits non-zero when unsuppressed
findings are detected, making it CI-ready out of the box.

## Usage

```text
graphql-linter [flags]
```

### Flags

| Flag          | Description                                                                                     |
| ------------- | ----------------------------------------------------------------------------------------------- |
| `-targetPath` | Directory or file containing the GraphQL schemas to check. Defaults to the project root.         |
| `-configPath` | Path to the configuration file. Defaults to `.graphql-linter.yml` in the project root.           |
| `-verbose`    | Enable verbose output.                                                                           |
| `-version`    | Print version information and exit.                                                              |

### Examples

```zsh
# Lint with a custom configuration file
graphql-linter -configPath ./config/.graphql-linter.yml -targetPath ./schema

# Verbose run
graphql-linter -targetPath ./schema -verbose

# Show help
graphql-linter --help
```

When running from a checkout of this repository you can invoke the linter
directly with `go run`:

```zsh
go run ./cmd/graphql-linter -targetPath test/testdata/graphql/base/invalid
```

## Configuration

When `-configPath` is not set, the linter looks for a `.graphql-linter.yml` file
in the project root. Use `-configPath` to point at a different file. If no
configuration is found, the built-in defaults below are used.

```yaml
---
# Global behaviour
settings:
  # Treat warnings as errors.
  strictMode: true
  # Validate Apollo Federation directives.
  validateFederation: true
  # Require descriptions on schema elements.
  checkDescriptions: true

# Findings to silence (see "Suppressing findings" below)
suppressions:
  - file: schema/user.graphqls
    line: 42
    rule: types-have-descriptions
    value: User
    reason: Documented in the federation gateway instead.
```

A fully commented reference configuration is available in
[.graphql-linter.yml.example](.graphql-linter.yml.example).

### Settings

| Setting              | Default | Description                                        |
| -------------------- | ------- | -------------------------------------------------- |
| `strictMode`         | `true`  | Treat warnings as errors.                          |
| `validateFederation` | `true`  | Validate Apollo Federation directives.             |
| `checkDescriptions`  | `true`  | Require descriptions on types, fields, and enums.  |

## Rules

### Schema rules

These mirror the `graphql-schema-linter` rule set:

- `arguments-have-descriptions`
- `defined-types-are-used`
- `deprecations-have-a-reason`
- `descriptions-are-capitalized`
- `enum-values-all-caps`
- `enum-values-have-descriptions`
- `enum-values-sorted-alphabetically`
- `fields-are-camel-cased`
- `fields-have-descriptions`
- `input-object-fields-sorted-alphabetically`
- `input-object-values-are-camel-cased`
- `input-object-values-have-descriptions`
- `interface-fields-sorted-alphabetically`
- `relay-connection-types-spec`
- `relay-connection-arguments-spec`
- `relay-page-info-spec`
- `type-fields-sorted-alphabetically`
- `types-are-capitalized`
- `types-have-descriptions`

### Federation rules

When `validateFederation` is enabled, the linter also verifies Apollo Federation
usage, including:

- Only valid federation directives are used on types and fields
  (`@key`, `@external`, `@requires`, `@provides`, `@extends`, `@shareable`,
  `@inaccessible`, `@override`, `@composeDirective`, `@interfaceObject`, `@tag`,
  `@deprecated`, `@specifiedBy`, `@oneOf`).
- Directive typos are detected and closest-match suggestions are offered.
- Composition-level validation of federated types.

## Suppressing findings

Individual findings can be suppressed in the configuration file. Every field is
optional and acts as a filter: an omitted field matches anything, so narrow the
suppression by combining fields. Always include a `reason` for auditability,
even though it is not enforced.

```yaml
suppressions:
  - file: test/testdata/graphql/base/invalid/07-enum-values-sorted-alphabetically.graphql
    line: 12
    rule: defined-types-are-used
    value: PageInfo
    reason: PageInfo is intentionally unused in this test schema.
```

| Field    | Matching behaviour                                                              |
| -------- | ------------------------------------------------------------------------------- |
| `file`   | Matches when the schema path ends with this value; omit to match any file.      |
| `line`   | Matches this line number; omit (or `0`) to match any line.                      |
| `rule`   | Matches this rule identifier; omit to match any rule.                           |
| `value`  | Matches a specific symbol (type, field, enum value); omit to match any value.   |
| `reason` | Free-form justification for the suppression (recommended, not enforced).        |

## Pre-commit hook

`graphql-linter` ships a [pre-commit](https://pre-commit.com) hook so schemas
are linted automatically before every commit.

Add the following to the `.pre-commit-config.yaml` in your repository:

```yaml
repos:
  - repo: https://github.com/schubergphilis/graphql-linter
    # Replace with the latest released tag; run `pre-commit autoupdate` to bump.
    rev: v0.1.4
    hooks:
      - id: graphql-linter
```

Then install and run it:

```zsh
pre-commit install
pre-commit run graphql-linter --all-files
```

The hook is triggered whenever a `.graphql` or `.graphqls` file is staged. It
lints the whole project (so cross-file Apollo Federation composition is
validated) and fails the commit when linting errors are found.

Configuration and suppressions are picked up from the `.graphql-linter.yml`
file in the repository root, as described above.

## Development

This project follows a Clean Architecture layout (presentation → application →
data) and uses [Task](https://taskfile.dev) for common workflows.

```zsh
# Run the full test suite
task remote:test

# Run integration and component tests
task remote:test-integration
task remote:test-component

# Lint and format
task remote:lint
task remote:format
task remote:fix-linting-issues

# Regenerate mocks (mockery)
task remote:mock-generate

# Regenerate test data fixtures
go run ./cmd/graphql-testdata-generator
```

### Project layout

```text
cmd/
  graphql-linter/             CLI entry point
  graphql-testdata-generator/ Test fixture generator
internal/
  app/graphql-linter/
    presentation/             CLI parsing and I/O
    application/              Linting orchestration and reporting
    data/                     Config, schema parsing, rule execution
      base/rules/             Schema rules
      federation/rules/       Apollo Federation rules
  pkg/                        Shared helpers and constants
test/                         Component tests and GraphQL fixtures
```

## Contributing

Contributions are welcome! To propose a change:

1. Fork the repository and create a feature branch.
2. Add or update tests for your change.
3. Ensure `task remote:test` and `task remote:lint` pass.
4. Open a pull request describing the motivation and behaviour change.

Please keep pull requests focused and include test coverage for new rules or
fixes.

## License

Released under the [MIT License](LICENSE). Copyright (c) 2025 Schuberg Philis.
