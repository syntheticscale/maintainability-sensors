# Maintainability Sensors — Agent Operational Protocol 📡

This document defines the high-assurance operational standards and workflows for the `maintainability-sensors` CLI repository. 

Every AI assistant or developer working on this codebase must respect these rules.

---

## 🏛️ Repository Architecture

This tool is designed to be a lightweight, zero-dependency, ultra-fast Go CLI utility (<15ms execution time) that orchestrates local static analysis and parses ASTs natively.

```
/
├── main.go               # CLI entrypoint
├── cli/
│   └── cmd.go            # Subcommands (run, bootstrap) & GitHub integrations
├── sensors/
│   ├── orchestrator.go   # Subprocess executor and linter JSON parser
│   ├── go_ast.go         # Native, zero-dependency Go AST metric collector
│   └── bootstrap.go      # Pristine config file template generator
└── tests/
    ├── orchestrator_test.go  # Go AST & Level 0 fallback unit tests
    └── bootstrap_test.go     # Bootstrap template and overwrite guardrail tests
```

---

## 🧩 Architectural Constraints (ADR Rules)

1. **Zero External Dependencies:** No external third-party Go dependencies. Only use Go's standard library (`go/*`, `os`, `exec`, `encoding/json`, `regexp`). This keeps the binary compilation instant and ensures frictionless Day 0 CI integration.
2. **Stateless Execution:** The CLI must remain completely stateless. It reads local files and writes to stdout or stderr. No database dependencies, no filesystem caches, and no remote telemetry.
3. **Safety Guardrails:** The `bootstrap` command must **never** destructive-overwrite existing custom configuration files. If an existing config is found, skip writing, alert the user, and output recommended addition snippets.
4. **Agent-Facing Output:** All warnings and errors must output clear, actionable, and structured **Refactoring Prompts** specifically optimized for LLM coding agents to ingest and act on.

---

## 🚀 Standard Operational Commands

```bash
# Run all tests
/usr/local/go/bin/go test ./...

# Force a clean, non-cached test run
/usr/local/go/bin/go test -count=1 ./...

# Build the static binary
go build -o bin/maintainability-sensors main.go

# Scan the current folder
./bin/maintainability-sensors run .

# Bootstrap a directory
./bin/maintainability-sensors bootstrap /path/to/repo
```

---

## 📋 Standard PR Checklists for AI Agents

When modifying existing sensors or adding a new language bootstrap:
1. **Spec First:** Define the language limits and expected linter patterns.
2. **Test First (TDD):** Implement table-driven tests inside the `tests/` directory and ensure they fail.
3. **Implement Cleanly:** Write the minimum code inside the `sensors/` package to pass the tests.
4. **Compile & Verify:** Confirm that `go test ./...` passes beautifully in milliseconds and the compiled binary functions as expected.
