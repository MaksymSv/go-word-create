# Quickstart: Label Report Percentage Columns

**Feature**: Label Report Percentage Columns
**Binary**: `bin/get-sprint-label-report` (unchanged binary name)
**Date**: 2026-06-02

## Prerequisites

Same as Feature 001 — no new configuration needed.

## 1. Build

```bash
make build-sprint-label-report
# or directly:
go build -o bin/get-sprint-label-report ./cmd/get-sprint-label-report
```

## 2. Generate Short-Format Report with Percentages

```bash
./bin/get-sprint-label-report -sprint="Sprint 42"
```

The output `.docx` now contains `Label | Count | Count,% | Total SP | Total SP,%` columns
plus a "Total" row at the bottom of each component's label table.

## 3. Generate Full-Format Report with Percentages

```bash
./bin/get-sprint-label-report -sprint="Sprint 42" -format=full -output="sprint-42-full.docx"
```

The full-format table now has 8 columns:
`Label | Count | Count,% | Total SP | Total SP,% | Key | Summary | SP`

## 4. Preview in Console

```bash
./bin/get-sprint-label-report -sprint="Sprint 42" -debug
./bin/get-sprint-label-report -sprint="Sprint 42" -format=full -debug
```

## Validation Checklist

- [ ] `make build-sprint-label-report` completes without errors
- [ ] Short-format: table header shows `Label | Count | Count,% | Total SP | Total SP,%`
- [ ] Short-format: each label row shows correct percentage values
- [ ] Short-format: "Total" row appears at bottom with summed Count,% and Total SP,%
- [ ] Short-format: "Total" row has empty Count and Total SP cells
- [ ] Full-format: table header shows 8 columns including the two % columns
- [ ] Full-format: percentage values repeat on every issue row within a label group
- [ ] Full-format: "Total" row appears at bottom of each component table
- [ ] `-debug` mode shows same percentage columns and "Total" row as Word doc
- [ ] Sprint with 0 total SP shows `0%` in Total SP,% column (no crash)
- [ ] `make test` passes with no regressions
