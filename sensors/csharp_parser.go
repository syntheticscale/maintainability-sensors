package sensors

import (
	"fmt"
)

// CSharpMetrics holds the parsed metrics for a C# file.
// Deprecated: This struct is retained for API compatibility, but
// ParseCSharp no longer returns meaningful metric values.
type CSharpMetrics struct {
	Complexity     int
	FunctionLength int
	ArgumentCount  int
}

// ParseCSharp returns an error because C# maintainability metrics cannot be
// accurately computed via native parsing. C# requires external tooling
// (e.g., dotnet build with Roslyn analyzers, or IDE analyzers).
//
// Regex-based parsing is fundamentally broken for real C# code due to:
//   - lambdas and expression-bodied members
//   - attributes (e.g., [Obsolete])
//   - properties with getters/setters
//   - generic type parameters (e.g., List<T>)
//   - async/await keywords
//   - verbatim strings and interpolated strings
//   - preprocessor directives
//
// To avoid generating false metrics, this function refuses to parse.
func ParseCSharp(filePath string) (CSharpMetrics, error) {
	return CSharpMetrics{}, fmt.Errorf(
		"C# analysis requires external tooling (e.g., dotnet build with Roslyn analyzers); " +
			"native parsing is not supported",
	)
}
