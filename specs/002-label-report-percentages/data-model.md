# Data Model: Label Report Percentage Columns

**Feature**: Label Report Percentage Columns
**Branch**: `002-label-report-percentages`
**Date**: 2026-06-02

## Modified Entities

### LabelGroup (extended)

Lives in `internal/labelreport/aggregator.go`.

```
LabelGroup
├── LabelName   string    — configured label string
├── Issues      []Issue   — sprint issues carrying this label (non-exclusive)
├── Count       int       — len(Issues)
├── TotalSP     float64   — sum of StoryPoints across Issues
├── CountPct    float64   — Count / componentTotalCount * 100; 0 when denominator is 0
└── TotalSPPct  float64   — TotalSP / componentTotalSP * 100; 0 when denominator is 0
```

**New fields**: `CountPct` and `TotalSPPct`. Computed inside `Aggregate()` after the
component-level totals are known.

### ComponentReport (unchanged)

Lives in `internal/labelreport/aggregator.go`.

```
ComponentReport
├── ComponentName    string        — Jira component name; "No Component" if none
├── LabelGroups      []LabelGroup  — one entry per configured label, in order
└── UnlabeledIssues  []Issue       — issues with no configured label
```

No changes to this struct. The component-level totals (used as denominators) are
computed transiently inside `Aggregate()` and are not stored in the struct — they are
a presentational concern satisfied by summing `Count` + `len(UnlabeledIssues)` and
`TotalSP` + unlabeled SP if needed.

> Note: The render layer accumulates percentage totals for the "Total" row by summing
> `g.CountPct` and `g.TotalSPPct` over all label groups during iteration — this is a
> rendering detail, not a data model addition.

## Computation Rules

| Field | Formula | Zero-guard |
|-------|---------|------------|
| `CountPct` | `float64(Count) / float64(componentTotalCount) * 100` | `0.0` when `componentTotalCount == 0` |
| `TotalSPPct` | `TotalSP / componentTotalSP * 100` | `0.0` when `componentTotalSP == 0.0` |

`componentTotalCount` = total number of issues in the component (labeled + unlabeled).
`componentTotalSP` = sum of `StoryPoints` across all issues in the component.

## Formatting Contract

Used in `cmd/get-sprint-label-report/main.go` via `formatPct(v float64) string`:

| Value | Formatted |
|-------|-----------|
| `0.0` | `"0%"` |
| `40.0` | `"40%"` |
| `33.333...` | `"33.3%"` |
| `100.0` | `"100%"` |
| `120.0` | `"120%"` (non-exclusive groups may exceed 100%) |
