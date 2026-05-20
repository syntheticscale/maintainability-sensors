package sensors

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeMockScript(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0755); err != nil {
		t.Fatalf("failed to write mock script %s: %v", name, err)
	}
}

func withMockLinterPath(t *testing.T) func() {
	t.Helper()
	mockDir := t.TempDir()

	writeMockScript(t, mockDir, "npx", `#!/bin/bash
FILE="${@: -1}"
BASENAME=$(basename "$FILE")
case "$BASENAME" in
  *exit0*) exit 0 ;;
  *exit1_valid*)
    echo '[{"filePath":"'"$FILE"'","messages":[{"ruleId":"complexity","message":"complexity of 15 (max 10)","line":1,"severity":2},{"ruleId":"max-params","message":"Function has 7 parameters","line":1,"severity":2},{"ruleId":"max-lines-per-function","message":"exceeds 80 lines","line":1,"severity":2}]}]'
    exit 1 ;;
  *exit1_invalid*)
    echo "ESLint internal error: Cannot read config file"
    exit 1 ;;
  *exit2*) exit 2 ;;
esac
`)

	writeMockScript(t, mockDir, "pylint", `#!/bin/bash
FILE="$2"
BASENAME=$(basename "$FILE")
case "$BASENAME" in
  *exit0*) exit 0 ;;
  *exit1_valid*)
    echo '[{"path":"'"$FILE"'","message":"Too many statements (72/50)","symbol":"too-many-statements","line":1},{"path":"'"$FILE"'","message":"McCabe rating is 12","symbol":"too-complex","line":1},{"path":"'"$FILE"'","message":"Too many arguments (8/5)","symbol":"too-many-arguments","line":1}]'
    exit 1 ;;
  *exit1_invalid*)
    echo "pylint crashed: invalid config"
    exit 1 ;;
  *exit2*) exit 2 ;;
esac
`)

	writeMockScript(t, mockDir, "rubocop", `#!/bin/bash
FILE="$3"
BASENAME=$(basename "$FILE")
case "$BASENAME" in
  *exit0*) exit 0 ;;
  *exit1_valid*)
    echo '{"files":[{"path":"'"$FILE"'","offenses":[{"cop_name":"Metrics/CyclomaticComplexity","message":"Cyclomatic complexity is too high: [15/10]"},{"cop_name":"Metrics/MethodLength","message":"Method is too long: [100/50]"},{"cop_name":"Metrics/ParameterLists","message":"Parameter list is too long: [8/5]"}]}]}'
    exit 1 ;;
  *exit1_invalid*)
    echo "rubocop config error"
    exit 1 ;;
  *exit2*) exit 2 ;;
esac
`)

	origPath := os.Getenv("PATH")
	os.Setenv("PATH", mockDir+":"+origPath)
	return func() { os.Setenv("PATH", origPath) }
}

// ─── ESLint ────────────────────────────────────────────────────────────────────

func TestRunESLint_Subprocess(t *testing.T) {
	defer withMockLinterPath(t)()

	tests := []struct {
		name      string
		filePath  string
		wantComp  int
		wantLen   int
		wantArgs  int
		wantErr   bool
		errSubstr string
	}{
		{"exit code 0", "exit0_test.ts", 0, 0, 0, false, ""},
		{"exit code 1 valid JSON", "exit1_valid_test.ts", 15, 80, 7, false, ""},
		{"exit code 1 invalid JSON", "exit1_invalid_test.ts", 0, 0, 0, true, "ESLint crashed"},
		{"exit code 2", "exit2_test.ts", 0, 0, 0, true, "unexpected code 2"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			metricsMap, err := runESLintBatch([]string{tc.filePath})
			metrics := metricsMap[tc.filePath]
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tc.errSubstr) {
					t.Errorf("error = %q, want containing %q", err.Error(), tc.errSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if metrics.Complexity != tc.wantComp {
				t.Errorf("Complexity = %d, want %d", metrics.Complexity, tc.wantComp)
			}
			if metrics.FunctionLength != tc.wantLen {
				t.Errorf("FunctionLength = %d, want %d", metrics.FunctionLength, tc.wantLen)
			}
			if metrics.ArgumentCount != tc.wantArgs {
				t.Errorf("ArgumentCount = %d, want %d", metrics.ArgumentCount, tc.wantArgs)
			}
		})
	}

	t.Run("tool not found", func(t *testing.T) {
		emptyDir := t.TempDir()
		origPath := os.Getenv("PATH")
		os.Setenv("PATH", emptyDir)
		defer os.Setenv("PATH", origPath)

		_, err := runESLintBatch([]string{"test.ts"})
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "not found in PATH") {
			t.Errorf("error = %q, want 'not found in PATH'", err.Error())
		}
	})
}

