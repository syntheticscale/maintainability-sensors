# Project Status 📡

**Last Updated:** 2026-05-24  
**Branch:** `main`  
**State:** 🟢 Stable (Two-Tier Architecture & Refactoring Complete)

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

## ✅ Completed Work (Two-Tier Architecture Sprint)

This sprint focused on resolving technical debt, finalizing the LSP server, and establishing the Agent Skills ecosystem.

| Category | Achievements |
|---|---|
| **LSP In-Memory** | Eliminated temp-file disk thrashing. LSP `didChange` events are now parsed entirely in-memory using `FileContext`. |
| **God File Dismantled** | Refactored `orchestrator.go` by extracting all specific linter logic into cohesive modules (`eslint_plugin.go`, etc.). |
| **Memory Leaks Fixed** | Added `defer tree.Close()` and `defer parser.Close()` to prevent CGO OOMs. |
| **Correctness Tools** | Added `gochecknoglobals` and `bodyclose` to `.golangci.yml`, and introduced `close_leak_test.go` to catch unclosed C-allocated objects. |
| **AI Skills** | Shipped `modularity-reviewer`, `pre-flight-check`, and `performance-benchmarker` as standard AI Agent Skills. |

---

## 🚧 Known Issues & Technical Debt

*   **CGO Dependency:** By adding `go-tree-sitter`, we broke the "Minimal External Dependencies" constraint. Compiling the CLI now requires a C compiler (gcc/clang), breaking the simple `go build` cross-compilation story.
*   **Brittle Parsing:** `internal/sensors/config_parsers.go` uses massive, fragile regular expressions to parse JavaScript (`.eslintrc.js`) configuration files.
*   **Naive Architecture Matching:** Layer matching in `CheckArchitectureDependencies` uses `strings.Contains(absPath, "/" + layerName + "/")`, which will easily yield false positives if a folder happens to share a name with a layer.