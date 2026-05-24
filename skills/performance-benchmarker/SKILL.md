---
name: performance-benchmarker
description: Empirically measures performance and non-functional requirements (NFRs) using microbenchmarks. Use when refactoring for performance, optimizing bottlenecks, or evaluating NFRs after major changes.
---

# Performance Benchmarker

This skill provides a standardized workflow for measuring performance and non-functional requirements (NFRs).

## Core Mandate
Never refactor for performance without establishing an empirical baseline first.

## Workflow

1.  **Write Microbenchmark:**
    Create a microbenchmark to isolate the specific operation you intend to measure. This should typically be done in a `*_bench_test.go` file (e.g., if optimizing `parser.go`, add benchmarks to `parser_bench_test.go`).
2.  **Establish Baseline:**
    Execute the benchmark to measure the current performance characteristics before making any changes.
    Use the following command to gather both execution time (`ns/op`) and memory allocation (`B/op`, `allocs/op`) metrics:
    ```bash
    go test -bench . -benchmem
    ```
3.  **Optimize:**
    Perform the intended refactoring or optimization.
4.  **Measure and Compare:**
    Run the benchmark command again and compare the new results against the established baseline to empirically validate the optimization.
5.  **Document Findings:**
    Include the before/after benchmark results in the commit message or PR description to justify the changes.
