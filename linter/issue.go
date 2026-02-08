// Package linter provides JSON Schema linting for Go compatibility.
package linter

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Severity indicates the severity of a lint issue.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

// IssueCode identifies a specific type of lint issue.
type IssueCode string

const (
	// Errors - these will cause problems in generated Go code
	CodeUnionNoDiscriminator      IssueCode = "union-no-discriminator"
	CodeInconsistentDiscriminator IssueCode = "inconsistent-discriminator"
	CodeMissingConst              IssueCode = "missing-const"
	CodeDuplicateConstValue       IssueCode = "duplicate-const-value"

	// Warnings - these may cause issues or indicate suboptimal patterns
	CodeLargeUnion        IssueCode = "large-union"
	CodeNestedUnion       IssueCode = "nested-union"
	CodeAdditionalProps   IssueCode = "additional-properties"
	CodeAmbiguousUnion    IssueCode = "ambiguous-union"
	CodeCircularReference IssueCode = "circular-reference"

	// Scale profile errors - strict rules for static type compatibility
	CodeCompositionDisallowed     IssueCode = "composition-disallowed"
	CodeAdditionalPropsDisallowed IssueCode = "additional-properties-disallowed"
	CodeMissingType               IssueCode = "missing-type"
	CodeMixedTypeDisallowed       IssueCode = "mixed-type-disallowed"
)

// Issue represents a single lint issue found in a schema.
type Issue struct {
	Code       IssueCode `json:"code"`
	Severity   Severity  `json:"severity"`
	Path       string    `json:"path"`
	Message    string    `json:"message"`
	Suggestion string    `json:"suggestion,omitempty"`
	TypeName   string    `json:"type_name,omitempty"`
}

// String returns a human-readable representation of the issue.
func (i Issue) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("[%s] %s: %s", i.Severity, i.Path, i.Message))
	if i.Suggestion != "" {
		sb.WriteString(fmt.Sprintf("\n  suggestion: %s", i.Suggestion))
	}
	return sb.String()
}

// Result contains all issues found during linting.
type Result struct {
	SchemaPath string  `json:"schema_path"`
	Issues     []Issue `json:"issues"`
}

// ErrorCount returns the number of error-severity issues.
func (r Result) ErrorCount() int {
	count := 0
	for _, issue := range r.Issues {
		if issue.Severity == SeverityError {
			count++
		}
	}
	return count
}

// WarningCount returns the number of warning-severity issues.
func (r Result) WarningCount() int {
	count := 0
	for _, issue := range r.Issues {
		if issue.Severity == SeverityWarning {
			count++
		}
	}
	return count
}

// HasErrors returns true if there are any error-severity issues.
func (r Result) HasErrors() bool {
	return r.ErrorCount() > 0
}

// JSON returns the result as JSON.
func (r Result) JSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

// String returns a human-readable summary.
func (r Result) String() string {
	var sb strings.Builder

	errors := r.ErrorCount()
	warnings := r.WarningCount()

	if len(r.Issues) == 0 {
		sb.WriteString("✅ No issues found\n")
		return sb.String()
	}

	for _, issue := range r.Issues {
		sb.WriteString(issue.String())
		sb.WriteString("\n")
	}

	sb.WriteString(fmt.Sprintf("\nSummary: %d error(s), %d warning(s)\n", errors, warnings))

	return sb.String()
}

// GitHubAnnotations returns issues formatted as GitHub Actions annotations.
func (r Result) GitHubAnnotations() string {
	var sb strings.Builder
	for _, issue := range r.Issues {
		// Format: ::{level} file={path}::{message}
		level := "warning"
		if issue.Severity == SeverityError {
			level = "error"
		}
		sb.WriteString(fmt.Sprintf("::%s file=%s::%s - %s\n",
			level, r.SchemaPath, issue.Code, issue.Message))
	}
	return sb.String()
}
