# Project Status

**Last Updated:** 2026-05-26
**Branch:** `main`
**State:** ✅ Stable — Sprint 1, 2, and 3 Complete

> See `docs/RADICAL_REVIEW.md` for the full audit.

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
18. **Naive architecture layer matching** — Not addressed
    *   `strings.Contains(absPath, "/"+layerName+"/")` is still used in `go_architecture.go`.
19. **CGO dependency** ✅
    *   **Fix:** Completely removed `go-tree-sitter` and all CGO dependencies. The core orchestrator is now a 100% statically compiled pure Go binary. Legacy AST parsers (`java`, `csharp`, `python`, `typescript`) were deleted from the core and moved/replaced by the legacy-plugin architecture.

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
│   ├── cli/                            # Subcommands & Output formatting
│   ├── legacy/                         # Legacy language plugins (Ruby, Python, JS/TS)
│   ├── lsp/                            # Language Server Protocol integration
│   ├── plugin/
│   │   └── protocol/                   # JSON standard I/O IPC schema
│   └── sensors/
│       ├── orchestrator.go             # Agent batching & sub-process routing
│       ├── plugin_runner.go            # IPC stdin/stdout JSON engine
│       ├── go_ast.go                   # Native pure-Go AST metrics
│       ├── go_architecture.go          # Native pure-Go dependency boundary rules
│       └── result.go                   # Orchestration result structures
├── skills/                             # AI Agent procedural guidelines
└── tests/                              # Golden snapshots, orchestrator & CLI tests
```

---

## Next Steps (Sprint 4)

1. **Naive architecture layer matching**
   - Address the string matching logic in `go_architecture.go`.
