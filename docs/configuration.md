# Configuration

## Usage

```text
graphql-linter [flags]
```

The linter walks the target path recursively and collects every `.graphql` and
`.graphqls` file. Below the target path it skips `node_modules`, `vendor` and
hidden directories (any name starting with `.`, such as `.git`). All files under
the target are linted together as one schema, so a schema split over several
files is checked as a whole. The linter exits non-zero when unsuppressed
findings are detected.

### Flags

| Flag          | Description                                                                                                           |
| ------------- | --------------------------------------------------------------------------------------------------------------------- |
| `-targetPath` | Directory or file containing the GraphQL schemas to check. Defaults to the current directory.                         |
| `-configPath` | Path to the configuration file. Defaults to `.graphql-linter.yml` or `.graphql-linter.yaml` in the current directory. |
| `-verbose`    | Enable verbose output.                                                                                                |
| `-version`    | Print version information and exit.                                                                                   |

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

## Configuration file

When `-configPath` is not set, the linter looks for `.graphql-linter.yml`, then
`.graphql-linter.yaml`, in the current directory. Use `-configPath` to point at
a different file. If no configuration is found, the built-in defaults below are
used.

```yaml
---
# Global behaviour
settings:
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
[.graphql-linter.yml.example](../.graphql-linter.yml.example).

### Settings

| Setting              | Default | Description                                                            |
| -------------------- | ------- | ---------------------------------------------------------------------- |
| `validateFederation` | `true`  | Build the federation schema and validate Apollo Federation directives. |
| `checkDescriptions`  | `true`  | Run the `*-have-descriptions` rules. Set to `false` to skip them.      |

## Suppressing findings

Individual findings can be suppressed in the configuration file. Every field is
optional and acts as a filter: an omitted field matches anything, so narrow the
suppression by combining fields. Always include a `reason` for auditability,
even though it is not enforced. See [rules.md](rules.md) for the rule ids.

```yaml
suppressions:
  - file: test/testdata/graphql/base/invalid/07-enum-values-sorted-alphabetically.graphql
    line: 12
    rule: defined-types-are-used
    value: PageInfo
    reason: PageInfo is intentionally unused in this test schema.
```

| Field    | Matching behaviour                                                                                                                       |
| -------- | ---------------------------------------------------------------------------------------------------------------------------------------- |
| `file`   | Matches when the schema path ends with this value; omit to match any file.                                                               |
| `line`   | Matches this line number; omit (or `0`) to match any line.                                                                               |
| `rule`   | Matches this rule identifier (see [rules.md](rules.md)); omit to match any rule.                                                         |
| `value`  | Matches the name of the offending type, field, argument or enum value (e.g. `User`, `firstName`, `PO4_VOLUME`); omit to match any value. |
| `reason` | Free-form justification for the suppression (recommended, not enforced).                                                                 |
