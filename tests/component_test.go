package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestPolyglotComponent is a high-level component test that verifies the full CLI pipeline
// (FindFiles -> ScanFiles -> FormatResultsCLI) on a simulated monorepo.
// It verifies that the tool correctly handles multiple languages, processes configurations,
// and correctly outputs the report summary without relying on mocked internal functions.
func TestPolyglotComponent(t *testing.T) {
	tempDir := t.TempDir()
	binPath := filepath.Join(tempDir, "maintainability-sensors")

	_, filename, _, _ := runtime.Caller(0)
	repoRoot := filepath.Dir(filepath.Dir(filename))

	// 1. Compile the CLI
	t.Logf("Compiling CLI to %s...", binPath)
	cmdBuild := exec.Command("go", "build", "-o", binPath, "main.go")
	cmdBuild.Dir = repoRoot
	if out, err := cmdBuild.CombinedOutput(); err != nil {
		t.Fatalf("failed to compile CLI: %v\nOutput: %s", err, out)
	}

	// 2. Scaffold a polyglot workspace
	workspace := filepath.Join(tempDir, "workspace")
	if err := os.Mkdir(workspace, 0755); err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}

	// Create a Go file
	goFile := filepath.Join(workspace, "main.go")
	if err := os.WriteFile(goFile, []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a Python file
	pyFile := filepath.Join(workspace, "script.py")
	if err := os.WriteFile(pyFile, []byte("def hello():\n    pass\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a JS file
	jsFile := filepath.Join(workspace, "app.js")
	if err := os.WriteFile(jsFile, []byte("function hello() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// 3. Test Bootstrap logic (Ensure it configures ALL languages in a monorepo)
	t.Log("Running bootstrap command on workspace...")
	cmdBootstrap := exec.Command(binPath, "bootstrap", workspace)
	if out, err := cmdBootstrap.CombinedOutput(); err != nil {
		t.Fatalf("bootstrap failed: %v\nOutput: %s", err, string(out))
	}

	// Verify all config files were created
	expectedConfigs := []string{".golangci.yml", ".pylintrc", ".eslintrc.json"}
	for _, cfg := range expectedConfigs {
		if _, err := os.Stat(filepath.Join(workspace, cfg)); os.IsNotExist(err) {
			t.Errorf("bootstrap failed to create %s in polyglot repo", cfg)
		}
	}

	// 4. Modify Go config to create a relaxed limit (testing the YAML parser & Exception reporter)
	goConfig := filepath.Join(workspace, ".golangci.yml")
	goConfigContent, _ := os.ReadFile(goConfig)
	// Replace default complexity (8) with a relaxed one (15) to trigger an exception
	relaxedConfig := strings.Replace(string(goConfigContent), "min-complexity: 8", "min-complexity: 15", -1)
	os.WriteFile(goConfig, []byte(relaxedConfig), 0644)

	// Verify guardrail: Run bootstrap again and ensure it does NOT overwrite our relaxed config
	cmdBootstrap2 := exec.Command(binPath, "bootstrap", workspace)
	if out, err := cmdBootstrap2.CombinedOutput(); err != nil {
		t.Fatalf("second bootstrap failed: %v\nOutput: %s", err, string(out))
	}
	newGoConfigContent, _ := os.ReadFile(goConfig)
	if string(newGoConfigContent) != relaxedConfig {
		t.Errorf("safety guardrail failed: existing config was overwritten by bootstrap!")
	}

	// 5. Test Run logic
	t.Log("Running run command on workspace...")
	cmdRun := exec.Command(binPath, "run", workspace)
	runOut, err := cmdRun.CombinedOutput()
	outStr := string(runOut)

	if err != nil {
		t.Fatalf("run failed: %v\nOutput:\n%s", err, outStr)
	}

	// Assertions on the output
	if !strings.Contains(outStr, "Maintainability Sensors Report Summary") {
		t.Error("Missing report summary header")
	}

	// It should list the files
	if !strings.Contains(outStr, "main.go") {
		t.Error("Missing main.go in output")
	}
	if !strings.Contains(outStr, "script.py") {
		t.Error("Missing script.py in output")
	}
	if !strings.Contains(outStr, "app.js") {
		t.Error("Missing app.js in output")
	}

	// It should flag the relaxed limit we injected
	if !strings.Contains(outStr, "Exceptions Created by AI (Relaxed Constraints)") {
		t.Error("Failed to detect relaxed constraint section")
	}
	if !strings.Contains(outStr, "Cyclomatic Complexity (15 vs baseline 8)") {
		t.Error("Failed to detect specifically relaxed Go cyclomatic complexity")
	}
}
