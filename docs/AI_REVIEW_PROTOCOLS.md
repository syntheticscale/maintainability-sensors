# AI Review Protocols

When acting as an AI coding assistant in this repository, you must execute these multi-persona review protocols whenever a significant architectural change or large feature is completed. This ensures high-assurance stability and prevents the "happy path" trap.

## Protocol 1: CISO & Enterprise Architect Review (Hostile Environment)
**Objective:** Evaluate the code assuming all external inputs and environments are hostile.
**Checklist:**
- **Injection:** Are there missing `--` delimiters before passing user paths to external CLI tools?
- **Timeouts:** Do all external API calls and subprocesses (`exec.CommandContext`) use contexts with strict timeouts to prevent hanging?
- **Bounds:** Are we loading files directly into memory without checking size limits (OOM risk)?
- **Scale:** Does the code iterate over thousands of files sequentially? Are we missing chunking or concurrent batching?
- **Path Traversal:** Are we blindly trusting absolute paths without verifying they sit strictly within the workspace boundary?

## Protocol 2: DevEx & AI Tooling Lead Review (Friction & Observability)
**Objective:** Ensure the tool is completely frictionless for human developers and other AI agents to use.
**Checklist:**
- **Stdout Pollution:** Are we logging human-readable `fmt.Println` diagnostics into standard output that would corrupt a structured JSON pipeline? (All diagnostics must go to stderr).
- **Persona Mismatch:** Is the tool output talking *about* the AI in the third person when the AI is the one actively reading the terminal?
- **Verbosity:** Are there leftover `DEBUG:` prints that will bloat an LLM's context window? Is there a way to run quietly?
- **Edge Cases:** Have we considered the "untracked file", "renamed file", or "deleted file" scenarios for Git diffing? Does it fail safely and explicitly?

## Protocol 3: Meta-Analysis & Root-Cause Review
**Objective:** When presented with a list of bugs or test failures, do not immediately write code to patch them individually.
**Checklist:**
- Group the failures into 3 or 4 broad systemic engineering flaws (e.g., "Lack of Defensive Programming", "Bypassing TDD", "Architectural Immaturity").
- Use these broad categories to perform a second, sweeping review of the entire codebase to identify hidden vulnerabilities that share the same root cause.
- Present the systemic findings before proposing or executing code changes.