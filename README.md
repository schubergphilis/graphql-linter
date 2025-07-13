# GraphQL Linter

An opinionated linter for GraphQL SDL (Schema Definition Language) files,
focused on federation and schema quality.

## Features

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
VERSION=v0.1.0-rc.6
curl --fail -L https://github.com/schubergphilis/graphql-linter/releases/download/${VERSION}/graphql-linter-${VERSION}-${OS}-${ARCH} \
-o graphql-linter && \
chmod +x graphql-linter && \
./graphql-linter --version | grep ${VERSION}
```

### Golang

```zsh
go install github.com/schubergphilis/graphql-linter/cmd/graphql-linter@v0.1.0-rc.6 && \
graphql-linter --version
```

## Usage

```zsh
go run cmd/graphql-linter/main.go
```
