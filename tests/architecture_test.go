package tests

//nolint // maintainability: highly cohesive test

import (
	"os"
	"path/filepath"
	"sync"
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
	defer sensors.ResetArchConfigCache()

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

func TestArchConfigCacheHit(t *testing.T) {
	defer sensors.ResetArchConfigCache()

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

	domainDir := filepath.Join(tempDir, "domain")
	os.Mkdir(domainDir, 0755)

	domainFile := filepath.Join(domainDir, "model.go")
	content := `package domain
import "myproject/api"
func Do() {}
`
	os.WriteFile(domainFile, []byte(content), 0644)

	plugin := sensors.GoPlugin{}

	res1, err := plugin.Analyze([]sensors.FileContext{{Path: domainFile}})
	if err != nil {
		t.Fatalf("Analyze (first call) failed: %v", err)
	}

	res2, err := plugin.Analyze([]sensors.FileContext{{Path: domainFile}})
	if err != nil {
		t.Fatalf("Analyze (second call) failed: %v", err)
	}

	violations1 := res1[domainFile]
	violations2 := res2[domainFile]

	if len(violations1) != len(violations2) {
		t.Fatalf("Expected stable violation count across cache hits, got %d then %d", len(violations1), len(violations2))
	}

	for i := range violations1 {
		if violations1[i].RuleName != violations2[i].RuleName {
			t.Errorf("RuleName mismatch at index %d: %q vs %q", i, violations1[i].RuleName, violations2[i].RuleName)
		}
		if violations1[i].Message != violations2[i].Message {
			t.Errorf("Message mismatch at index %d: %q vs %q", i, violations1[i].Message, violations2[i].Message)
		}
	}

	found := false
	for _, v := range violations2 {
		if v.RuleName == "DependencyBoundary" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected DependencyBoundary violation in second (cache-hit) call")
	}
}

func TestArchConfigCacheNil(t *testing.T) {
	defer sensors.ResetArchConfigCache()

	tempDir := t.TempDir()

	file1 := filepath.Join(tempDir, "first.go")
	content1 := `package main
import "fmt"
func First() {}
`
	os.WriteFile(file1, []byte(content1), 0644)

	file2 := filepath.Join(tempDir, "second.go")
	content2 := `package main
import "fmt"
func Second() {}
`
	os.WriteFile(file2, []byte(content2), 0644)

	plugin := sensors.GoPlugin{}

	res1, err := plugin.Analyze([]sensors.FileContext{{Path: file1}})
	if err != nil {
		t.Fatalf("Analyze first file failed: %v", err)
	}

	res2, err := plugin.Analyze([]sensors.FileContext{{Path: file2}})
	if err != nil {
		t.Fatalf("Analyze second file failed: %v", err)
	}

	for _, v := range res1[file1] {
		if v.RuleName == "DependencyBoundary" {
			t.Errorf("First file: unexpected DependencyBoundary violation with no config: %v", v)
		}
	}
	for _, v := range res2[file2] {
		if v.RuleName == "DependencyBoundary" {
			t.Errorf("Second file: unexpected DependencyBoundary violation with no config: %v", v)
		}
	}
}

func TestArchConfigCacheConcurrent(t *testing.T) {
	defer sensors.ResetArchConfigCache()

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

	domainDir := filepath.Join(tempDir, "domain")
	os.Mkdir(domainDir, 0755)

	domainFile := filepath.Join(domainDir, "model.go")
	content := `package domain
import "myproject/api"
func Do() {}
`
	os.WriteFile(domainFile, []byte(content), 0644)

	plugin := sensors.GoPlugin{}
	fileCtx := []sensors.FileContext{{Path: domainFile}}

	const numGoroutines = 20

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	results := make([]map[string][]sensors.Violation, numGoroutines)
	var mu sync.Mutex

	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			res, err := plugin.Analyze(fileCtx)
			if err != nil {
				t.Errorf("Goroutine %d Analyze failed: %v", idx, err)
				return
			}
			mu.Lock()
			results[idx] = res
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	var firstResult []sensors.Violation
	for i, res := range results {
		if res == nil {
			t.Fatalf("Goroutine %d returned nil result map", i)
		}
		violations := res[domainFile]
		if i == 0 {
			firstResult = violations
		}
		if len(violations) != len(firstResult) {
			t.Errorf("Goroutine %d returned %d violations, expected %d", i, len(violations), len(firstResult))
		}
	}

	foundViolation := false
	for _, v := range firstResult {
		if v.RuleName == "DependencyBoundary" {
			foundViolation = true
			break
		}
	}
	if !foundViolation {
		t.Errorf("Expected DependencyBoundary violation in concurrent analysis results")
	}
}
