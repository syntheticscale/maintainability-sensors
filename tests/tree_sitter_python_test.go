package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/syntheticscale/maintainability-sensors/sensors"
)

func TestTreeSitterPythonParser(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test.py")

	pythonCode := `
def simple_function(a, b):
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
	if err := os.WriteFile(filePath, []byte(pythonCode), 0644); err != nil {
		t.Fatalf("Failed to write test python file: %v", err)
	}

	violations, err := sensors.ParsePythonTreeSitter(filePath)
	if err != nil {
		t.Fatalf("ParsePythonTreeSitter failed: %v", err)
	}

	// Helper to find a specific rule violation for a function line
	findViolation := func(ruleName string, startLine int) *sensors.Violation {
		for _, v := range violations {
			if v.RuleName == ruleName && v.StartLine == startLine {
				return &v
			}
		}
		return nil
	}

	// simple_function is at line 2. 
	// FunctionLength: line 2 to line 4 -> 3 lines
	// ArgumentCount: 2
	// Complexity: 1 (base)
	
	if v := findViolation("FunctionLength", 2); v == nil || v.Value != 3 {
		t.Errorf("Expected simple_function length 3, got: %+v", v)
	}
	if v := findViolation("ArgumentCount", 2); v == nil || v.Value != 2 {
		t.Errorf("Expected simple_function arg count 2, got: %+v", v)
	}
	if v := findViolation("Complexity", 2); v == nil || v.Value != 1 {
		t.Errorf("Expected simple_function complexity 1, got: %+v", v)
	}

	// complex_function is at line 6.
	// FunctionLength: line 6 to line 18 -> 13 lines
	// ArgumentCount: 4
	// Complexity: 
	// 1 (base) + 1 (if) + 1 (for) + 1 (elif doesn't exist explicitly in tree-sitter python, usually it's an if inside elif or just elif block) 
	// + 1 (while) + 1 (except) = 6
	
	if v := findViolation("FunctionLength", 6); v == nil || v.Value != 13 {
		t.Errorf("Expected complex_function length 13, got: %+v", v)
	}
	if v := findViolation("ArgumentCount", 6); v == nil || v.Value != 4 {
		t.Errorf("Expected complex_function arg count 4, got: %+v", v)
	}
	if v := findViolation("Complexity", 6); v == nil || v.Value != 6 {
		t.Errorf("Expected complex_function complexity 6, got: %+v", v)
	}
}
