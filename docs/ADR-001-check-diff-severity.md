# ADR-001: Configurable Severity/Tolerance Thresholds for `check-diff`

**Status:** Proposed
**Date:** 2026-05-22
**Author:** maintainability-sensors maintainers

---

## 1. Context & Problem Statement

The `check-diff` command is the primary gating mechanism for CI pipelines and pre-commit hooks. Its job is to flag maintainability regressions introduced by an agent's code changes. However, the current implementation is **dogmatic**: every threshold exceeded in a modified code block is treated as a hard-blocking error that returns a non-zero exit code and aborts the commit.

Baseline thresholds are currently hardcoded in `sensors/constants.go`:

| Rule               | Baseline |
|--------------------|----------|
| Cyclomatic Complexity | 8        |
| Function Length       | 50       |
| Argument Count        | 4        |

In `cli/cmd.go`, `CheckDiffCmd.Run()` calls `processViolationsMap()`, which unconditionally sets `hasDeltaViolations = true` for any `isTrueViolation()` that overlaps with the diff range, and then returns `fmt.Errorf("Delta violations found")`. There is no escape valve.

### Why This Is a Problem

1. **Legacy Code Incompatibility:** Teams cannot adopt `check-diff` in existing repositories that contain decades-old code because a single-line fix to a 200-line function immediately blocks the commit.
2. **Frustration for Gradual Adoption:** Teams that want to deploy the CLI incrementally must either perfect every baseline or bypass the hook entirely, defeating the purpose.
3. **No Warning-then-Fail Semantics:** Users cannot opt to see violations as **warnings** during a transitional period before upgrading them to **errors**.
4. **Prevents Tooling Rollout:** A hard-blocking message forces teams to either fix all debt first (big-bang) or not use the tool at all.

The goal of this ADR is to make `check-diff` configurable so that teams can define **severity levels** (`error`, `warn`, `ignore`) and **per-rule tolerances** without changing the default hard-blocking behavior for users who are already satisfied with it.

---

## 2. Decision

We will introduce a **two-tier configuration mechanism**:

1. **CLI Flags on `check-diff`** for quick overrides (e.g., `--severity-warn-for=Complexity,FunctionLength`).
2. **Project-level YAML file** (`.maintainability-sensors.yml`) for persistent, version-controlled policy per repository.

The CLI flags take precedence over the YAML file. If neither is present, behavior remains **exactly** as it is today (all violations are errors), ensuring **zero breaking changes**.

---

## 3. Severity Levels

Three levels are supported for every rule (and for the global fallback):

| Level   | Behavior in `check-diff`                     | Exit Code |
|---------|----------------------------------------------|-----------|
| `error` | Print to `stderr` and fail the command       | `1`       |
| `warn`  | Print to `stderr` but allow the commit/CI to pass | `0`       |
| `ignore`| Silently suppress the violation entirely     | `0`       |

These map cleanly to the existing `AI WARNING` output stream in `cli/cmd.go`, allowing downstream CI systems to parse warnings without failing.

---

## 4. Configuration Format

### 4.1 Project-Level YAML File (`.maintainability-sensors.yml`)

This file is optional and lives in the repository root (or the directory pointed to by `--config`).

```yaml
version: "1"

check-diff:
  # Global default for any rule not listed below.
  default-severity: error

  rules:
    - name: Complexity
      # Override severity for this rule. Allowed: error, warn, ignore.
      severity: warn
      # Optional: override the baseline threshold itself.
      # If omitted, the linter/AST baseline constant is used.
      threshold: 12

    - name: FunctionLength
      severity: warn

    - name: ArgumentCount
      severity: error
      # Keep blocking on parameter bloat, but raise the tolerance.
      threshold: 5
```

- `name` corresponds to the human-readable rule identifiers already used in the codebase (`Complexity`, `FunctionLength`, `ArgumentCount`).
- `severity` defaults to `error` if omitted, matching current behavior.
- `threshold` is optional. When present, it overrides the hardcoded `BaselineComplexity`, `BaselineFunctionLength`, and `BaselineArgumentCount` values for `check-diff` only.

### 4.2 CLI Flags

For CI pipelines or one-off runs, `check-diff` gains three repeatable flags:

