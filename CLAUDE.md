# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

GraphQL Linter extends graphql-schema-linter with Apollo Federation support. It validates GraphQL SDL files (`.graphql`, `.graphqls`) for syntax, best practices, and Apollo Federation directive usage. All rules from graphql-schema-linter are supported, plus additional federation-specific validation.

## Common Commands

### Running the Linter

```bash
# Run linter on a directory
go run cmd/graphql-linter/main.go -targetPath test/testdata/graphql/base/invalid

# Run linter on a specific file
go run cmd/graphql-linter/main.go -targetPath test/testdata/graphql/base/invalid/20-suspicious-enum-value.graphql

# Enable verbose output
go run cmd/graphql-linter/main.go -targetPath <path> -verbose

# Use custom config file
go run cmd/graphql-linter/main.go -configPath /path/to/.graphql-linter.yml -targetPath <path>

# Show version
go run cmd/graphql-linter/main.go -version
```

### Testing

```bash
# Run all tests
task remote:test

# Run unit tests only
task remote:test

# Run integration tests
task remote:test-integration

# Run component tests
task remote:test-component

# Run specific test
go test -v ./internal/app/graphql-linter/application -run TestExecute_Run

# Coverage
task remote:coverage          # Check coverage
task remote:coverage-visual   # Show visual coverage report
```

### Linting and Formatting

```bash
# Run golangci-lint
task remote:lint

# Auto-fix linting issues
task remote:fix-linting-issues

# Format code
task remote:format

# Run without cache
task remote:golangci-lint-run-without-cache
```

## Architecture

This project follows **Clean Architecture** with three distinct layers:

### 1. Presentation Layer
- **Location**: `cmd/graphql-linter/main.go`, `internal/app/graphql-linter/presentation/`
- **Responsibility**: CLI flag parsing, user input handling

### 2. Application Layer
- **Location**: `internal/app/graphql-linter/application/`
- **Responsibility**: Business logic orchestration, coordinates linting workflow
- **Key components**:
  - `Execute`: Main executor that orchestrates the linting process
  - `report/`: Handles error reporting and output formatting

### 3. Data Layer
- **Location**: `internal/app/graphql-linter/data/`
- **Responsibility**: File I/O, config loading, schema parsing, rule execution
- **Key components**:
  - `Store`: Central data store handling config, schema files, and validation
  - `base/rules/`: Core linting rules from graphql-schema-linter
  - `base/models/`: Data models for config, errors, suppressions
  - `federation/`: Apollo Federation-specific validation
  - `federation/rules/`: Federation directive validation

### Shared Packages
- **Location**: `internal/pkg/`
- **Contents**:
  - `logging/`: slog handler setup
  - `rules/`: Common rule helpers used across base and federation rules

### Flow
1. **Presentation** receives CLI input → creates **Application** executor
2. **Application** coordinates the linting:
   - Uses **Data** layer to load config (`.graphql-linter.yml`)
   - Discovers GraphQL files (`.graphql`, `.graphqls`)
   - For each file: reads, parses, filters comments, validates federation
   - Executes rules from both `base/rules` and `federation/rules`
   - Applies suppressions from config
   - Collects errors
3. **Application** uses `report` package to format and print results

### Key Dependencies
- `wundergraph/graphql-go-tools/v2`: GraphQL AST parsing and federation support
- `log/slog`: Structured logging (stdlib; handler installed in `presentation.Run()`)
- `gopkg.in/yaml.v3`: Config file parsing

### Testing Strategy
- **Unit tests** (`*_test.go`): Test individual functions and methods
- **Integration tests** (`*_integration_test.go`): Test interactions between layers
- **Component tests** (`test/component/`): End-to-end tests with real GraphQL files
- **Test fixtures** (`test/testdata/graphql/`): Hand-written `.graphql` files; edit them directly

## Configuration

Default config location: `.graphql-linter.yml`, then `.graphql-linter.yaml`, in the current directory.

Example config structure:
```yaml
suppressions:
  - file: "test/testdata/graphql/invalid/ignore_this.graphql"
    line: 12
    rule: "types-have-descriptions"
    value: "User"
    reason: "Documented elsewhere"
settings:
  validateFederation: true   # false skips federation build and directive checks
  checkDescriptions: true    # false skips the *-have-descriptions rules
```

See `.graphql-linter.yml.example` for full configuration options.

## Linting Rules

Rules are organized in two locations:

### Base Rules (`internal/app/graphql-linter/data/base/rules/`)
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

### Federation Rules (`internal/app/graphql-linter/data/federation/rules/`)
- Apollo Federation directive validation
- Composition validation

## Important Notes

- The linter automatically skips `node_modules`, `vendor` and hidden directories (starting with `.`) below the target path
- GraphQL files must have `.graphql` or `.graphqls` extensions
- Comments using `//` are filtered before parsing but preserved for line number reporting
- Suppressions can be applied per-file or per-line via config
- Code coverage target: 80.2%
- CI uses `mcvs-golang-action@v3.11.2` with multiple test types
