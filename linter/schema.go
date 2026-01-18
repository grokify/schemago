package linter

import (
	"encoding/json"
)

// Schema represents a JSON Schema document or subschema.
// This is a simplified representation focused on the fields needed for linting.
type Schema struct {
	// Core
	Schema      string             `json:"$schema,omitempty"`
	ID          string             `json:"$id,omitempty"`
	Ref         string             `json:"$ref,omitempty"`
	Defs        map[string]*Schema `json:"$defs,omitempty"`
	Definitions map[string]*Schema `json:"definitions,omitempty"`

	// Type
	Type string `json:"type,omitempty"`

	// Composition
	AnyOf []*Schema `json:"anyOf,omitempty"`
	OneOf []*Schema `json:"oneOf,omitempty"`
	AllOf []*Schema `json:"allOf,omitempty"`

	// Object
	Properties                 map[string]*Schema `json:"properties,omitempty"`
	Required                   []string           `json:"required,omitempty"`
	AdditionalProperties       *bool              `json:"-"` // Handled specially
	AdditionalPropertiesSchema *Schema            `json:"-"` // Handled specially

	// Array
	Items *Schema `json:"items,omitempty"`

	// Validation
	Const any   `json:"const,omitempty"`
	Enum  []any `json:"enum,omitempty"`

	// Metadata
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Default     any    `json:"default,omitempty"`

	// Extension
	XAbstractComponent *bool `json:"x-abstract-component,omitempty"`
}

// UnmarshalJSON implements custom unmarshalling to handle additionalProperties.
func (s *Schema) UnmarshalJSON(data []byte) error {
	// Use an alias to avoid infinite recursion
	type schemaAlias Schema
	var alias schemaAlias

	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}

	*s = Schema(alias)

	// Handle additionalProperties which can be bool or schema
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	if apRaw, ok := raw["additionalProperties"]; ok {
		// Try as bool first
		var boolVal bool
		if err := json.Unmarshal(apRaw, &boolVal); err == nil {
			s.AdditionalProperties = &boolVal
		} else {
			// Try as schema
			var schemaVal Schema
			if err := json.Unmarshal(apRaw, &schemaVal); err == nil {
				s.AdditionalPropertiesSchema = &schemaVal
				// If it's a schema, we treat it as allowing additional properties
				trueVal := true
				s.AdditionalProperties = &trueVal
			}
		}
	}

	return nil
}

// IsObject returns true if this schema describes an object type.
func (s *Schema) IsObject() bool {
	return s.Type == "object" || len(s.Properties) > 0
}

// IsArray returns true if this schema describes an array type.
func (s *Schema) IsArray() bool {
	return s.Type == "array" || s.Items != nil
}

// IsUnion returns true if this schema is a union type (anyOf or oneOf).
func (s *Schema) IsUnion() bool {
	return len(s.AnyOf) > 0 || len(s.OneOf) > 0
}

// IsRef returns true if this schema is a reference.
func (s *Schema) IsRef() bool {
	return s.Ref != ""
}

// GetUnionVariants returns the union variants (anyOf takes precedence over oneOf).
func (s *Schema) GetUnionVariants() []*Schema {
	if len(s.AnyOf) > 0 {
		return s.AnyOf
	}
	return s.OneOf
}
