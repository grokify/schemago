package linter

import (
	"encoding/json"
	"fmt"
	"os"
)

// Profile represents a linting profile with predefined rules.
type Profile string

const (
	// ProfileDefault is the standard linting profile.
	ProfileDefault Profile = "default"
	// ProfileScale is a strict profile for static type compatibility (jsonschema4scale).
	ProfileScale Profile = "scale"
)

// Config holds linter configuration options.
type Config struct {
	// Profile is the linting profile to use.
	Profile Profile
	// MaxUnionVariants is the threshold for large union warnings (default: 10)
	MaxUnionVariants int
	// MaxUnionNestingDepth is the threshold for nested union warnings (default: 2)
	MaxUnionNestingDepth int
	// DiscriminatorFields are the field names to look for as discriminators
	DiscriminatorFields []string
}

// DefaultConfig returns the default linter configuration.
func DefaultConfig() Config {
	return Config{
		Profile:              ProfileDefault,
		MaxUnionVariants:     10,
		MaxUnionNestingDepth: 2,
		DiscriminatorFields:  []string{"component_type", "type", "kind"},
	}
}

// IsScaleProfile returns true if the scale profile is active.
func (c Config) IsScaleProfile() bool {
	return c.Profile == ProfileScale
}

// Linter checks JSON Schemas for Go compatibility issues.
type Linter struct {
	config Config
}

// New creates a new Linter with the given configuration.
func New(config Config) *Linter {
	return &Linter{config: config}
}

// NewWithDefaults creates a new Linter with default configuration.
func NewWithDefaults() *Linter {
	return New(DefaultConfig())
}

// LintFile lints a JSON Schema file.
func (l *Linter) LintFile(path string) (*Result, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	result, err := l.Lint(data)
	if err != nil {
		return nil, err
	}
	result.SchemaPath = path
	return result, nil
}

// Lint lints JSON Schema data.
func (l *Linter) Lint(data []byte) (*Result, error) {
	var schema Schema
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, fmt.Errorf("failed to parse JSON Schema: %w", err)
	}

	result := &Result{
		Issues: []Issue{},
	}

	// Lint the root schema
	l.lintSchema(&schema, "$", result, 0)

	// Lint definitions ($defs)
	for name, def := range schema.Defs {
		path := fmt.Sprintf("$/$defs/%s", name)
		l.lintSchema(def, path, result, 0)
	}

	// Lint legacy definitions (definitions)
	for name, def := range schema.Definitions {
		path := fmt.Sprintf("$/definitions/%s", name)
		l.lintSchema(def, path, result, 0)
	}

	return result, nil
}

func (l *Linter) lintSchema(schema *Schema, path string, result *Result, unionDepth int) {
	if schema == nil {
		return
	}

	// Scale profile: strict checks for static type compatibility
	if l.config.IsScaleProfile() {
		l.lintScaleProfile(schema, path, result)
	}

	// Check for union types
	if len(schema.AnyOf) > 0 {
		l.lintUnion(schema.AnyOf, path+"/anyOf", result, unionDepth, "anyOf")
	}
	if len(schema.OneOf) > 0 {
		l.lintUnion(schema.OneOf, path+"/oneOf", result, unionDepth, "oneOf")
	}

	// Check properties
	for propName, propSchema := range schema.Properties {
		propPath := fmt.Sprintf("%s/properties/%s", path, propName)
		l.lintSchema(propSchema, propPath, result, unionDepth)
	}

	// Check items
	if schema.Items != nil {
		l.lintSchema(schema.Items, path+"/items", result, unionDepth)
	}

	// Check additionalProperties
	if schema.AdditionalPropertiesSchema != nil {
		l.lintSchema(schema.AdditionalPropertiesSchema, path+"/additionalProperties", result, unionDepth)
	}
}

