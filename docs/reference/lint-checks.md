# Lint Checks Reference

Complete reference of all lint checks performed by `schemakit lint`.

## Default Profile

### Errors

Errors indicate patterns that will cause problems in generated code.

| Code | Name | Description |
|------|------|-------------|
| `union-no-discriminator` | Missing Discriminator | Union (`anyOf`/`oneOf`) has no discriminator field |
| `inconsistent-discriminator` | Inconsistent Discriminator | Variants use different discriminator field names |
| `missing-const` | Missing Const | Union variant lacks `const` value for discriminator |
| `duplicate-const-value` | Duplicate Const | Multiple variants have the same discriminator value |
| `invalid-property-case` | Invalid Property Case | Property name does not follow the configured case convention |

### Warnings

Warnings indicate patterns that may cause issues or are suboptimal.

| Code | Name | Description |
|------|------|-------------|
| `large-union` | Large Union | Union has more than 10 variants |
| `nested-union` | Nested Union | Union nested more than 2 levels deep |
| `additional-properties` | Additional Properties | Union variant has `additionalProperties: true` |
| `ambiguous-union` | Ambiguous Union | Union variants cannot be distinguished |
| `circular-reference` | Circular Reference | Schema contains circular `$ref` |

## Scale Profile

The scale profile includes all default checks plus these additional errors:

| Code | Name | Description |
|------|------|-------------|
| `composition-disallowed` | Composition Disallowed | Disallow `anyOf`, `oneOf`, `allOf` |
| `additional-properties-disallowed` | Additional Props Disallowed | Disallow `additionalProperties: true` |
| `missing-type` | Missing Type | Require explicit `type` field |
| `mixed-type-disallowed` | Mixed Type Disallowed | Disallow type arrays like `["string", "number"]` |

## Navigable Profile

The navigable profile enforces flat, locally-comprehensible schemas for human
review and AI authoring. See [Profiles](profiles.md#navigable-profile) for the
full rationale.

| Code | Name | Severity | Description |
|------|------|----------|-------------|
| `deep-nesting` | Deep Nesting | Error | Object nesting exceeds 2 levels |
| `deep-array-nesting` | Deep Array Nesting | Warning | Arrays of arrays of objects reduce navigability |
| `missing-id-field` | Missing ID Field | Warning | Array items lack an ID field for cross-referencing |

## Examples

### union-no-discriminator

**Problem:**

```json
{
  "anyOf": [
    {"type": "object", "properties": {"data": {"type": "string"}}},
    {"type": "object", "properties": {"error": {"type": "string"}}}
  ]
}
```

**Fix:** Add a discriminator field with `const` values:

```json
{
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
```

### invalid-property-case

**Problem (with `--property-case camelCase`):**

```json
{
  "properties": {
    "user_name": {"type": "string"}
  }
}
```

**Fix:**

```json
{
  "properties": {
    "userName": {"type": "string"}
  }
}
```

### composition-disallowed (Scale Profile)

**Problem:**

```json
{
  "allOf": [
    {"$ref": "#/$defs/Base"},
    {"$ref": "#/$defs/Extended"}
  ]
}
```

**Fix:** Flatten into a single object type or use explicit typing.

## Property Case Conventions

| Convention | Pattern | Example |
|------------|---------|---------|
| `none` | No validation | Any case allowed |
| `camelCase` | lowerCamelCase | `userName`, `createdAt` |
| `snake_case` | lower_snake_case | `user_name`, `created_at` |
| `kebab-case` | lower-kebab-case | `user-name`, `created-at` |
| `PascalCase` | UpperCamelCase | `UserName`, `CreatedAt` |
