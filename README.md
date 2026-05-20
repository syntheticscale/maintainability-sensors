# Maintainability Sensors for Coding Agents 📡

> **The Community Reference Implementation**  
> *Inspired by Birgitta Böckeler's article ["Maintainability sensors for coding agents"](https://martinfowler.com/articles/sensors-for-coding-agents.html) on MartinFowler.com.*

---

## 🏛️ What are Maintainability Sensors?

In an AI-augmented software development lifecycle (SDLC), **generating code is no longer the bottleneck—validating and maintaining it is.**

When coding agents (such as Cursor, Claude Code, GitHub Copilot, or autonomous workspace agents) work in a codebase, they can rapidly introduce **"Verification Debt"** and **"Maintainability Decay."** They might write monolithic functions, duplicate logic, generate excessive parameter lists, or create deeply nested conditional structures (cyclomatic complexity) that human PR reviewers struggle to catch in high-velocity pipelines.

**Maintainability Sensors** are active, automated feedback loops (ran locally or in CI) that measure code health in real-time. Instead of passive instructions (guides), sensors act as the **active guardrails** that detect code decay early and **force coding agents to self-correct** before the code ever reaches human eyes.

---

## 🚀 Key Features

* **Zero External Dependencies:** Built as a single, static Go binary. No databases, no telemetry, no warm-up times. Runs in microseconds (<15ms per file).
* **Orchestration Architecture:** Auto-detects and orchestrates your local language-specific static analysis tools (ESLint, PyLint, Go Vet). It respects your project's custom linting rules rather than forcing a proprietary model.
* **Native Go AST Engine:** Contains a built-in, native Go Abstract Syntax Tree (AST) parser to collect precise complexity and method metrics without installing any external tools.
* **The `bootstrap` Accelerator:** An interactive setup engine supporting **TypeScript/React, Python, Go, and Java**. If your repo is "working blind" (Level 0), `bootstrap` programmatically writes pristine linter/compiler configs with strict maintainability thresholds directly to your project root.
* **Self-Correction Formatter:** Formats standard linter outputs into rich, high-context prompts specifically designed for AI agents to ingest and use to rewrite their own code safely.

---

## 📦 Installation

```bash
# Build the binary
go build -o bin/maintainability-sensors main.go

# Install to path
chmod +x bin/maintainability-sensors
mv bin/maintainability-sensors /usr/local/bin/
```

---

## 🚦 Usage & Commands

### 1. `run` (Scan Repository or File)
Scan a specific file or your entire repository. The CLI checks for local static analysis tools (Orchestrated Mode). If none are found, it runs in **Level 0 (Working Blind)** mode, logging a warning and suggesting the `bootstrap` command.

```bash
# Scan a specific file
maintainability-sensors run src/components/MyComponent.tsx

# Scan the entire repository
maintainability-sensors run .

# Output JSON payload to stdout
maintainability-sensors run . --json

# Write reports to files
maintainability-sensors run . --markdown-out=report.md --html-out=report.html --json-out=report.json
```

### 2. `generate` (Reconstruct Reports from JSON)
Reconstruct beautiful Markdown and HTML scorecards from a previously saved JSON payload. This is useful for CI pipelines that archive raw metrics and later publish visual reports.

```bash
# Generate HTML and Markdown reports from a saved JSON scorecard
maintainability-sensors generate report.json --html-out=report.html --markdown-out=report.md
```

### 3. `bootstrap` (Environment-Hardening)
Auto-detects the primary codebase language and writes pristine, ready-to-use maintainability configurations enforcing strict thresholds:
* **File Length:** max 300 lines
* **Function Length:** max 50 lines
* **Argument Count:** max 4 parameters
* **Cyclomatic Complexity:** max 8 limit

```bash
# Bootstrap local maintainability configurations
maintainability-sensors bootstrap /path/to/repo
```

Supported Blueprints:
* **TypeScript/React:** Generates `.eslintrc.json` (max-params, complexity, max-lines-per-function) and outputs the `npm install` development dependencies.
* **Python:** Generates `.pylintrc` (McCabe complexity, method limits) and outputs the `pip` packages.
* **Go:** Generates `.golangci.yml` (configuring `gocognit`, `funlen`, `gocyclo`, `lll`) and outputs the installation curl command.
* **Java:** Generates a standard Checkstyle configuration `checkstyle.xml` (method length, complexity, parameter limits) and provides Maven/Gradle integration instructions.
* **Ruby:** Generates `.rubocop.yml` (cyclomatic complexity, method length, parameter limits) and outputs the `gem install` command.
* **C#:** Generates `.editorconfig` (Roslyn analyzer rules for complexity) and outputs .NET build integration instructions.

---

## 🔬 Real-World Case Studies & Code Decay Analysis

To prove the accuracy, speed, and real-world utility of `maintainability-sensors`, we audited six high-profile, production-grade repositories across **Go, Python, TypeScript, and C#**.

When static analysis engines flag high complexity, they are capturing **real architectural decay (Verification Debt)**—points where humans and AI coding agents experience a "reasoning freeze."

### 🧪 Production Complexity Validation Matrix

| Repository | Language | Target File / Function | Smell Type | Underlying Force | AI Agent Risk |
|---|---|---|---|---|---|
| **1. `go-chi/chi`** | Go | `tree.go` -> `findRoute` | Monolithic Algorithm | Extreme execution speed & zero heap-allocations | High risk of introducing concurrency or path matching regressions. |
| **2. `psf/requests`** | Python | `adapters.py` -> `HTTPAdapter.send` | Overloaded Method | Evolutionary drag & accumulating OS exceptions | Fragmented error handling; high risk of breaking exception mappings. |
| **3. Go Std Library** | Go | `net/http/server.go` -> `serve` | Stateful Monolith | Inherent complexity of stateful network protocols | Code cannot be safely modified without breaking HTTP/1.x invariants. |
| **4. `tiangolo/fastapi`** | Python | `dependencies/utils.py` | Procedural Bottleneck | High-throughput polymorphic request parsing | Breaking type-casting or security parameter validation. |
| **5. `nestjs/nest`** | TypeScript | `packages/core/injector` | Deeply Coupled State | Runtime reflection & complex DAG DI resolution | AI recursion traps, stack overflows, or memory leaks on custom scopes. |
| **6. `dotnet/aspnetcore`** | C# | `Routing/EndpointRoutingMiddleware.cs` -> `Invoke` | Layered Middleware Stack | Framework extensibility & cross-cutting concern composition | AI introduces middleware ordering bugs or breaks request pipeline短路invariants. |

### 📖 Read the Full Deep-Dive
For a detailed analysis of why these systems were written this way, how they act as AI traps, and exactly how they should be refactored, read our:
👉 **[Architectural Case Studies deep-dive (docs/CASE_STUDIES.md)](docs/CASE_STUDIES.md)**

---

## 🧩 Architectural Philosophy (ADR Standards)

This CLI is designed around three strict architectural rules:
1. **Statelessness (Local-First):** Inputs are your local files and git; outputs are stdout. No external telemetry or remote databases.
2. **Orchestration over Re-implementation:** Leverage standard local compilers and AST tools (ESLint, PyLint) rather than re-writing syntax parsers.
3. **Deterministic Feedback Loop:** Errors and warnings are structured to provide explicit refactoring guidance specifically formatted for LLM coding agents.

---

## 📄 License

MIT © 2026 Paulo Lai & Contributors.  
*This is an independent open-source community reference implementation and is not affiliated with Thoughtworks or Martin Fowler.*
