package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/syntheticscale/maintainability-sensors/internal/sensors"
)

func TestGoArchitectureCheckFalsePositive(t *testing.T) {
	tempDir := t.TempDir()

	archYaml := `
layers:
  user:
    allowed_imports: []
  utils:
    allowed_imports:
      - user
`
	os.WriteFile(filepath.Join(tempDir, ".sensors-architecture.yml"), []byte(archYaml), 0644)

	utilsUserDir := filepath.Join(tempDir, "src", "utils", "user")
	os.MkdirAll(utilsUserDir, 0755)
	
	utilsUserFile := filepath.Join(utilsUserDir, "auth.go")
	os.WriteFile(utilsUserFile, []byte(`package user
import "myproject/src/utils"
`), 0644)

	utilsDir := filepath.Join(tempDir, "src", "utils")
	os.MkdirAll(utilsDir, 0755)
	utilsFile := filepath.Join(utilsDir, "utils.go")
	os.WriteFile(utilsFile, []byte(`package utils
`), 0644)

	config, _ := sensors.ParseArchitectureConfig(filepath.Join(tempDir, ".sensors-architecture.yml"))

	// We want to test CheckGoArchitecture directly
	violations, err := sensors.CheckGoArchitecture(utilsUserFile, config)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	for _, v := range violations {
		if v.RuleName == "DependencyBoundary" {
			t.Logf("Violation: %v", v)
			// If it matched "user", it would have a violation for importing "utils"
			t.Errorf("False positive matched!")
		}
	}
}
