package sensors

import (
	"fmt"
	"os"
	"path/filepath"
)

const golangciTemplate = `run:
  timeout: 5m

linters-settings:
  gocognit:
    min-complexity: %d
  funlen:
    lines: %d
    statements: 40
  cyclop:
    max-complexity: %d
  lll:
    line-length: 120
  revive:
    rules:
      - name: argument-limit
        arguments: [%d]

linters:
  enable:
    - gocognit
    - funlen
    - cyclop
    - lll
    - revive
`

func bootstrapGo(absPath string) error {
	gociPath := filepath.Join(absPath, ".golangci.yml")
	if _, err := os.Stat(gociPath); err == nil {
		printExistingConfigBanner(".golangci.yml", fmt.Sprintf(`
- gocognit: { min-complexity: %d }
- funlen: { lines: %d }
- gocyclo: { min-complexity: %d }
- revive: { argument-limit: %d }`, BaselineComplexity, BaselineFunctionLength, BaselineComplexity, BaselineArgumentCount))
	} else {
		if err := os.WriteFile(gociPath, []byte(fmt.Sprintf(golangciTemplate, BaselineComplexity, BaselineFunctionLength, BaselineComplexity, BaselineArgumentCount)), 0644); err != nil {
			return fmt.Errorf("failed to write .golangci.yml: %w", err)
		}
		fmt.Fprintf(os.Stderr, "- [CREATED] .golangci.yml (Pristine Go Vet / Gocognit Complexity Rules)\n\n")
	}
	printInstallerInstructions("go")
	return nil
}

func printGoInstaller() {
	fmt.Fprintf(os.Stderr, "Execute this command to install the golangci-lint meta-linter:\n")
	fmt.Fprintf(os.Stderr, "  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.60.0\n\n")
	fmt.Fprintf(os.Stderr, "Run checks with:\n")
	fmt.Fprintf(os.Stderr, "  golangci-lint run ./...\n")
}
