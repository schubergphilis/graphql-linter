# Rules

## Overview

The widely used [`graphql-schema-linter`](https://github.com/cjoudrey/graphql-schema-linter)
does not support Apollo Federation, and the
[request to add it](https://github.com/cjoudrey/graphql-schema-linter/issues/210)
has been open since 2020.

`graphql-linter` fills that gap. It honours the `graphql-schema-linter` rule set
that teams already rely on and adds Apollo Federation directive validation and
subgraph validation of the merged target files on top. Because it ships as a
single static Go binary, there is no Node.js toolchain to install and it drops
cleanly into CI pipelines and pre-commit hooks.

- **Drop-in rule parity**: implements the rules from `graphql-schema-linter`.
- **Apollo Federation aware**: recognizes and validates federation directives
  (`@key`, `@external`, `@requires`, `@provides`, `@shareable`, `@override`,
  `@inaccessible`, `@tag`, and more) and flags invalid directives or typos.
- **Schema hygiene checks**: enforces descriptions, naming conventions,
  alphabetical sorting, deprecation reasons, and Relay connection specs.
- **Clear diagnostics**: reports the rule, file, line number, and context for
  every finding.
- **Flexible suppressions**: silence specific findings per file, line, rule and
  value through the config file, see
  [Suppressing findings](configuration.md#suppressing-findings).
- **Single binary**: no runtime dependencies; runs anywhere Go binaries run.

## Schema rules

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

Additional rules:

- `invalid-graphql-schema`: a `Query` root type must be provided.
- `suspicious-enum-value`: enum values with digits, e.g. `STRING2`.
- `invalid-graphql-syntax`: the file does not parse; the other files are still
  linted.
- `failed-to-read-schema-file`: the file could not be read.

Description, capitalization and deprecation rules cover every definition kind:
object, interface, input object, enum, union and scalar types, and their
fields, arguments, input values and enum values.

`defined-types-are-used`, `invalid-graphql-schema` and `relay-page-info-spec`
are schema wide: they run once on all target files together, so a schema that
is split over several files is checked as a whole. Line numbers come from the
parsed schema, so `line` suppressions stay stable.

Setting `checkDescriptions: false` skips the `*-have-descriptions` rules, see
[Settings](configuration.md#settings).

## Federation rules

When `validateFederation` is enabled (the default), all target files are
checked together as one subgraph. Set `validateFederation: false` to skip these
rules, see [Settings](configuration.md#settings).

- `invalid-federation-directive`: a directive that is not allowed, with a
  suggestion for typos, e.g. `Did you mean '@key'?`.
- `invalid-federation-schema`: the merged subgraph schema is invalid.

`invalid-federation-directive` checks every directive on types, fields,
arguments, input values, enum values, unions and scalars. A directive must be a
Federation v2.x directive (`@key`, `@external`, `@requires`, `@provides`,
`@extends`, `@shareable`, `@inaccessible`, `@override`, `@composeDirective`,
`@interfaceObject`, `@tag`, `@link`, `@authenticated`, `@requiresScopes`,
`@policy`, `@context`, `@fromContext`, `@cost`, `@listSize`), a built-in
directive (`@deprecated`, `@specifiedBy`, `@oneOf`), defined in the schema, or
named in `@composeDirective`. Namespaced imports such as `@federation__key` are
accepted.

`invalid-federation-schema` requires unique type, field and enum value names,
known types, non-empty types and correct interface implementations. Extending
an entity owned by another subgraph and repeating `@key` or `@tag` are allowed.

This is subgraph validation, not supergraph composition: lint each subgraph
with its own `-targetPath`.
