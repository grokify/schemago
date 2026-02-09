package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
)

var (
	genOutput string
	genIndent bool
)

func init() {
	rootCmd.AddCommand(generateCmd)

	generateCmd.Flags().StringVarP(&genOutput, "output", "o", "", "Output file (default: stdout)")
	generateCmd.Flags().BoolVar(&genIndent, "indent", true, "Indent JSON output")
}

var generateCmd = &cobra.Command{
	Use:   "generate <package> <type>",
	Short: "Generate JSON Schema from Go struct type",
	Long: `Generate a JSON Schema from a Go struct type using reflection.

This command creates a temporary Go program that imports your type and
uses github.com/invopop/jsonschema to generate the schema.

Examples:
  # Generate schema for TaskList type from structured-tasks
  schemalint generate github.com/grokify/structured-tasks/tasks TaskList

  # Generate and save to file
  schemalint generate -o schema.json github.com/myorg/myproject/types Config

  # Generate without indentation
  schemalint generate --indent=false github.com/myorg/myproject/types Config

Notes:
  - The package must be importable (available locally or via go get)
  - The type must be exported (start with uppercase)
  - Uses struct tags: json, jsonschema, title, description, etc.`,
	Args: cobra.ExactArgs(2),
	RunE: runGenerate,
}

const genTemplate = `//go:build ignore

package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/invopop/jsonschema"
	target "{{.Package}}"
)

func main() {
	r := jsonschema.Reflector{
		DoNotReference: false,
		ExpandedStruct: false,
	}
	schema := r.Reflect(&target.{{.Type}}{})
	{{if .Indent}}
	data, err := json.MarshalIndent(schema, "", "  ")
	{{else}}
	data, err := json.Marshal(schema)
	{{end}}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error marshaling schema: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(data))
}
`

func runGenerate(cmd *cobra.Command, args []string) error {
	pkgPath := args[0]
	typeName := args[1]

	// Validate type name starts with uppercase (exported)
	if len(typeName) == 0 || typeName[0] < 'A' || typeName[0] > 'Z' {
		return fmt.Errorf("type name must be exported (start with uppercase): %s", typeName)
	}

	// Find the module root and module name
	modRoot, modName := findModule(pkgPath)

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "schemalint-gen-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Generate the temporary program
	tmpl, err := template.New("gen").Parse(genTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, map[string]any{
		"Package": pkgPath,
		"Type":    typeName,
		"Indent":  genIndent,
	})
	if err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	// Write the temporary program
	genFile := filepath.Join(tmpDir, "gen.go")
	if err := os.WriteFile(genFile, buf.Bytes(), 0600); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Helper function to run go commands and check for errors
	goCmd := func(args ...string) error {
		c := exec.Command("go", args...)
		c.Dir = tmpDir
		c.Env = append(os.Environ(), "GO111MODULE=on")
		var stderr bytes.Buffer
		c.Stderr = &stderr
		if err := c.Run(); err != nil {
			return fmt.Errorf("go %v failed: %w\n%s", args, err, stderr.String())
		}
		return nil
	}

	// Initialize the go module
	if err := goCmd("mod", "init", "schemalint-gen"); err != nil {
		return err
	}

	// Fetch jsonschema dependency first (before any replace directives)
	if err := goCmd("get", "github.com/invopop/jsonschema@latest"); err != nil {
		return err
	}

	// Fetch the target module
	if modRoot != "" {
		// For local modules: add replace directive, then get the module
		if err := goCmd("mod", "edit", "-replace", modName+"="+modRoot); err != nil {
			return err
		}
		// Use go get with the package path to add the require
		if err := goCmd("get", pkgPath); err != nil {
			return err
		}
	} else {
		// Remote module - fetch from network
		if err := goCmd("get", modName+"@latest"); err != nil {
			return err
		}
	}

	// Note: We skip `go mod tidy` because gen.go has //go:build ignore
	// which causes tidy to remove all requires since it sees no imports.

	// Run the generator
	genCmd := exec.Command("go", "run", "gen.go")
	genCmd.Dir = tmpDir
	genCmd.Env = append(os.Environ(), "GO111MODULE=on")
	var stdout, stderr bytes.Buffer
	genCmd.Stdout = &stdout
	genCmd.Stderr = &stderr

	if err := genCmd.Run(); err != nil {
		return fmt.Errorf("failed to generate schema: %w\n%s", err, stderr.String())
	}

	// Output result
	output := strings.TrimSpace(stdout.String())
	if genOutput != "" {
		if err := os.WriteFile(genOutput, []byte(output+"\n"), 0600); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "Generated %s\n", genOutput)
	} else {
		fmt.Println(output)
	}

	return nil
}

// findModule finds the module root directory and module name for a package path.
// Returns (moduleRoot, moduleName).
// If the module is found locally, moduleRoot is the filesystem path.
// If not found locally, moduleRoot is empty and moduleName is set for remote fetch.
func findModule(pkgPath string) (string, string) {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		home, _ := os.UserHomeDir()
		gopath = filepath.Join(home, "go")
	}

	// Split package path into parts
	parts := strings.Split(pkgPath, "/")

	// Try progressively shorter paths to find the module root
	for i := len(parts); i > 0; i-- {
		candidate := filepath.Join(parts[:i]...)
		candidatePath := filepath.Join(gopath, "src", candidate)
		goModPath := filepath.Join(candidatePath, "go.mod")

		if _, err := os.Stat(goModPath); err == nil {
			// Found go.mod - read the module name
			content, err := os.ReadFile(goModPath)
			if err != nil {
				continue
			}
			modName := parseModuleName(string(content))
			if modName != "" {
				return candidatePath, modName
			}
		}
	}

	// Not found locally - assume it's the package path itself (remote module)
	// Try to guess the module name (first 3 parts for github.com/org/repo pattern)
	if len(parts) >= 3 && (parts[0] == "github.com" || parts[0] == "gitlab.com" || parts[0] == "bitbucket.org") {
		return "", strings.Join(parts[:3], "/")
	}

	return "", pkgPath
}

// parseModuleName extracts the module name from go.mod content.
func parseModuleName(content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}
