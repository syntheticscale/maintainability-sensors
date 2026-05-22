# Hybrid Plugin Architecture: Migration Plan

This document stages the refactoring of `maintainability-sensors` from a monolithic orchestrator into a hybrid plugin architecture. 

## Phase 1: The Plugin Interface (Core Framework)
- **Goal:** Define the boundary between the CLI core and the language parsers.
- **Tasks:**
  1. Create a `sensors/plugin.go` file.
  2. Define the `Plugin` interface:
     ```go
     type Plugin interface {
         Name() string
         Analyze(filePaths []string) (map[string]MaintainabilityMetrics, error)
     }
     ```
  3. Create a central `PluginRegistry` to map languages (e.g., "js", "python", "go") to their respective plugins.

## Phase 2: Refactor Tier 2 (Orchestrated Ecosystems)
- **Goal:** Port the existing ecosystem wrappers to the new Plugin interface.
- **Tasks:**
  1. Refactor `runESLintBatch` and `runBiomeBatch` into `ESLintPlugin` and `BiomePlugin`.
  2. Refactor `runPyLintBatch` and `runRuffBatch` into `PyLintPlugin` and `RuffPlugin`.
  3. Refactor `runRuboCopBatch` and `runStandardRBBatch` into `RuboCopPlugin` and `StandardRBPlugin`.
  4. Register these in the `PluginRegistry`.

## Phase 3: Refactor Tier 1 (Native AST)
- **Goal:** Port the native AST parsers (`go/ast` for Go, `tree-sitter` for C# and Java) to the Plugin interface to unify the execution pipeline.
- **Tasks:**
  1. Refactor the `ParseGoAST`, `ParseCSharp`, and `ParseJava` logic to wrap them in `GoPlugin`, `CSharpPlugin`, and `JavaPlugin` structs that implement the `Plugin` interface.
  2. Ensure they accept a batch of files (even if they loop internally) to fulfill the `Analyze(filePaths []string)` contract.

## Phase 4: Pipeline Wiring & Cleanup
- **Goal:** Remove the legacy switch statements.
- **Tasks:**
  1. Update `OrchestratedScanBatch` (or `scanWithLocalAnalyzersBatch`/`scanNativeFile`) to simply query the `PluginRegistry` for the given language.
  2. Execute `plugin.Analyze(paths)` directly.
  3. Remove all leftover legacy orchestrator switch statements and cleanup `orchestrator.go`.
