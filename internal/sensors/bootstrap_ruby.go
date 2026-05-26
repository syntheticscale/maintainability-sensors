package sensors

import (
	"fmt"
	"os"
	"path/filepath"
)

const rubocopTemplate = `# Pristine RuboCop Maintainability Rules
Metrics/CyclomaticComplexity:
  Max: %d
  Enabled: true

Metrics/MethodLength:
  Max: %d
  CountComments: false
  Enabled: true

Metrics/ParameterLists:
  Max: %d
  Enabled: true

Metrics/ModuleLength:
  Max: %d
  Enabled: true
`

func bootstrapRuby(absPath string) error {
	ruboPath := filepath.Join(absPath, ".rubocop.yml")
	if _, err := os.Stat(ruboPath); err == nil {
		printExistingConfigBanner(".rubocop.yml", fmt.Sprintf(`
- Metrics/CyclomaticComplexity: { Max: %d }
- Metrics/MethodLength: { Max: %d }
- Metrics/ParameterLists: { Max: %d }`, BaselineComplexity, BaselineFunctionLength, BaselineArgumentCount))
	} else {
		if err := os.WriteFile(ruboPath, []byte(fmt.Sprintf(rubocopTemplate, BaselineComplexity, BaselineFunctionLength, BaselineArgumentCount, BaselineFileLength)), 0644); err != nil {
			return fmt.Errorf("failed to write .rubocop.yml: %w", err)
		}
		fmt.Fprintf(os.Stderr, "- [CREATED] .rubocop.yml (Pristine Ruby RuboCop Complexity Rules)\n\n")
	}
	printInstallerInstructions("ruby")
	return nil
}

func printRubyInstaller() {
	fmt.Fprintf(os.Stderr, "Execute this command to install the RuboCop engine:\n")
	fmt.Fprintf(os.Stderr, "  gem install rubocop\n\n")
	fmt.Fprintf(os.Stderr, "To run checks natively:\n")
	fmt.Fprintf(os.Stderr, "  rubocop --format json your_code_directory/\n")
}
