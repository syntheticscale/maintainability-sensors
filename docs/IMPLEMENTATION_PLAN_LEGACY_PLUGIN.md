# Implementation Plan: The Legacy Plugin Extraction

**Goal:** Execute the Polyglot Plugin pivot by extracting all non-Go language logic out of the core orchestrator and into a standalone "Legacy Go Plugin". This proves the new `stdio` JSON protocol, drops CGO, and preserves existing language support while isolating technical debt.

## Phase 1: The Protocol Implementation
1. **Define the Schema:** Create a shared Go package (e.g., `internal/plugin/protocol`) that defines the Go structs for the standard I/O JSON protocol (`AnalyzeRequest`, `AnalyzeResponse`, `Violation`, `Handshake`).
2. **Implement Core Runner:** Create `internal/sensors/plugin_runner.go` in the core orchestrator. This logic will spawn a subprocess, send the batched `AnalyzeRequest` over `stdin`, read `stdout`, and parse the `AnalyzeResponse`. It must handle the "Tolerant Reader" JSON unmarshalling.

## Phase 2: Create the Legacy Plugin
1. **New Entrypoint:** Create `cmd/legacy-plugin/main.go`. This is a standalone Go binary.
2. **Implement Handshake:** Make the binary respond to `--handshake` or a specific handshake JSON payload over `stdin` to announce its supported languages (Python, Ruby, etc.).
3. **Port Existing Logic:** Move `ruff_plugin.go`, `pylint_plugin.go`, `rubocop_plugin.go`, `biome_plugin.go`, `standardrb_plugin.go`, and their corresponding JSON parsers into an `internal/legacy` folder.
4. **Wire to Stdio:** Hook up the legacy plugins to read the incoming JSON payload from `stdin`, execute their existing subprocess logic (`os.Exec` to ruff, pylint, etc.), and emit the standard `AnalyzeResponse` to `stdout`.

## Phase 3: The Great Core Deletion
1. **Delete CGO:** Remove `github.com/smacker/go-tree-sitter` and all `tree_sitter_*.go` files from the core repository. 
2. **Delete Old Parsers:** Remove all foreign language plugins and configs from the core `internal/sensors` package.
3. **Wire Orchestrator:** Update `orchestrator.go` (and related routing logic) so that when it encounters a non-Go file, it automatically invokes the `legacy-plugin` binary via the new `plugin_runner.go`.

## Phase 4: Validation
1. Build both `maintainability-sensors` and `legacy-plugin`.
2. Ensure both exist in the system `$PATH` (or relative directories).
3. Run the test suite (`go test ./...`). Note: tests will need to be refactored to either run against the new plugin architecture or moved to the plugin's package.
4. Verify that running the core CLI on a Python repository successfully invokes the legacy plugin, runs Ruff/PyLint, and reports back the violations correctly.