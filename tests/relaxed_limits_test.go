package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/syntheticscale/maintainability-sensors/cli"
	"github.com/syntheticscale/maintainability-sensors/sensors"
)

func TestOrchestratedScan_RelaxedLimits_ESLint(t *testing.T) {
	tempDir := t.TempDir()

	// Write relaxed ESLint config
	eslintConfig := `{
		"rules": {
			"complexity": ["error", 12],
			"max-params": ["error", 6],
			"max-lines-per-function": ["error", 100],
			"max-lines": ["error", 500]
		}
	}`
	if err := os.WriteFile(filepath.Join(tempDir, ".eslintrc.json"), []byte(eslintConfig), 0644); err != nil {
		t.Fatalf("failed to write ESLint config: %v", err)
	}

	mockTS := filepath.Join(tempDir, "index.ts")
	if err := os.WriteFile(mockTS, []byte("const x = 1;\n"), 0644); err != nil {
		t.Fatalf("failed to write mock TS file: %v", err)
	}

	res, err := sensors.OrchestratedScan(mockTS)
	if err != nil {
		t.Fatalf("OrchestratedScan failed: %v", err)
	}

	if !res.ToolingDetected {
		t.Fatal("expected ToolingDetected to be true")
	}

	// Verify exceptions parsed
	if len(res.Exceptions) != 4 {
		t.Errorf("expected 4 exceptions, got %d: %+v", len(res.Exceptions), res.Exceptions)
	}

	expectedMap := map[string]int{
		"Cyclomatic Complexity": 12,
		"Argument Count":        6,
		"Function Length":       100,
		"File Length":           500,
	}

	for _, exc := range res.Exceptions {
		expectedVal, exists := expectedMap[exc.RuleName]
		if !exists {
			t.Errorf("unexpected rule name in exceptions: %s", exc.RuleName)
			continue
		}
		if exc.ConfiguredVal != expectedVal {
			t.Errorf("for %s, expected configured val %d, got %d", exc.RuleName, expectedVal, exc.ConfiguredVal)
		}
	}
}

func TestOrchestratedScan_RelaxedLimits_PyLint(t *testing.T) {
	tempDir := t.TempDir()

	// Write relaxed PyLint config
	pylintConfig := `[DESIGN]
max-args=7
max-statements=80
max-complexity=11
max-module-lines=450
`
	if err := os.WriteFile(filepath.Join(tempDir, ".pylintrc"), []byte(pylintConfig), 0644); err != nil {
		t.Fatalf("failed to write PyLint config: %v", err)
	}

	mockPy := filepath.Join(tempDir, "app.py")
	if err := os.WriteFile(mockPy, []byte("print('hello')\n"), 0644); err != nil {
		t.Fatalf("failed to write mock Py file: %v", err)
	}

	res, err := sensors.OrchestratedScan(mockPy)
	if err != nil {
		t.Fatalf("OrchestratedScan failed: %v", err)
	}

	if !res.ToolingDetected {
		t.Fatal("expected ToolingDetected to be true")
	}

	// Verify exceptions parsed
	if len(res.Exceptions) != 4 {
		t.Errorf("expected 4 exceptions, got %d: %+v", len(res.Exceptions), res.Exceptions)
	}

	expectedMap := map[string]int{
		"Cyclomatic Complexity": 11,
		"Argument Count":        7,
		"Function Length":       80,
		"File Length":           450,
	}

	for _, exc := range res.Exceptions {
		expectedVal, exists := expectedMap[exc.RuleName]
		if !exists {
			t.Errorf("unexpected rule name in exceptions: %s", exc.RuleName)
			continue
		}
		if exc.ConfiguredVal != expectedVal {
			t.Errorf("for %s, expected configured val %d, got %d", exc.RuleName, expectedVal, exc.ConfiguredVal)
		}
	}
}

func TestOrchestratedScan_RelaxedLimits_GolangCI(t *testing.T) {
	tempDir := t.TempDir()

	// Write relaxed golangci config
	golangciConfig := `
linters-settings:
  gocognit:
    min-complexity: 15
  funlen:
    lines: 75
`
	if err := os.WriteFile(filepath.Join(tempDir, ".golangci.yml"), []byte(golangciConfig), 0644); err != nil {
		t.Fatalf("failed to write golangci config: %v", err)
	}

	mockGo := filepath.Join(tempDir, "main.go")
	if err := os.WriteFile(mockGo, []byte("package main\n"), 0644); err != nil {
		t.Fatalf("failed to write mock Go file: %v", err)
	}

	res, err := sensors.OrchestratedScan(mockGo)
	if err != nil {
		t.Fatalf("OrchestratedScan failed: %v", err)
	}

	if !res.ToolingDetected {
		t.Fatal("expected ToolingDetected to be true")
	}

	if len(res.Exceptions) != 2 {
		t.Errorf("expected 2 exceptions (Complexity and Function Length), got %d: %+v", len(res.Exceptions), res.Exceptions)
	}

	expectedMap := map[string]int{
		"Cyclomatic Complexity": 15,
		"Function Length":       75,
	}

	for _, exc := range res.Exceptions {
		expectedVal, exists := expectedMap[exc.RuleName]
		if !exists {
			t.Errorf("unexpected rule name in exceptions: %s", exc.RuleName)
			continue
		}
		if exc.ConfiguredVal != expectedVal {
			t.Errorf("for %s, expected configured val %d, got %d", exc.RuleName, expectedVal, exc.ConfiguredVal)
		}
	}
}

func TestGenerateMarkdownScorecard(t *testing.T) {
	results := []sensors.OrchestratorResult{
		{
			FilePath:        "src/auth.ts",
			Language:        "typescript",
			ToolingDetected: true,
			Metrics: sensors.MaintainabilityMetrics{
				Complexity:     12,
				FunctionLength: 45,
				ArgumentCount:  5,
			},
			Exceptions: []sensors.RelaxedLimit{
				{RuleName: "Cyclomatic Complexity", ConfiguredVal: 15, BaselineVal: 8},
			},
		},
		{
			FilePath:        "src/util.py",
			Language:        "python",
			ToolingDetected: false,
		},
	}

	scorecard := cli.GenerateMarkdownScorecard(results)

	if !strings.Contains(scorecard, "📡 Maintainability Sensors Scorecard") {
		t.Error("scorecard missing main title")
	}

	if !strings.Contains(scorecard, "auth.ts") || !strings.Contains(scorecard, "util.py") {
		t.Error("scorecard missing scanned file names")
	}

	// Verify exceptions are printed in markdown
	if !strings.Contains(scorecard, "Exceptions Created by AI (Relaxed Constraints)") {
		t.Error("scorecard missing exceptions section header")
	}
	if !strings.Contains(scorecard, "Cyclomatic Complexity") || !strings.Contains(scorecard, "15") {
		t.Error("scorecard missing exception details")
	}

	// Verify warnings are printed
	if !strings.Contains(scorecard, "AI Agent Self-Correction Prompts") {
		t.Error("scorecard missing self-correction prompts header")
	}
	if !strings.Contains(scorecard, "Complexity is 12") {
		t.Error("scorecard missing complexity violation warning")
	}
}
