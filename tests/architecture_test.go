package tests

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
	
	resValid, err := plugin.Analyze([]string{apiFileValid})
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}
	
	validViolations := resValid[apiFileValid]
	for _, v := range validViolations {
		if v.RuleName == "DependencyBoundary" {
			t.Errorf("Expected no dependency boundary violations for api file, got: %v", v)
		}
	}

	resInvalid, err := plugin.Analyze([]string{domainFileInvalid})
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

func TestTypeScriptArchitectureCheck(t *testing.T) {
	_, apiDir, domainDir := setupArchitectureWorkspace(t)

	apiFileValid := filepath.Join(apiDir, "handler.ts")
	validContent := `import { Something } from "myproject/domain";
const req = require("myproject/domain/other");
function handle() {}
`
	os.WriteFile(apiFileValid, []byte(validContent), 0644)

	domainFileInvalid := filepath.Join(domainDir, "model.ts")
	invalidContent := `import { Controller } from "myproject/api";
const api = require("myproject/api/handler");
function do() {}
`
	os.WriteFile(domainFileInvalid, []byte(invalidContent), 0644)

	plugin := sensors.TypeScriptTreeSitterPlugin{}
	
	resValid, err := plugin.Analyze([]string{apiFileValid})
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}
	
	for _, v := range resValid[apiFileValid] {
		if v.RuleName == "DependencyBoundary" {
			t.Errorf("Expected no dependency boundary violations for api file, got: %v", v)
		}
	}

	resInvalid, err := plugin.Analyze([]string{domainFileInvalid})
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	invalidViolations := resInvalid[domainFileInvalid]
	violationCount := 0
	for _, v := range invalidViolations {
		if v.RuleName == "DependencyBoundary" {
			violationCount++
			if v.Message != "Illegal import: layer 'domain' is not allowed to import layer 'api'" {
				t.Errorf("Unexpected violation message: %s", v.Message)
			}
		}
	}
	
	if violationCount != 2 { // One for import, one for require
		t.Errorf("Expected 2 DependencyBoundary violations for domain file importing api, got %d", violationCount)
	}
}

func TestPythonArchitectureCheck(t *testing.T) {
	_, apiDir, domainDir := setupArchitectureWorkspace(t)

	apiFileValid := filepath.Join(apiDir, "handler.py")
	validContent := `import myproject.domain
from myproject.domain import Something
def handle(): pass
`
	os.WriteFile(apiFileValid, []byte(validContent), 0644)

	domainFileInvalid := filepath.Join(domainDir, "model.py")
	invalidContent := `import myproject.api.controller
from myproject.api import handler
def do(): pass
`
	os.WriteFile(domainFileInvalid, []byte(invalidContent), 0644)

	plugin := sensors.PythonTreeSitterPlugin{}
	
	resValid, err := plugin.Analyze([]string{apiFileValid})
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}
	
	for _, v := range resValid[apiFileValid] {
		if v.RuleName == "DependencyBoundary" {
			t.Errorf("Expected no dependency boundary violations for api file, got: %v", v)
		}
	}

	resInvalid, err := plugin.Analyze([]string{domainFileInvalid})
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	invalidViolations := resInvalid[domainFileInvalid]
	violationCount := 0
	for _, v := range invalidViolations {
		if v.RuleName == "DependencyBoundary" {
			violationCount++
			if v.Message != "Illegal import: layer 'domain' is not allowed to import layer 'api'" {
				t.Errorf("Unexpected violation message: %s", v.Message)
			}
		}
	}
	
	if violationCount != 2 { // One for import, one for from import
		t.Errorf("Expected 2 DependencyBoundary violations for domain file importing api, got %d", violationCount)
	}
}
