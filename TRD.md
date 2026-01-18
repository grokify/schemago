# Technical Requirements Document: schemago

## 1. Overview

This document describes the technical architecture and implementation requirements for **schemago**, a JSON Schema to Go code generator with first-class union type support.

### 1.1 Context

JSON Schema (Draft 2020-12) is widely used to define data models for APIs and specifications. While tools like `go-jsonschema` can generate Go types from JSON Schema, they produce suboptimal code for certain schema patterns that don't map cleanly to Go's type system.

### 1.2 Scope

This document covers:

- System architecture and component design
- Semantic IR (Intermediate Representation) specification
- Union type handling strategies
- Code generation patterns
- Testing requirements

## 2. Architecture

### 2.1 High-Level Pipeline

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│  JSON Schema    │────▶│  Schema Parser   │────▶│  Semantic IR    │
│  (input.json)   │     │  (jsonschema-go) │     │                 │
└─────────────────┘     └──────────────────┘     └────────┬────────┘
                                                          │
                                                          ▼
                                               ┌──────────────────┐
                                               │  Union Analyzer  │
                                               │  - Detect unions │
                                               │  - Find discrim. │
                                               │  - Classify      │
                                               └────────┬─────────┘
                                                        │
                                                        ▼
                                               ┌──────────────────┐
                                               │  Go Generator    │
                                               │  - Structs       │
                                               │  - Unions        │
                                               │  - Marshal/Unm.  │
                                               └────────┬─────────┘
                                                        │
                                                        ▼
                                               ┌──────────────────┐
                                               │  Generated .go   │
                                               │  files           │
                                               └──────────────────┘
```

### 2.2 Package Structure

```
schemago/
├── cmd/
│   └── schemago/           # CLI entry point           ✅ Implemented
│       └── main.go
├── linter/                  # Schema linting           ✅ Implemented
│   ├── linter.go           # Core linting logic
│   ├── linter_test.go      # Unit tests
│   ├── schema.go           # JSON Schema types
│   └── issue.go            # Issue/Result types
├── parser/                  # JSON Schema parsing      🔲 Planned
│   ├── parser.go           # Schema loader
│   └── resolver.go         # $ref resolution
├── ir/                      # Intermediate Rep.        🔲 Planned
│   ├── types.go            # IR type definitions
│   ├── builder.go          # Schema → IR conversion
│   └── analyzer.go         # Union/pattern detection
├── generator/               # Go code generation       🔲 Planned
│   ├── generator.go        # Main generator
│   ├── struct.go           # Struct generation
│   ├── union.go            # Union generation
│   ├── marshal.go          # Marshal/Unmarshal methods
│   └── templates/          # Go templates
├── testdata/                # Test schemas             ✅ Implemented
└── examples/                # Example schemas          🔲 Planned
```

## 2.3 Linter Package (Implemented)

The `linter/` package provides schema validation for Go compatibility. It detects patterns that would cause problems during code generation.

### Issue Codes

| Code | Severity | Description |
|------|----------|-------------|
| `union-no-discriminator` | Error | Union lacks discriminator field |
| `inconsistent-discriminator` | Error | Variants use different discriminators |
| `missing-const` | Error | Variant lacks const value |
| `duplicate-const-value` | Error | Multiple variants have same value |
| `large-union` | Warning | Union has >10 variants |
| `nested-union` | Warning | Union nested >2 levels deep |
| `additional-properties` | Warning | Variant has `additionalProperties: true` |

### Configuration

```go
type Config struct {
    MaxUnionVariants     int      // Default: 10
    MaxUnionNestingDepth int      // Default: 2
    DiscriminatorFields  []string // Default: ["component_type", "type", "kind"]
}
```

### Pattern Detection

The linter correctly identifies and skips:

- **Nullable patterns**: `anyOf: [T, null]` - Not flagged as missing discriminator
- **Reference patterns**: `anyOf: [ComponentReference, BaseXxx]` - Recognized by `$component_ref` property
- **All-$ref unions**: Unions where all variants are `$ref` - Skipped (requires resolution)

## 3. Semantic IR Specification (Planned)

The Semantic IR is the core abstraction that preserves union semantics lost by direct AST approaches.

### 3.1 Type Definitions

```go
// TypeKind classifies the semantic type
type TypeKind int

const (
    KindStruct TypeKind = iota
    KindUnion
    KindEnum
    KindArray
    KindMap
    KindPrimitive
    KindRef
)

