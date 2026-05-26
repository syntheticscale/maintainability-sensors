# Project Status

**Last Updated:** 2026-05-26
**Branch:** `main`
**State:** ✅ Stable — Sprints 1–6 Complete

> See `docs/RADICAL_REVIEW.md` for the full audit and remaining items.

---

## Sprint 1 Summary (Hardening Sprint)

All three CRITICAL items and partial HIGH items resolved.

### CRITICAL — Resolved

1.  **Fix LSP Race Condition** ✅
2.  **Fix `hasViolations` Config Exception Bug** ✅
3.  **Fix Python Complexity Under-Reporting** ✅

### HIGH — Partially Resolved

4.  **Harden `internal/sensors` Test Coverage** ⚠️ Partial
5.  **Refactor `cmd.go` into Focused Files** ✅

---

## Sprint 2 Summary

All MEDIUM items (6–9) resolved plus audit-discovered items.

### MEDIUM — Resolved

6.  **Fix `logStderr` String-Matching Anti-Pattern** ✅
7.  **Fix Python Function Length Calculation** ✅
8.  **Complete GitHub PR Reporting** ✅
9.  **Deduplicate Skill Definitions** ✅

### Audit-Discovered Items — Resolved

10. **`.java` missing from `isValidExtension`** ✅
11. **Dead code removal** ✅
12. **Magic number extraction** ✅
13. **Consistent logging** ✅
14. **`os.Exit(1)` removal** ✅
15. **`checkWalkDirPath` path prefix hardening** ✅

---

## Sprint 3 Summary (The Great Core Deletion)

### Technical Debt Items — Resolved

16. **Complete `orchestrator.go` Dismantling** ✅
    *   **Fix:** Extracted `result.go`, `delta.go`, `metric_updater.go`, `legacy_config_parsers.go`.
17. **Brittle JS config parsing** ✅
    *   **Fix:** Removed fragile Tree-sitter AST parsing for JS configurations entirely. Now relies on robust fallback string tokenization.
18. **Naive architecture layer matching** ✅
    *   **Fix:** Replaced with robust path segment evaluation.
19. **CGO dependency** ✅
    *   **Fix:** Completely removed `go-tree-sitter` and all CGO dependencies. The core orchestrator is now a 100% statically compiled pure Go binary.

---

## Sprint 4 Summary (Structural Precision)

20. **Naive architecture layer matching** ✅
    *   **Fix:** Replaced naive `strings.Contains` layer matching with robust path segment evaluation.
21. **CLI Domain Purification** ✅
    *   **Fix:** Centralized violation evaluation logic in `evaluator.go`, removing domain leakage from HTML and PR output formatters.

---

## Sprint 5 Summary (Radical Audit Cleanup)

All P0 items and most P1 items from the `docs/RADICAL_REVIEW.md` audit resolved.

### P0 — Resolved

22. **Repo hygiene** ✅
    *   **Fix:** `git rm`'d compiled binary, runtime artifacts, generated lint configs, `.skill` archives, `dist/reports/`. Rewrote `.gitignore`.
23. **Contradictory duplication of `EffectiveLimits`** ✅
    *   **Fix:** Canonical definition in `evaluator.go`. `github.go` uses type alias + delegates. `format.go` calls `sensors.Evaluate()` directly.
24. **Contradictory config parser anchor lists** ✅
    *   **Fix:** Anchor lists consolidated to superset. `ParserRule` struct and all baseline/rule constants moved to `internal/plugin/protocol/` as single source of truth. Both `sensors` and `legacy` use type aliases.

### P1 — Resolved

25. **Silent error swallowing** ✅
    *   **Fix:** `findAllConfigValsYAML` falls back on unmarshal failure. `findAllConfigValsTOML` returns early. `parseAndCacheArchConfig` caches nil on parse error.
26. **The magic "100"** ✅
    *   **Fix:** All 6 legacy plugins now use named `FallbackEndLineOffset` constant.
27. **IPC protocol hardening** ✅
    *   **Fix:** Plugin binary existence check with clear error. `ProtocolVersion = 1` in IPC schema; core and plugin both validate version.

### P2 — Resolved

28. **Formatting bug in `run.go`** ✅
29. **`bootstrap.go` God Object (509 lines)** ✅
    *   **Fix:** Split into 7 per-language files: `bootstrap.go`, `bootstrap_go.go`, `bootstrap_python.go`, `bootstrap_tsjs.go`, `bootstrap_java.go`, `bootstrap_ruby.go`, `bootstrap_csharp.go`.

