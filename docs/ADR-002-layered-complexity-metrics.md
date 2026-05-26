# ADR-002: Layered Complexity Metrics for AI-Safe Modularity

**Status:** Accepted
**Date:** 2026-05-22
**Author:** maintainability-sensors maintainers

---

## 1. Context & Problem Statement

Classic Cyclomatic Complexity (e.g., McCabe's algorithm) heavily penalizes highly cohesive, flat `switch` statements. For instance, a flat `switch` mapping 20 JSON error codes is perfectly readable to a human but generates a cyclomatic complexity score of 20+, violating strict baselines (e.g., max 8).

To prevent fracturing readable code, we introduced the **Honest Exception Protocol**, allowing developers and AI agents to use standard suppression comments (e.g., `//nolint:gocognit`) with a documented justification. 

**The Risk:** Suppressing the complexity metric for the entire function creates a dangerous blind spot. An AI agent or a rushed developer could hide massive amounts of technical debt (e.g., 50 lines of nested `if/for` loops or linear spaghetti code) *inside* one of the suppressed switch cases, completely evading CI detection.

## 2. Decision

To achieve true maintainability without forcing unnatural code fragmentation, we are adopting a **3-Layer Defense System** that balances human readability with strict AI safety.

### Layer 1: Cyclomatic Complexity (The Broad Net)
We will maintain Cyclomatic Complexity as the universal default metric across all languages due to widespread linter support.
- **The Guardrail:** Catches mathematically convoluted code globally.
- **The Escape Hatch:** The "Honest Exception Protocol" (e.g., `//nolint:cyclop`) is explicitly permitted for cohesive mapping logic.

### Layer 2: Cognitive Complexity (The Semantic Check)
We will introduce Cognitive Complexity as an additional, mandatory baseline check.
- **The Mechanism:** Cognitive complexity increments by +1 for the `switch` itself, but does not penalize individual flat cases. However, it severely penalizes nested `if/for` loops *inside* those cases.
- **The Rule:** Even if Cyclomatic Complexity is suppressed via a linter comment, the Cognitive Complexity score must never exceed the baseline. Flat switches pass automatically; hidden nested loops fail instantly.

### Layer 3: Native AST `MaxCaseLength` (The AI Specific Rule)
To prevent agents from hiding large blocks of linear code inside a `case` block (which passes Cognitive Complexity but still causes context-window fragmentation), we will enforce a strict line limit on individual cases via our native AST parsers.
- **The Rule:** A `switch` block can have an unlimited number of cases, but no single `case` block can exceed 5-10 lines of logic. Complex case logic MUST be delegated to isolated helper functions.

## 3. Consequences

- **Positive:** We stop penalizing human-readable flat switches, eliminating false positives and friction.
- **Positive:** We close the `//nolint` blind spot, ensuring AI agents cannot sneak technical debt into suppressed functions.
- **Positive:** Forces the adoption of the Strategy Pattern or granular delegation for complex routing.
- **Negative:** Requires updating underlying linter dependencies (e.g., transitioning to `gocognit` and `eslint-plugin-sonarjs`) to explicitly support Cognitive Complexity.
- **Negative:** Requires extending our native AST parsers (e.g., `sensors/go_ast.go`) to natively detect and fail `ast.CaseClause` nodes that exceed length thresholds.