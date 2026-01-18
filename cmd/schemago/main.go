// Package main provides the schemago CLI.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/grokify/schemago/linter"
)

var version = "dev"

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "schemago",
	Short: "JSON Schema to Go code generator with union type support",
	Long: `schemago is a JSON Schema to Go code generator that correctly handles
union types (anyOf/oneOf) by generating proper tagged unions with
discriminator-based unmarshalling.

Use 'schemago lint' to check schemas for Go compatibility issues.
Use 'schemago generate' to generate Go code from schemas.`,
}

var lintCmd = &cobra.Command{
	Use:   "lint <schema.json>",
	Short: "Lint JSON Schema for Go compatibility issues",
	Long: `Lint a JSON Schema file and report any patterns that would cause
problems when generating Go code.

Issues checked include:
  - Unions without discriminator fields (error)
  - Inconsistent discriminator field names (error)
  - Missing const values in union variants (error)
  - Large unions with many variants (warning)
  - Deeply nested unions (warning)
  - additionalProperties on union variants (warning)

Exit codes:
  0 - No issues found
  1 - Errors found (schema has problems)
  2 - Warnings found but no errors`,
	Args: cobra.ExactArgs(1),
	RunE: runLint,
}

var (
	lintOutput string
)

func init() {
	rootCmd.AddCommand(lintCmd)
	rootCmd.AddCommand(versionCmd)

	lintCmd.Flags().StringVarP(&lintOutput, "output", "o", "text", "Output format: text, json, github")
}

func runLint(cmd *cobra.Command, args []string) error {
	schemaPath := args[0]

	l := linter.NewWithDefaults()
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
		fmt.Printf("schemago version %s\n", version)
	},
}
