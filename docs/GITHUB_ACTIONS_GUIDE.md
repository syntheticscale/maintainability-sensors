# GitHub Actions CI Integration Guide 🐙

This guide details how to integrate `maintainability-sensors` directly into your GitHub Actions CI pipelines. This automates the PR review gate and outputs an executive scorecard directly onto your Pull Requests.

---

## 1. Quick Start: Standard CI Step Summary

You can run `maintainability-sensors` on every commit. If it runs in a GitHub Actions runner, the CLI will automatically output a beautiful, rich Markdown scorecard directly into the GitHub Actions Job Summary (`GITHUB_STEP_SUMMARY`).

Add this step to your existing `.github/workflows/ci.yml` pipeline:

```yaml
name: Continuous Integration

on: [push, pull_request]

jobs:
  validate-quality:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout Code
        uses: actions/checkout@v4

      # 1. Download and install the Maintainability Sensors binary
      - name: Install Maintainability Sensors
        run: |
          curl -sSfL -o /usr/local/bin/maintainability-sensors https://github.com/syntheticscale/maintainability-sensors/releases/latest/download/maintainability-sensors-linux
          chmod +x /usr/local/bin/maintainability-sensors

      # 2. Run the scan (triggers orchestrated linter and writes summaries)
      - name: Run Maintainability Scan
        run: maintainability-sensors run .
```

---

## 2. Advanced: Inline PR Review Comments (`--github-pr`)

To have the CLI **directly review your active Pull Requests** (posting inline comments on the exact lines of code that exceed complexity limits), enable the `--github-pr` flag and provide the `GITHUB_TOKEN` secret. The CLI natively uses the GitHub Pull Request Review API to create actionable, localized feedback.

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

      - name: Install Maintainability Sensors
        run: |
          curl -sSfL -o /usr/local/bin/maintainability-sensors https://github.com/syntheticscale/maintainability-sensors/releases/latest/download/maintainability-sensors-linux
          chmod +x /usr/local/bin/maintainability-sensors

      # 3. Scan and write directly back as a PR Issue comment
      - name: Post PR Scorecard Review
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          maintainability-sensors run . --github-pr
```

---

## 🔒 3. Required Permissions

For the `--github-pr` commenting feature to succeed, your GitHub Actions token must have permission to write comments to pull requests. Ensure your job has these permissions configured:

```yaml
permissions:
  pull-requests: write  # Required for posting comments
  contents: read        # Required for checkout
```
