package tests

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestMultiRepoValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping multi-repo integration test in short mode")
	}

	tempDir := t.TempDir()
	binPath := filepath.Join(tempDir, "maintainability-sensors")

	_, filename, _, _ := runtime.Caller(0)
	repoRoot := filepath.Dir(filepath.Dir(filename))

	// 1. Compile the CLI
	t.Logf("Compiling CLI to %s...", binPath)
	cmdBuild := exec.Command("go", "build", "-o", binPath, "main.go")
	cmdBuild.Dir = repoRoot
	var buildErr bytes.Buffer
	cmdBuild.Stderr = &buildErr
	if err := cmdBuild.Run(); err != nil {
		t.Fatalf("failed to compile CLI: %v\nBuild stderr: %s", err, buildErr.String())
	}

	repos := []struct {
		name string
		path string
	}{
		{
			name: "maintainability-sensors (itself)",
			path: repoRoot,
		},
	}

	for _, repo := range repos {
		t.Run(repo.name, func(t *testing.T) {
			// Check if repository exists
			if _, err := os.Stat(repo.path); os.IsNotExist(err) {
				t.Skipf("Skipping %s, path does not exist: %s", repo.name, repo.path)
			}

			t.Logf("Running CLI scan on %s...", repo.path)
			cmdRun := exec.Command(binPath, "run", repo.path)
			var stdout, stderr bytes.Buffer
			cmdRun.Stdout = &stdout
			cmdRun.Stderr = &stderr

			err := cmdRun.Run()
			stdoutStr := stdout.String()
			stderrStr := stderr.String()

			t.Logf("STDOUT:\n%s", stdoutStr)
			t.Logf("STDERR:\n%s", stderrStr)

			if err != nil {
				t.Fatalf("CLI exited with error: %v\nStderr:\n%s", err, stderrStr)
			}

			// Ensure no panic or severe errors
			if strings.Contains(stdoutStr, "panic:") || strings.Contains(stderrStr, "panic:") {
				t.Fatal("CLI run caused a panic")
			}

			// Verify that either the report summary or a single-file result header is printed
			if !strings.Contains(stdoutStr, "Maintainability Sensors Report Summary") && !strings.Contains(stdoutStr, "Maintainability Sensor Result") {
				t.Error("Output does not seem to contain standard report summary or single-file result")
			}
		})
	}
}
