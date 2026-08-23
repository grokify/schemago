package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

func TestCheckSchemaDrift(t *testing.T) {
	dir := t.TempDir()
	cmd := &cobra.Command{}

	match := filepath.Join(dir, "match.json")
	if err := os.WriteFile(match, []byte("{\"a\":1}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	drift := filepath.Join(dir, "drift.json")
	if err := os.WriteFile(drift, []byte("{\"a\":2}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(dir, "missing.json")

	// Matching content (ignoring trailing newline) is not drift.
	if err := checkSchemaDrift(cmd, match, "{\"a\":1}"); err != nil {
		t.Errorf("matching content should not report drift: %v", err)
	}

	// Differing content is drift.
	if err := checkSchemaDrift(cmd, drift, "{\"a\":1}"); err == nil {
		t.Error("differing content should report drift")
	}

	// A missing file is drift (must be generated first).
	if err := checkSchemaDrift(cmd, missing, "{\"a\":1}"); err == nil {
		t.Error("missing file should report drift")
	}
}
