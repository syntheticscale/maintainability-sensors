# Architectural Gaps & Technical Debt

This document tracks the architectural flaws and incorrect trade-offs identified during the deep review, which prioritized development speed over robustness and correctness. 

## 1. [RESOLVED] Subprocess OOM Risks (`sensors/orchestrator.go`)
- **The Flaw:** The tool buffered the *entire* output stream of orchestrated linters (ESLint, PyLint) into memory using `CombinedOutput()` and then parsed it directly via `json.Unmarshal`. For large enterprise monorepos, this could easily cause an Out-of-Memory (OOM) crash.
- **The Fix:** Streamed `stdout` into an `io.Reader` and parsed using `json.NewDecoder().Decode()`.

## 2. Regex Parsing of English Linter Output (`sensors/orchestrator.go`)
- **The Flaw:** To extract metrics, the orchestrator runs regex against the *English text* of the linter warnings (e.g., `regexp.MustCompile("complexity of (\\d+)")`). If maintainers change their error string formatting, the tool breaks silently.
- **The Worst Offense:** For **Biome**, the code hardcodes a dummy value of `2` for violations. This completely sacrifices accuracy.
- **The Fix:** Parse the structured JSON output properties (or equivalent error codes/data fields) instead of relying on regex against message strings.

## 3. [RESOLVED] Inaccurate Go AST Complexity Measurement (`sensors/go_ast.go`)
- **The Flaw:** The native Go AST parser used `ast.Inspect` to traverse all nodes recursively indefinitely. If a function contained a nested closure (`func() { ... }`), the complexity of the inner closure was added directly to the outer parent function's score.
- **The Fix:** Traversal updated to not bleed complexity scores from inner `*ast.FuncLit` nodes into the parent function.

## 4. [RESOLVED] Performance Penalty in Report Generation (`cli/html.go`)
- **The Flaw:** The HTML template was re-compiled via `template.New("report").Parse(...)` on every single run.
- **The Fix:** Cached statically using `template.Must(...)` at package initialization.

## 5. [RESOLVED] Fragile File Path Tracking (`sensors/orchestrator.go`)
- **The Flaw:** It used `strings.HasSuffix(outAbs, cleanPath)` to map linter results back to files, which could cause false positives (e.g., matching `my_util.go` to `util.go`).
- **The Fix:** Updated to use exact path matching and proper path comparison logic.
