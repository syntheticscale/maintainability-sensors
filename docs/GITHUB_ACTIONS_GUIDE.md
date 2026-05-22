# GitHub Actions CI Integration Guide 🐙

This guide details how to integrate `maintainability-sensors` directly into your GitHub Actions CI pipelines. This automates the PR review gate and outputs an executive scorecard directly onto your Pull Requests.

---

## 1. The Strategy: Check-Diff over Full Scans

Legacy codebases often contain hundreds of existing maintainability violations. Failing the CI build because a developer touched a file that *already* had issues is counterproductive and punishes them for legacy debt. 

Instead, we run the sensors in **Delta Mode** using `check-diff`. This ensures that PRs are only blocked if the *changed lines* (the delta) introduce **new** rot.

## 2. Quick Start: Standard CI Step Summary

Add this step to your existing `.github/workflows/ci.yml` pipeline to scan the PR's delta against the `main` branch. The CLI will automatically output a rich Markdown scorecard into the GitHub Actions Job Summary.

```yaml
name: Continuous Integration

on: [push, pull_request]

jobs:
  validate-quality:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout Code
        uses: actions/checkout@v4
        with:
          fetch-depth: 0 # Required to fetch origin/main for check-diff

      # 1. Download and install the Maintainability Sensors binary
      - name: Install Maintainability Sensors
        run: |
          curl -sSfL -o /usr/local/bin/maintainability-sensors https://github.com/syntheticscale/maintainability-sensors/releases/latest/download/maintainability-sensors-linux
          chmod +x /usr/local/bin/maintainability-sensors

      # 2. Run the delta scan (only checks changed lines against main)
      - name: Run Maintainability Scan (Delta)
        run: maintainability-sensors check-diff origin/main
```

---

### Gradual Adoption with Severity Configuration

For legacy codebases, add a `.maintainability-sensors.yml` instead of blocking immediately:

```yaml
version: "1"
check-diff:
  default-severity: warn
  rules:
    - name: ArgumentCount
      severity: error
```

Then CI stays green while the team observes patterns:
```yaml
- run: maintainability-sensors check-diff origin/main
```

You can also override severity directly from the CLI without a config file:
```bash
--default-severity warn
--severity Complexity:warn
```

---

## 3. Advanced: Inline PR Review Comments (`--github-pr`)

To have the CLI **directly review your active Pull Requests** (posting inline comments on the exact lines of code that exceed complexity limits), enable the `--github-pr` flag and provide the `GITHUB_TOKEN` secret. 

```yaml
name: Quality Gate & PR Review

on:
  pull_request:
    types: [opened, synchronize, reopened]

jobs:
  pr-maintainability-review:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout Code
        uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Install Maintainability Sensors
        run: |
          curl -sSfL -o /usr/local/bin/maintainability-sensors https://github.com/syntheticscale/maintainability-sensors/releases/latest/download/maintainability-sensors-linux
          chmod +x /usr/local/bin/maintainability-sensors

      # 3. Scan the diff and write directly back as a PR Issue comment
      - name: Post PR Scorecard Review
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          maintainability-sensors check-diff origin/main --github-pr
```

---

## 🔒 4. Required Permissions

For the `--github-pr` commenting feature to succeed, your GitHub Actions token must have permission to write comments to pull requests. Ensure your job has these permissions configured:

```yaml
permissions:
  pull-requests: write  # Required for posting comments
  contents: read        # Required for checkout
```
