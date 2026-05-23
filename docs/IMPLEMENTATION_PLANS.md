# Implementation Plans for Strategic Architectural Gaps

This document provides detailed context, architectural direction, and step-by-step implementation plans for the four major strategic gaps identified during our review of Birgitta Böckeler's "Maintainability sensors for coding agents". 

Any human engineer or AI coding agent can pick up a section below and execute it to evolve `maintainability-sensors` beyond micro-level syntax checks into a macro-level architectural guardrail.

---

## Gap 1: Architectural & Dependency Rules (Macro-Coupling)

### Context & Why it Matters
Currently, the sensors are blind to architectural boundary violations. An AI agent could write a beautifully simple 5-line function in a React UI component that connects directly to a PostgreSQL database. The code would pass all cyclomatic complexity checks (`Delta clean`), but it catastrophically violates layered architecture principles. We need to measure and enforce dependency boundaries.

### Implementation Strategy
Do not write a dependency graph parser from scratch. We will orchestrate existing, battle-tested tools and map their output into our `Violation` struct, just as we did for ESLint and Ruff.

#### Target Tools:
*   **JavaScript/TypeScript:** `dependency-cruiser`
*   **Go:** `golangci-lint`'s `depguard` linter
*   **Python:** `import-linter`

#### Step-by-Step Implementation:
1.  **Configuration Boostrap:** 
    *   Update `sensors/bootstrap.go` to generate strict baseline configurations for these tools (e.g., a `.dependency-cruiser.js` file that forbids cross-layer imports).
2.  **Plugin Creation:** 
    *   Create `sensors/dependency_cruiser_parser.go` implementing the `Plugin` interface.
    *   Implement the `Analyze` method to execute `npx depcruise src --include-only "^src" --output-type json`.
3.  **Mapping Violations:**
    *   Parse the JSON output of the underlying tool. Extract the source file, the offending import path, and the rule violated (e.g., `not-to-dev-dep`).
    *   Map these to the `Violation` struct: `RuleName: "DependencyBoundary"`, `Message: "Illegal import of module X from layer Y"`.
4.  **CLI Output Updates:**
    *   Update `cli/cmd.go` to recognize the `DependencyBoundary` rule and ensure it outputs explicit AI refactoring prompts (e.g., *"REFACTORING PROMPT: Remove illegal dependency. Move this logic to the orchestration layer."*).
5.  **Testing:** Add golden tests with intentionally broken imports to verify the pipeline.

---

## Gap 2: Fan-in / Fan-out Coupling Metrics (Blast Radius)

### Context & Why it Matters
We currently do not understand the "blast radius" of a code change. If an AI modifies a 5-line utility function that is imported by 500 other modules (High Fan-in), the risk of regression is massive. Conversely, a 20-complexity legacy function with 0 dependents (a leaf node) is relatively safe to refactor. We need metrics to calculate coupling so we can warn agents about highly-coupled files.

### Implementation Strategy
This is mathematically intensive. For Go, we can do it natively via our AST tools. For other languages, we need to lean on external static analyzers.

#### Step-by-Step Implementation:
1.  **Metric Definition:** 
    *   Update `MaintainabilityMetrics` in `sensors/orchestrator.go` to include `FanIn int` and `FanOut int`.
2.  **Native Go Implementation (`sensors/go_ast.go`):**
    *   Currently, `ParseGoAST` only looks at a single file. We must change the architecture to optionally load the `packages.Config` across the entire workspace (using `golang.org/x/tools/go/packages`).
    *   Traverse the workspace. For each exported function/struct in the target file, count how many external packages reference it (`FanIn`).
    *   Count the number of external packages the target file imports (`FanOut`).
3.  **Threshold Enforcement:**
    *   Add baseline limits in `sensors/constants.go`: e.g., `BaselineFanOut = 15`.
    *   If a file exceeds the Fan-Out limit (it depends on too many things), flag it as a "God Module" violation.
4.  **AI Prompts:**
    *   Generate a specific warning: *"REFACTORING PROMPT: High Fan-Out (X > 15). This file is highly coupled. Delegate responsibilities to smaller modules."*

---

## Gap 3: Semantic "Inferential" Modularity Review

### Context & Why it Matters
Mathematical linters lack semantic context. An AI agent might accidentally duplicate the exact same business logic in two different files, just using different variable names. Our tools will report both files as clean. We need an inferential sensor to evaluate the *meaning* of the code, not just its syntax.

### Implementation Strategy
This requires bridging traditional static analysis with LLM capabilities. We will build an "LLM-as-a-Judge" sensor that runs strictly in Delta Mode.

#### Step-by-Step Implementation:
1.  **API Integration:**
    *   Add a new configuration flag to the CLI: `--ai-reviewer-token`.
    *   Create a new package `sensors/inferential_reviewer.go`.
2.  **Context Construction:**
    *   When `check-diff` is run, extract the raw `git diff` string.
    *   Gather the full text of the modified files to provide surrounding context.
3.  **The Modularity Prompt:**
    *   Construct a rigid system prompt based on Vlad Khononov’s modularity principles. Ask the LLM to return a strict JSON schema evaluating three things:
        1. *Semantic Duplication:* Does this code exist elsewhere?
        2. *Inefficient Arguments:* Are primitive arguments passed too deeply instead of using a Parameter Object?
        3. *Misplaced Responsibilities:* Is data access mixed with UI/business logic?
4.  **Execution & Mapping:**
    *   Make an HTTP POST to an LLM provider (e.g., Gemini 1.5 Pro).
    *   Parse the JSON response and map any identified issues into our standard `Violation` struct with `RuleName: "SemanticModularity"`.

---

## Gap 4: Mutation Testing (Test Quality Sensors)

### Context & Why it Matters
AI agents are excellent at writing tests that achieve 100% line coverage but assert absolutely nothing (tautological tests). If our sensors ensure the codebase is clean, we also need a sensor to ensure the test harness protecting that code is actually capable of catching bugs.

### Implementation Strategy
We will orchestrate incremental mutation testing tools to run exclusively against the lines of code the AI modified.

#### Target Tools:
*   **Go:** `go-mutesting`
*   **JavaScript/TypeScript:** `Stryker`
*   **Python:** `mutmut`

#### Step-by-Step Implementation:
1.  **Plugin Creation:** 
    *   Create `sensors/mutation_parser.go`.
2.  **Delta Execution:**
    *   Instead of running the mutation tool against the whole repository (which takes hours), use the `git diff` line ranges calculated in `sensors/git_diff.go` to instruct the mutation tool to *only* mutate the actively modified lines.
    *   *Example:* `mutmut run --paths-to-mutate=src/utils.py`
3.  **Evaluating the Score:**
    *   Parse the output. A mutation test creates a bug (a "mutant"). If the test suite fails, the mutant is "killed" (good). If the test suite passes, the mutant "survived" (bad).
    *   Calculate the Mutation Score (Killed / Total Mutants).
4.  **Threshold & Feedback:**
    *   Set `BaselineMutationScore = 80%`.
    *   If the score is lower, output: *"REFACTORING PROMPT: Tests lack assertion strength (Mutation Score X%). The AI tests did not catch injected bugs. Write stronger assertions."*