### P3 — Resolved

30. **Corrupted text in `CASE_STUDIES.md`** ✅

---

## Current Architecture (Two-Tier)

```
maintainability-sensors/
├── cmd/
│   ├── maintainability-sensors/
│   │   └── main.go                     # Core CLI entrypoint
│   └── legacy-plugin/
│       └── main.go                     # Polyglot plugin entrypoint
├── internal/
│   ├── cli/                            # Subcommands & output formatting
│   │   ├── cmd.go                      # Flag parsing & dispatch
│   │   ├── run.go                      # `run` command
│   │   ├── format.go                   # CLI & JSON formatting
│   │   ├── html.go                     # HTML scorecard generator
│   │   ├── github.go                   # GitHub Actions step summary & PR comments
│   │   ├── generate.go                 # `generate` report command
│   │   ├── violations.go              # Violation evaluation (evaluator integration)
│   │   └── bootstrap_exec.go           # `bootstrap` command dispatch
│   ├── legacy/                         # Legacy language plugins (Ruby, Python, JS/TS)
│   │   ├── constants.go               # Delegates to protocol package
│   │   ├── types.go                   # Type aliases to protocol package
│   │   └── *_plugin.go               # Per-linter subprocess parsers
│   ├── lsp/                            # Language Server Protocol (experimental)
│   │   ├── server.go
│   │   └── server_test.go
│   ├── plugin/
│   │   └── protocol/                   # IPC schema + shared domain constants
│   │       └── schema.go               # ProtocolVersion, ParserRule, baselines, rule names
│   └── sensors/
│       ├── orchestrator.go             # Agent batching & sub-process routing
│       ├── plugin_runner.go            # IPC stdin/stdout JSON engine (version-checked)
│       ├── plugin.go                   # Plugin registry
│       ├── evaluator.go               # Canonical EffectiveLimits & violation evaluation
│       ├── go_ast.go                   # Native pure-Go AST metrics
│       ├── go_architecture.go          # Native pure-Go dependency boundary rules
│       ├── architecture_parser.go      # YAML parser for dependency rules
│       ├── result.go                   # Orchestration result structures
│       ├── delta.go                    # Check-diff delta calculation
│       ├── metric_updater.go           # Metric map update helpers
│       ├── config_parsers.go           # ConfigParser interface + shared utilities
│       ├── config_detector.go          # Config file discovery
│       ├── legacy_config_parsers.go    # Per-linter config parsers (ESLint, RuboCop, PyLint, etc.)
│       ├── constants.go               # Delegates to protocol package
│       ├── bootstrap.go               # Bootstrap orchestration & language detection
│       ├── bootstrap_*.go             # Per-language template + writer (7 files)
│       ├── git_diff.go                # Git diff parsing
│       └── pathutils.go              # Path sanitization
├── skills/                             # AI Agent procedural guidelines
└── tests/                              # Component, architecture, golden, & CLI tests
```

---

## Remaining Items (from Radical Audit)

### P1 — Protocol schema bloat

`internal/plugin/protocol/schema.go` now holds domain constants (`ParserRule`, baselines, rule names, `FallbackEndLineOffset`) alongside wire types. The protocol should define message types only. These constants should live in a dedicated domain package.

### P2 — LSP server is experimental stub

3 message handlers, no `didOpen`/`shutdown`, URI parser breaks on `file:///` and Windows. README now correctly marks it as experimental. Needs either completion or continued honest labeling.

### P2 — Golden tests are fragile

Clone 5 external repos, skipped in short mode (never in CI), `-update` overwrites golden files. Need deterministic in-process tests with checked-in fixtures.

### P2 — `archConfigCache` untested

No reset helper, no cache-hit/miss tests, no concurrent access tests.

### P2 — No `Makefile`

Two-binary deployment with no build script. Plugin path is hardcoded to `./bin/legacy-plugin`.

### P3 — Skills are thin prompt wrappers

`pre-flight-check` and `performance-benchmarker` wrap basic bash commands. `modularity-reviewer` has genuine value but is thin.

### P3 — Inconsistent error wrapping

20 `fmt.Errorf` calls use `%v` instead of `%w`. Should bulk-replace.
