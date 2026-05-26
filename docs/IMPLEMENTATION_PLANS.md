# Implementation Plans: The Two-Tier Sensor Strategy

This document outlines the architectural roadmap for `maintainability-sensors`, explicitly aligned with the original vision from Birgitta Böckeler's "Maintainability sensors for coding agents" and our core technical constraints.

To prevent scope creep and maintain the CLI as a lightweight, ultra-fast Go utility, we employ a **Two-Tier Architecture**:
1. **Tier 1: Syntactic Sensors (The Go CLI):** Sub-millisecond, stateless, local AST analysis executed continuously during the agent's coding loop (`check-diff` Delta Mode).
2. **Tier 2: Semantic & Inferential Sensors (AI Skills & CI):** Slower, context-heavy, or network-dependent analysis deferred to higher-level AI Agent Skills or asynchronous CI pipelines.

---

## Implemented Features (Tier 1: The Go CLI)

These features have been successfully built directly into the ultra-fast, stateless Go binary.

### 1. Native AST Dependency Rules (Macro-Coupling)
*   **Context:** AI agents need instant nudges when they violate layered architecture boundaries (e.g., a React UI component importing directly from the PostgreSQL DB layer).
*   **Implementation:** Instead of orchestrating slow external tools like `dependency-cruiser` (Node) or `import-linter` (Python), we will implement dependency checking natively.
*   **Execution Plan:**
    *   Leverage our existing Go AST parser (and upcoming Tree-sitter integration) to extract `import` statements.
    *   Implement a lightweight `.sensors-architecture.yml` configuration to define allowed layer dependencies.
    *   Flag illegal cross-layer imports instantly in `check-diff` mode.

### 2. Universal AST via Legacy Plugin Architecture (Stdio JSON)
*   **Context:** Relying on external linters (ESLint, pylint) requires the agent to have language-specific environments installed. However, attempting to parse foreign languages (Python, TypeScript) natively in Go using CGO (`go-tree-sitter`) destroyed the portability and build speed of the core CLI.
*   **Implementation:** We pivoted to a Two-Tier architecture where the core Go CLI orchestrates a standalone `legacy-plugin` subprocess via a standard I/O JSON protocol.
*   **Execution Plan:**
    *   Extract all non-Go language logic into the `legacy-plugin` binary.
    *   Maintain the core `maintainability-sensors` CLI as a 100% pure, statically compiled Go binary with zero CGO dependencies.
    *   Ensure the core CLI simply pipes file contents and thresholds to the `legacy-plugin` and parses the returned JSON `AnalyzeResponse` in sub-milliseconds.

---

## External Features (Tier 2: AI Skills & CI)

These features are highly valuable to the overarching goal of maintaining code quality, but violate the CLI's strict constraints (they are slow, stateful, or require network/LLM calls). They have been implemented externally as Agent Skills or CI configurations.

### 3. Semantic "Inferential" Modularity Review (LLM-as-a-Judge)
*   **Context:** Detecting duplicated business logic or inefficient argument passing requires semantic understanding of the code's *intent*, which mathematical AST analysis cannot provide.
*   **Why Defer:** Making HTTP calls to LLM APIs from the Go CLI destroys its sub-millisecond performance profile and breaks the stateless constraint.
*   **Where it belongs (AI Skill):** Implement this as a specialized **AI Agent Skill**. An orchestrating AI (like Claude or Gemini) can invoke this skill *after* finishing a major feature. The skill will provide rigid prompts based on Vlad Khononov’s modularity principles, allowing the AI to evaluate its own work for semantic duplication and misplaced responsibilities before human review.

### 4. Mutation Testing (Test Quality Sensors)
*   **Context:** Ensuring tests actually catch bugs, preventing AI agents from generating tautological tests that achieve 100% coverage but assert nothing.
*   **Why Defer:** Mutation testing requires running the test suite multiple times (sometimes taking minutes). It cannot be part of the instant `check-diff` inner loop.
*   **Where it belongs (CI Pipeline):** Defer this to CI/CD (e.g., GitHub Actions). Create a workflow script that runs tools like `Stryker` or `go-mutesting` strictly against the `git diff` line ranges before a PR merge, feeding the results back as an asynchronous PR comment.

### 5. Fan-in / Fan-out Coupling Metrics
*   **Context:** Calculating how many modules depend on a file to assess the "blast radius" of an AI's code change.
*   **Why Defer:** Parsing the entire workspace AST to map all references is computationally heavy. Furthermore, as noted in the original blog post, mathematical coupling metrics are often "lackluster" without semantic AI context to explain *why* the coupling exists.
*   **Where it belongs (Discard or CI):** Discard from the real-time CLI. If needed, this logic is better suited for a slow, asynchronous nightly CI report.
