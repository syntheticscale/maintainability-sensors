# Maintainability Sensors — Implementation Plan

**Updated:** 2026-05-20  
**State:** Milestone "Architecture & Modernization Rewrite" Complete

---

## 🏆 Sprint 1: Architecture & Modernization (Completed)

All architectural flaws and outstanding technical debt from the initial release have been addressed. The CLI is now robust, testable, and ready for modern polyglot environments.

| Task | Outcome | Delivered |
|---|---|---|
| **CLI Monolith Refactor** | `executeRun` split into a clean `FindFiles`, `ScanFiles`, `FormatResultsCLI` pipeline. | ✅ |
| **Polyglot Bootstrapping** | `BootstrapRepo` now detects and configures all languages in a monorepo, rather than just the majority language. | ✅ |
| **Native Config Parser** | Replaced fragile line-oriented regex with a native, standard library-only stack-based YAML/INI parser for `.rubocop.yml`, `.pylintrc`, etc. | ✅ |
| **ESLint 9 Flat Config** | Added robust parsing for `eslint.config.js` and `eslint.config.mjs` flat configuration structures. | ✅ |
| **Input Validation** | Added strict schema validation to the `generate` subcommand to prevent cryptic JSON unmarshal crashes. | ✅ |
| **Golden Snapshots** | Regenerated test snapshots against active local linters to capture real, deterministic code metrics. | ✅ |
| **CLI & Subprocess Tests**| Extensive test suites for CLI package commands, subprocess error boundaries, and environment fallbacks. | ✅ |
| **Ecosystem Modernization** | Added batched subprocess execution and native parsers for Biome, Ruff, and StandardRB. | ✅ |
| **Visionary Features** | Added `baseline` command for legacy debt suppression and inline GitHub PR review comments. | ✅ |

---

## 🎯 Future Explorations (Unscheduled)

With the core architecture stabilized, future work will focus on expanding native parsing capabilities and keeping up with the evolving static analysis ecosystem.

1. **Native C# AST Parsing:** Investigate utilizing native Go ports of Roslyn (if they ever exist) or cross-compiling analyzers to remove the reliance on the host `.NET` SDK.
2. **Native Java AST Parsing:** Explore parsing Java ASTs in Go to drop the `Checkstyle` dependency.
3. **Modernize Templates:** Periodically review and update the `.golangci.yml` and `.eslintrc.json` templates to ensure they align with the latest community best practices.

---

## How to cut a release

```bash
# 1. Run full suite
go test -count=1 -race ./...

# 2. Build binary
go build -o bin/maintainability-sensors main.go

# 3. Verify no linting issues
go vet ./...

# 4. Commit
git add -A && git commit -m "chore: prepare release <version>"
```