// TypeDef represents a schema type definition
type TypeDef struct {
    Name        string
    Kind        TypeKind
    Description string

    // For KindStruct
    Fields      []Field

    // For KindUnion
    Variants      []Variant
    Discriminator *Discriminator
    UnionKind     UnionKind  // Reference, Nullable, Polymorphic

    // For KindEnum
    EnumValues  []string
    EnumType    string  // "string", "integer"

    // For KindArray
    ItemType    *TypeRef

    // For KindMap
    ValueType   *TypeRef

    // For KindPrimitive
    PrimitiveType string  // "string", "integer", "number", "boolean"

    // Metadata
    Abstract    bool    // x-abstract-component
    Extensions  map[string]any
}

// Field represents a struct field
type Field struct {
    Name         string
    JSONName     string
    Type         TypeRef
    Required     bool
    Nullable     bool  // anyOf [T, null]
    Description  string
    Default      any
    Const        any   // const value
}

// Variant represents a union variant
type Variant struct {
    Name     string
    TypeRef  TypeRef
    Const    string  // discriminator const value
}

// Discriminator identifies how to distinguish variants
type Discriminator struct {
    PropertyName string            // e.g., "component_type"
    Mapping      map[string]string // const value → variant name
}

// UnionKind classifies the union pattern
type UnionKind int

const (
    UnionReference   UnionKind = iota  // ComponentReference | BaseXxx
    UnionNullable                       // T | null
    UnionPolymorphic                    // A | B | C with discriminator
    UnionUntyped                        // Fallback to interface{}
)

// TypeRef is a reference to a type
type TypeRef struct {
    Name     string
    Pointer  bool
    Package  string
}
```

### 3.2 IR Building Rules

| JSON Schema Pattern | IR Result |
|---------------------|-----------|
| `type: object` with `properties` | `KindStruct` |
| `anyOf: [T, null]` | Field with `Nullable: true` |
| `anyOf: [$ref A, $ref B]` with discriminator | `KindUnion` with `UnionPolymorphic` |
| `anyOf: [ComponentReference, BaseXxx]` | `KindUnion` with `UnionReference` |
| `enum: [...]` | `KindEnum` |
| `type: array` | `KindArray` |
| `additionalProperties: {...}` | `KindMap` |
| `$ref: #/$defs/Foo` | `KindRef` |

## 4. Union Type Handling

### 4.1 Pattern Detection Algorithm

```go
func ClassifyUnion(schema *jsonschema.Schema) UnionKind {
    variants := schema.AnyOf // or OneOf

    // Check for nullable pattern
    if len(variants) == 2 && hasNullType(variants) {
        return UnionNullable
    }

    // Check for reference pattern
    if len(variants) == 2 && hasComponentReference(variants) {
        return UnionReference
    }

    // Check for discriminator
    if disc := findDiscriminator(variants); disc != nil {
        return UnionPolymorphic
    }

    return UnionUntyped
}

func findDiscriminator(variants []Schema) *Discriminator {
    // Look for common property with const values
    candidates := make(map[string][]string)

    for _, v := range variants {
        for propName, prop := range v.Properties {
            if prop.Const != nil {
                candidates[propName] = append(candidates[propName], prop.Const.(string))
            }
        }
    }

    // Find property present in all variants with unique const values
    for propName, values := range candidates {
        if len(values) == len(variants) && allUnique(values) {
            return &Discriminator{
                PropertyName: propName,
                Mapping:      buildMapping(variants, propName),
            }
        }
    }

    return nil
}
```

### 4.2 Generated Code Patterns

#### Pattern A: Nullable Field

**Schema:**
```json
{
  "description": {
    "anyOf": [{"type": "string"}, {"type": "null"}],
    "default": null
  }
}
```

**Generated Go:**
```go
type MyType struct {
    Description *string `json:"description,omitempty"`
}
```

#### Pattern B: Reference Union

**Schema:**
```json
{
  "Agent": {
    "anyOf": [
      {"$ref": "#/$defs/ComponentReference"},
      {"$ref": "#/$defs/BaseAgent"}
    ]
  }
}
```

**Generated Go:**
```go
type Agent struct {
    // Reference to external component (mutually exclusive with inline)
    ComponentRef string `json:"$component_ref,omitempty"`

    // Inline component definition
    *BaseAgent
}

func (a *Agent) IsReference() bool {
    return a.ComponentRef != ""
}

func (a *Agent) UnmarshalJSON(data []byte) error {
    var probe struct {
        ComponentRef string `json:"$component_ref"`
    }
    if err := json.Unmarshal(data, &probe); err != nil {
        return err
    }

    if probe.ComponentRef != "" {
        a.ComponentRef = probe.ComponentRef
        return nil
    }

    a.BaseAgent = new(BaseAgent)
    return json.Unmarshal(data, a.BaseAgent)
}

func (a Agent) MarshalJSON() ([]byte, error) {
    if a.ComponentRef != "" {
        return json.Marshal(struct {
            ComponentRef string `json:"$component_ref"`
        }{a.ComponentRef})
    }
    return json.Marshal(a.BaseAgent)
}
```

