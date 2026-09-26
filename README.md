# GraphQL Linter

[![GitHub release](https://img.shields.io/github/v/release/schubergphilis/graphql-linter)](https://github.com/schubergphilis/graphql-linter/releases)
[![License](https://img.shields.io/github/license/schubergphilis/graphql-linter)](LICENSE)

<img src="./assets/logos/graphql-linter.png" width="250" alt="GraphQL Linter logo">

The [`graphql-schema-linter`](https://github.com/cjoudrey/graphql-schema-linter)
rules plus Apollo Federation validation, in a single Go binary. It lints
`.graphql` and `.graphqls` files for syntax, schema design best practices and
federation directive usage. No Node.js toolchain is needed, so it fits in CI
pipelines and pre-commit hooks.

## Install

Pre-built binary (`linux/amd64`, `linux/arm64`, `darwin/arm64`):

```zsh
ARCH=$(uname -m | sed -e 's/x86_64/amd64/' -e 's/aarch64/arm64/')
OS=$(uname | tr '[:upper:]' '[:lower:]')
VERSION=v0.2.4
curl --fail -L -o graphql-linter \
  "https://github.com/schubergphilis/graphql-linter/releases/download/${VERSION}/graphql-linter-${VERSION}-${OS}-${ARCH}"
chmod +x graphql-linter
./graphql-linter -version | grep "${VERSION}"
```

Or with Go:

```zsh
go install github.com/schubergphilis/graphql-linter/cmd/graphql-linter@v0.2.4
```

## Quick start

```zsh
graphql-linter -targetPath ./schema
```

```text
schema/user.graphql:7: types-have-descriptions: Object type 'User' is missing a description
  type User @key(fields: "id") {
schema/user.graphql:10: invalid-federation-directive: Invalid federation directive '@shareble' on field 'User.name'. Did you mean '@shareable'?
  name: String @shareble
level=ERROR msg="totalErrors: 2"
```

All files under `-targetPath` (default: the current directory) are linted as
one schema. The exit code is non-zero when there are findings.

## Configuration

Put a `.graphql-linter.yml` (or `.graphql-linter.yaml`) in the directory you
run the linter from:

```yaml
---
settings:
  validateFederation: true
  checkDescriptions: true
suppressions:
  - file: schema/user.graphql
    rule: types-have-descriptions
    value: User
    reason: Documented in the gateway instead.
```

All flags, settings and suppressions: [docs/configuration.md](docs/configuration.md).

## Use in CI

GitHub Actions, with the `graphql-lint` testing-type of mcvs-general-action
(proposed in [schubergphilis/mcvs-general-action#63](https://github.com/schubergphilis/mcvs-general-action/issues/63)):

```yaml
- uses: schubergphilis/mcvs-general-action@<sha> # vX.Y.Z
  with:
    testing-type: graphql-lint
```

Pre-commit, in `.pre-commit-config.yaml` (details in [docs/ci.md](docs/ci.md)):

```yaml
repos:
  - repo: https://github.com/schubergphilis/graphql-linter
    rev: v0.2.4
    hooks:
      - id: graphql-linter
```

## Documentation

- [Configuration](docs/configuration.md): flags, settings and suppressing findings
- [Rules](docs/rules.md): schema rules and Apollo Federation rules
- [CI](docs/ci.md): GitHub Actions and the pre-commit hook
- [Development](docs/development.md): building from source, tests, project layout and contributing

## License

Released under the [MIT License](LICENSE). Copyright (c) 2025 Schuberg Philis.
