package sensors

import (
	"fmt"
	"os"
	"path/filepath"
)

const pylintTemplate = `[MASTER]
load-plugins=pylint.extensions.mccabe

[DESIGN]
max-args=%d
max-statements=%d
max-complexity=%d
max-module-lines=%d
`

func bootstrapPython(absPath string) error {
	pylintPath := filepath.Join(absPath, ".pylintrc")
	if _, err := os.Stat(pylintPath); err == nil {
		printExistingConfigBanner(".pylintrc", fmt.Sprintf(`
- [DESIGN]
  max-args=%d
  max-statements=%d
  max-complexity=%d`, BaselineArgumentCount, BaselineFunctionLength, BaselineComplexity))
	} else {
		if err := os.WriteFile(pylintPath, []byte(fmt.Sprintf(pylintTemplate, BaselineArgumentCount, BaselineFunctionLength, BaselineComplexity, BaselineFileLength)), 0644); err != nil {
			return fmt.Errorf("failed to write .pylintrc: %w", err)
		}
		fmt.Fprintf(os.Stderr, "- [CREATED] .pylintrc (Pristine McCabe / PyLint Complexity Rules)\n\n")
	}
	printInstallerInstructions("python")
	return nil
}

func printPythonInstaller() {
	fmt.Fprintf(os.Stderr, "Execute this command to install the required PyLint engine:\n")
	fmt.Fprintf(os.Stderr, "  pip install pylint\n\n")
	fmt.Fprintf(os.Stderr, "To run McCabe cyclomatic checks with pylint:\n")
	fmt.Fprintf(os.Stderr, "  pylint --load-plugins=pylint.extensions.mccabe your_code_directory/\n")
}
