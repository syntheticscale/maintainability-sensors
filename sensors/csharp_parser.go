package sensors

import (
	"os"
	"regexp"
	"strings"
)

// CSharpMetrics holds the parsed metrics for a C# file.
type CSharpMetrics struct {
	Complexity     int
	FunctionLength int
	ArgumentCount  int
}

// ParseCSharp parses a C# file natively using structural bracket-tracking.
func ParseCSharp(filePath string) (CSharpMetrics, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return CSharpMetrics{}, err
	}
	return AnalyzeCSharpSource(string(data)), nil
}

// AnalyzeCSharpSource computes complexity, function lengths, and parameter counts natively.
func AnalyzeCSharpSource(content string) CSharpMetrics {
	var metrics CSharpMetrics

	lines := strings.Split(content, "\n")
	
	// Pre-compile Regex to find potential method signatures:
	// Access modifier, return type, name, parenthesized parameters, and optional open brace
	reMethod := regexp.MustCompile(`(?:public|private|protected|internal|static|async|virtual|override|readonly)?\s+[\w<>[\]]+\s+(\w+)\s*\(([^)]*)\)`)

	inMethod := false
	methodDepth := 0
	currentMethodLines := 0
	currentMethodComplexity := 1

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if inMethod {
				currentMethodLines++
			}
			continue
		}

		// Track braces to determine boundaries
		for i := 0; i < len(trimmed); i++ {
			char := trimmed[i]
			if char == '{' {
				if inMethod {
					methodDepth++
				} else {
					// Check if this brace opens a newly detected method
					// Standard C# methods are declared outside and open with '{'
					// If we matched a method signature recently, we enter method context
					// We'll use a simpler heuristic: if depth is at method entry level
				}
			} else if char == '}' {
				if inMethod {
					methodDepth--
					if methodDepth == 0 {
						// Method ended! Save metrics if they are maximums
						if currentMethodLines > metrics.FunctionLength {
							metrics.FunctionLength = currentMethodLines
						}
						if currentMethodComplexity > metrics.Complexity {
							metrics.Complexity = currentMethodComplexity
						}
						inMethod = false
					}
				}
			}
		}

		if inMethod {
			currentMethodLines++
			
			// Calculate Cyclomatic Complexity within method body
			// Increment on branching keywords and logical operators:
			// if, while, for, foreach, catch, case, ??, &&, ||, ? (ternary)
			currentMethodComplexity += countKeywords(trimmed)
		} else {
			// Check if this line is a method definition declaration
			if matches := reMethod.FindStringSubmatch(trimmed); matches != nil {
				// Ignore control flow keywords that look like methods (e.g. if, while, switch)
				methodName := matches[1]
				if methodName != "if" && methodName != "while" && methodName != "switch" && methodName != "for" && methodName != "foreach" && methodName != "catch" {
					inMethod = true
					methodDepth = 1
					currentMethodLines = 1
					currentMethodComplexity = 1

					// Parse parameter count
					paramsStr := strings.TrimSpace(matches[2])
					paramCount := 0
					if paramsStr != "" {
						// Count commas to deduce parameters: count = commas + 1
						paramCount = strings.Count(paramsStr, ",") + 1
					}
					if paramCount > metrics.ArgumentCount {
						metrics.ArgumentCount = paramCount
					}
				}
			}
		}
	}

	return metrics
}

func countKeywords(line string) int {
	count := 0
	
	// Skip comments
	if strings.HasPrefix(strings.TrimSpace(line), "//") || strings.HasPrefix(strings.TrimSpace(line), "/*") {
		return 0
	}

	// Tokenize/search for branching statements
	keywords := []string{"if(", "if ", "while(", "while ", "for(", "for ", "foreach(", "foreach ", "case ", "catch(", "catch ", "&&", "||", "??", " ? "}
	for _, kw := range keywords {
		count += strings.Count(line, kw)
	}

	return count
}
