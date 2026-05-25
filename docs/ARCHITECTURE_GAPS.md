# Architectural Gaps & Technical Debt

This document tracks the architectural flaws and incorrect trade-offs identified during the deep review, which prioritized development speed over robustness and correctness. 

## 1. [RESOLVED] Subprocess OOM Risks (`sensors/orchestrator.go`)
- **The Flaw:** The tool buffered the *entire* output stream of orchestrated linters (ESLint, PyLint) into memory using `CombinedOutput()` and then parsed it directly via `json.Unmarshal`. For large enterprise monorepos, this could easily cause an Out-of-Memory (OOM) crash.
- **The Fix:** Streamed `stdout` into an `io.Reader` and parsed using `json.NewDecoder().Decode()`.

## 2. Regex Parsing of English Linter Output (`sensors/orchestrator.go`) — ⚠️ Partially Addressed
- **The Flaw:** To extract metrics, the orchestrator runs regex against the *English text* of the linter warnings (e.g., `regexp.MustCompile("complexity of (\\d+)")`). If maintainers change their error string formatting, the tool breaks silently.
- **The Worst Offense:** For **Biome**, the code hardcodes a dummy value of `2` for violations. This completely sacrifices accuracy.
- **Current State:** Magic numbers (`100` endLine offset, `2*1024*1024` file size limit) extracted into named constants. Biome dummy value and regex-based extraction remain. Structured JSON output parsing (where available) is the long-term fix.

## 3. [RESOLVED] Inaccurate Go AST Complexity Measurement (`sensors/go_ast.go`)
- **The Flaw:** The native Go AST parser used `ast.Inspect` to traverse all nodes recursively indefinitely. If a function contained a nested closure (`func() { ... }`), the complexity of the inner closure was added directly to the outer parent function's score.
- **The Fix:** Traversal updated to not bleed complexity scores from inner `*ast.FuncLit` nodes into the parent function.

## 4. [RESOLVED] Performance Penalty in Report Generation (`cli/html.go`)
- **The Flaw:** The HTML template was re-compiled via `template.New("report").Parse(...)` on every single run.
- **The Fix:** Cached statically using `template.Must(...)` at package initialization.

## 5. [RESOLVED] Fragile File Path Tracking (`sensors/orchestrator.go`)
- **The Flaw:** It used `strings.HasSuffix(outAbs, cleanPath)` to map linter results back to files, which could cause false positives (e.g., matching `my_util.go` to `util.go`).
- **The Fix:** Updated to use exact path matching and proper path comparison logic.

---

## Strategic Gaps (from the Original Concept)

Based on Birgitta Böckeler’s original "Maintainability sensors for coding agents" article, our repository has mastered **Micro-Level "Computational" Sensors** (linting, function size, complexity), but lacks visibility into Macro-Level architecture and semantic intent.

### Gap 1: Architectural & Dependency Rules
*   **The Flaw:** We only measure *internal* function complexity. An AI can write a perfectly simple 5-line function that illegally imports a database package directly into a UI component, and our sensors will report `Delta clean`.
*   **Effort Estimate: Medium (2-3 Days)**
*   **Implementation Path:** We would integrate tools like `dependency-cruiser` (JS/TS) or `golangci-lint`'s `depguard` to enforce strict architectural boundaries (e.g., API layer cannot depend on DB orchestration). The orchestration logic already supports external linters, so we would just need to map output violations to our `OrchestratorResult`.

### Gap 2: Fan-in / Fan-out Coupling Metrics
*   **The Flaw:** We are entirely blind to the "blast radius" of a code change. A high-complexity function with 0 dependents is often safer for an AI to modify than a simple 5-line function called by 500 other modules.
*   **Effort Estimate: Large (1-2 Weeks)**
*   **Implementation Path:** This requires parsing the entire dependency graph. For Go, we could extend `sensors/go_ast.go` to count exported symbols and their usages across the workspace. For other languages, we would need to integrate a robust static analysis graph tool. It requires fundamental changes to the `MaintainabilityMetrics` struct to capture graph-level metrics.

### Gap 3: Semantic "Inferential" Modularity Review
*   **The Flaw:** Our tool stops at syntax. An AI can duplicate the exact same business logic in two different files with different variable names, and mathematical linters will never notice.
*   **Effort Estimate: Very Large (Research Phase)**
*   **Implementation Path:** We would need to build an "LLM-as-a-Judge" sensor. This involves orchestrating a secondary AI pass (using a framework like LangChain or direct LLM API calls) that reads the `git diff` alongside the full file context, specifically prompting it to look for Semantic Duplication, Inefficient Arguments, and Misplaced Responsibilities.

### Gap 4: Mutation Testing (Test Quality Sensors)
*   **The Flaw:** AI agents are notoriously good at writing tests that achieve 100% line coverage but actually assert nothing. Our sensors currently validate code structure, but not test quality.
*   **Effort Estimate: Medium-Large (1 Week)**
*   **Implementation Path:** We would orchestrate mutation testing tools (like `go-mutesting` or `Stryker` for JS). The tool would run in the background, mutating the lines of code the AI just changed, and verifying that the AI's tests actually fail when bugs are injected.
