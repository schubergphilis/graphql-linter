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
curl -L https://github.com/schubergphilis/graphql-linter/releases/download/v0.1.0/graphql-linter-v0.1.0-linux-arm64 \
-o graphql-linter && \
chmod +x graphql-linter && \
graphql-linter --version
```

### Golang

```zsh
go install github.com/schubergphilis/graphql-linter/cmd/graphql-linter@v0.1.0 && \
graphql-linter --version
```

## Usage

```zsh
go run cmd/graphql-linter/main.go
```
