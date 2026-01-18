# Product Requirements Document: schemago

## Executive Summary

**schemago** is a JSON Schema to Go code generator designed to handle union types (`anyOf`, `oneOf`) correctly. Unlike existing generators that degrade unions to `interface{}`, schemago produces idiomatic Go code with proper tagged unions, discriminator-based unmarshalling, and nullable pointer types.

## Problem Statement

### Current State

Existing Go JSON Schema generators (e.g., `go-jsonschema`, `quicktype`) produce suboptimal code for schemas that use union types:

| JSON Schema Pattern | Current Output | Desired Output |
|---------------------|----------------|----------------|
| `anyOf: [A, B]` | `interface{}` | Tagged union struct |
| `anyOf: [T, null]` | `interface{}` | `*T` (pointer) |
| `const: "value"` | `interface{}` | Typed const |
| Discriminated unions | `interface{}` | Switch on discriminator |

### Impact

- **Type safety lost**: Runtime errors instead of compile-time errors
- **Poor developer experience**: No IDE autocomplete or type checking
- **Boilerplate required**: Manual type assertions everywhere
- **Incorrect semantics**: JSON unmarshalling doesn't enforce schema rules

### Root Cause

JSON Schema's type system includes constructs (unions, nullable types, discriminators) that have no direct Go equivalent. Existing generators take the easy path of emitting `interface{}` rather than generating proper wrapper types with custom marshalling logic.

## Target Users

1. **SDK Authors**: Building Go clients for APIs defined by JSON Schema or OpenAPI 3.1
2. **Agent Framework Developers**: Working with Agent Spec, MCP, A2A protocols
3. **Data Pipeline Engineers**: Processing JSON data with strict schema validation
4. **API Developers**: Generating server/client code from schemas

## Goals

### Primary Goals

1. **Correct Union Handling**: Generate tagged union types for `anyOf`/`oneOf` with proper `UnmarshalJSON`/`MarshalJSON`
2. **Nullable Type Support**: Convert `anyOf: [T, null]` to `*T` pointers
3. **Discriminator Detection**: Auto-detect and use discriminator fields for union decoding
4. **Idiomatic Go Output**: Generate code that follows Go conventions and passes `golangci-lint`

### Secondary Goals

1. **Enum Generation**: Generate typed consts for enum values
2. **Abstract Type Interfaces**: Generate interfaces for `x-abstract-component` types
3. **Reference Resolution**: Handle `$ref` correctly including circular references
4. **Custom Extensions**: Support `x-*` extension properties

### Non-Goals (v1)

1. Full JSON Schema 2020-12 support (defer: `if/then/else`, `unevaluatedProperties`)
2. Validation code generation
3. OpenAPI-specific features (defer to wrapper tool)
4. Multiple output languages

## Functional Requirements

### FR-1: Schema Parsing

| ID | Requirement | Priority | Status |
|----|-------------|----------|--------|
| FR-1.1 | Parse JSON Schema Draft 2020-12 | High | ⚠️ Partial (linter only) |
| FR-1.2 | Resolve `$ref` references (local and remote) | High | 🔲 Planned |
| FR-1.3 | Handle circular references without infinite loops | High | 🔲 Planned |
| FR-1.4 | Extract `x-*` extension properties | Medium | ⚠️ Partial (x-abstract-component) |

### FR-2: Union Detection & Classification

| ID | Requirement | Priority | Status |
|----|-------------|----------|--------|
| FR-2.1 | Detect `anyOf`/`oneOf` unions | High | ✅ Implemented |
| FR-2.2 | Classify nullable patterns: `anyOf: [T, null]` → pointer | High | ✅ Implemented |
| FR-2.3 | Detect discriminator fields (`const` properties) | High | ✅ Implemented |
| FR-2.4 | Detect reference vs inline patterns | High | ✅ Implemented |
| FR-2.5 | Support explicit `discriminator` keyword | Medium | 🔲 Planned |

### FR-3: Code Generation

| ID | Requirement | Priority | Status |
|----|-------------|----------|--------|
| FR-3.1 | Generate Go structs for object types | High | 🔲 Planned |
| FR-3.2 | Generate tagged union types for `anyOf`/`oneOf` | High | 🔲 Planned |
| FR-3.3 | Generate `UnmarshalJSON` for unions | High | 🔲 Planned |
| FR-3.4 | Generate `MarshalJSON` for unions | High | 🔲 Planned |
| FR-3.5 | Generate pointer types for nullable fields | High | 🔲 Planned |
| FR-3.6 | Generate typed consts for enums | Medium | 🔲 Planned |
| FR-3.7 | Generate interfaces for abstract types | Medium | 🔲 Planned |
| FR-3.8 | Support custom JSON field names | High | 🔲 Planned |

### FR-4: CLI Interface

| ID | Requirement | Priority | Status |
|----|-------------|----------|--------|
| FR-4.1 | `schemago lint` command (check Go compatibility) | High | ✅ Implemented |
| FR-4.2 | `schemago generate` command | High | 🔲 Planned |
| FR-4.3 | `schemago validate` command | Medium | 🔲 Planned |
| FR-4.4 | `schemago analyze` command (show detected patterns) | Medium | 🔲 Planned |
| FR-4.5 | Configuration file support | Medium | 🔲 Planned |

## Non-Functional Requirements

### NFR-1: Code Quality

- Generated code must pass `gofmt`
- Generated code must pass `golangci-lint` with default rules
- Generated code must compile with Go 1.21+
- No external dependencies in generated code (stdlib only)

### NFR-2: Performance

- Generate 100+ types in under 5 seconds
- Handle schemas up to 50,000 lines

### NFR-3: Maintainability

- Generated files clearly marked with generation comments
- Regeneration produces identical output (deterministic)
- Support `go:generate` directive

## Success Metrics

1. **Correctness**: 100% of Agent Spec types generate valid, compilable Go code
2. **Union Coverage**: 0 types degrade to `interface{}` when discriminator is available
3. **Round-trip**: JSON → Go → JSON produces identical output
4. **Adoption**: Used as official Go SDK generator for at least one major spec

## Competitive Analysis

| Tool | Union Support | Discriminators | Nullable | Go Idioms |
|------|---------------|----------------|----------|-----------|
| go-jsonschema | `interface{}` | No | `interface{}` | Partial |
| quicktype | Better | Partial | Yes | Partial |
| **schemago** | Tagged unions | Yes | Pointer | Full |

## Milestones

| Version | Scope | Status |
|---------|-------|--------|
| v0.1 | Schema linter with union detection | ✅ Complete |
| v0.2 | Core IR + basic struct generation | 🔲 Planned |
| v0.3 | Union types + discriminator codegen | 🔲 Planned |
| v0.4 | Full Agent Spec support | 🔲 Planned |
| v1.0 | Production ready | 🔲 Planned |

## References

- [JSON Schema Draft 2020-12](https://json-schema.org/draft/2020-12/json-schema-core)
- [Oracle Agent Spec](https://oracle.github.io/agent-spec/)
- [go-jsonschema](https://github.com/omissis/go-jsonschema)
- [google/jsonschema-go](https://github.com/google/jsonschema-go)
