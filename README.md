# SchemaKit

[![Go CI][go-ci-svg]][go-ci-url]
[![Go Lint][go-lint-svg]][go-lint-url]
[![Go SAST][go-sast-svg]][go-sast-url]
[![Docs][docs-godoc-svg]][docs-godoc-url]
[![Visualization][viz-svg]][viz-url]
[![License][license-svg]][license-url]

 [go-ci-svg]: https://github.com/grokify/schemakit/actions/workflows/go-ci.yaml/badge.svg?branch=main
 [go-ci-url]: https://github.com/grokify/schemakit/actions/workflows/go-ci.yaml
 [go-lint-svg]: https://github.com/grokify/schemakit/actions/workflows/go-lint.yaml/badge.svg?branch=main
 [go-lint-url]: https://github.com/grokify/schemakit/actions/workflows/go-lint.yaml
 [go-sast-svg]: https://github.com/grokify/schemakit/actions/workflows/go-sast-codeql.yaml/badge.svg?branch=main
 [go-sast-url]: https://github.com/grokify/schemakit/actions/workflows/go-sast-codeql.yaml
 [docs-godoc-svg]: https://pkg.go.dev/badge/github.com/grokify/schemakit
 [docs-godoc-url]: https://pkg.go.dev/github.com/grokify/schemakit
 [viz-svg]: https://img.shields.io/badge/visualizaton-Go-blue.svg
 [viz-url]: https://mango-dune-07a8b7110.1.azurestaticapps.net/?repo=grokify%2Fschemakit
 [loc-svg]: https://tokei.rs/b1/github/grokify/schemakit
 [repo-url]: https://github.com/grokify/schemakit
 [license-svg]: https://img.shields.io/badge/license-MIT-blue.svg
 [license-url]: https://github.com/grokify/schemakit/blob/master/LICENSE

JSON Schema toolkit for Go developers.

## Overview

**schemakit** is a toolkit for working with JSON Schema in Go projects. It provides:

- **Linting** - Validate schemas for compatibility with statically-typed languages
- **Generation** - Generate JSON Schema from Go struct types
- **Documentation** - Generate Markdown specification docs from Go types

## Installation

### Homebrew

```bash
brew install grokify/tap/schemakit
```

### Go Install

```bash
go install github.com/grokify/schemakit/cmd/schemakit@latest
```

## Commands

| Command | Description |
|---------|-------------|
| `schemakit lint` | Check schemas for static type compatibility |
| `schemakit generate` | Generate JSON Schema from Go struct types |
| `schemakit doc` | Generate Markdown documentation from Go types |
| `schemakit version` | Print version information |

## Usage

### Generate Schema from Go Types

Generate a JSON Schema from Go struct types:

```bash
schemakit generate github.com/myorg/myproject/types Config
schemakit generate -o schema.json github.com/myorg/myproject/types Config

# CI drift guard: fail if the committed schema is out of sync with the Go structs
schemakit generate -o schema.json --check github.com/myorg/myproject/types Config
```