#### Pattern C: Polymorphic Union

**Schema:**
```json
{
  "BaseAgenticComponent": {
    "anyOf": [
      {"$ref": "#/$defs/OciAgent"},
      {"$ref": "#/$defs/RemoteAgent"},
      {"$ref": "#/$defs/Flow"},
      {"$ref": "#/$defs/Agent"}
    ],
    "x-abstract-component": true
  }
}
```

**Generated Go:**
```go
type AgenticComponent struct {
    ComponentType string `json:"component_type"`

    OciAgent    *BaseOciAgent
    RemoteAgent *BaseRemoteAgent
    Flow        *BaseFlow
    Agent       *BaseAgent
}

func (c *AgenticComponent) UnmarshalJSON(data []byte) error {
    var probe struct {
        ComponentType string `json:"component_type"`
    }
    if err := json.Unmarshal(data, &probe); err != nil {
        return err
    }

    c.ComponentType = probe.ComponentType

    switch probe.ComponentType {
    case "OciAgent":
        c.OciAgent = new(BaseOciAgent)
        return json.Unmarshal(data, c.OciAgent)
    case "RemoteAgent":
        c.RemoteAgent = new(BaseRemoteAgent)
        return json.Unmarshal(data, c.RemoteAgent)
    case "Flow":
        c.Flow = new(BaseFlow)
        return json.Unmarshal(data, c.Flow)
    case "Agent":
        c.Agent = new(BaseAgent)
        return json.Unmarshal(data, c.Agent)
    default:
        return fmt.Errorf("unknown component_type: %q", probe.ComponentType)
    }
}

func (c AgenticComponent) MarshalJSON() ([]byte, error) {
    switch c.ComponentType {
    case "OciAgent":
        return json.Marshal(c.OciAgent)
    case "RemoteAgent":
        return json.Marshal(c.RemoteAgent)
    case "Flow":
        return json.Marshal(c.Flow)
    case "Agent":
        return json.Marshal(c.Agent)
    default:
        return nil, fmt.Errorf("no variant set")
    }
}

// Helper methods for type-safe access
func (c *AgenticComponent) AsOciAgent() (*BaseOciAgent, bool) {
    return c.OciAgent, c.OciAgent != nil
}

func (c *AgenticComponent) AsRemoteAgent() (*BaseRemoteAgent, bool) {
    return c.RemoteAgent, c.RemoteAgent != nil
}
// ... etc
```

## 5. Enum Generation

**Schema:**
```json
{
  "ModelProvider": {
    "enum": ["META", "GROK", "COHERE", "OTHER"],
    "type": "string"
  }
}
```

**Generated Go:**
```go
type ModelProvider string

const (
    ModelProviderMeta   ModelProvider = "META"
    ModelProviderGrok   ModelProvider = "GROK"
    ModelProviderCohere ModelProvider = "COHERE"
    ModelProviderOther  ModelProvider = "OTHER"
)

func (m ModelProvider) IsValid() bool {
    switch m {
    case ModelProviderMeta, ModelProviderGrok, ModelProviderCohere, ModelProviderOther:
        return true
    }
    return false
}

func (m ModelProvider) String() string {
    return string(m)
}
```

## 6. Configuration

### 6.1 Configuration File

```yaml
# schemago.yaml
input:
  schema: agentspec_v25.4.1.json

output:
  package: agentspec
  dir: ./agentspec
  file_per_type: false

options:
  # Union handling
  nullable_as_pointer: true
  generate_union_helpers: true
  unknown_union_error: true  # vs return nil

  # Enum handling
  generate_enum_consts: true
  generate_enum_validators: true

  # Naming
  struct_name_prefix: ""
  struct_name_suffix: ""

  # Extensions
  abstract_extension: "x-abstract-component"

  # Discriminators
  discriminator_priority:
    - component_type
    - type
    - kind
```

### 6.2 CLI Interface

#### Implemented Commands

```bash
# Lint schema for Go compatibility issues
schemago lint schema.json                    # Text output (default)
schemago lint --output json schema.json      # JSON output
schemago lint --output github schema.json    # GitHub Actions annotations

# Show version
schemago version
```

#### Planned Commands

```bash
# Generate from schema
schemago generate -i schema.json -o ./pkg/types

# Generate with config
schemago generate -c schemago.yaml

# Validate schema (check for unsupported patterns)
schemago validate schema.json

# Analyze schema (show detected patterns)
schemago analyze schema.json
```

