package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func compileBinary(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	bin := filepath.Join(tmpDir, "maintainability-sensors")
	_, filename, _, _ := runtime.Caller(0)
	repoRoot := filepath.Dir(filepath.Dir(filename))
	cmd := exec.Command("go", "build", "-buildvcs=false", "-o", bin, "./cmd/maintainability-sensors") // -buildvcs=false avoids build failures in shallow worktrees or Docker without git history
	cmd.Dir = repoRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to compile CLI binary: %v\nOutput: %s", err, out)
	}
	return bin
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git command failed (%v): %v\nOutput: %s", args, err, out)
	}
}

func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test")
}

func writeConfig(t *testing.T, dir, content string) {
	t.Helper()
	p := filepath.Join(dir, ".maintainability-sensors.yml")
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
}

func writeGoFile(t *testing.T, dir, filename, content string) {
	t.Helper()
	p := filepath.Join(dir, filename)
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write Go file: %v", err)
	}
}

const simpleGoFile = `package main

func SimpleFunc() {}
`

// A Go function with cyclomatic complexity 9 (> 8 baseline)
const complexGoFile = `package main

func ComplexFunc(x int) {
	if x > 1 { return }
	if x > 2 { return }
	if x > 3 { return }
	if x > 4 { return }
	if x > 5 { return }
	if x > 6 { return }
	if x > 7 { return }
	if x > 8 { return }
}
`

// A Go function with complexity 10 (violates baseline 8 but NOT threshold 12)
const underThresholdGoFile = `package main

func UnderThreshold(x int) {
	if x > 1 { return }
	if x > 2 { return }
	if x > 3 { return }
	if x > 4 { return }
	if x > 5 { return }
	if x > 6 { return }
	if x > 7 { return }
	if x > 8 { return }
	if x > 9 { return }
}
`

// A Go function that only violates ArgumentCount (5 > 4), nothing else
const argsOnlyGoFile = `package main

func OnlyArgs(a, b, c, d, e int) {}
`

func TestCheckDiffWithWarnSeverity(t *testing.T) {
	bin := compileBinary(t)
	dir := t.TempDir()
	initGitRepo(t, dir)

	writeConfig(t, dir, `version: "1"
check-diff:
  default-severity: warn
`)
	writeGoFile(t, dir, "main.go", simpleGoFile)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "test")

	writeGoFile(t, dir, "main.go", complexGoFile)

	cmd := exec.Command(bin, "check-diff", dir)
	out, err := cmd.CombinedOutput()
	output := string(out)

	if err != nil {
		t.Fatalf("unexpected error for warn severity: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "REFACTORING PROMPT") {
		t.Errorf("expected output to contain 'REFACTORING PROMPT', got:\n%s", output)
	}
	if strings.Contains(output, "Delta violations found") {
		t.Errorf("expected no 'Delta violations found' in output, got:\n%s", output)
	}
}

func TestCheckDiffWithErrorSeverity(t *testing.T) {
	bin := compileBinary(t)
	dir := t.TempDir()
	initGitRepo(t, dir)

	writeConfig(t, dir, `version: "1"
check-diff:
  default-severity: error
`)
	writeGoFile(t, dir, "main.go", simpleGoFile)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "test")

	writeGoFile(t, dir, "main.go", complexGoFile)

	cmd := exec.Command(bin, "check-diff", dir)
	out, err := cmd.CombinedOutput()
	output := string(out)

	if err == nil {
		t.Fatalf("expected non-zero exit code for error severity, got exit 0. Output:\n%s", output)
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("unexpected error type: %T: %v", err, err)
	}
	if exitErr.ExitCode() != 1 {
		t.Fatalf("expected exit code 1, got %d. Output:\n%s", exitErr.ExitCode(), output)
	}
	if !strings.Contains(output, "REFACTORING PROMPT") {
		t.Errorf("expected output to contain 'REFACTORING PROMPT', got:\n%s", output)
	}
	if !strings.Contains(output, "Delta violations found") {
		t.Errorf("expected output to contain 'Delta violations found', got:\n%s", output)
	}
}

func TestCheckDiffWithPerRuleIgnore(t *testing.T) {
	bin := compileBinary(t)
	dir := t.TempDir()
	initGitRepo(t, dir)

	writeConfig(t, dir, `version: "1"
check-diff:
  default-severity: error
  rules:
    - name: ArgumentCount
      severity: ignore
`)
	writeGoFile(t, dir, "main.go", simpleGoFile)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "test")

	writeGoFile(t, dir, "main.go", argsOnlyGoFile)

	cmd := exec.Command(bin, "check-diff", dir)
	out, err := cmd.CombinedOutput()
	output := string(out)

	if err != nil {
		t.Fatalf("unexpected error when rule is ignored: %v\nOutput: %s", err, output)
	}
	if strings.Contains(output, "REFACTORING PROMPT") {
		t.Errorf("expected no REFACTORING PROMPT when ArgumentCount is ignored, got:\n%s", output)
	}
	if strings.Contains(output, "Delta violations found") {
		t.Errorf("expected no 'Delta violations found', got:\n%s", output)
	}
}

func TestCheckDiffWithThresholdOverride(t *testing.T) {
	bin := compileBinary(t)
	dir := t.TempDir()
	initGitRepo(t, dir)

	writeConfig(t, dir, `version: "1"
check-diff:
  default-severity: error
  rules:
    - name: Complexity
      threshold: 12
    - name: CognitiveComplexity
      threshold: 12
`)
	writeGoFile(t, dir, "main.go", simpleGoFile)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "test")

	writeGoFile(t, dir, "main.go", underThresholdGoFile)

	cmd := exec.Command(bin, "check-diff", dir)
	out, err := cmd.CombinedOutput()
	output := string(out)

	if err != nil {
		t.Fatalf("unexpected error when below threshold: %v\nOutput: %s", err, output)
	}
	if strings.Contains(output, "REFACTORING PROMPT") {
		t.Errorf("expected no REFACTORING PROMPT when complexity is below threshold, got:\n%s", output)
	}
}

func TestCheckDiffCLIOverridesConfig(t *testing.T) {
	bin := compileBinary(t)
	dir := t.TempDir()
	initGitRepo(t, dir)

	writeConfig(t, dir, `version: "1"
check-diff:
  default-severity: error
`)
	writeGoFile(t, dir, "main.go", simpleGoFile)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "test")

	writeGoFile(t, dir, "main.go", complexGoFile)

	cmd := exec.Command(bin, "check-diff", "--default-severity", "warn", dir)
	out, err := cmd.CombinedOutput()
	output := string(out)

	if err != nil {
		t.Fatalf("unexpected error when CLI overrides config: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "REFACTORING PROMPT") {
		t.Errorf("expected output to contain 'REFACTORING PROMPT', got:\n%s", output)
	}
	if strings.Contains(output, "Delta violations found") {
		t.Errorf("expected no 'Delta violations found' when CLI overrides to warn, got:\n%s", output)
	}
}