// lintScaleProfile applies strict checks for the scale profile.
func (l *Linter) lintScaleProfile(schema *Schema, path string, result *Result) {
	// Disallow composition keywords (anyOf, oneOf, allOf)
	if len(schema.AnyOf) > 0 {
		result.Issues = append(result.Issues, Issue{
			Code:       CodeCompositionDisallowed,
			Severity:   SeverityError,
			Path:       path + "/anyOf",
			Message:    "anyOf is disallowed in scale profile",
			Suggestion: "Use separate schema definitions instead of unions",
		})
	}
	if len(schema.OneOf) > 0 {
		result.Issues = append(result.Issues, Issue{
			Code:       CodeCompositionDisallowed,
			Severity:   SeverityError,
			Path:       path + "/oneOf",
			Message:    "oneOf is disallowed in scale profile",
			Suggestion: "Use separate schema definitions instead of unions",
		})
	}
	if len(schema.AllOf) > 0 {
		result.Issues = append(result.Issues, Issue{
			Code:       CodeCompositionDisallowed,
			Severity:   SeverityError,
			Path:       path + "/allOf",
			Message:    "allOf is disallowed in scale profile",
			Suggestion: "Flatten the schema structure instead of using composition",
		})
	}

	// Disallow additionalProperties: true
	if schema.AdditionalProperties != nil && *schema.AdditionalProperties {
		result.Issues = append(result.Issues, Issue{
			Code:       CodeAdditionalPropsDisallowed,
			Severity:   SeverityError,
			Path:       path,
			Message:    "additionalProperties: true is disallowed in scale profile",
			Suggestion: "Set additionalProperties: false or remove it to ensure strict type mapping",
		})
	}

	// Require explicit type (unless it's a $ref or boolean schema or container)
	if !schema.HasType() && !schema.IsRef() && !schema.IsBooleanSchema {
		// Only report if this is a meaningful schema (has properties, items, etc.)
		if len(schema.Properties) > 0 || schema.Items != nil || schema.Const != nil || len(schema.Enum) > 0 {
			result.Issues = append(result.Issues, Issue{
				Code:       CodeMissingType,
				Severity:   SeverityError,
				Path:       path,
				Message:    "missing explicit type field in scale profile",
				Suggestion: "Add a 'type' field to specify the schema type",
			})
		}
	}

	// Disallow mixed types (type arrays like ["string", "number"])
	if schema.HasMixedType() {
		result.Issues = append(result.Issues, Issue{
			Code:       CodeMixedTypeDisallowed,
			Severity:   SeverityError,
			Path:       path,
			Message:    fmt.Sprintf("mixed type array %v is disallowed in scale profile", schema.TypeList),
			Suggestion: "Use a single type; for nullable types, use a separate null check",
		})
	}
}

func (l *Linter) lintUnion(variants []*Schema, path string, result *Result, unionDepth int, unionType string) {
	// Skip nullable patterns (anyOf with null)
	if l.isNullablePattern(variants) {
		return
	}

	// Skip if all variants are $refs (need resolution to verify discriminators)
	if l.allRefs(variants) {
		return
	}

	// Check union size
	if len(variants) > l.config.MaxUnionVariants {
		result.Issues = append(result.Issues, Issue{
			Code:       CodeLargeUnion,
			Severity:   SeverityWarning,
			Path:       path,
			Message:    fmt.Sprintf("Union has %d variants (threshold: %d)", len(variants), l.config.MaxUnionVariants),
			Suggestion: "Consider splitting into smaller, more focused unions",
		})
	}

	// Check nesting depth
	if unionDepth >= l.config.MaxUnionNestingDepth {
		result.Issues = append(result.Issues, Issue{
			Code:       CodeNestedUnion,
			Severity:   SeverityWarning,
			Path:       path,
			Message:    fmt.Sprintf("Union nested %d levels deep (threshold: %d)", unionDepth+1, l.config.MaxUnionNestingDepth),
			Suggestion: "Flatten the union hierarchy for better Go compatibility",
		})
	}

	// Check for discriminator
	discriminator := l.findDiscriminator(variants)
	if discriminator == nil && len(variants) > 1 && !l.isReferencePattern(variants) {
		result.Issues = append(result.Issues, Issue{
			Code:       CodeUnionNoDiscriminator,
			Severity:   SeverityError,
			Path:       path,
			Message:    fmt.Sprintf("%s union has no discriminator field", unionType),
			Suggestion: "Add a const property (e.g., 'type' or 'kind') to each variant with a unique value",
		})
	}

	// If we found a discriminator, verify all variants have it
	if discriminator != nil {
		l.verifyDiscriminator(variants, discriminator, path, result)
	}

	// Check for additionalProperties on union variants
	for i, variant := range variants {
		if variant == nil || variant.Ref != "" {
			continue
		}
		if variant.AdditionalProperties != nil && *variant.AdditionalProperties {
			result.Issues = append(result.Issues, Issue{
				Code:       CodeAdditionalProps,
				Severity:   SeverityWarning,
				Path:       fmt.Sprintf("%s/%d", path, i),
				Message:    "Union variant has additionalProperties: true",
				Suggestion: "Set additionalProperties: false to avoid ambiguous JSON decoding",
			})
		}
	}

	// Recursively lint nested schemas in variants
	for i, variant := range variants {
		if variant != nil && variant.Ref == "" {
			variantPath := fmt.Sprintf("%s/%d", path, i)
			l.lintSchema(variant, variantPath, result, unionDepth+1)
		}
	}
}

