package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/paulolai/maintainability-sensors/sensors"
)

func TestBootstrapRepo_AutoDetect(t *testing.T) {
	// Create a temporary directory structure mimicking a Python repository
	tempDir := t.TempDir()
	
	pyFile := filepath.Join(tempDir, "app.py")
	if err := os.WriteFile(pyFile, []byte("print('hello')\n"), 0644); err != nil {
		t.Fatalf("failed to write mock python file: %v", err)
	}

	// Run Bootstrap on it
	err := sensors.BootstrapRepo(tempDir)
	if err != nil {
		t.Fatalf("BootstrapRepo failed on mock python repo: %v", err)
	}

	// Verify .pylintrc was created
	pylintPath := filepath.Join(tempDir, ".pylintrc")
	data, err := os.ReadFile(pylintPath)
	if err != nil {
		t.Errorf("expected .pylintrc to be created, but got error: %v", err)
	}

	if !strings.Contains(string(data), "max-complexity=8") {
		t.Errorf("created .pylintrc missing max-complexity rule. Content:\n%s", string(data))
	}
}

func TestBootstrapRepo_DoNotOverwrite(t *testing.T) {
	// Create temporary directory mimicking a TypeScript repo with existing config
	tempDir := t.TempDir()
	
	tsFile := filepath.Join(tempDir, "index.ts")
	if err := os.WriteFile(tsFile, []byte("const x: number = 5;\n"), 0644); err != nil {
		t.Fatalf("failed to write mock ts file: %v", err)
	}

	eslintPath := filepath.Join(tempDir, ".eslintrc.json")
	originalConfig := `{"rules": {"semi": "off"}}`
	if err := os.WriteFile(eslintPath, []byte(originalConfig), 0644); err != nil {
		t.Fatalf("failed to write pre-existing config: %v", err)
	}

	// Run Bootstrap
	err := sensors.BootstrapRepo(tempDir)
	if err != nil {
		t.Fatalf("BootstrapRepo failed on mock TS repo: %v", err)
	}

	// Verify .eslintrc.json was NOT overwritten
	data, err := os.ReadFile(eslintPath)
	if err != nil {
		t.Fatalf("failed to read .eslintrc.json: %v", err)
	}

	if string(data) != originalConfig {
		t.Errorf("safety guardrail failed: original config was overwritten. Got:\n%s\nExpected:\n%s", string(data), originalConfig)
	}
}
