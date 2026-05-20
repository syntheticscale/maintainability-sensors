package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/syntheticscale/maintainability-sensors/sensors"
)

func TestParseGoAST_ValidFile(t *testing.T) {
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

	metrics, err := sensors.ParseGoAST(goFile)
	if err != nil {
		t.Fatalf("ParseGoAST failed: %v", err)
	}

	// Verify maximum argument count (ComplexFunc has 4 parameters: a, b, name, flag)
	if metrics.ArgumentCount != 4 {
		t.Errorf("expected max ArgumentCount to be 4, got %d", metrics.ArgumentCount)
	}

	// Verify maximum function length (ComplexFunc has body from L12 to L22 -> ~10 lines)
	if metrics.FunctionLength < 9 || metrics.FunctionLength > 12 {
		t.Errorf("expected max FunctionLength to be around 10, got %d", metrics.FunctionLength)
	}

	// Verify maximum cyclomatic complexity (ComplexFunc has 1 (base) + 1 (for) + 1 (if) + 1 (&&) + 1 (else-if) + 1 (||) = 6)
	if metrics.Complexity != 6 {
		t.Errorf("expected max Complexity to be 6, got %d", metrics.Complexity)
	}
}

func TestOrchestratedScan_WorkingBlindFallback(t *testing.T) {
	tempDir := t.TempDir()
	pyFile := filepath.Join(tempDir, "app.py")

	if err := os.WriteFile(pyFile, []byte("def hello():\n    pass\n"), 0644); err != nil {
		t.Fatalf("failed to write mock python file: %v", err)
	}

	// Scan file in clean temp directory (no lint configs exist)
	res, err := sensors.OrchestratedScan(pyFile)
	if err != nil {
		t.Fatalf("OrchestratedScan failed: %v", err)
	}

	if res.ToolingDetected {
		t.Error("expected ToolingDetected to be false in un-configured directory")
	}

	if res.Metrics.Complexity != 0 || res.Metrics.FunctionLength != 0 {
		t.Errorf("expected metrics to be zero in fallback mode, got %+v", res.Metrics)
	}
}

func TestParseCSharp_ReturnsError(t *testing.T) {
	tempDir := t.TempDir()
	csFile := filepath.Join(tempDir, "sample.cs")

	content := `using System;

namespace Sample {
    public class Processor {
        public void ProcessData(int x, string name, bool flag, double score) {
            int result = 0;
            if (x > 10 && flag) {
                result = 1;
            } else if (score < 5.0 || name == "skip") {
                result = 2;
            }
            Console.WriteLine(result);
        }
    }
}
`
	if err := os.WriteFile(csFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test C# file: %v", err)
	}

	// C# native parsing is not supported and should always return an error.
	_, err := sensors.ParseCSharp(csFile)
	if err == nil {
		t.Fatalf("expected ParseCSharp to return an error, got nil")
	}
}
