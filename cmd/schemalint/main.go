// Package main provides the schemalint CLI.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/grokify/schemalint/linter"
)

var version = "dev"

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "schemalint",
	Short: "JSON Schema linter and generator for static type compatibility",
	Long: `schemalint validates and generates JSON Schema files for compatibility
with statically-typed languages like Go, Rust, TypeScript, and others.

Commands:
  lint      - Check schemas for type compatibility issues
  generate  - Generate JSON Schema from Go struct types

Profiles:
  default  - Check for common issues (discriminators, large unions)
  scale    - Strict mode for static type generation (no composition keywords)`,
}

var lintCmd = &cobra.Command{
	Use:   "lint <schema.json>",
	Short: "Lint JSON Schema for static type compatibility",
	Long: `Lint a JSON Schema file and report patterns that cause problems
when generating code for statically-typed languages.

Default profile checks:
  - Unions without discriminator fields (error)
  - Inconsistent discriminator field names (error)
  - Missing const values in union variants (error)
  - Large unions with many variants (warning)
  - Deeply nested unions (warning)
  - additionalProperties on union variants (warning)

Scale profile additionally checks:
  - Composition keywords anyOf/oneOf/allOf (error)
  - additionalProperties: true (error)
  - Missing explicit type field (error)
  - Mixed type arrays like ["string", "number"] (error)

Exit codes:
  0 - No issues found
  1 - Errors found (schema has problems)
  2 - Warnings found but no errors`,
	Args: cobra.ExactArgs(1),
	RunE: runLint,
}

var (
	lintOutput       string
	lintProfile      string
	lintPropertyCase string
)

func init() {
	rootCmd.AddCommand(lintCmd)
	rootCmd.AddCommand(versionCmd)

	lintCmd.Flags().StringVarP(&lintOutput, "output", "o", "text", "Output format: text, json, github")
	lintCmd.Flags().StringVarP(&lintProfile, "profile", "p", "default", "Linting profile: default, scale")
	lintCmd.Flags().StringVar(&lintPropertyCase, "property-case", "camelCase", "Property case convention: none, camelCase, snake_case, kebab-case, PascalCase")
}

func runLint(cmd *cobra.Command, args []string) error {
	schemaPath := args[0]

	config := linter.DefaultConfig()
	switch lintProfile {
	case "scale":
		config.Profile = linter.ProfileScale
	case "default":
		config.Profile = linter.ProfileDefault
	default:
		return fmt.Errorf("unknown profile: %s (use 'default' or 'scale')", lintProfile)
	}

	switch lintPropertyCase {
	case "none":
		config.PropertyCase = linter.CaseNone
	case "camelCase":
		config.PropertyCase = linter.CaseCamel
	case "snake_case":
		config.PropertyCase = linter.CaseSnake
	case "kebab-case":
		config.PropertyCase = linter.CaseKebab
	case "PascalCase":
		config.PropertyCase = linter.CasePascal
	default:
		return fmt.Errorf("unknown property case: %s", lintPropertyCase)
	}

	l := linter.New(config)
	result, err := l.LintFile(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to lint schema: %w", err)
	}

	switch lintOutput {
	case "json":
		data, err := result.JSON()
		if err != nil {
			return fmt.Errorf("failed to serialize result: %w", err)
		}
		fmt.Println(string(data))
	case "github":
		fmt.Print(result.GitHubAnnotations())
	default:
		fmt.Print(result.String())
	}

	if result.HasErrors() {
		os.Exit(1)
	}
	if result.WarningCount() > 0 {
		os.Exit(2)
	}

	return nil
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("schemalint version %s\n", version)
	},
}