## 7. Testing Requirements

### 7.1 Unit Tests

| Component | Coverage Target | Current |
|-----------|-----------------|---------|
| Linter | 90% | 60.8% |
| Parser | 90% | 🔲 Not implemented |
| IR Builder | 95% | 🔲 Not implemented |
| Generator | 90% | 🔲 Not implemented |

### 7.2 Implemented Tests

The linter package includes 7 unit tests:

- `TestLintNullablePattern` - Verifies nullable patterns are skipped
- `TestLintUnionWithDiscriminator` - Verifies valid discriminated unions pass
- `TestLintUnionWithoutDiscriminator` - Verifies missing discriminator is flagged
- `TestLintLargeUnion` - Verifies large union warning
- `TestLintAdditionalProperties` - Verifies additionalProperties warning
- `TestLintAllRefs` - Verifies all-$ref unions are skipped
- `TestResultCounts` - Verifies error/warning counting

### 7.3 Planned Integration Tests

- Round-trip tests: JSON → Go → JSON equality
- Compile tests: Generated code must compile
- Lint tests: Generated code must pass golangci-lint

### 7.4 Test Schemas

Current:

- `testdata/good_schema.json` - Valid schema with discriminators
- `testdata/bad_schema.json` - Schema with lint issues

Planned:

- `testdata/nullable.json` - Nullable field patterns
- `testdata/union_reference.json` - Reference vs inline
- `testdata/union_polymorphic.json` - Discriminated unions
- `testdata/enum.json` - Enum types
- `testdata/circular.json` - Circular references
- `testdata/agentspec.json` - Full Agent Spec

## 8. Dependencies

### 8.1 Current Dependencies

| Dependency | Purpose | Status |
|------------|---------|--------|
| `github.com/spf13/cobra` | CLI framework | ✅ In use |

### 8.2 Planned Dependencies

| Dependency | Purpose |
|------------|---------|
| `github.com/google/jsonschema-go` | Full schema parsing with $ref resolution |
| `gopkg.in/yaml.v3` | Config file parsing |

### 8.3 Generated Code Dependencies

**None** - Generated code uses only Go standard library.

## 9. Error Handling

### 9.1 Parse Errors

- Invalid JSON syntax
- Invalid JSON Schema
- Unresolved `$ref`
- Circular reference detection

### 9.2 Generation Errors

- Unsupported schema patterns (with clear message)
- Name collisions
- Invalid Go identifiers

### 9.3 Warnings

- Fallback to `interface{}` (with reason)
- Unused type definitions
- Deprecated patterns

## 10. Implementation Phases

### Phase 1: Linter ✅ Complete

- [x] Project structure and CI setup
- [x] CLI with cobra (`lint`, `version` commands)
- [x] JSON Schema parsing (simplified for linting)
- [x] Union detection algorithm
- [x] Nullable pattern detection
- [x] Reference pattern detection
- [x] Discriminator detection
- [x] Multiple output formats (text, JSON, GitHub)
- [x] Unit tests (60.8% coverage)

### Phase 2: Foundation 🔲 Planned

- [ ] Full schema parsing with $ref resolution
- [ ] Basic IR types
- [ ] Schema → IR conversion
- [ ] Struct generation (no unions)

### Phase 3: Unions 🔲 Planned

- [ ] Nullable field handling (pointer types)
- [ ] Reference union generation
- [ ] Polymorphic union generation
- [ ] Marshal/Unmarshal generation

### Phase 4: Polish 🔲 Planned

- [ ] Configuration file support
- [ ] Enum generation
- [ ] Abstract type interfaces
- [ ] Agent Spec validation
- [ ] `generate`, `validate`, `analyze` commands

### Phase 5: Release 🔲 Planned

- [ ] Documentation
- [ ] Examples
- [ ] GoReleaser setup
- [ ] v1.0.0 release

## 11. Appendix

### A. Agent Spec Statistics

| Metric | Count |
|--------|-------|
| Total type definitions | ~140 |
| Union types (anyOf) | ~50 |
| Nullable field patterns | ~100+ |
| Discriminator (component_type) | 32 |
| Abstract types | 9 |
| Enums | 4 |

### B. References

- [JSON Schema Draft 2020-12](https://json-schema.org/draft/2020-12/json-schema-core)
- [Oracle Agent Spec](https://oracle.github.io/agent-spec/)
- [google/jsonschema-go](https://github.com/google/jsonschema-go)
- [Go JSON Marshaling](https://pkg.go.dev/encoding/json)
