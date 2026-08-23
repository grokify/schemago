# schemakit generate

Generate JSON Schema from Go struct types using reflection.

## Usage

```bash
schemakit generate <package> <type> [flags]
```

## Flags

| Flag | Description |
|------|-------------|
| `-o, --output` | Output file (default: stdout) |
| `--indent` | Indent JSON output (default: true) |
| `--check` | Verify the committed `-o` file matches freshly generated output; exit non-zero on drift (no write). Requires `-o`. |

## Examples

```bash
# Generate to stdout
schemakit generate github.com/myorg/myproject/types Config

# Save to file
schemakit generate -o schema.json github.com/myorg/myproject/types Config

# Without indentation
schemakit generate --indent=false github.com/myorg/myproject/types Config

# CI drift guard: fail if the committed schema is out of sync with the Go structs
schemakit generate -o schema.json --check github.com/myorg/myproject/types Config
```

## Drift Guard (`--check`)

`--check` regenerates the schema in memory and compares it to the committed
file at `-o`, exiting non-zero on any difference without rewriting the file.
This keeps a committed schema honest against its Go source of truth. Pair it
with a `//go:generate` directive so `go generate ./...` refreshes the schema
and CI enforces it:

```go
//go:generate schemakit generate -o schema.json github.com/myorg/myproject/types Config
```

```bash
# In CI
schemakit generate -o schema.json --check github.com/myorg/myproject/types Config
```

## How It Works

The command creates a temporary Go program that:

1. Imports your target package
2. Uses [invopop/jsonschema](https://github.com/invopop/jsonschema) to reflect on the type
3. Generates a JSON Schema with `$defs` for nested types

## Module Resolution

The command supports both local and remote packages:

- **Local modules**: Packages in `$GOPATH/src` are resolved via replace directives
- **Remote modules**: Packages are fetched via `go get`

## Struct Tags

The generated schema respects these struct tags:

| Tag | Effect |
|-----|--------|
| `json:"name"` | Sets the property name |
| `json:",omitempty"` | Marks field as optional |
| `json:"-"` | Excludes field from schema |
| `jsonschema:"title=X"` | Sets schema title |
| `jsonschema:"description=X"` | Sets schema description |
| `jsonschema:"enum=a,b,c"` | Defines enum values |

## Example Output

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "config",
  "$defs": {
    "Database": {
      "type": "object",
      "properties": {
        "host": { "type": "string" },
        "port": { "type": "integer" }
      },
      "required": ["host", "port"]
    }
  },
  "type": "object",
  "properties": {
    "database": { "$ref": "#/$defs/Database" }
  }
}
```
