# Release Notes: schemago v0.1.0

**Release Date:** 2025-01-17

## Overview

This is the initial release of **schemago**, a JSON Schema to Go code generator with first-class union type support. This release focuses on the schema linting functionality, which validates JSON Schemas for Go compatibility issues before code generation.

## Highlights

- **Schema Linter**: Detect union patterns that cause problems in Go code generation
- **Smart Pattern Detection**: Correctly identifies nullable, reference, and polymorphic union patterns
- **Multiple Output Formats**: Text, JSON, and GitHub Actions annotations
- **Configurable**: Customizable discriminator field priority and thresholds

## Installation

```bash
go install github.com/grokify/schemago/cmd/schemago@v0.1.0
```

## Quick Start

```bash
# Lint a JSON Schema file
schemago lint schema.json

# Output as JSON
schemago lint --output json schema.json

# Output as GitHub Actions annotations
schemago lint --output github schema.json
```

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

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | No issues found |
| 1 | Errors found (schema has problems) |
| 2 | Warnings found but no errors |

## What's Next

Future releases will add:

- `schemago generate` - Generate Go code from validated schemas
- Full `$ref` resolution including circular references
- Configuration file support (`schemago.yaml`)
- Oracle Agent Spec compatibility

See [ROADMAP.md](ROADMAP.md) for the complete roadmap.

## Documentation

- [README.md](README.md) - Quick start guide
- [PRD.md](PRD.md) - Product requirements
- [TRD.md](TRD.md) - Technical requirements

## License

MIT License
