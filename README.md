# SchemaGo

[![Build Status][build-status-svg]][build-status-url]
[![Lint Status][lint-status-svg]][lint-status-url]
[![Go Report Card][goreport-svg]][goreport-url]
[![Docs][docs-godoc-svg]][docs-godoc-url]
[![Visualization][viz-svg]][viz-url]
[![License][license-svg]][license-url]

JSON Schema to Go code generator with first-class union type support.

## Overview

**schemago** is a JSON Schema to Go code generator that correctly handles union types (`anyOf`/`oneOf`). Unlike existing generators that degrade unions to `interface{}`, schemago produces idiomatic Go code with:

- Proper tagged union structs
- Discriminator-based `UnmarshalJSON`/`MarshalJSON`
- Nullable pointer types for `anyOf [T, null]`

## Installation

```bash
go install github.com/grokify/schemago/cmd/schemago@latest
```

## Usage

### Lint Schema for Go Compatibility

Check a JSON Schema for patterns that cause problems in Go code generation:

```bash
schemago lint schema.json
```

Output formats:

```bash
schemago lint --output text schema.json   # Human-readable (default)
schemago lint --output json schema.json   # Machine-readable JSON
schemago lint --output github schema.json # GitHub Actions annotations
```

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | No issues found |
| 1 | Errors found (schema has problems) |
| 2 | Warnings found but no errors |

## Lint Checks

### Errors

| Code | Description |
|------|-------------|
| `union-no-discriminator` | Union (`anyOf`/`oneOf`) has no discriminator field |
| `inconsistent-discriminator` | Variants use different discriminator field names |
| `missing-const` | Union variant lacks `const` value for discriminator |
| `duplicate-const-value` | Multiple variants have the same discriminator value |

### Warnings

| Code | Description |
|------|-------------|
| `large-union` | Union has more than 10 variants |
| `nested-union` | Union nested more than 2 levels deep |
| `additional-properties` | Union variant has `additionalProperties: true` |

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

Running `schemago lint` will report:

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

## Roadmap

See [ROADMAP.md](ROADMAP.md) for planned features including:

- `schemago generate` - Generate Go code from schemas
- Full Oracle Agent Spec compatibility
- OpenAPI 3.1 support

## References

- [PRD.md](PRD.md) - Product requirements
- [TRD.md](TRD.md) - Technical requirements
- [JSON Schema Draft 2020-12](https://json-schema.org/draft/2020-12/json-schema-core)
- [Oracle Agent Spec](https://oracle.github.io/agent-spec/)

## License

MIT License - see [LICENSE](LICENSE) for details.

 [build-status-svg]: https://github.com/grokify/schemago/actions/workflows/ci.yaml/badge.svg?branch=main
 [build-status-url]: https://github.com/grokify/schemago/actions/workflows/ci.yaml
 [lint-status-svg]: https://github.com/grokify/schemago/actions/workflows/lint.yaml/badge.svg?branch=main
 [lint-status-url]: https://github.com/grokify/schemago/actions/workflows/lint.yaml
 [goreport-svg]: https://goreportcard.com/badge/github.com/grokify/schemago
 [goreport-url]: https://goreportcard.com/report/github.com/grokify/schemago
 [docs-godoc-svg]: https://pkg.go.dev/badge/github.com/grokify/schemago
 [docs-godoc-url]: https://pkg.go.dev/github.com/grokify/schemago
 [viz-svg]: https://img.shields.io/badge/visualizaton-Go-blue.svg
 [viz-url]: https://mango-dune-07a8b7110.1.azurestaticapps.net/?repo=grokify%2Fschemago
 [loc-svg]: https://tokei.rs/b1/github/grokify/schemago
 [repo-url]: https://github.com/grokify/schemago
 [license-svg]: https://img.shields.io/badge/license-MIT-blue.svg
 [license-url]: https://github.com/grokify/schemago/blob/master/LICENSE