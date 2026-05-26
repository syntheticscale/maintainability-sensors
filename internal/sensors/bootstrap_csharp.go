package sensors

import (
	"fmt"
	"os"
	"path/filepath"
)

const editorconfigTemplate = `# Pristine Microsoft .editorconfig Maintainability Rules
root = true

[*.cs]
# Roslyn CA1502: Avoid excessive cyclomatic complexity (Limit: %d)
dotnet_diagnostic.CA1502.severity = warning
dotnet_code_quality.CA1502.maximum_cyclomatic_complexity = %d

# Roslyn CA1506: Avoid excessive class coupling
dotnet_diagnostic.CA1506.severity = warning

# Enforce standard method length / formatting rules
dotnet_sort_system_directives_first = true
`

func bootstrapCSharp(absPath string) error {
	editorPath := filepath.Join(absPath, ".editorconfig")
	if _, err := os.Stat(editorPath); err == nil {
		printExistingConfigBanner(".editorconfig", fmt.Sprintf(`
- dotnet_code_quality.CA1502.maximum_cyclomatic_complexity = %d
- dotnet_diagnostic.CA1502.severity = warning`, BaselineComplexity))
	} else {
		if err := os.WriteFile(editorPath, []byte(fmt.Sprintf(editorconfigTemplate, BaselineComplexity, BaselineComplexity)), 0644); err != nil {
			return fmt.Errorf("failed to write .editorconfig: %w", err)
		}
		fmt.Fprintf(os.Stderr, "- [CREATED] .editorconfig (Pristine Microsoft C# EditorConfig Analyzers)\n\n")
	}
	printInstallerInstructions("csharp")
	return nil
}

func printCSharpInstaller() {
	fmt.Fprintf(os.Stderr, "Microsoft C# Analyzers are built natively into the .NET SDK.\n")
	fmt.Fprintf(os.Stderr, "To verify code formatting and analyzer rules, run standard .NET commands:\n\n")
	fmt.Fprintf(os.Stderr, "Run static code analysis:\n")
	fmt.Fprintf(os.Stderr, "  dotnet build /p:TreatWarningsAsErrors=true\n\n")
	fmt.Fprintf(os.Stderr, "Or run automatic formatting verification:\n")
	fmt.Fprintf(os.Stderr, "  dotnet format --verify-no-changes\n")
}
