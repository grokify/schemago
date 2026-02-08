# Release Notes: schemalint v0.2.0

**Release Date:** 2026-02-08

## Overview

This release renames the project from **schemago** to **schemalint** and introduces the **scale profile** for strict static type compatibility. The new name better reflects the tool's purpose as a JSON Schema linter for statically-typed languages.

## Breaking Changes

- **Binary renamed**: `schemago` is now `schemalint`
- **Module path changed**: `github.com/grokify/schemago` is now `github.com/grokify/schemalint`

Update your installation:

```bash
# Old
go install github.com/grokify/schemago/cmd/schemago@latest

# New
go install github.com/grokify/schemalint/cmd/schemalint@latest
```

## Highlights

- **Scale Profile**: New `--profile scale` option for strict static type compatibility
- **Homebrew Support**: Install via `brew install grokify/tap/schemalint`
- **Multi-Language Focus**: Rebranded to emphasize support for all statically-typed languages (Go, Rust, TypeScript, etc.)

## Installation

### Homebrew

```bash
brew install grokify/tap/schemalint
```

### Go Install

```bash
go install github.com/grokify/schemalint/cmd/schemalint@latest
```

## New Features

### Scale Profile

The scale profile enforces strict rules for clean code generation in statically-typed languages:

```bash
schemalint lint schema.json --profile scale
```

| Code | Description |
|------|-------------|
| `composition-disallowed` | Disallow `anyOf`, `oneOf`, `allOf` |
| `additional-properties-disallowed` | Disallow `additionalProperties: true` |
| `missing-type` | Require explicit `type` field |
| `mixed-type-disallowed` | Disallow type arrays like `["string", "number"]` |

### Profile Selection

Use `--profile` or `-p` to select a linting profile:

```bash
schemalint lint schema.json                  # default profile
schemalint lint schema.json -p scale         # strict scale profile
```

### Property Case Convention

Use `--property-case` to enforce a naming convention for object properties. The default is `camelCase`.

```bash
# Enforce snake_case
schemalint lint schema.json --property-case snake_case

# Disable case checking
schemalint lint schema.json --property-case none
```

| Convention   | Description                 |
|--------------|-----------------------------|
| `none`       | No case validation          |
| `camelCase`  | e.g., `myProperty` (default) |
| `snake_case` | e.g., `my_property`          |
| `kebab-case` | e.g., `my-property`          |
| `PascalCase` | e.g., `MyProperty`           |

## What's Next

Future releases will add:

- Code generation for Go, Rust, TypeScript
- Full `$ref` resolution including circular references
- Configuration file support
- OpenAPI 3.1 support

See [ROADMAP.md](ROADMAP.md) for the complete roadmap.

## Documentation

- [README.md](README.md) - Quick start guide
- [CHANGELOG.md](CHANGELOG.md) - Full changelog
- [PRD.md](PRD.md) - Product requirements
- [TRD.md](TRD.md) - Technical requirements

## License

MIT License