// ─── PyLint ────────────────────────────────────────────────────────────────────

func TestRunPyLint_Subprocess(t *testing.T) {
	defer withMockLinterPath(t)()

	tests := []struct {
		name      string
		filePath  string
		wantComp  int
		wantLen   int
		wantArgs  int
		wantErr   bool
		errSubstr string
	}{
		{"exit code 0", "exit0_test.py", 0, 0, 0, false, ""},
		{"exit code 1 valid JSON", "exit1_valid_test.py", 12, 72, 8, false, ""},
		{"exit code 1 invalid JSON", "exit1_invalid_test.py", 0, 0, 0, true, "pylint crashed"},
		{"exit code 2", "exit2_test.py", 0, 0, 0, true, "pylint crashed"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			metricsMap, err := runPyLintBatch([]string{tc.filePath})
			metrics := metricsMap[tc.filePath]
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tc.errSubstr) {
					t.Errorf("error = %q, want containing %q", err.Error(), tc.errSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if metrics.Complexity != tc.wantComp {
				t.Errorf("Complexity = %d, want %d", metrics.Complexity, tc.wantComp)
			}
			if metrics.FunctionLength != tc.wantLen {
				t.Errorf("FunctionLength = %d, want %d", metrics.FunctionLength, tc.wantLen)
			}
			if metrics.ArgumentCount != tc.wantArgs {
				t.Errorf("ArgumentCount = %d, want %d", metrics.ArgumentCount, tc.wantArgs)
			}
		})
	}

	t.Run("tool not found", func(t *testing.T) {
		emptyDir := t.TempDir()
		origPath := os.Getenv("PATH")
		os.Setenv("PATH", emptyDir)
		defer os.Setenv("PATH", origPath)

		_, err := runPyLintBatch([]string{"test.py"})
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "not found in PATH") {
			t.Errorf("error = %q, want 'not found in PATH'", err.Error())
		}
	})
}

// ─── RuboCop ──────────────────────────────────────────────────────────────────

func TestRunRuboCop_Subprocess(t *testing.T) {
	defer withMockLinterPath(t)()

	tests := []struct {
		name      string
		filePath  string
		wantComp  int
		wantLen   int
		wantArgs  int
		wantErr   bool
		errSubstr string
	}{
		{"exit code 0", "exit0_test.rb", 0, 0, 0, false, ""},
		{"exit code 1 valid JSON", "exit1_valid_test.rb", 15, 100, 8, false, ""},
		{"exit code 1 invalid JSON", "exit1_invalid_test.rb", 0, 0, 0, true, "rubocop crashed"},
		{"exit code 2", "exit2_test.rb", 0, 0, 0, true, "unexpected code 2"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			metricsMap, err := runRuboCopBatch([]string{tc.filePath})
			metrics := metricsMap[tc.filePath]
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tc.errSubstr) {
					t.Errorf("error = %q, want containing %q", err.Error(), tc.errSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if metrics.Complexity != tc.wantComp {
				t.Errorf("Complexity = %d, want %d", metrics.Complexity, tc.wantComp)
			}
			if metrics.FunctionLength != tc.wantLen {
				t.Errorf("FunctionLength = %d, want %d", metrics.FunctionLength, tc.wantLen)
			}
			if metrics.ArgumentCount != tc.wantArgs {
				t.Errorf("ArgumentCount = %d, want %d", metrics.ArgumentCount, tc.wantArgs)
			}
		})
	}

	t.Run("tool not found", func(t *testing.T) {
		emptyDir := t.TempDir()
		origPath := os.Getenv("PATH")
		os.Setenv("PATH", emptyDir)
		defer os.Setenv("PATH", origPath)

		_, err := runRuboCopBatch([]string{"test.rb"})
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "not found in PATH") {
			t.Errorf("error = %q, want 'not found in PATH'", err.Error())
		}
	})
}