```
--severity <rule>:<level>
  e.g., --severity Complexity:warn --severity FunctionLength:warn

--default-severity <level>
  e.g., --default-severity warn

--config <path>
  e.g., --config .maintainability-sensors.yml
```

- If `--severity` is specified, it wins over both the YAML and the default.
- If `--default-severity` is specified, it overrides the `default-severity` key in the YAML.

---

## 5. Per-Rule vs. Global Configuration

### Global Fallback
A top-level `default-severity: warn` converts all violations to warnings without listing each rule individually. This is the recommended starting point for teams bootstrapping the tool into a legacy codebase.

### Per-Rule Override
Users can layer exceptions on top of the global fallback:

```yaml

check-diff:
  default-severity: warn
  rules:
    - name: ArgumentCount
      severity: error   # Still block on parameter bloat
```

### Why Not File-Level or Directory-Level?
File-level severity is explicitly **out of scope** for this ADR. The existing linter configs (`.eslintrc`, `pylintrc`, `.golangci.yml`) already handle threshold overrides per file via native suppression comments and ignore patterns. Replicating that inside `.maintainability-sensors.yml` would create a competing configuration surface. If future demand is strong, a follow-up ADR can add `glob` matches.

---

## 6. Default Behavior & Backwards Compatibility

### Defaults
- `default-severity` defaults to `error`.
- Every rule, if unspecified, inherits `default-severity`.
- Baseline thresholds (`8`, `50`, `4`) remain unchanged and are still sourced from `sensors/constants.go` unless a YAML/CLI `threshold` overrides them.

### Backwards Compatibility Guarantee
If a repository has **no** `.maintainability-sensors.yml` and **no** new CLI flags are passed, `check-diff` behaves identically to today:
- All overlapping violations are treated as errors.
- Exit code is `1` when violations are found.
- Stderr output format is unchanged.

This prevents any existing CI pipelines or pre-commit hooks from breaking on upgrade.

---

## 7. CLI Integration

### 7.1 `cli/cmd.go` Changes

1. Extend `CheckDiffCmd`:
   ```go
   type CheckDiffCmd struct {
       TargetBranch    string   `optional:"" default:"HEAD"`
       TargetPath      string   `arg:"" optional:"" default:"."`
       Config          string   `help:"Path to .maintainability-sensors.yml config file."`
       DefaultSeverity string   `optional:"" default:"error" help:"Default severity level for rules not explicitly configured (error|warn|ignore)."`
       Severity        []string `optional:"" name:"severity" help:"Per-rule severity overrides (format: Rule:level)."`
   }
   ```

2. Introduce a resolved `CheckDiffPolicy` struct that both the YAML and CLI flags collapse into:
   ```go
   type Severity string
   const (
       SeverityError Severity = "error"
       SeverityWarn  Severity = "warn"
       SeverityIgnore Severity = "ignore"
   )

   type RulePolicy struct {
       Name      string
       Severity  Severity
       Threshold *int // nil means "use baseline constant"
   }

   type CheckDiffPolicy struct {
       DefaultSeverity Severity
       Rules           map[string]RulePolicy // keyed by rule name
   }
   ```

3. Modify `processViolationsMap()` to accept a `*CheckDiffPolicy`. For each violation:
   - Look up the rule's policy (e.g., `"Complexity"`).
   - If missing, fallback to `DefaultSeverity`.
   - If severity is `ignore`, skip printing and do not set `hasDeltaViolations`.
   - If severity is `warn`, print the existing `AI WARNING` line but do not set `hasDeltaViolations = true`.
   - If severity is `error`, print and set `hasDeltaViolations = true`.

### 7.2 Threshold Override Logic

`isTrueViolation()` currently compares against constants like `sensors.BaselineComplexity`. This function (or its caller) should accept the resolved policy so that a per-rule `Threshold` can override the constant.

```go
func isTrueViolation(v sensors.Violation, policy CheckDiffPolicy) bool {
    rulePolicy, ok := policy.Rules[v.RuleName]
    if !ok {
        // fallback to baseline constants
        rulePolicy = RulePolicy{Severity: policy.DefaultSeverity}
    }
    if rulePolicy.Severity == SeverityIgnore {
        return false
    }

    limit := baselineForRule(v.RuleName) // returns hardcoded constant
    if rulePolicy.Threshold != nil {
        limit = *rulePolicy.Threshold
    }

    return v.Value > limit
}
```

