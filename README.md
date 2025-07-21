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
VERSION=v0.1.0-rc.11
curl --fail -L https://github.com/schubergphilis/graphql-linter/releases/download/${VERSION}/graphql-linter-${VERSION}-${OS}-${ARCH} \
-o graphql-linter && \
chmod +x graphql-linter && \
./graphql-linter --version | grep ${VERSION}
```

### Golang

```zsh
go install github.com/schubergphilis/graphql-linter/cmd/graphql-linter@v0.1.0-rc.11 && \
graphql-linter --version
```

## Usage

### Testdata generation

```zsh
go run cmd/graphql-testdata-generator/main.go
```

### Help

```zsh
go run cmd/graphql-linter/main.go --help
```

### Lint specific directory

```zsh
go run cmd/graphql-linter/main.go -targetPath test/testdata/graphql/invalid
```

```zsh
graphql-schema-linter test/testdata/graphql/invalid/*
```

```zsh
graphql-schema-linter test/testdata/graphql/valid/*
```

- 10-input-object-fields-sorted-alphabetically.graphql
- 12-input-object-values-have-descriptions.graphql
- 14-relay-connection-types-spec.graphql
- 15-relay-connection-arguments-spec.graphql
- 17-types-are-capitalized.graphql
- 18-types-have-descriptions.graphql
- 19-invalid-graphql-schema.graphql
