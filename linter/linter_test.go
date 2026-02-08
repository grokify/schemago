package linter

import (
	"testing"
)

func TestLintNullablePattern(t *testing.T) {
	schema := `{
		"$defs": {
			"NullableString": {
				"anyOf": [
					{"type": "string"},
					{"type": "null"}
				]
			}
		}
	}`

	l := NewWithDefaults()
	result, err := l.Lint([]byte(schema))
	if err != nil {
		t.Fatalf("Failed to lint: %v", err)
	}

	if len(result.Issues) != 0 {
		t.Errorf("Expected no issues for nullable pattern, got %d: %v", len(result.Issues), result.Issues)
	}
}

func TestLintUnionWithDiscriminator(t *testing.T) {
	schema := `{
		"$defs": {
			"Animal": {
				"anyOf": [
					{
						"type": "object",
						"properties": {
							"type": {"const": "dog"},
							"name": {"type": "string"}
						}
					},
					{
						"type": "object",
						"properties": {
							"type": {"const": "cat"},
							"name": {"type": "string"}
						}
					}
				]
			}
		}
	}`

	l := NewWithDefaults()
	result, err := l.Lint([]byte(schema))
	if err != nil {
		t.Fatalf("Failed to lint: %v", err)
	}

	if result.HasErrors() {
		t.Errorf("Expected no errors for union with discriminator, got: %v", result.Issues)
	}
}

