# CI

`graphql-linter` exits non-zero when it finds unsuppressed issues, so any CI
system can use it as a gate. Configuration and suppressions come from
`.graphql-linter.yml` or `.graphql-linter.yaml`, see
[configuration.md](configuration.md).

## GitHub Actions

Use the `graphql-lint` testing-type of
[schubergphilis/mcvs-general-action](https://github.com/schubergphilis/mcvs-general-action).

> [!NOTE]
> This testing-type is proposed in
> [schubergphilis/mcvs-general-action#63](https://github.com/schubergphilis/mcvs-general-action/issues/63)
> and is not released yet.

```yaml
---
name: graphql-lint
"on": pull_request
permissions:
  contents: read
jobs:
  graphql-lint:
    runs-on: ubuntu-24.04
    steps:
      - uses: actions/checkout@<sha> # vX.Y.Z
      - uses: schubergphilis/mcvs-general-action@<sha> # vX.Y.Z
        with:
          testing-type: graphql-lint
```

Optional inputs (proposed):

| Input                        | Default                                                      |
| ---------------------------- | ------------------------------------------------------------ |
| `graphql-linter-target-path` | `.`                                                          |
| `graphql-linter-config-path` | Auto-discover `.graphql-linter.yml` / `.graphql-linter.yaml` |

## Other CI systems

Download the binary as shown in the [README](../README.md#install) and run it
from the repository root:

```zsh
./graphql-linter -targetPath .
```

The job fails when the linter reports issues, because the exit code is
non-zero.

## Pre-commit hook

`graphql-linter` ships a [pre-commit](https://pre-commit.com) hook so schemas
are linted automatically before every commit.

Add the following to the `.pre-commit-config.yaml` in your repository:

```yaml
repos:
  - repo: https://github.com/schubergphilis/graphql-linter
    # Replace with the latest released tag; run `pre-commit autoupdate` to bump.
    rev: v0.2.4
    hooks:
      - id: graphql-linter
```

Then install and run it:

```zsh
pre-commit install
pre-commit run graphql-linter --all-files
```

The hook uses `language: golang`, so pre-commit builds the linter itself. It
works in repositories that do not contain Go code.

The hook is triggered whenever a `.graphql` or `.graphqls` file is staged. It
does not lint only the staged files: it lints the whole project as one schema,
so cross-file checks such as `defined-types-are-used` and the federation
subgraph validation see all files. The commit fails when issues are found.

Configuration and suppressions are read from `.graphql-linter.yml` or
`.graphql-linter.yaml` in the directory the hook runs from, which is the
repository root. See [configuration.md](configuration.md).