This creates a temporary Go program that uses [invopop/jsonschema](https://github.com/invopop/jsonschema) to reflect on your type and generate the schema. The target package can be local (in GOPATH/src) or remote.

Use `--check` (requires `-o`) to verify that a committed schema still matches its Go structs without rewriting it — it exits non-zero on drift. Pair it with a `//go:generate schemakit generate ...` directive so `go generate ./...` refreshes the schema and CI catches drift.

### Generate Documentation from Go Types

Generate Markdown specification documentation from Go struct types:

```bash
# Generate to stdout
schemakit doc github.com/grokify/threat-model-spec/ir ThreatModel

# Generate with title and version, save to file
schemakit doc -t "Threat Model Specification" -v v0.4.0 \
  github.com/grokify/threat-model-spec/ir ThreatModel -o spec.md

# Prepend custom header (overview, examples, references)
schemakit doc --prepend header.md \
  github.com/grokify/threat-model-spec/ir ThreatModel -o spec.md
```

The `doc` command extracts:

- Package doc comments as overview
- Type descriptions from doc comments
- Field names and JSON tags
- Required vs optional fields (based on `omitempty`)
- Enum types (`type X string`) and their const values
- Enum value descriptions from const doc comments
- Generates Type Reference and Enum Reference sections

### Lint Schema

Check a JSON Schema for patterns that cause problems in code generation:

```bash
schemakit lint schema.json
```

### Profiles

Use `--profile` to select a linting profile:

```bash
schemakit lint schema.json                  # default profile
schemakit lint schema.json --profile scale  # strict scale profile
```

| Profile | Description |
|---------|-------------|
| `default` | Standard checks for discriminators, union size, nesting |
| `scale` | Strict mode that disallows composition keywords for clean static types |

### Output Formats

```bash
schemakit lint --output text schema.json   # Human-readable (default)
schemakit lint --output json schema.json   # Machine-readable JSON
schemakit lint --output github schema.json # GitHub Actions annotations
```

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | No issues found |
| 1 | Errors found (schema has problems) |
| 2 | Warnings found but no errors |

## Lint Checks

### Default Profile

#### Errors

| Code | Description |
|------|-------------|
| `union-no-discriminator` | Union (`anyOf`/`oneOf`) has no discriminator field |
| `inconsistent-discriminator` | Variants use different discriminator field names |
| `missing-const` | Union variant lacks `const` value for discriminator |
| `duplicate-const-value` | Multiple variants have the same discriminator value |
| `invalid-property-case` | Property name does not follow the configured case convention |

#### Warnings

| Code | Description |
|------|-------------|
| `large-union` | Union has more than 10 variants |
| `nested-union` | Union nested more than 2 levels deep |
| `additional-properties` | Union variant has `additionalProperties: true` |

### Scale Profile

The scale profile includes all default checks plus these additional errors:

| Code | Description |
|------|-------------|
| `composition-disallowed` | Disallow `anyOf`, `oneOf`, `allOf` |
| `additional-properties-disallowed` | Disallow `additionalProperties: true` |
| `missing-type` | Require explicit `type` field |
| `mixed-type-disallowed` | Disallow type arrays like `["string", "number"]` |

## Example

Given this schema with a union that lacks a discriminator:

```json
{
  "$defs": {
    "Response": {
      "anyOf": [
        {"type": "object", "properties": {"data": {"type": "string"}}},
        {"type": "object", "properties": {"error": {"type": "string"}}}
      ]
    }
  }
}
```

Running `schemakit lint` will report:

```
[error] $/$defs/Response/anyOf: anyOf union has no discriminator field
  suggestion: Add a const property (e.g., 'type' or 'kind') to each variant with a unique value

Summary: 1 error(s), 0 warning(s)
```

Fix by adding a discriminator:

```json
{
  "$defs": {
    "Response": {
      "anyOf": [
        {
          "type": "object",
          "properties": {
            "type": {"const": "success"},
            "data": {"type": "string"}
          }
        },
        {
          "type": "object",
          "properties": {
            "type": {"const": "error"},
            "error": {"type": "string"}
          }
        }
      ]
    }
  }
}
```

## Go-First Workflow

schemakit supports a Go-first approach where Go structs are the source of truth:

```
Go Structs (with doc comments)
         │
         ├──► schemakit generate ──► JSON Schema
         │
         └──► schemakit doc ──► Markdown Specification
```

This ensures your JSON Schema and documentation are always in sync with your Go types.

## Roadmap

See [TASKS.md](TASKS.md) for planned features including:

- Code generation for Go, Rust, TypeScript
- Full `$ref` resolution
- OpenAPI 3.1 support

## Documentation

Full documentation is available at [grokify.github.io/schemakit](https://grokify.github.io/schemakit/).

Key guides:

- [Spec Documentation Guide](https://grokify.github.io/schemakit/guides/spec-documentation/) - Creating human-audience specs with auto-generation
- [Go-First Workflow](https://grokify.github.io/schemakit/guides/go-first-workflow/) - Complete workflow guide
- [Lint Checks Reference](https://grokify.github.io/schemakit/reference/lint-checks/) - All lint check codes

## References

- [PRD.md](PRD.md) - Product requirements
- [TRD.md](TRD.md) - Technical requirements
- [JSON Schema Draft 2020-12](https://json-schema.org/draft/2020-12/json-schema-core)

## License

MIT License - see [LICENSE](LICENSE) for details.
