package sensors

//nolint // maintainability: highly cohesive test

import (
	"os"
	"path/filepath"
	"testing"
)

func getMaxViolation(violations []Violation, ruleName string) int {
	max := 0
	for _, v := range violations {
		if v.RuleName == ruleName && v.Value > max {
			max = v.Value
		}
	}
	return max
}

func TestOrchestratedScan_GoFile(t *testing.T) {
	tempDir := t.TempDir()
	goFile := filepath.Join(tempDir, "sample_test.go")

	content := `package sample

func SimpleFunc(a int) bool {
	if a > 10 {
		return true
	}
	return false
}

func ComplexFunc(a, b int, name string, flag bool) int {
	sum := 0
	for i := 0; i < a; i++ {
		if b > 5 && flag {
			sum += i
		} else if b < 2 || !flag {
			sum -= i
		}
	}
	return sum
}
`
	if err := os.WriteFile(goFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test Go file: %v", err)
	}

	res, err := OrchestratedScan(goFile)
	if err != nil {
		t.Fatalf("OrchestratedScan failed: %v", err)
	}

	if res.Language != "go" {
		t.Errorf("expected language 'go', got %q", res.Language)
	}
	if !res.ToolingDetected {
		t.Error("expected ToolingDetected to be true for Go native AST")
	}

	// Verify metrics for the Go file (same as tests/orchestrator_test.go but through public API)
	if res.Metrics.Complexity != 6 {
		t.Errorf("expected Complexity 6, got %d", res.Metrics.Complexity)
	}
	if res.Metrics.ArgumentCount != 4 {
		t.Errorf("expected ArgumentCount 4, got %d", res.Metrics.ArgumentCount)
	}
	if res.Metrics.FunctionLength < 9 || res.Metrics.FunctionLength > 15 {
		t.Errorf("expected FunctionLength around 10, got %d", res.Metrics.FunctionLength)
	}
}


