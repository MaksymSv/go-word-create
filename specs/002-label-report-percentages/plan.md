# Implementation Plan: Label Report Percentage Columns

**Branch**: `002-label-report-percentages` | **Date**: 2026-06-02 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/002-label-report-percentages/spec.md`

## Summary

Extend the existing `get-sprint-label-report` command to show `Count,%` and `Total SP,%`
columns in both short-format and full-format label aggregation tables, plus a "Total" row
at the bottom of each component table that sums those two percentage columns only
(Count and Total SP cells in the Total row are left empty). Percentages are computed
against the total issues and total SP of all issues within the same component (labeled +
unlabeled combined). The change touches two files: `internal/labelreport/aggregator.go`
(adds percentage fields and computation) and `cmd/get-sprint-label-report/main.go`
(updates all four renderers).

## Technical Context

**Language/Version**: Go 1.24.0

**Primary Dependencies**:
- `github.com/carmel/gooxml` — Word document generation (reused, no new import)
- `github.com/andygrunwald/go-jira` — Jira API client (unchanged)
- `github.com/joho/godotenv` — `.env` loading (unchanged)

**Storage**: N/A — no new storage; percentage values are computed in-process

**Testing**: `go test ./...` via `make test`

**Target Platform**: Linux/macOS CLI binary (no change)

**Project Type**: CLI tool — extension to existing binary `cmd/get-sprint-label-report`

**Performance Goals**: Negligible impact; percentages are simple arithmetic on already-
loaded data, well within the existing 30-second performance target.

**Constraints**: No new flags, no new env vars, no new dependencies. Must not break the
existing `001-sprint-label-report` column contract for the Count and Total SP columns.

**Scale/Scope**: Touches 2 files; 2 struct fields added; 4 render functions updated.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. CLI-First Design | ✅ Pass | No new flags; extends existing binary output; `-debug` mode updated |
| II. Environment-Driven Config | ✅ Pass | No new configuration; percentages always shown (no opt-in needed) |
| III. Package Separation | ✅ Pass | Percentage computation stays in `internal/labelreport`; `cmd/` only formats |
| IV. Consistent Document Output | ✅ Pass | Reuses `word.NewTable`, `AddHeaderRow`, `AddDataRow` — no new formatting primitives |
| V. Observability | ✅ Pass | Console/debug output updated to include same percentage columns as Word output |

No violations. Complexity Tracking table not required.

## Project Structure

### Documentation (this feature)

```text
specs/002-label-report-percentages/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/
│   └── cli.md           # Updated CLI output schema
└── tasks.md             # Phase 2 output (/speckit-tasks command)
```

### Source Code (repository root)

```text
internal/
└── labelreport/
    └── aggregator.go    # Add CountPct, TotalSPPct to LabelGroup; compute in Aggregate()

cmd/
└── get-sprint-label-report/
    └── main.go          # Update all 4 renderers + add formatPct() helper
```

No new files or directories. No Makefile changes needed (binary already built by
`build-sprint-label-report`).

**Structure Decision**: Single-project extension (same pattern as Feature 001). Percentage
computation lives in `internal/labelreport` to respect Package Separation; formatting
lives in `cmd/`.

## Complexity Tracking

> No constitution violations — this section is left intentionally blank.
