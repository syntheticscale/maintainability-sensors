# Maintainability Sensors — Implementation Plan

**Updated:** 2026-05-26
**State:** Milestone "Two-Tier Architecture Refactor" Complete

---

## 🏆 Completed Sprints

All architectural flaws and outstanding technical debt from the initial release and the Radical Audit have been addressed across 5 consecutive sprints. The CLI is now robust, testable, and completely statically compiled (zero CGO).

| Sprint | Focus | Outcome |
|---|---|---|
| **Sprint 1** | Hardening & Bug Fixes | Fixed LSP race conditions, rule-name mismatches, and Python AST metric bugs. |
| **Sprint 2** | UX & Output Quality | Fixed deceptive log-matching, improved function-length accuracy, and enabled full GitHub PR comments. |
| **Sprint 3** | The Great Core Deletion | Dismantled `orchestrator.go`, purged the `go-tree-sitter` CGO dependency, and pivoted to a Two-Tier plugin architecture. |
| **Sprint 4** | Structural Precision | Replaced naive layer matching strings with robust path segment evaluation. |
| **Sprint 5** | CLI Domain Purification | Centralized the violation evaluation logic, removing domain leakage from HTML and PR output formatters. |

---

## 🎯 Future Explorations

> See `STATUS.md` for the up-to-date roadmap and active tracking.

With the core architecture stabilized and the Two-Tier IPC plugin model established, future work will focus on:
1. **Expanding the Legacy Plugin:** Adding support for more languages (e.g., Rust, Kotlin) by bolting new subprocess linters onto the standalone legacy plugin without needing to recompile the core Go CLI.
2. **Modernize Templates:** Periodically review and update the `.golangci.yml` and `.eslintrc.json` templates to ensure they align with the latest community best practices.

---

## How to cut a release

```bash
# 1. Run full suite
go test -count=1 -race ./...

# 2. Build binaries
go build -o bin/maintainability-sensors ./cmd/maintainability-sensors
go build -o bin/legacy-plugin ./cmd/legacy-plugin

# 3. Verify no linting issues
go vet ./...

# 4. Commit
git add -A && git commit -m "chore: prepare release <version>"
```