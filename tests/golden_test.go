package tests

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/paulolai/maintainability-sensors/sensors"
)

// Define -update flag to update golden files: go test ./tests -run TestGoldenSnapshots -update
var updateGolden = flag.Bool("update", false, "update golden snapshot files")

type GoldenCase struct {
	repoName   string
	repoURL    string
	commitHash string
	scanPath   string
	goldenFile string
}

func TestGoldenSnapshots(t *testing.T) {
	// Skip if running a short test, as this requires downloading/syncing repositories
	if testing.Short() {
		t.Skip("skipping golden snapshots test in short mode")
	}

	_, filename, _, _ := runtime.Caller(0)
	repoRoot := filepath.Dir(filepath.Dir(filename))
	cacheDir := filepath.Join(repoRoot, ".cache")
	goldenDir := filepath.Join(repoRoot, "tests", "golden")

	// Ensure directories exist
	_ = os.MkdirAll(cacheDir, 0755)
	_ = os.MkdirAll(goldenDir, 0755)

	cases := []GoldenCase{
		{
			repoName:   "go-chi",
			repoURL:    "https://github.com/go-chi/chi.git",
			commitHash: "v5.1.0", // Tag commit
			scanPath:   "tree.go",
			goldenFile: filepath.Join(goldenDir, "go-chi-tree-report.json"),
		},
		{
			repoName:   "requests",
			repoURL:    "https://github.com/psf/requests.git",
			commitHash: "v2.31.0",
			scanPath:   "src/requests/adapters.py",
			goldenFile: filepath.Join(goldenDir, "requests-adapters-report.json"),
		},
		{
			repoName:   "go-std-net",
			repoURL:    "https://github.com/golang/go.git",
			commitHash: "go1.22.0",
			scanPath:   "src/net/http/server.go",
			goldenFile: filepath.Join(goldenDir, "go-std-http-server-report.json"),
		},
		{
			repoName:   "fastapi",
			repoURL:    "https://github.com/tiangolo/fastapi.git",
			commitHash: "0.110.0",
			scanPath:   "fastapi/dependencies/utils.py",
			goldenFile: filepath.Join(goldenDir, "fastapi-dependencies-report.json"),
		},
		{
			repoName:   "nestjs",
			repoURL:    "https://github.com/nestjs/nest.git",
			commitHash: "v10.3.3",
			scanPath:   "packages/core/injector/injector.ts",
			goldenFile: filepath.Join(goldenDir, "nestjs-injector-report.json"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.repoName, func(t *testing.T) {
			repoRoot := filepath.Join(cacheDir, tc.repoName)

			// 1. Sync and checkout exact stable commit
			syncAndCheckout(t, tc.repoName, tc.repoURL, tc.commitHash, repoRoot)

			// 2. Bootstrap linter configs so orchestrated runs work correctly
			err := sensors.BootstrapRepo(repoRoot)
			if err != nil {
				t.Fatalf("failed to bootstrap repo '%s': %v", tc.repoName, err)
			}

			// 3. Scan the target file
			filePath := tc.scanPath
			origWD, _ := os.Getwd()
			os.Chdir(repoRoot)
			result, err := sensors.OrchestratedScan(filePath)
			os.Chdir(origWD)
			if err != nil {
				t.Fatalf("OrchestratedScan failed for %s: %v", filePath, err)
			}

			// Clean absolute paths in result to make it reproducible across environments
			result.FilePath = tc.scanPath

			// Sanitize dynamic Node.js process IDs in error messages (e.g. "(node:12345)") to make snapshots deterministic
			if strings.Contains(result.Message, "(node:") {
				reNodePID := regexp.MustCompile(`\(node:\d+\)`)
				result.Message = reNodePID.ReplaceAllString(result.Message, "(node:PID)")
			}

			// Convert current result to JSON
			currentJSON, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				t.Fatalf("failed to marshal scan result: %v", err)
			}

			// 4. Update golden file if requested or missing
			if *updateGolden || isFileMissing(tc.goldenFile) {
				t.Logf("Writing golden snapshot for %s to %s", tc.repoName, tc.goldenFile)
				if err := os.WriteFile(tc.goldenFile, currentJSON, 0644); err != nil {
					t.Fatalf("failed to write golden file: %v", err)
				}
				return
			}

			// 5. Read golden snapshot
			goldenJSON, err := os.ReadFile(tc.goldenFile)
			if err != nil {
				t.Fatalf("failed to read golden file: %v", err)
			}

			// 6. Compare actual with golden
			if string(currentJSON) != string(goldenJSON) {
				// Format a friendly comparison diff
				diff := compareJSON(string(goldenJSON), string(currentJSON))
				t.Errorf("Mismatched Golden Snapshot for %s!\n\nFRIENDLY DIFF:\n%s\nRun 'go test ./tests -run TestGoldenSnapshots -update' to accept these changes if they are intended.", tc.repoName, diff)
			}
		})
	}
}

func isFileMissing(path string) bool {
	_, err := os.Stat(path)
	return os.IsNotExist(err)
}

func syncAndCheckout(t *testing.T, name, url, commit, path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Logf("Cloning %s at %s (shallow)...", name, commit)
		cmd := exec.Command("git", "clone", "--depth", "1", "--branch", commit, url, path)
		if err := cmd.Run(); err != nil {
			t.Logf("Shallow clone failed, falling back to full clone...")
			cmdFull := exec.Command("git", "clone", url, path)
			if err := cmdFull.Run(); err != nil {
				t.Fatalf("failed to clone %s: %v", name, err)
			}
		} else {
			return // Successfully cloned shallow branch/tag
		}
	}

	t.Logf("Checking out %s at %s...", name, commit)
	cmdFetch := exec.Command("git", "fetch", "--tags")
	cmdFetch.Dir = path
	_ = cmdFetch.Run()

	cmdCheckout := exec.Command("git", "checkout", commit)
	cmdCheckout.Dir = path
	if err := cmdCheckout.Run(); err != nil {
		t.Fatalf("failed to checkout %s to %s: %v", name, commit, err)
	}
}

// compareJSON performs a simple line-by-line diff of two JSON strings to provide a friendly comparison
func compareJSON(golden, current string) string {
	goldenLines := strings.Split(golden, "\n")
	currentLines := strings.Split(current, "\n")

	var diff strings.Builder

	maxLines := len(goldenLines)
	if len(currentLines) > maxLines {
		maxLines = len(currentLines)
	}

	for i := 0; i < maxLines; i++ {
		var gLine, cLine string
		if i < len(goldenLines) {
			gLine = strings.TrimSpace(goldenLines[i])
		}
		if i < len(currentLines) {
			cLine = strings.TrimSpace(currentLines[i])
		}

		if gLine != cLine {
			diff.WriteString(fmt.Sprintf("Line %d:\n", i+1))
			diff.WriteString(fmt.Sprintf("  - GOLDEN:  %s\n", gLine))
			diff.WriteString(fmt.Sprintf("  + CURRENT: %s\n\n", cLine))
		}
	}

	return diff.String()
}
