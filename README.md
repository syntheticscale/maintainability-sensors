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
```

### 2. `bootstrap` (Environment-Hardening)
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
