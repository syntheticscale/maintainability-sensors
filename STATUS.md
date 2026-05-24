# Project Status 📡

**Last Updated:** 2026-05-24  
**Branch:** `main`  
**State:** 🟡 Technical Debt Accumulating (Two-Tier Architecture Complete, but needs refactoring)

---

## 🏗️ Current Architecture (Two-Tier Ecosystem)

The codebase has transitioned to a "Two-Tier Ecosystem" separating fast syntactic CLI checks from slow semantic AI Agent Skills.

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
│       └── bootstrap.go             # Enterprise-safe config generator
├── skills/
│   ├── modularity-reviewer/         # Tier 2 Agent Skill (Semantic Review)
│   └── pre-flight-check/            # Tier 2 Agent Skill (Enforces Checks)
└── tests/
    └── golden_test.go               # Validates formatted LLM prompts
```

---

## 🚨 Radically Candid Review Findings (May 2026)

A rigorous architectural review of the recent sprint uncovered significant technical debt that must be addressed:

### 1. Critical Flaws (Fix Immediately)
*   **Memory Leaks:** `tree_sitter_python.go` and `tree_sitter_typescript.go` allocate C memory for ASTs but **never call `Close()`** on the `Tree` or `Parser`. Because the LSP server runs as a daemon parsing on every keystroke, this will rapidly OOM the IDE.
*   **Constraint Violation (Statelessness):** `go_architecture.go` introduced a global `archConfigCache` map. `AGENTS.md` strictly mandates: *"The CLI must remain completely stateless... no filesystem caches."* This breaks thread safety and the core architectural philosophy.

### 2. Architectural Debt (Refactor Soon)
*   **The God File:** `internal/sensors/orchestrator.go` is over 900 lines long. It mixes high-level orchestration, plugin registry logic, subprocess execution, and specific linter implementations (`ESLintPlugin`, `PyLintPlugin`). It desperately needs to be broken up.
*   **Naive LSP:** `internal/lsp/server.go` processes requests sequentially (blocking on large files). Worse, on every `didChange` keystroke, it creates a temporary file on disk just so `OrchestratedScan` can read it. It needs an in-memory buffer system.
*   **CGO Dependency:** By adding `go-tree-sitter`, we broke the "Minimal External Dependencies" constraint. Compiling the CLI now requires a C compiler (gcc/clang), breaking the simple `go build` cross-compilation story.

### 3. Code Smells
*   **Brittle Parsing:** `internal/sensors/config_parsers.go` uses massive, fragile regular expressions to parse JavaScript (`.eslintrc.js`) configuration files.
*   **Naive Architecture Matching:** Layer matching in `CheckArchitectureDependencies` uses `strings.Contains(absPath, "/" + layerName + "/")`, which will easily yield false positives if a folder happens to share a name with a layer.

---

## ✅ Completed Work (Enterprise Hardening Sprint)

This sprint focused on making the tool robust enough for strictly regulated, massive-scale CI/CD pipelines.

| Category | Achievements |
|---|---|
| **Hybrid Plugins** | Replaced brittle monolithic switch statements with a scalable `Plugin` interface and global registry. |
| **Native Execution** | Built native AST parsers (`go-tree-sitter`) for C# and Java. No JVM or .NET SDK required. |
| **OOM Protection** | Implemented streaming JSON parsing and 2MB file-size limits. Replaced `CombinedOutput()` buffers. |
| **Scale & Speed** | Upgraded to `filepath.WalkDir` and implemented 300-file argument chunking to prevent OS `ARG_MAX` panics on huge monorepos. |
| **Security Bounding**| Enforced a strict Workspace Jail to prevent absolute path traversal and added POSIX `--` delimiters to prevent command injection. |
| **Compliance** | Added GitHub API URL overrides (`GITHUB_API_URL`) and 10-second HTTP timeouts for air-gapped CI proxies. |
| **Correctness** | Fixed Go AST metrics leaking complexity from inner closures and fixed error swallowing during initializations. |