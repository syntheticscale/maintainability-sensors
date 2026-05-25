package tests

//nolint // maintainability: highly cohesive test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/syntheticscale/maintainability-sensors/internal/sensors"
)

func getMax(violations []sensors.Violation, ruleName string) int {
	max := 0
	for _, v := range violations {
		if v.RuleName == ruleName && v.Value > max {
			max = v.Value
		}
	}
	return max
}

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

	metrics, err := sensors.ParseGoAST(sensors.FileContext{Path: goFile})
	if err != nil {
		t.Fatalf("ParseGoAST failed: %v", err)
	}

	// Verify maximum argument count (ComplexFunc has 4 parameters: a, b, name, flag)
	args := getMax(metrics, "ArgumentCount")
	if args != 4 {
		t.Errorf("expected max ArgumentCount to be 4, got %d", args)
	}

	// Verify maximum function length (ComplexFunc has body from L12 to L22 -> ~10 lines)
	flen := getMax(metrics, "FunctionLength")
	if flen < 9 || flen > 12 {
		t.Errorf("expected max FunctionLength to be around 10, got %d", flen)
	}

	// Verify maximum cyclomatic complexity (ComplexFunc has 1 (base) + 1 (for) + 1 (if) + 1 (&&) + 1 (else-if) + 1 (||) = 6)
	comp := getMax(metrics, "Complexity")
	if comp != 6 {
		t.Errorf("expected max Complexity to be 6, got %d", comp)
	}
}

func TestOrchestratedScan_WorkingBlindFallback(t *testing.T) {
	tempDir := t.TempDir()
	rbFile := filepath.Join(tempDir, "app.rb")

	if err := os.WriteFile(rbFile, []byte("def hello\nend\n"), 0644); err != nil {
		t.Fatalf("failed to write mock ruby file: %v", err)
	}

	// Scan file in clean temp directory (no lint configs exist)
	res, err := sensors.OrchestratedScan(rbFile)
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

func TestParseCSharp_ValidFile(t *testing.T) {
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

	metrics, err := sensors.ParseCSharp(sensors.FileContext{Path: csFile})
	if err != nil {
		t.Fatalf("ParseCSharp failed: %v", err)
	}

	args := getMax(metrics, "ArgumentCount")
	if args != 4 {
		t.Errorf("expected max ArgumentCount to be 4, got %d", args)
	}
	comp := getMax(metrics, "Complexity")
	if comp != 5 {
		t.Errorf("expected max Complexity to be 5, got %d", comp)
	}
	flen := getMax(metrics, "FunctionLength")
	if flen == 0 {
		t.Errorf("expected FunctionLength > 0, got 0")
	}
}

func TestParseJava_ValidFile(t *testing.T) {
	tempDir := t.TempDir()
	javaFile := filepath.Join(tempDir, "Sample.java")

	content := `
package com.example;

public class Sample {
    public void processData(int x, String name, boolean flag, double score) {
        int result = 0;
        if (x > 10 && flag) {
            result = 1;
        } else if (score < 5.0 || name.equals("skip")) {
            result = 2;
        }
        System.out.println(result);
    }
}
`
	if err := os.WriteFile(javaFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test Java file: %v", err)
	}

	metrics, err := sensors.ParseJava(sensors.FileContext{Path: javaFile})
	if err != nil {
		t.Fatalf("ParseJava failed: %v", err)
	}

	args := getMax(metrics, "ArgumentCount")
	if args != 4 {
		t.Errorf("expected max ArgumentCount to be 4, got %d", args)
	}
	comp := getMax(metrics, "Complexity")
	if comp != 5 {
		t.Errorf("expected max Complexity to be 5, got %d", comp)
	}
	flen := getMax(metrics, "FunctionLength")
	if flen == 0 {
		t.Errorf("expected FunctionLength > 0, got 0")
	}
}

func TestOrchestratedScan_PythonAdvancedComplexity(t *testing.T) {
	tempDir := t.TempDir()
	pyFile := filepath.Join(tempDir, "advanced.py")

	content := `def advanced_complexity(a, b, c):
    try:
        if a > 0 and b > 0:
            with open("file.txt") as f:
                data = f.read()
        elif a < 0 or c < 0:
            data = "fallback"
    except Exception:
        data = "error"
    result = [x for x in range(10) if x > 5]
    return 1 if result else 0
`
	if err := os.WriteFile(pyFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test Python file: %v", err)
	}

	res, err := sensors.OrchestratedScan(pyFile)
	if err != nil {
		t.Fatalf("OrchestratedScan failed: %v", err)
	}

	if !res.ToolingDetected {
		t.Error("expected ToolingDetected to be true for python-ast plugin")
	}

	// Complexity breakdown:
	// 1 base + 1 (try) + 1 (if) + 1 (and) + 1 (with) + 1 (elif) + 1 (or) + 1 (except) + 1 (list_comprehension) + 1 (conditional_expression) = 10
	if res.Metrics.Complexity != 10 {
		t.Errorf("expected complexity 10, got %d", res.Metrics.Complexity)
	}
}
