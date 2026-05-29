# Implementation Plan: Sprint Label Aggregation Report

**Branch**: `001-sprint-label-report` | **Date**: 2026-05-29 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/001-sprint-label-report/spec.md`

## Summary

Add a new CLI command (`get-sprint-label-report`) that fetches all issues from a named
Jira sprint, groups them by an ordered list of configured labels, and outputs a Word
document (or console table) showing label-level aggregates (count + total SP) with an
optional per-issue breakdown. Issues that carry none of the configured labels are listed
in a separate unlabeled issues table. The report is organized at the top level by Jira
component (team), consistent with the existing month-issues report.

## Technical Context

**Language/Version**: Go 1.24.0

**Primary Dependencies**:
- `github.com/andygrunwald/go-jira` — Jira API client (sprint issue fetching)
- `github.com/carmel/gooxml` — Word document generation
- `github.com/joho/godotenv` — `.env` loading

**Storage**: N/A — generates `.docx` files; no database

**Testing**: `go test ./...` via `make test`

**Target Platform**: Linux/macOS CLI binary

**Project Type**: CLI tool (standalone binary, same pattern as existing commands)

**Performance Goals**: Generate report for a sprint of up to 200 issues in under 30 seconds

**Constraints**: No new external services; reuse existing Jira auth and Word formatting
patterns. Must not break existing binaries.

**Scale/Scope**: Single sprint per invocation, ~50–200 issues typical

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. CLI-First Design | ✅ Pass | New binary in `cmd/get-sprint-label-report/`; `-format` and `-debug` flags exposed |
| II. Environment-Driven Config | ✅ Pass | Labels via `REPORT_LABELS` env var; all other config follows existing `.env` pattern |
| III. Package Separation | ✅ Pass | Aggregation logic in new `internal/labelreport/` (no Jira/Word imports); `cmd/` only wires |
| IV. Consistent Document Output | ✅ Pass | Reuses `word.NewTable`, `AddHeaderRow`, `AddDataRow`; same font/margin/colour standard |
| V. Observability | ✅ Pass | `-debug` flag prints to stdout; error messages include sprint name, component, label context |

No violations. Complexity Tracking table not required.

## Project Structure

### Documentation (this feature)

```text
specs/001-sprint-label-report/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/
│   └── cli.md           # CLI flags, env vars, output schema
└── tasks.md             # Phase 2 output (/speckit-tasks command)
```

### Source Code (repository root)

```text
cmd/
└── get-sprint-label-report/
    └── main.go                          # Entry point: flags, wiring, output routing

internal/
├── config/
│   └── config.go                        # Add ReportLabels []string field
├── jiraservice/
│   └── jira.go                          # Add Labels []string and Components []string
│                                        # to Issue struct; populate from Fields.Labels
│                                        # and Fields.Components in LoadIssuesFromSprint
└── labelreport/                         # NEW package — pure aggregation, no Jira/Word imports
    └── aggregator.go                    # LabelGroup, ComponentReport, Aggregate()
```

**Structure Decision**: Single project (Option 1), extending the existing `internal/`
hierarchy. The new `internal/labelreport` package keeps label aggregation logic isolated
and independently testable. No new top-level directories are introduced.

## Complexity Tracking

> No constitution violations — this section is left intentionally blank.