func TestLintUnionWithoutDiscriminator(t *testing.T) {
	schema := `{
		"$defs": {
			"BadUnion": {
				"anyOf": [
					{
						"type": "object",
						"properties": {
							"name": {"type": "string"}
						}
					},
					{
						"type": "object",
						"properties": {
							"title": {"type": "string"}
						}
					}
				]
			}
		}
	}`

	l := NewWithDefaults()
	result, err := l.Lint([]byte(schema))
	if err != nil {
		t.Fatalf("Failed to lint: %v", err)
	}

	if !result.HasErrors() {
		t.Error("Expected error for union without discriminator")
	}

	found := false
	for _, issue := range result.Issues {
		if issue.Code == CodeUnionNoDiscriminator {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected union-no-discriminator error")
	}
}

func TestLintLargeUnion(t *testing.T) {
	schema := `{
		"$defs": {
			"LargeUnion": {
				"oneOf": [
					{"type": "object", "properties": {"type": {"const": "v1"}}},
					{"type": "object", "properties": {"type": {"const": "v2"}}},
					{"type": "object", "properties": {"type": {"const": "v3"}}},
					{"type": "object", "properties": {"type": {"const": "v4"}}},
					{"type": "object", "properties": {"type": {"const": "v5"}}},
					{"type": "object", "properties": {"type": {"const": "v6"}}},
					{"type": "object", "properties": {"type": {"const": "v7"}}},
					{"type": "object", "properties": {"type": {"const": "v8"}}},
					{"type": "object", "properties": {"type": {"const": "v9"}}},
					{"type": "object", "properties": {"type": {"const": "v10"}}},
					{"type": "object", "properties": {"type": {"const": "v11"}}}
				]
			}
		}
	}`

	l := NewWithDefaults()
	result, err := l.Lint([]byte(schema))
	if err != nil {
		t.Fatalf("Failed to lint: %v", err)
	}

	found := false
	for _, issue := range result.Issues {
		if issue.Code == CodeLargeUnion && issue.Severity == SeverityWarning {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected large-union warning")
	}
}

func TestLintAdditionalProperties(t *testing.T) {
	schema := `{
		"$defs": {
			"OpenUnion": {
				"anyOf": [
					{
						"type": "object",
						"properties": {
							"type": {"const": "open"}
						},
						"additionalProperties": true
					},
					{
						"type": "object",
						"properties": {
							"type": {"const": "closed"}
						},
						"additionalProperties": false
					}
				]
			}
		}
	}`

	l := NewWithDefaults()
	result, err := l.Lint([]byte(schema))
	if err != nil {
		t.Fatalf("Failed to lint: %v", err)
	}

	found := false
	for _, issue := range result.Issues {
		if issue.Code == CodeAdditionalProps {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected additional-properties warning")
	}
}

func TestLintAllRefs(t *testing.T) {
	schema := `{
		"$defs": {
			"Animal": {
				"anyOf": [
					{"$ref": "#/$defs/Dog"},
					{"$ref": "#/$defs/Cat"}
				]
			},
			"Dog": {
				"type": "object",
				"properties": {
					"type": {"const": "dog"}
				}
			},
			"Cat": {
				"type": "object",
				"properties": {
					"type": {"const": "cat"}
				}
			}
		}
	}`

	l := NewWithDefaults()
	result, err := l.Lint([]byte(schema))
	if err != nil {
		t.Fatalf("Failed to lint: %v", err)
	}

	// Unions with all $refs should be skipped (no error for Animal)
	for _, issue := range result.Issues {
		if issue.Path == "$/$defs/Animal/anyOf" && issue.Code == CodeUnionNoDiscriminator {
			t.Error("Should not report error for all-refs union")
		}
	}
}

func TestResultCounts(t *testing.T) {
	result := Result{
		Issues: []Issue{
			{Severity: SeverityError},
			{Severity: SeverityError},
			{Severity: SeverityWarning},
			{Severity: SeverityInfo},
		},
	}

	if result.ErrorCount() != 2 {
		t.Errorf("Expected 2 errors, got %d", result.ErrorCount())
	}
	if result.WarningCount() != 1 {
		t.Errorf("Expected 1 warning, got %d", result.WarningCount())
	}
	if !result.HasErrors() {
		t.Error("Expected HasErrors to be true")
	}
}

func TestScaleProfileDisallowsAnyOf(t *testing.T) {
	schema := `{
		"$defs": {
			"Animal": {
				"anyOf": [
					{"type": "object", "properties": {"type": {"const": "dog"}}},
					{"type": "object", "properties": {"type": {"const": "cat"}}}
				]
			}
		}
	}`

	config := DefaultConfig()
	config.Profile = ProfileScale
	l := New(config)

	result, err := l.Lint([]byte(schema))
	if err != nil {
		t.Fatalf("Failed to lint: %v", err)
	}

	found := false
	for _, issue := range result.Issues {
		if issue.Code == CodeCompositionDisallowed && issue.Severity == SeverityError {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected composition-disallowed error for anyOf")
	}
}

func TestScaleProfileDisallowsOneOf(t *testing.T) {
	schema := `{
		"$defs": {
			"Animal": {
				"oneOf": [
					{"type": "object", "properties": {"type": {"const": "dog"}}},
					{"type": "object", "properties": {"type": {"const": "cat"}}}
				]
			}
		}
	}`

	config := DefaultConfig()
	config.Profile = ProfileScale
	l := New(config)

	result, err := l.Lint([]byte(schema))
	if err != nil {
		t.Fatalf("Failed to lint: %v", err)
	}

	found := false
	for _, issue := range result.Issues {
		if issue.Code == CodeCompositionDisallowed && issue.Severity == SeverityError {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected composition-disallowed error for oneOf")
	}
}

func TestScaleProfileDisallowsAllOf(t *testing.T) {
	schema := `{
		"$defs": {
			"Combined": {
				"allOf": [
					{"type": "object", "properties": {"name": {"type": "string"}}},
					{"type": "object", "properties": {"age": {"type": "integer"}}}
				]
			}
		}
	}`

	config := DefaultConfig()
	config.Profile = ProfileScale
	l := New(config)

	result, err := l.Lint([]byte(schema))
	if err != nil {
		t.Fatalf("Failed to lint: %v", err)
	}

	found := false
	for _, issue := range result.Issues {
		if issue.Code == CodeCompositionDisallowed && issue.Severity == SeverityError {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected composition-disallowed error for allOf")
	}
}

func TestScaleProfileDisallowsAdditionalProperties(t *testing.T) {
	schema := `{
		"type": "object",
		"properties": {
			"name": {"type": "string"}
		},
		"additionalProperties": true
	}`

	config := DefaultConfig()
	config.Profile = ProfileScale
	l := New(config)

	result, err := l.Lint([]byte(schema))
	if err != nil {
		t.Fatalf("Failed to lint: %v", err)
	}

	found := false
	for _, issue := range result.Issues {
		if issue.Code == CodeAdditionalPropsDisallowed && issue.Severity == SeverityError {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected additional-properties-disallowed error")
	}
}

func TestScaleProfileRequiresType(t *testing.T) {
	schema := `{
		"$defs": {
			"Person": {
				"properties": {
					"name": {"type": "string"}
				}
			}
		}
	}`

	config := DefaultConfig()
	config.Profile = ProfileScale
	l := New(config)

	result, err := l.Lint([]byte(schema))
	if err != nil {
		t.Fatalf("Failed to lint: %v", err)
	}

	found := false
	for _, issue := range result.Issues {
		if issue.Code == CodeMissingType && issue.Severity == SeverityError {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected missing-type error")
	}
}

func TestScaleProfileDisallowsMixedTypes(t *testing.T) {
	schema := `{
		"$defs": {
			"StringOrNumber": {
				"type": ["string", "number"]
			}
		}
	}`

	config := DefaultConfig()
	config.Profile = ProfileScale
	l := New(config)

	result, err := l.Lint([]byte(schema))
	if err != nil {
		t.Fatalf("Failed to lint: %v", err)
	}

	found := false
	for _, issue := range result.Issues {
		if issue.Code == CodeMixedTypeDisallowed && issue.Severity == SeverityError {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected mixed-type-disallowed error")
	}
}

func TestDefaultProfileAllowsComposition(t *testing.T) {
	schema := `{
		"$defs": {
			"Animal": {
				"anyOf": [
					{
						"type": "object",
						"properties": {"type": {"const": "dog"}}
					},
					{
						"type": "object",
						"properties": {"type": {"const": "cat"}}
					}
				]
			}
		}
	}`

	l := NewWithDefaults()
	result, err := l.Lint([]byte(schema))
	if err != nil {
		t.Fatalf("Failed to lint: %v", err)
	}

	for _, issue := range result.Issues {
		if issue.Code == CodeCompositionDisallowed {
			t.Error("Default profile should not report composition-disallowed")
		}
	}
}

func TestScaleProfileValidSchema(t *testing.T) {
	schema := `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "integer"}
		},
		"additionalProperties": false
	}`

	config := DefaultConfig()
	config.Profile = ProfileScale
	l := New(config)

	result, err := l.Lint([]byte(schema))
	if err != nil {
		t.Fatalf("Failed to lint: %v", err)
	}

	if result.HasErrors() {
		t.Errorf("Expected no errors for valid scale profile schema, got: %v", result.Issues)
	}
}
