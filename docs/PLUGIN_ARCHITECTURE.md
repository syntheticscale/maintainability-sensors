# Maintainability Sensors: Plugin Architecture

## The Balanced Strategy
The ideal architecture blends the speed of native execution with the ecosystem power of the orchestrator, managed via a standard Plugin Interface.

### Core Responsibilities
The Go binary (`maintainability-sensors`) acts purely as the **CLI framework, CI gatekeeper, and Agent Prompt Generator**. It handles:
- File discovery and filtering.
- Managing baseline suppressions (`maintainability-baseline.json`).
- Generating GitHub PR comments and HTML scorecards.
- Formatting the "Agent Self-Correction Prompts".

### Plugin Responsibilities
Language parsing is delegated to plugins that implement a standard JSON-over-STDIO contract. A plugin takes a list of file paths and returns an array of `MaintainabilityMetrics`.

### Plugin Tiers
1. **Tier 1 (Built-in / Native):** High-speed `tree-sitter` or Go AST parsers compiled directly into the binary for maximum performance (e.g., Go, C#, Java).
2. **Tier 2 (Orchestrated Wrappers):** Built-in adapters that shell out to ecosystem tools (ESLint, Ruff, Biome) when installed on the host.
3. **Tier 3 (External WASM / Binaries):** Custom plugins loaded at runtime for niche languages (e.g., Swift, Kotlin) without modifying the core orchestrator codebase.

This allows the tool to be "zero-dependency" for compiled languages, while still piggybacking on the massive ecosystems of JavaScript and Python without forcing the core team to reinvent their AST parsers.