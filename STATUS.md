# Project Status

**Last Updated:** 2026-05-24
**Branch:** `main`
**State:** ⚠️ Pre-Stable: Undergoing Hardening Sprint

> **Do not tag a 1.0 release until all CRITICAL items below are resolved.**
> See `docs/RADICAL_REVIEW.md` for the full audit.

---

## Remediation Plan

These items are listed in strict severity order. Work top-to-bottom.

### CRITICAL — Do Not Ship 1.0 Without These

1.  **Fix LSP Race Condition**
    *   **Location:** `internal/lsp/server.go:200-242`
    *   **Problem:** The `textDocument/didChange` handler spawns an anonymous goroutine per keystroke. Multiple goroutines write to `os.Stdout` without synchronization. This interleaves JSON-RPC messages and crashes the LSP client.
    *   **Fix:** Serialize all writes to `io.Writer` with a mutex or a dedicated writer goroutine.

2.  **Fix `hasViolations` Config Exception Bug**
    *   **Location:** `internal/cli/cmd.go:488-496` vs. `799-817`
    *   **Problem:** `hasViolations` checks exception rule names using human-readable strings (e.g., `"Cyclomatic Complexity"`), while parsers return canonical names (e.g., `"Complexity"`). Files within relaxed thresholds are incorrectly flagged, causing CI to fail PRs for no reason.
    *   **Fix:** Introduce canonical rule-name constants (e.g., `RuleComplexity`, `RuleFunctionLength`) in a shared `const` block. Replace every scattered string literal across the codebase.

3.  **Fix Python Complexity Under-Reporting**
    *   **Location:** `internal/sensors/tree_sitter_python.go:101-127`
    *   **Problem:** The complexity counter intentionally ignores `try`, `with`, Boolean operators, ternary expressions, and comprehensions to keep tests simple. This gives Python developers false confidence.
    *   **Fix:** Count all decision points per the standard McCabe/Sonarsource definition. Update tests to assert *correct* values, not convenient ones.

### HIGH

4.  **Harden `internal/sensors` Test Coverage**
    *   **Current:** 25.4%
    *   **Target:** 70%+
    *   **Focus Areas:** `orchestrator.go` (plugin filtering, blind-mode fallback, chunking edge cases), config parsers (exception extraction), and plugin dispatch logic.

5.  **Refactor `cmd.go` into Focused Sub-Packages**
    *   **Location:** `internal/cli/cmd.go` (872 lines)
    *   **Problem:** Single file defines CLI structs, scanning, delta analysis, file discovery, output formatting (JSON/Markdown/HTML/table), self-correction guidance, report writing, GitHub review posting, and JSON scorecard parsing.
    *   **Fix:** Extract at minimum: `internal/cli/reports/`, `internal/cli/github/`, `internal/cli/scan/`, `internal/cli/delta/`.

### MEDIUM

6.  **Fix `logStderr` String-Matching Anti-Pattern**
    *   **Location:** `internal/cli/cmd.go:26-41`
    *   **Problem:** Quiet-mode suppression uses `strings.Contains(format, "[ERROR]")`, so properly formatted log messages can be accidentally suppressed.
    *   **Fix:** Introduce a structured `LogLevel` enum (`Debug`, `Info`, `Warn`, `Error`) and pass it explicitly.

7.  **Fix Python Function Length Calculation**
    *   **Location:** `internal/sensors/tree_sitter_python.go:72-79`
    *   **Problem:** Length is `endLine - startLine + 1`, which includes decorators and docstrings, systematically inflating the metric.
    *   **Fix:** Measure body length (from colon to end) or subtract the leading decorator/docstring span.

8.  **Complete GitHub PR Reporting**
    *   **Location:** `internal/cli/github.go:213`
    *   **Problem:** `buildPRCommentBody` only reports Complexity, Function Length, and Argument Count. It silently ignores Cognitive Complexity and Max Case Length violations.
    *   **Fix:** Include all rule types in the PR comment body generation.

9.  **Deduplicate Skill Definitions**
    *   **Location:** `skills/` and `.gemini/skills/`
    *   **Problem:** Identical `SKILL.md` files exist in both paths for all three skills.
    *   **Fix:** Make `skills/` the canonical location. Delete `.gemini/skills/` or replace with symlinks. Update `.gitignore` (it already ignores `*.skill`).

10. **Complete `orchestrator.go` Dismantling**
    *   **Location:** `internal/sensors/orchestrator.go` (452 lines)
    *   **Problem:** Still contains `OrchestratorResult`, `RelaxedLimit`, `OrchestratedScan`, `OrchestratedScanBatch`, `ScanDeltaBatch`, map-update helpers, result builders, config finders, and dead code (`updateMetric`).
    *   **Fix:** Extract into `result.go`, `delta.go`, and `metric_updater.go`. Remove dead code.

11. **Retire `docs/FUTURE_PLAN.md`**
    *   **Status:** Already removed in this commit.
    *   **Rationale:** It was a victory-lap document claiming all roadmap items were complete. It no longer served a planning purpose.

---

## Current Architecture (Two-Tier Ecosystem)

```
maintainability-sensors/
├── cmd/
│   └── maintainability-sensors/
│       └── main.go                  # CLI entrypoint
├── internal/
│   ├── cli/
│   │   ├── cmd.go                   # Subcommands & Workspace Jailing
│   │   ├── html.go                  # Statically cached HTML scorecard generator
│   │   └── github.go                # Enterprise GitHub integration
│   ├── lsp/
│   │   └── server.go                # Real-time IDE feedback server
│   └── sensors/
│       ├── plugin.go                # Core Plugin Interface & Registry
│       ├── orchestrator.go          # Argument Chunking & Plugin Invocation
│       ├── go_ast.go                # Native Go Plugin
│       ├── tree_sitter_python.go    # Native Python Plugin (Tree-sitter)
│       ├── tree_sitter_typescript.go # Native TS/JS Plugin (Tree-sitter)
│       ├── csharp_parser.go         # Native C# Plugin (Tree-sitter)
│       ├── java_parser.go           # Native Java Plugin (Tree-sitter)
│       ├── eslint_plugin.go         # (Tier 2 wrappers implement Plugin interface)
│       └── bootstrap.go             # Enterprise-safe config generator
├── skills/
│   ├── modularity-reviewer/         # Tier 2 Agent Skill (Semantic Review)
│   ├── pre-flight-check/            # Tier 2 Agent Skill (Enforces Checks)
│   └── performance-benchmarker/     # Tier 2 Agent Skill (Empirical NFRs)
└── tests/
    └── golden_test.go               # Validates formatted LLM prompts
```

---

## Known Issues Not Yet Addressed

These are historical items from prior audits that have not been assigned to the sprint above.

*   **CGO Dependency:** `go-tree-sitter` broke the "Minimal External Dependencies" constraint. Cross-compilation now requires a C compiler. Consider vendoring pre-built C libs or documenting this clearly.
*   **Brittle Parsing:** `internal/sensors/config_parsers.go` uses massive, fragile regular expressions to parse JavaScript (`.eslintrc.js`) configuration files.
*   **Naive Architecture Matching:** Layer matching in `CheckArchitectureDependencies` uses `strings.Contains(absPath, "/"+layerName+"/")`, which will easily yield false positives if a folder happens to share a name with a layer.
