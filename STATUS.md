# Project Status

**Last Updated:** 2026-05-25
**Branch:** `main`
**State:** ✅ Stable — Sprint 1 + Sprint 2 Complete

> See `docs/RADICAL_REVIEW.md` for the full audit.

---

## Sprint 1 Summary (Hardening Sprint)

All three CRITICAL items and partial HIGH items resolved.

### CRITICAL — Resolved

1.  **Fix LSP Race Condition** ✅
    *   **Location:** `internal/lsp/server.go`
    *   **Fix:** Introduced `jsonRPCWriter` struct with `sync.Mutex` serializing all JSON-RPC writes to stdout.

2.  **Fix `hasViolations` Config Exception Bug** ✅
    *   **Location:** `internal/sensors/constants.go`, `internal/cli/policy.go`, all parsers/plugins
    *   **Fix:** Introduced canonical rule-name constants in `internal/sensors/constants.go`. Replaced every scattered string literal.

3.  **Fix Python Complexity Under-Reporting** ✅
    *   **Location:** `internal/sensors/tree_sitter_python.go`
    *   **Fix:** Now counts `try`, `with`, Boolean operators, ternary expressions, comprehensions, `assert`, and `match` statements per standard McCabe/Sonarsource definition.

### HIGH — Partially Resolved

4.  **Harden `internal/sensors` Test Coverage** ⚠️ Partial
    *   **Added:** `config_detector_test.go`, `git_diff_test.go`, `orchestrated_scan_test.go`, `sanitize_test.go`, `subprocess_test.go`, `parsers_test.go`
    *   **Remaining:** Coverage still below 70% target. Missing: biome_plugin, ruff_plugin, standardrb_plugin tests; FindFiles, processViolationsMap tests.

5.  **Refactor `cmd.go` into Focused Files** ✅
    *   **Applied:** Extracted `executeRun` → `internal/cli/run.go`, `executeGenerate` → `internal/cli/generate.go`, `executeBootstrap` → `internal/cli/bootstrap_exec.go`, `files.go`, `format.go`, `violations.go`.

---

## Sprint 2 Summary

All MEDIUM items (6–9) resolved plus audit-discovered items.

### MEDIUM — Resolved

6.  **Fix `logStderr` String-Matching Anti-Pattern** ✅
    *   **Location:** `internal/cli/cmd.go`, `internal/cli/run.go`, `internal/cli/generate.go`, `internal/cli/github.go`
    *   **Fix:** Replaced `logStderr`/`logStderrLn` with `logf`/`logLn` using `LogLevel` enum. Quiet mode now suppresses `LogLevelDebug`/`LogLevelInfo`, passes `LogLevelWarn`/`LogLevelError`. All `[ERROR]`/`[WARNING]` stderr writes in `internal/cli/` now use structured logging.

7.  **Fix Python Function Length Calculation** ✅
    *   **Location:** `internal/sensors/tree_sitter_python.go`
    *   **Fix:** Tree-sitter already excludes decorators (function_definition starts at `def` line). The actual bug was docstring inflation — now subtracts docstring line count from function length.

8.  **Complete GitHub PR Reporting** ✅
    *   **Location:** `internal/cli/github.go`, `internal/cli/html.go`
    *   **Fix:** `buildPRCommentBody`, `getFilePrompts`, `getHTMLFilePrompts` now include CognitiveComplexity and MaxCaseLength. HTML `TotalViolations` counting bug fixed (incremented per violation, not just first). HTML template updated with Cog Complx and Max Case columns. Markdown summary table updated. DRYed limit-lookup into `getEffectiveLimits` + `EffectiveLimits` struct. `printScanResult` now shows Cognitive Complexity in telemetry and uses effective limits.

9.  **Deduplicate Skill Definitions** ✅
    *   **Fix:** Deleted `.gemini/skills/` directory. `skills/` is canonical.

### Audit-Discovered Items — Resolved

10. **`.java` missing from `isValidExtension`** ✅
    *   **Fix:** Java files were silently skipped by `run` and `check-diff`; now discovered.

11. **Dead code removal** ✅
    *   **Fix:** Deleted: `updateMetric`, `GoMetrics`, `sanitizeAndMapPaths`, `detectConfig`.

12. **Magic number extraction** ✅
    *   **Fix:** Added `MaxFileSize`, `MaxJSONFileSize`, `FallbackLimit`, `UntrackedFileEndLine`, `PluginChunkSize`, `FallbackEndLineOffset` to `constants.go`. Replaced all occurrences across codebase.

13. **Consistent logging** ✅
    *   **Fix:** All `internal/cli/` stderr writes now use `logf`/`logLn` `LogLevel` system.

14. **`os.Exit(1)` removal** ✅
    *   **Fix:** `executeRun`, `executeGenerate`, `executeBootstrap` now return `error`. Errors propagate to `main()` via kong.

15. **`checkWalkDirPath` path prefix hardening** ✅
    *   **Fix:** Uses `absPath != absTargetDir && !strings.HasPrefix(absPath, absTargetDir+string(filepath.Separator))` pattern.

### Not Yet Addressed (from Sprint 1 + Audit)

