# Maintainability Sensors — Implementation Plan

**Updated:** 2026-05-26
**State:** Sprint 6 (Radical Audit Cleanup) Complete

---

## 🏆 Completed Sprints

| Sprint | Focus | Outcome |
|---|---|---|
| **Sprint 1** | Hardening & Bug Fixes | Fixed LSP race conditions, rule-name mismatches, and Python AST metric bugs. |
| **Sprint 2** | UX & Output Quality | Fixed deceptive log-matching, improved function-length accuracy, and enabled full GitHub PR comments. |
| **Sprint 3** | The Great Core Deletion | Dismantled `orchestrator.go`, purged the `go-tree-sitter` CGO dependency, and pivoted to a Two-Tier plugin architecture. |
| **Sprint 4** | Structural Precision | Replaced naive layer matching strings with robust path segment evaluation. |
| **Sprint 5** | CLI Domain Purification | Centralized the violation evaluation logic in `evaluator.go`, removing domain leakage from HTML and PR output formatters. |
| **Sprint 6** | Radical Audit Cleanup | Resolved all P0/P1 items from radical audit: repo hygiene, EffectiveLimits duplication, config parser contradictions, silent error swallowing, magic numbers, IPC hardening, bootstrap God Object split. |

---

## 🎯 Future Explorations

> See `STATUS.md` for the up-to-date roadmap and active tracking.

With the core architecture stabilized and the Two-Tier IPC plugin model established, future work will focus on:
1. **Protocol Schema Cleanup:** Move domain constants out of `internal/plugin/protocol/schema.go` into a dedicated domain package. The protocol should define wire types only.
2. **Expanding the Legacy Plugin:** Adding support for more languages (e.g., Rust, Kotlin) by bolting new subprocess linters onto the standalone legacy plugin without needing to recompile the core Go CLI.
3. **Modernize Templates:** Periodically review and update the `.golangci.yml` and `.eslintrc.json` templates to ensure they align with the latest community best practices.
4. **Pure-Go TypeScript Parser:** Investigate or build a pure-Go AST parser for TypeScript/JavaScript. This would allow TS/JS to be promoted back to a Tier 1 native sensor, eliminating the overhead of the Node.js/ESLint legacy plugin subprocess without re-introducing CGO.
5. **LSP Server:** Either finish the LSP implementation (add `didOpen`, `shutdown`, proper URI parsing, incremental sync) or document it as experimental and remove production claims.
6. **Golden Test Reform:** Replace external-repo golden tests with deterministic in-process tests using checked-in fixture files.
7. **Build Tooling:** Add a `Makefile` with `build`, `test`, and `install` targets for the two-binary deployment.

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
