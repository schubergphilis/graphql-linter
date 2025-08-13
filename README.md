# GraphQL Linter

As [this issue](https://github.com/cjoudrey/graphql-schema-linter/issues/210)
has been open since 2020, the decision was made to add federation support to a
new linter. In other words, the `graphql-schema-linter` spec is honoured and
federation support is added.

## Features

- All rules from `graphql-schema-linter` supported.
- Federation directives and composition are recognized and validated.
- Validates GraphQL SDL files for syntax and best practices.
- Checks for Apollo Federation directive usage and typos.
- Ensures all types, enums, and fields have descriptions.
- Detects undefined types and invalid enum values.
- Reports errors with line numbers and context.
- Supports error suppression via comments.

## Installation

### Binary

```zsh
ARCH=$(uname -m | awk '{if ($1=="x86_64") print "amd64"; else if ($1=="arm64" || $1=="aarch64") print "arm64"; else { print "Unsupported architecture: " $1 > "/dev/stderr"; exit 1 }}')
OS=$(uname | tr '[:upper:]' '[:lower:]')
VERSION=v0.1.0
curl --fail -L https://github.com/schubergphilis/graphql-linter/releases/download/${VERSION}/graphql-linter-${VERSION}-${OS}-${ARCH} \
-o graphql-linter && \
chmod +x graphql-linter && \
./graphql-linter --version | grep ${VERSION}
```

### Golang

```zsh
go install github.com/schubergphilis/graphql-linter/cmd/graphql-linter@v0.1.0 && \
graphql-linter --version
```

## Usage

### Command-line Parameters

- `-configPath string`
  The path to the configuration file (optional, defaults to `.graphql-linter.yaml` in the current directory).
- `-targetPath string`
  The directory with GraphQL files that should be checked.
- `-verbose`
  Enable verbose output.
- `-version`
  Show version information and exit.

### Testdata generation

```zsh
go run cmd/graphql-testdata-generator/main.go
```

### Help

```zsh
go run cmd/graphql-linter/main.go --help
```

### Lint a specific directory

```zsh
go run cmd/graphql-linter/main.go \
  -targetPath test/testdata/graphql/base/invalid
```

### Lint a specific file

```zsh
go run cmd/graphql-linter/main.go \
  -targetPath test/testdata/graphql/base/invalid/20-suspicious-enum-value.graphql
```

### Example configuration file

By default, the linter looks for a `.graphql-linter.yaml` file in the current directory.

For a full example with all options and comments, see the provided `.graphql-linter.yml.example` file in the repository root.

Example minimal configuration:

```yaml
rules:
  require-descriptions:
    types: true
    fields: true
    enums: true
suppressions:
  - path: "test/testdata/graphql/invalid/ignore_this.graphql"
    rules: ["require-descriptions"]
```
