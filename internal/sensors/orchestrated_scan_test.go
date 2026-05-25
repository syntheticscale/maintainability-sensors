package sensors

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

func TestOrchestratedScan_TypeScriptFile(t *testing.T) {
	tempDir := t.TempDir()
	tsFile := filepath.Join(tempDir, "sample_test.ts")

	content := `
function simpleFunc(a: string, b: number) {
	console.log(a, b);
}

function complexFunc(x: number, y: number, z: number) {
	let result = 0;
	if (x > 0) {
		result += x;
	} else if (x < 0) {
		result -= x;
	}
	for (let i = 0; i < y; i++) {
		result += i;
	}
	return result;
}

const arrowFunc = (a: string) => {
	return a;
};
`
	if err := os.WriteFile(tsFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test TS file: %v", err)
	}

	res, err := OrchestratedScan(tsFile)
	if err != nil {
		t.Fatalf("OrchestratedScan failed: %v", err)
	}

	if !res.ToolingDetected {
		t.Error("expected ToolingDetected to be true for TypeScript native AST")
	}

	// simpleFunc: 2 args, complexity 1
	// complexFunc: 3 args, complexity 4 (base + if + else-if + for)
	// arrowFunc: 1 arg, complexity 1
	if res.Metrics.Complexity != 4 {
		t.Errorf("expected Complexity 4, got %d", res.Metrics.Complexity)
	}
	if res.Metrics.ArgumentCount != 3 {
		t.Errorf("expected ArgumentCount 3, got %d", res.Metrics.ArgumentCount)
	}
}

func TestOrchestratedScan_PythonFile(t *testing.T) {
	tempDir := t.TempDir()
	pyFile := filepath.Join(tempDir, "sample_test.py")

	content := `def simple_function(a, b):
    # This is a comment
    return a + b

def complex_function(x, y, z, options):
    if x > 10:
        for i in range(x):
            print(i)
    elif y < 0:
        while z > 0:
            z -= 1
    else:
        try:
            pass
        except Exception:
            pass
    return True
`
	if err := os.WriteFile(pyFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test Python file: %v", err)
	}

	res, err := OrchestratedScan(pyFile)
	if err != nil {
		t.Fatalf("OrchestratedScan failed: %v", err)
	}

	if !res.ToolingDetected {
		t.Error("expected ToolingDetected to be true for Python native AST")
	}

	// simple_function: 2 args, length 3, complexity 1
	// complex_function: 4 args, length 13, complexity 7
	if res.Metrics.Complexity != 7 {
		t.Errorf("expected Complexity 7, got %d", res.Metrics.Complexity)
	}
	if res.Metrics.ArgumentCount != 4 {
		t.Errorf("expected ArgumentCount 4, got %d", res.Metrics.ArgumentCount)
	}
	if res.Metrics.FunctionLength != 13 {
		t.Errorf("expected FunctionLength 13, got %d", res.Metrics.FunctionLength)
	}
}

func TestOrchestratedScan_JavaFile(t *testing.T) {
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

	res, err := OrchestratedScan(javaFile)
	if err != nil {
		t.Fatalf("OrchestratedScan failed: %v", err)
	}

	if !res.ToolingDetected {
		t.Error("expected ToolingDetected to be true for Java native AST")
	}

	if res.Metrics.Complexity != 5 {
		t.Errorf("expected Complexity 5, got %d", res.Metrics.Complexity)
	}
	if res.Metrics.ArgumentCount != 4 {
		t.Errorf("expected ArgumentCount 4, got %d", res.Metrics.ArgumentCount)
	}
	if res.Metrics.FunctionLength == 0 {
		t.Error("expected FunctionLength > 0, got 0")
	}
}

func TestOrchestratedScan_CSharpFile(t *testing.T) {
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

	res, err := OrchestratedScan(csFile)
	if err != nil {
		t.Fatalf("OrchestratedScan failed: %v", err)
	}

	if !res.ToolingDetected {
		t.Error("expected ToolingDetected to be true for C# native AST")
	}

	if res.Metrics.Complexity != 5 {
		t.Errorf("expected Complexity 5, got %d", res.Metrics.Complexity)
	}
	if res.Metrics.ArgumentCount != 4 {
		t.Errorf("expected ArgumentCount 4, got %d", res.Metrics.ArgumentCount)
	}
	if res.Metrics.FunctionLength == 0 {
		t.Error("expected FunctionLength > 0, got 0")
	}
}

func TestOrchestratedScan_UnsupportedFile(t *testing.T) {
	tempDir := t.TempDir()
	rsFile := filepath.Join(tempDir, "main.rs")
	if err := os.WriteFile(rsFile, []byte("fn main() {}\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	_, err := OrchestratedScan(rsFile)
	if err == nil {
		t.Fatal("expected error for unsupported file type")
	}
}

func TestOrchestratedScan_EmptyPath(t *testing.T) {
	_, err := OrchestratedScan("")
	if err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestOrchestratedScanBatch_MultipleGoFiles(t *testing.T) {
	tempDir := t.TempDir()
	goFile1 := filepath.Join(tempDir, "a_test.go")
	goFile2 := filepath.Join(tempDir, "b_test.go")

	content1 := `package sample
func A(a, b, c int) bool {
	if a > 10 { return true }
	return false
}
`
	content2 := `package sample
func B(x int) int {
	return x * 2
}
`
	if err := os.WriteFile(goFile1, []byte(content1), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	if err := os.WriteFile(goFile2, []byte(content2), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	files := []FileContext{{Path: goFile1}, {Path: goFile2}}
	results, err := OrchestratedScanBatch(files, "go")
	if err != nil {
		t.Fatalf("OrchestratedScanBatch failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	for _, res := range results {
		if !res.ToolingDetected {
			t.Errorf("expected ToolingDetected to be true for %s", res.FilePath)
		}
	}
}
