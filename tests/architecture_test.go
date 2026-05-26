package tests

//nolint // maintainability: highly cohesive test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/syntheticscale/maintainability-sensors/internal/sensors"
)

func setupArchitectureWorkspace(t *testing.T) (string, string, string) {
	tempDir := t.TempDir()

	archYaml := `
layers:
  api:
    allowed_imports:
      - domain
  domain:
    allowed_imports: []
`
	if err := os.WriteFile(filepath.Join(tempDir, ".sensors-architecture.yml"), []byte(archYaml), 0644); err != nil {
		t.Fatal(err)
	}

	apiDir := filepath.Join(tempDir, "api")
	domainDir := filepath.Join(tempDir, "domain")
	os.Mkdir(apiDir, 0755)
	os.Mkdir(domainDir, 0755)

	return tempDir, apiDir, domainDir
}

func TestGoArchitectureCheck(t *testing.T) {
	_, apiDir, domainDir := setupArchitectureWorkspace(t)

	apiFileValid := filepath.Join(apiDir, "handler.go")
	validContent := `package api
import (
	"fmt"
	"myproject/domain"
)
func Handle() {}
`
	os.WriteFile(apiFileValid, []byte(validContent), 0644)

	domainFileInvalid := filepath.Join(domainDir, "model.go")
	invalidContent := `package domain
import (
	"myproject/api"
)
func Do() {}
`
	os.WriteFile(domainFileInvalid, []byte(invalidContent), 0644)

	plugin := sensors.GoPlugin{}

	resValid, err := plugin.Analyze([]sensors.FileContext{{Path: apiFileValid}})
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	validViolations := resValid[apiFileValid]
	for _, v := range validViolations {
		if v.RuleName == "DependencyBoundary" {
			t.Errorf("Expected no dependency boundary violations for api file, got: %v", v)
		}
	}

	resInvalid, err := plugin.Analyze([]sensors.FileContext{{Path: domainFileInvalid}})
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	invalidViolations := resInvalid[domainFileInvalid]
	foundViolation := false
	for _, v := range invalidViolations {
		if v.RuleName == "DependencyBoundary" {
			foundViolation = true
			if v.Message != "Illegal import: layer 'domain' is not allowed to import layer 'api'" {
				t.Errorf("Unexpected violation message: %s", v.Message)
			}
		}
	}

	if !foundViolation {
		t.Errorf("Expected DependencyBoundary violation for domain file importing api")
	}
}