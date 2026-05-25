# Project Status

**Last Updated:** 2026-05-25
**Branch:** `hardening-sprint-final`
**State:** ✅ Stable — Hardening Sprint Complete (Critical Items Resolved)

> See `docs/RADICAL_REVIEW.md` for the full audit.

---

## Hardening Sprint Summary (2026-05-25)

All three CRITICAL items and partial HIGH items have been resolved. The codebase is now safe to gate CI pipelines.

### CRITICAL — Resolved

1.  **Fix LSP Race Condition** ✅
    *   **Location:** `internal/lsp/server.go`
    *   **Fix Applied:** Introduced `jsonRPCWriter` struct with `sync.Mutex` serializing all JSON-RPC writes to stdout. Concurrent `didChange` goroutines no longer interleave messages.

2.  **Fix `hasViolations` Config Exception Bug** ✅
    *   **Location:** `internal/sensors/constants.go`, `internal/cli/policy.go`, all parsers/plugins
    *   **Fix Applied:** Introduced canonical rule-name constants (`RuleComplexity`, `RuleFunctionLength`, `RuleArgumentCount`, `RuleCognitiveComplexity`, `RuleCaseBlockLength`) in `internal/sensors/constants.go`. Replaced every scattered string literal across all parsers, plugins, CLI, and LSP code.

3.  **Fix Python Complexity Under-Reporting** ✅
    *   **Location:** `internal/sensors/tree_sitter_python.go`
    *   **Fix Applied:** Now counts `try`, `with`, Boolean operators (`and`/`or`), ternary expressions, comprehensions, `assert`, and `match` statements per the standard McCabe/Sonarsource definition. Tests updated to assert correct values.

### HIGH — Partially Resolved

4.  **Harden `internal/sensors` Test Coverage** ⚠️ Partial
    *   **Added:** `config_detector_test.go`, `git_diff_test.go`, `orchestrated_scan_test.go`
    *   **Remaining:** Coverage still below 70% target. Orchestrator plugin filtering and chunking edge cases need more tests.

5.  **Refactor `cmd.go` into Focused Files** ⚠️ Partial
    *   **Applied:** Extracted `executeRun` → `internal/cli/run.go`, `executeGenerate` → `internal/cli/generate.go`, `executeBootstrap` → `internal/cli/bootstrap_exec.go`
    *   **Remaining:** Further sub-package extraction (`reports/`, `github/`, `scan/`, `delta/`) not yet done.

### MEDIUM — Not Yet Addressed

6.  **Fix `logStderr` String-Matching Anti-Pattern**
    *   **Location:** `internal/cli/cmd.go:26-41`
    *   **Problem:** Quiet-mode suppression uses `strings.Contains(format, "[ERROR]")`, so properly formatted log messages can be accidentally suppressed.
    *   **Fix:** Introduce a structured `LogLevel` enum (`Debug`, `Info`, `Warn`, `Error`) and pass it explicitly.

7.  **Fix Python Function Length Calculation**
    *   **Location:** `internal/sensors/tree_sitter_python.go`
    *   **Problem:** Length is `endLine - startLine + 1`, which includes decorators and docstrings, systematically inflating the metric.
    *   **Fix:** Measure body length (from colon to end) or subtract the leading decorator/docstring span.

8.  **Complete GitHub PR Reporting**
    *   **Location:** `internal/cli/github.go:213`
    *   **Problem:** `buildPRCommentBody` only reports Complexity, Function Length, and Argument Count. It silently ignores Cognitive Complexity and Max Case Length violations.
    *   **Fix:** Include all rule types in the PR comment body generation.

9.  **Deduplicate Skill Definitions**
    *   **Location:** `skills/` and `.gemini/skills/`
    *   **Problem:** Identical `SKILL.md` files exist in both paths for all three skills.
    *   **Fix:** Make `skills/` the canonical location. Delete `.gemini/skills/` or replace with symlinks.

10. **Complete `orchestrator.go` Dismantling**
    *   **Location:** `internal/sensors/orchestrator.go`
    *   **Problem:** Still contains `OrchestratorResult`, `RelaxedLimit`, `OrchestratedScan`, `OrchestratedScanBatch`, `ScanDeltaBatch`, map-update helpers, result builders, config finders, and dead code (`updateMetric`).
    *   **Fix:** Extract into `result.go`, `delta.go`, and `metric_updater.go`. Remove dead code.

11. **Retire `docs/FUTURE_PLAN.md`** ✅
    *   **Status:** Already removed.
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
│   │   ├── run.go                   # Scan execution (extracted from cmd.go)
│   │   ├── generate.go              # Report generation (extracted from cmd.go)
│   │   ├── bootstrap_exec.go        # Bootstrap execution (extracted from cmd.go)
│   │   ├── html.go                  # Statically cached HTML scorecard generator
│   │   ├── github.go                # Enterprise GitHub integration
│   │   └── policy.go                # Check-diff severity & rule policy
│   ├── lsp/
│   │   └── server.go                # Real-time IDE feedback (mutex-protected writes)
│   └── sensors/
│       ├── plugin.go                # Core Plugin Interface & Registry
│       ├── orchestrator.go          # Argument Chunking & Plugin Invocation
│       ├── constants.go             # Canonical rule-name constants
│       ├── go_ast.go                # Native Go Plugin
│       ├── tree_sitter_python.go    # Native Python Plugin (full McCabe complexity)
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
    ├── golden_test.go               # Validates formatted LLM prompts
    ├── orchestrator_test.go         # Go AST & fallback unit tests
    └── relaxed_limits_test.go       # Relaxed limit detection tests
```

---

## Known Issues Not Yet Addressed

These are historical items from prior audits that have not been assigned to the sprint above.

*   **CGO Dependency:** `go-tree-sitter` broke the "Minimal External Dependencies" constraint. Cross-compilation now requires a C compiler. Consider vendoring pre-built C libs or documenting this clearly.
*   **Brittle Parsing:** `internal/sensors/config_parsers.go` uses massive, fragile regular expressions to parse JavaScript (`.eslintrc.js`) configuration files.
*   **Naive Architecture Matching:** Layer matching in `CheckArchitectureDependencies` uses `strings.Contains(absPath, "/"+layerName+"/")`, which will easily yield false positives if a folder happens to share a name with a layer.
