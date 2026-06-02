# Research: Label Report Percentage Columns

**Feature**: Label Report Percentage Columns
**Branch**: `002-label-report-percentages`
**Date**: 2026-06-02

## Percentage Computation Location

**Decision**: Compute `CountPct` and `TotalSPPct` inside `Aggregate()` in
`internal/labelreport/aggregator.go`.

**Rationale**: Per the Package Separation principle, business logic belongs in
`internal/`. The denominator values (total component issue count and total component SP)
are naturally available during aggregation — the component's full issue slice is already
iterated. Computing percentages at render time would require the `cmd/` layer to
re-implement business logic or carry additional state, which violates the principle.

**Alternatives considered**:
- Computing percentages in the render functions in `cmd/main.go` — violates Package
  Separation; `cmd/` should only wire and format.
- Adding a separate `ComputePercentages()` function called after `Aggregate()` — adds
  unnecessary indirection; the computation is simple arithmetic that fits naturally within
  the existing aggregation loop.

---

## Percentage Denominator Scope

**Decision**: Use per-component totals as the denominator (total issues and total SP of
all issues in the component, labeled + unlabeled combined).

**Rationale**: FR-003 is explicit: "Percentage denominators MUST be the total count and
total SP of all issues in the component (labeled + unlabeled combined)." This is also
consistent with the existing top-level grouping by component — each `ComponentReport` is
already an isolated view of one component's sprint work.

**Alternatives considered**:
- Sprint-level totals as denominator — would make percentages less meaningful within each
  component view; also contradicts FR-003.

---

## Percentage Formatting

**Decision**: Format percentages to one decimal place; omit the trailing `.0` for whole
numbers. Use a `formatPct(v float64) string` helper in `cmd/main.go`.

**Rationale**: FR-007 requires one decimal place and no trailing zero for whole numbers.
In Go, `strconv.FormatFloat(v, 'f', 1, 64)` always produces one decimal place (e.g.,
`40.0`). To satisfy "no trailing zero for whole numbers", check if `math.Trunc(v) == v`
and use `strconv.Itoa(int(v))` in that case; otherwise use `FormatFloat`. Append `%`
suffix in both cases.

**Example output**:
- `40.0` → `"40%"`
- `33.333...` → `"33.3%"`
- `0.0` → `"0%"`
- `100.0` → `"100%"`

**Alternatives considered**:
- Using `fmt.Sprintf("%.1f%%", v)` always — produces `"40.0%"` which violates FR-007.
- Using a third-party formatting library — unnecessary for this simple requirement.

---

## Division-by-Zero Protection

**Decision**: When the component's total issue count is 0, set `CountPct = 0` for all
groups. When the component's total SP is 0, set `TotalSPPct = 0` for all groups.

**Rationale**: FR-005 and FR-006 require `0%` display rather than a runtime panic.
In Go, a float64 division by zero does not panic (it produces `+Inf`), but displaying
`+Inf%` is incorrect. An explicit zero-check before the division is the safest and
clearest approach.

**Alternatives considered**:
- Catching `+Inf` after division — fragile; explicitly checking the denominator is more
  readable and self-documenting.

---

## "Total" Row Rendering

**Decision**: Render the "Total" row entirely within the render functions in `cmd/main.go`
rather than storing it in the data model.

**Rationale**: The "Total" row is a presentational artifact — the sum of the `CountPct`
and `TotalSPPct` values for a component. It is not a first-class entity in the data model.
Render functions already iterate `LabelGroups` to produce rows; they can accumulate
`CountPct` and `TotalSPPct` totals during the loop and emit one final row afterwards.
Storing the total in `ComponentReport` would pollute the data model with presentation
concerns.

**Alternatives considered**:
- Adding a `TotalCountPct` and `TotalSPPct` field to `ComponentReport` — leaks
  presentation state into the data model.

---

## LabelGroup Struct Extension

**Decision**: Add two new fields to `LabelGroup` in `internal/labelreport/aggregator.go`:

```go
CountPct   float64  // group count / component total count * 100
TotalSPPct float64  // group TotalSP / component total SP * 100
```

**Rationale**: These fields mirror the existing `Count` and `TotalSP` fields. Adding them
to the struct keeps all label-group data co-located and makes the data available to all
renderers without passing extra arguments.

**Impact on implementation**: `Aggregate()` computes component-level totals first (a
single pre-pass over `compIssues[compName]`), then fills `CountPct` and `TotalSPPct`
alongside `Count` and `TotalSP` in the existing group-computation loop.

---

## All NEEDS CLARIFICATION items resolved

No NEEDS CLARIFICATION markers were present in the spec. All design decisions are derived
from the spec requirements and existing codebase patterns.
