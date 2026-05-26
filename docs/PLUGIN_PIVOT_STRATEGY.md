# Strategic Pivot: The Polyglot Plugin Architecture

**Date:** May 26, 2026
**Status:** Approved

## 1. The Core Problem
Our attempt to build a monolithic Go CLI that parses native ASTs (via `go-tree-sitter`) and configuration files for 6 different language ecosystems simultaneously resulted in unmanageable horizontal scope creep. 
*   **CGO Dependency:** Relying on `go-tree-sitter` broke the "single static binary" promise of Go, making cross-compilation fragile.
*   **Ecosystem Friction:** Parsing TS configurations, resolving `tsconfig.json` path aliases, and matching ESLint heuristics in Go is an infinite game of catch-up.
*   **Feedback Noise:** Discrepancies between our Go-based AST parser and the developer's native linter (e.g., ESLint) caused frustrating loops for AI coding agents.

## 2. The Architectural Pivot
We are adopting a **Thin Orchestrator / Fat Native Plugin** model, similar to Terraform or Pulumi.

1.  **Go (Tier 1 - Native):** The Go CLI is the orchestrator. It handles the `check-diff` logic, parses GitHub PR events, formats the UI (HTML/Markdown/CLI), runs the concurrent file discovery, and manages the LSP server lifecycle. It natively parses Go files using `go/ast` (dropping CGO entirely).
2.  **TypeScript (Tier 1 - Supported Plugin):** TypeScript/JavaScript analysis is delegated to a native Node.js sidecar plugin (`@syntheticscale/sensor-ts`).
3.  **Other Languages (Tier 2/3 - Community Plugins):** Python, Ruby, C#, and Java are relegated to community-driven templates. We will define the JSON contract, but we will not actively maintain their AST parsing logic.

## 3. The Contract (Pragmatic JSON over Standard I/O)
The Go Orchestrator will spawn the native plugin as a subprocess and communicate via standard input/output using JSON. To avoid the complexity of gRPC/Protobufs while maintaining compatibility, we employ a "Tolerant Reader" pattern combined with semantic versioning.

**Input (from Go to Plugin via `stdin`):**
```json
{
  "protocol_version": "1.0",
  "command": "analyze",
  "files": [
    {
      "path": "/absolute/path/to/src/index.ts",
      "content": "..." 
    }
  ],
  "config": {
    "targetDir": "/workspace/root"
  }
}
```

**Output (from Plugin to Go via `stdout`):**
```json
{
  "protocol_version": "1.0",
  "status": "success",
  "results": {
    "/absolute/path/to/src/index.ts": [
      {
        "RuleName": "CognitiveComplexity",
        "Value": 12,
        "StartLine": 45,
        "EndLine": 60,
        "Message": "Cognitive Complexity is 12 (Max 8)"
      }
    ]
  }
}
```

## 4. Execution Plan (The Path Forward)

### Phase 1: The Legacy Plugin Extraction
*   **Create `legacy-plugin`:** Move all existing external linter wrappers (Ruff, PyLint, Rubocop, Biome) into a new standalone Go binary (`cmd/legacy-plugin`).
*   **Implement Protocol:** Wire this legacy plugin to speak the new JSON `stdio` contract. This acts as our reference implementation for the protocol.
*   **Drop CGO:** Remove `github.com/smacker/go-tree-sitter` and all native foreign AST parsing from the core.
*   **Clean Core:** Delete the brittle regex config parsers and subprocess wrappers from the core CLI.

### Phase 2: Core Orchestration
*   Implement `internal/sensors/plugin_runner.go` in the Go orchestrator. It will spawn external plugins, batch files into the `stdin` JSON payload, and read the `stdout` response. The core will default to calling the `legacy-plugin` for non-Go files.

### Phase 3: The TypeScript Native Plugin
*   Create a separate package (or a `plugins/typescript` folder).
*   Write a Node.js script utilizing the TS Compiler API and ESLint programmatic API.
*   This script will natively handle `tsconfig.json` path aliases for architecture boundary checks and perfectly match ESLint's cognitive complexity logic.

### Phase 4: Tiered Rollout
*   Update `README.md` to establish the Tiered Support Matrix (Go as Native, TS as Supported Plugin, others as Community Templates).
*   Publish the Node.js plugin to npm (e.g., `npx @syntheticscale/maintainability-sensor-ts`). Go will attempt to run it via `npx` if it detects TS files and the plugin isn't installed locally.