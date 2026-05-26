# Polyglot Plugin Architecture: Traps & Execution Plan

**Date:** May 26, 2026

While pivoting from a monolithic CGO `go-tree-sitter` design to a "Thin Orchestrator / Fat Native Plugin" architecture solves massive ecosystem and distribution headaches, it introduces new structural traps that must be managed before enterprise rollout.

## ⚠️ Trap 1: The Subprocess "Cold Start" Penalty
**The Flaw:** Go is blazingly fast (< 5ms boot). Node.js and Python are not (~150-300ms boot). If the Go orchestrator spawns a new Node.js sidecar process for *every single file* discovered in a monorepo, a 1,000-file repository will take several minutes to scan, completely destroying the sub-millisecond feedback loop promise.
**The Plan:**
1. **Batching:** The `plugin_runner.go` must never spawn a process per file. It must aggregate all files of a specific language (e.g., all `.ts` and `.tsx` files) into a single massive JSON array payload.
2. **Daemonization (LSP Context):** For the LSP server, spawning a Node.js process on every keystroke (`didChange`) is fatal. The Go LSP must spawn the Node plugin *once* as a persistent background daemon, streaming JSON-RPC payloads to it over a dedicated `stdio` pipe, and killing the child process when the IDE closes.

## ⚠️ Trap 2: Plugin Discovery and Distribution Dependency
**The Flaw:** By removing CGO, we regained the magical "single static Go binary." However, if a user runs `maintainability-sensors run .` in a TypeScript project, the Go CLI now inherently depends on the existence of `@syntheticscale/sensor-ts`. If Node.js isn't installed, or the plugin isn't in the `node_modules`, the scan fails.
**The Plan:**
1. **Graceful Degradation:** If the TS plugin cannot be found locally, the Go orchestrator should cleanly fallback to Level 0 ("BLIND") mode, warning the user rather than crashing.
2. **Auto-Bootstrapping (`npx` fallback):** In the `plugin_runner.go`, if local resolution fails, Go can attempt to run `npx --yes @syntheticscale/sensor-ts`. This trades a one-time network latency hit for seamless onboarding.
3. **Explicit Documentation:** The `README.md` and `bootstrap` commands must explicitly output the necessary `npm install` commands required to wire up the native sensors.

## ⚠️ Trap 3: Zombie Processes and Resource Leaks
**The Flaw:** When Go spawns sidecar plugins, unexpected crashes in the Go orchestrator (or a user hitting `Ctrl+C`) can strand orphaned Node.js or Python processes in the background, eventually eating up the developer's RAM.
**The Plan:**
1. **Context Context Context:** Every `exec.CommandContext` in Go must be tightly bound to the application's root `context.Context`. 
2. **Signal Trapping:** The Go orchestrator must explicitly listen for `SIGINT` and `SIGTERM`, ensuring it sends termination signals to all active plugin subprocesses before exiting.
3. **Parent Death Polling:** The native plugins (e.g., the Node script) should periodically poll to ensure their parent PID (the Go CLI) is still alive. If the parent dies, the sidecar must commit suicide.

## ⚠️ Trap 4: Contract Versioning Mismatches
**The Flaw:** As the Go CLI evolves, new fields (like `MaintainabilityMetrics.MaxCaseLength`) will be added to the payload. If we relied on strict schema enforcement, an outdated local version of the `@syntheticscale/sensor-ts` plugin might panic or drop data. Forcing community members to compile Protocol Buffers for simple Ruby or Python plugins is an massive barrier to entry.
**The Plan:**
1. **The Tolerant Reader (JSON):** We explicitly revert back to JSON but enforce the "Tolerant Reader" pattern. Go's `encoding/json` natively ignores unknown fields by default. Old plugins simply ignore new fields, and new orchestrators receive zero-values for missing fields from old plugins.
2. **Semantic Versioning (`protocol_version`):** The `stdin` payload must include a `"protocol_version": "1.0"` field. Go will only gracefully abort if the *major* version mismatches. If the minor version differs, it proceeds on a best-effort basis, allowing plugins to lag slightly behind the Go CLI without forcing constant rebuilds.