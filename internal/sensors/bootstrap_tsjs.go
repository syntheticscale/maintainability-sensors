package sensors

import (
	"fmt"
	"os"
	"path/filepath"
)

const eslintFlatTemplate = `import typescriptEslint from "@typescript-eslint/eslint-plugin";
import tsParser from "@typescript-eslint/parser";

export default [
  {
    files: ["**/*.ts", "**/*.tsx", "**/*.js", "**/*.jsx"],
    languageOptions: {
      parser: tsParser,
    },
    plugins: {
      "@typescript-eslint": typescriptEslint,
    },
    rules: {
      "complexity": ["error", %d],
      "max-params": ["error", %d],
      "max-lines-per-function": ["error", { "max": %d, "skipBlankLines": true, "skipComments": true }],
      "max-lines": ["error", { "max": %d, "skipBlankLines": true, "skipComments": true }],
      "@typescript-eslint/no-explicit-any": "warn"
    }
  }
];
`

func bootstrapTSJS(absPath string) error {
	existingConfig := findExistingESLintConfig(absPath)

	if existingConfig != "" {
		printExistingConfigBanner(existingConfig, fmt.Sprintf(`
- "complexity": ["error", %d]
- "max-params": ["error", %d]
- "max-lines-per-function": ["error", { "max": %d }]
- "max-lines": ["error", { "max": %d }]`, BaselineComplexity, BaselineArgumentCount, BaselineFunctionLength, BaselineFileLength))
	} else {
		eslintPath := filepath.Join(absPath, "eslint.config.mjs")
		content := fmt.Sprintf(eslintFlatTemplate, BaselineComplexity, BaselineArgumentCount, BaselineFunctionLength, BaselineFileLength)
		if err := os.WriteFile(eslintPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write eslint.config.mjs: %w", err)
		}
		fmt.Fprintf(os.Stderr, "- [CREATED] eslint.config.mjs (Pristine Maintainability Rule Suite)\n\n")
	}
	printInstallerInstructions("tsjs")
	return nil
}

func printTSJSInstaller() {
	fmt.Fprintf(os.Stderr, "Execute this command to install the required development engines:\n")
	fmt.Fprintf(os.Stderr, "  npm install --save-dev eslint @typescript-eslint/parser @typescript-eslint/eslint-plugin\n\n")
	fmt.Fprintf(os.Stderr, "Or for Yarn / PNPM:\n")
	fmt.Fprintf(os.Stderr, "  pnpm add -D eslint @typescript-eslint/parser @typescript-eslint/eslint-plugin\n")
}
