package sensors

//nolint // maintainability: highly cohesive test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBootstrapRepoWithPolicy_CreatesConfig(t *testing.T) {
	tempDir := t.TempDir()
	// Create a dummy file to satisfy language detection
	os.WriteFile(filepath.Join(tempDir, "main.go"), []byte("package main\n"), 0644)

	err := BootstrapRepoWithPolicy(tempDir, true)
	if err != nil {
		t.Fatalf("BootstrapRepoWithPolicy failed: %v", err)
	}

	configPath := filepath.Join(tempDir, ".maintainability-sensors.yml")
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("expected .maintainability-sensors.yml to be created: %v", err)
	}

	if !strings.Contains(string(content), "default-severity: warn") {
		t.Errorf("expected config to contain 'default-severity: warn', got:\n%s", string(content))
	}
}

func TestBootstrapRepoWithPolicy_NoPolicy_DoesNotCreateConfig(t *testing.T) {
	tempDir := t.TempDir()
	os.WriteFile(filepath.Join(tempDir, "main.go"), []byte("package main\n"), 0644)

	err := BootstrapRepoWithPolicy(tempDir, false)
	if err != nil {
		t.Fatalf("BootstrapRepoWithPolicy failed: %v", err)
	}

	configPath := filepath.Join(tempDir, ".maintainability-sensors.yml")
	if _, err := os.Stat(configPath); err == nil {
		t.Error("expected .maintainability-sensors.yml NOT to be created when warnPolicy is false")
	}
}

func TestBootstrapRepoWithPolicy_SkipExistingConfig(t *testing.T) {
	tempDir := t.TempDir()
	os.WriteFile(filepath.Join(tempDir, "main.go"), []byte("package main\n"), 0644)

	configPath := filepath.Join(tempDir, ".maintainability-sensors.yml")
	originalContent := []byte("existing: config\n")
	if err := os.WriteFile(configPath, originalContent, 0644); err != nil {
		t.Fatalf("failed to write existing config: %v", err)
	}

	err := BootstrapRepoWithPolicy(tempDir, true)
	if err != nil {
		t.Fatalf("BootstrapRepoWithPolicy failed: %v", err)
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read existing config: %v", err)
	}

	if string(content) != string(originalContent) {
		t.Errorf("existing .maintainability-sensors.yml was overwritten! expected:\n%s\ngot:\n%s", string(originalContent), string(content))
	}
}

func TestBootstrapRepo_BackwardsCompatibility(t *testing.T) {
	tempDir := t.TempDir()
	os.WriteFile(filepath.Join(tempDir, "main.go"), []byte("package main\n"), 0644)

	err := BootstrapRepo(tempDir)
	if err != nil {
		t.Fatalf("BootstrapRepo failed: %v", err)
	}

	configPath := filepath.Join(tempDir, ".maintainability-sensors.yml")
	if _, err := os.Stat(configPath); err == nil {
		t.Error("BootstrapRepo (backwards compat) should NOT create .maintainability-sensors.yml")
	}

	// Ensure language config was still created
	gociPath := filepath.Join(tempDir, ".golangci.yml")
	if _, err := os.Stat(gociPath); err != nil {
		t.Errorf("expected .golangci.yml to be created: %v", err)
	}
}