### 7.3 Exit Code Contract

`CheckDiffCmd.Run()` final exit logic changes from:
```go
if hasDeltaViolations {
    return fmt.Errorf("Delta violations found")
}
```
...to a two-pass evaluation:
1. Print warnings.
2. If any violation has severity `error`, return error and exit `1`.
3. If none are `error`, log `Delta clean.` and exit `0`.

---

## 8. Example Adoption Walkthrough

### Scenario A: Wishy-Washy Rollout on a Monorepo
A team wants to run `check-diff` in CI but not block builds for the first 3 months.

```yaml
# .maintainability-sensors.yml
version: "1"
check-diff:
  default-severity: warn
```

CI now prints violations for every PR, engineers see the warnings, but pipelines stay green. After 90 days, the team changes `warn` to `error` and the tool becomes a gate.

### Scenario B: Blocking One Rule Only
Senior engineers want to prevent parameter bloat (`ArgumentCount > 4`) from creeping in, while tolerating 10-point complexity in legacy files.

```yaml
# .maintainability-sensors.yml
version: "1"
check-diff:
  default-severity: warn
  rules:
    - name: ArgumentCount
      severity: error
```

Only `ArgumentCount` regressions fail the build.

### Scenario C: Command-Line Override in a Migration Script
```bash
maintainability-sensors check-diff \
  --default-severity warn \
  --severity ArgumentCount:error \
  --config /repo/.maintainability-sensors.yml
```

### Scenario D: Pre-Commit Hook (Current Behavior Unchanged)
If no config file exists and no flags are passed, `check-diff` blocks on everything exactly as it does today.

---

## 9. Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| **Scope creep:** Users ask for file-level, directory-level, or author-level severity. | Explicitly out of scope. Native linter configs already solve this. ADR reserves a path for a future `glob` extension but does not implement it now. |
| **Silent failures:** If a user typos a rule name (`Complexty:warn`), it is silently ignored. | Validate all rule names at startup against the canonical list (`Complexity`, `FunctionLength`, `ArgumentCount`). Unknown rules print a clear error and abort. |
| **Config drift:** Teams commit `.maintainability-sensors.yml` with `ignore` on all rules. | Not a technical risk. That is an organizational choice. The tool's job is to make the policy explicit. |
| **Threshold override confusion:** Users set `threshold: 200` for `FunctionLength` and then complain the tool is useless. | Threshold overrides are coupled with severity. If they override a threshold, they still see the violation at the new limit unless severity is `ignore`. |

---

## 10. Consequences

### Positive
- Teams can adopt the CLI in legacy repositories without a massive refactoring sprint.
- CI/CD owners can run `check-diff` as an **advisory** step before promoting it to a **blocking** step.
- The tool remains stateless; the YAML file is a standard artifact read from disk, aligned with the existing architecture constraints.
- Zero breaking changes for existing users.

### Negative
- Adds a new config surface (`.maintainability-sensors.yml`) that must be parsed, validated, and documented.
- Increases decision fatigue: users now have to choose between `error`, `warn`, and `ignore`.
- A new dependency on `gopkg.in/yaml.v3` (already present for config parsing) is used for the new file.

---

## 11. Next Steps

1. Implement the YAML loader and CLI flag parser in `cli/cmd.go`.
2. Define the `CheckDiffPolicy` types in a new `sensors/policy.go` file (or inside `cli/policy.go`).
3. Update `isTrueViolation()` and `processViolationsMap()` to accept policy.
4. Add table-driven unit tests for policy resolution, severity classification, and exit-code contracts.
5. Update `docs/GITHUB_ACTIONS_GUIDE.md` and `docs/AI_AGENT_FEEDBACK_LOOP.md` to document the new `warn` mode.
6. (Future) Add `--with-warn-policy` flag to the `bootstrap` command to generate a starter `.maintainability-sensors.yml` with `default-severity: warn`. **Not yet implemented.**

---

## 12. References

- `cli/cmd.go` — `CheckDiffCmd.Run()` and `processViolationsMap()`
- `sensors/constants.go` — Baseline threshold constants
- `sensors/config_parsers.go` — Existing YAML/TOML/JSON parsing utilities
- `docs/DELTA_MODE_STRATEGY.md` — Original delta-mode design rationale