// allRefs checks if all variants are $ref references.
func (l *Linter) allRefs(variants []*Schema) bool {
	for _, v := range variants {
		if v == nil {
			continue
		}
		if v.Ref == "" {
			return false
		}
	}
	return true
}

// isNullablePattern checks if this is a simple nullable pattern: anyOf [T, null]
func (l *Linter) isNullablePattern(variants []*Schema) bool {
	if len(variants) != 2 {
		return false
	}
	hasNull := false
	hasType := false
	for _, v := range variants {
		if v == nil {
			continue
		}
		if v.Type == "null" {
			hasNull = true
		} else if v.Type != "" || v.Ref != "" {
			hasType = true
		}
	}
	return hasNull && hasType
}

// isReferencePattern checks if this is a reference pattern: anyOf [ComponentReference, BaseXxx]
func (l *Linter) isReferencePattern(variants []*Schema) bool {
	if len(variants) != 2 {
		return false
	}
	for _, v := range variants {
		if v == nil {
			continue
		}
		// Check if one variant is a reference type (has $component_ref property)
		if prop, ok := v.Properties["$component_ref"]; ok && prop != nil {
			return true
		}
		// Check if it's a $ref to something with "Reference" in the name
		if v.Ref != "" && (contains(v.Ref, "Reference") || contains(v.Ref, "Ref")) {
			return true
		}
	}
	return false
}

// findDiscriminator looks for a common discriminator field across variants.
func (l *Linter) findDiscriminator(variants []*Schema) *discriminatorInfo {
	if len(variants) < 2 {
		return nil
	}

	// Count const values for each potential discriminator field
	candidates := make(map[string]map[string]int) // field -> const value -> count

	for _, fieldName := range l.config.DiscriminatorFields {
		candidates[fieldName] = make(map[string]int)
	}

	resolvedVariants := 0
	for _, variant := range variants {
		if variant == nil || variant.Ref != "" {
			// Skip $ref variants - they need to be resolved
			continue
		}
		resolvedVariants++

		for _, fieldName := range l.config.DiscriminatorFields {
			if prop, ok := variant.Properties[fieldName]; ok && prop != nil {
				if prop.Const != nil {
					if strVal, ok := prop.Const.(string); ok {
						candidates[fieldName][strVal]++
					}
				}
			}
		}
	}

	// Find a field where all resolved variants have unique const values
	for _, fieldName := range l.config.DiscriminatorFields {
		values := candidates[fieldName]
		if len(values) == resolvedVariants && resolvedVariants > 0 {
			// Check all values are unique (count == 1)
			allUnique := true
			for _, count := range values {
				if count != 1 {
					allUnique = false
					break
				}
			}
			if allUnique {
				return &discriminatorInfo{
					fieldName: fieldName,
					values:    values,
				}
			}
		}
	}

	return nil
}

type discriminatorInfo struct {
	fieldName string
	values    map[string]int
}

func (l *Linter) verifyDiscriminator(variants []*Schema, disc *discriminatorInfo, path string, result *Result) {
	seenValues := make(map[string]bool)

	for i, variant := range variants {
		if variant == nil || variant.Ref != "" {
			continue
		}

		prop, ok := variant.Properties[disc.fieldName]
		if !ok || prop == nil {
			result.Issues = append(result.Issues, Issue{
				Code:       CodeMissingConst,
				Severity:   SeverityError,
				Path:       fmt.Sprintf("%s/%d", path, i),
				Message:    fmt.Sprintf("Variant missing discriminator property '%s'", disc.fieldName),
				Suggestion: fmt.Sprintf("Add '%s' property with a const value to this variant", disc.fieldName),
			})
			continue
		}

		if prop.Const == nil {
			result.Issues = append(result.Issues, Issue{
				Code:       CodeMissingConst,
				Severity:   SeverityError,
				Path:       fmt.Sprintf("%s/%d/properties/%s", path, i, disc.fieldName),
				Message:    fmt.Sprintf("Discriminator property '%s' has no const value", disc.fieldName),
				Suggestion: fmt.Sprintf("Add 'const' to the '%s' property with a unique string value", disc.fieldName),
			})
			continue
		}

		strVal, ok := prop.Const.(string)
		if !ok {
			continue
		}

		if seenValues[strVal] {
			result.Issues = append(result.Issues, Issue{
				Code:       CodeDuplicateConstValue,
				Severity:   SeverityError,
				Path:       fmt.Sprintf("%s/%d/properties/%s", path, i, disc.fieldName),
				Message:    fmt.Sprintf("Duplicate discriminator value '%s'", strVal),
				Suggestion: "Each variant must have a unique const value for the discriminator",
			})
		}
		seenValues[strVal] = true
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsImpl(s, substr))
}

func containsImpl(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
