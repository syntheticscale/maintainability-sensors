# LSP Benchmark & In-Memory Refactor Plan

## Objective
Measure the performance cost of the current "Naive LSP" implementation (which creates temporary files on disk for every `didChange` event) using Go benchmarks. If the resources needed are deemed excessive, refactor the `sensors.Plugin` interface and the LSP server to operate entirely in-memory.

## Key Files & Context
- `internal/lsp/server_test.go`: Where the benchmarks will be added.
- `internal/lsp/server.go`: The LSP server implementation that currently uses `os.CreateTemp`.
- `internal/sensors/plugin.go`: The core `Plugin` interface that needs to be updated to accept in-memory buffers.
- `internal/sensors/orchestrator.go`: The orchestrator that needs to pass buffers to the plugins.
- All plugin implementations in `internal/sensors/`: Need to be updated to read from the buffer if provided, falling back to disk if not.

## Implementation Steps

### 1. Verification & Commits
- Verify the git working directory is clean. If there are any lingering changes from the previous refactor, commit them.

### 2. Setup Benchmarks (Empirical Measurement)
- Create a `BenchmarkLSPDidChange` function in `internal/lsp/server_test.go`.
- The benchmark will simulate a rapid succession of `textDocument/didChange` events being piped into the LSP server's `stdio` stream.
- Run the benchmark with memory profiling: `go test -bench=BenchmarkLSPDidChange -benchmem ./internal/lsp`.
- Analyze the `ns/op` (nanoseconds per operation) and `B/op` (bytes allocated per operation) to judge if the temp file creation is causing excessive overhead.

### 3. Refactor Plugin Interface (If Excessive)
- Introduce a new struct in `internal/sensors/plugin.go`:
  ```go
  type FileContext struct {
      Path    string
      Content []byte // In-memory buffer
  }
  ```
- Update the `Plugin` interface:
  ```go
  type Plugin interface {
      Name() string
      Analyze(files []FileContext) (map[string][]Violation, error)
  }
  ```
- Update `OrchestratedScanBatch` to accept `[]FileContext`.

### 4. Update Plugins
- Modify all native AST plugins (`tree_sitter_python.go`, `tree_sitter_typescript.go`, `go_ast.go`, etc.) to check if `FileContext.Content` is populated. If so, parse the buffer directly instead of calling `os.ReadFile`.
- For external linters (ESLint, PyLint), we may still need to use files, or we can pipe the buffer to their `stdin` if supported. For this iteration, we will prioritize optimizing the native Tree-sitter plugins used for fast feedback.

### 5. Update LSP Server
- Modify `internal/lsp/server.go` to construct a `FileContext` with the `textDocument/didChange` payload's content and pass it directly to `OrchestratedScanBatch`, eliminating the `os.CreateTemp` call entirely.

### 6. Verification
- Re-run the `BenchmarkLSPDidChange` benchmark.
- Compare the "Before" and "After" metrics to empirically prove the performance gains of the in-memory architecture.
- Run `go test ./...` to ensure no regressions.

## Migration & Rollback
- The `FileContext` struct allows a graceful fallback: if `Content` is `nil`, plugins can safely revert to reading from the `Path` on disk, preserving compatibility for the CLI `run` and `check-diff` modes.