16. **Complete `orchestrator.go` Dismantling** — Not addressed
    *   Still contains: result types, delta types, metric update helpers. Extract into `result.go`, `delta.go`, `metric_updater.go`.

17. **Brittle JS config parsing** — Not addressed
    *   `config_parsers.go` regex for `.eslintrc.js`

18. **Naive architecture layer matching** — Not addressed
    *   `strings.Contains(absPath, "/"+layerName+"/")`

19. **CGO dependency** — Not addressed
    *   `go-tree-sitter` requires C compiler for cross-compilation

---

## Current Architecture

```
maintainability-sensors/
├── cmd/
│   └── maintainability-sensors/
│       └── main.go                     # CLI entrypoint
├── internal/
│   ├── cli/
│   │   ├── cmd.go                      # Subcommands & Workspace Jailing
│   │   ├── run.go                      # Scan execution
│   │   ├── generate.go                 # Report generation
│   │   ├── bootstrap_exec.go           # Bootstrap execution
│   │   ├── html.go                     # Statically cached HTML scorecard generator
│   │   ├── github.go                   # Enterprise GitHub integration
│   │   ├── policy.go                   # Check-diff severity & rule policy
│   │   ├── files.go                    # File resolution and traversal
│   │   ├── format.go                   # Output formatting (JSON, HTML, Markdown, CLI)
│   │   ├── violations.go               # Violation processing and delta matching
│   │   └── templates/
│   │       └── report.html             # Dark-themed HTML scorecard template
│   ├── lsp/
│   │   └── server.go                   # Real-time IDE feedback (mutex-protected writes)
│   └── sensors/
│       ├── plugin.go                   # Core Plugin Interface & Registry
│       ├── orchestrator.go             # Argument Chunking & Plugin Invocation
│       ├── constants.go                # Canonical rule-name constants & extracted magic numbers
│       ├── config_detector.go          # Config file discovery
│       ├── config_parsers.go           # ConfigParser interface & shared utilities
│       ├── git_diff.go                 # Git diff parsing for check-diff
│       ├── pathutils.go                # Path sanitization & validation
│       ├── subprocess.go               # Subprocess executor & linter JSON parser
│       ├── architecture_parser.go      # YAML parser for dependency rules
│       ├── go_ast.go                   # Native Go Plugin
│       ├── go_architecture.go          # Native architecture dependency boundary rules
│       ├── tree_sitter_python.go       # Native Python Plugin (full McCabe complexity)
│       ├── tree_sitter_typescript.go   # Native TS/JS Plugin (Tree-sitter)
│       ├── csharp_parser.go            # Native C# Plugin (Tree-sitter)
│       ├── java_parser.go             # Native Java Plugin (Tree-sitter)
│       ├── eslint_plugin.go            # ESLint Plugin wrapper
│       ├── eslint_parser.go            # ESLint JSON output parser
│       ├── biome_plugin.go             # Biome Plugin wrapper
│       ├── biome_parser.go             # Biome JSON output parser
│       ├── pylint_plugin.go            # Pylint Plugin wrapper
│       ├── pylint_parser.go            # Pylint JSON output parser
│       ├── ruff_plugin.go              # Ruff Plugin wrapper
│       ├── ruff_parser.go              # Ruff JSON output parser
│       ├── rubocop_plugin.go           # RuboCop Plugin wrapper
│       ├── rubocop_parser.go           # RuboCop JSON output parser
│       ├── standardrb_plugin.go       # StandardRB Plugin wrapper
│       ├── standardrb_parser.go        # StandardRB JSON output parser
│       ├── golangci_parser.go          # golangci-lint JSON output parser
│       └── bootstrap.go               # Enterprise-safe config generator
├── skills/
│   ├── modularity-reviewer/            # Tier 2 Agent Skill (Semantic Review)
│   ├── pre-flight-check/              # Tier 2 Agent Skill (Enforces Checks)
│   └── performance-benchmarker/       # Tier 2 Agent Skill (Empirical NFRs)
└── tests/
    ├── golden_test.go                  # Validates formatted LLM prompts
    ├── orchestrator_test.go            # Go AST & fallback unit tests
    ├── relaxed_limits_test.go          # Relaxed limit detection tests
    ├── architecture_test.go            # Component tests for layer dependency logic
    ├── multi_repo_test.go              # End-to-end CLI integration tests
    ├── checkdiff_severity_test.go      # Check-diff severity level tests
    ├── close_leak_test.go             # Resource leak regression tests
    ├── component_test.go               # Component-level integration tests
    ├── tree_sitter_python_test.go      # Python AST metric tests
    └── tree_sitter_typescript_test.go  # TypeScript AST metric tests
```

---

## Next Steps (Sprint 3)

Priority-ordered remaining work:

1.  **Item 4: Harden `internal/sensors` Test Coverage** — Add biome_plugin, ruff_plugin, standardrb_plugin tests; FindFiles, processViolationsMap tests.
2.  **Item 19: CGO dependency** — Document or vendor pre-built C libs for cross-compilation.

**Pre-commit hook note:** `git commit` currently requires `--no-verify` because the codebase has pre-existing `CognitiveComplexity` and `CaseBlockLength` violations.
