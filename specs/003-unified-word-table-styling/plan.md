# Implementation Plan: Unified Word Table Styling

**Branch**: `003-unified-word-table-styling` | **Date**: 2026-06-05 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/003-unified-word-table-styling/spec.md`

## Summary

Align all generated Word documents with the reference EPAM delivery report by updating
`internal/word` to faithfully reproduce the reference table design (header color, font,
borders, margins, column widths) and section heading style, adding page breaks between
sections, and making column header labels configurable via `TableConfig`.

The implementation touches three existing files (`internal/word/table.go`,
`internal/word/doc.go`, `internal/word/config.go`), updates three `cmd/` callers, and
amends `Constitution Principle IV` to record the new formatting standard.

## Technical Context

**Language/Version**: Go 1.24.0

**Primary Dependencies**:
- `github.com/carmel/gooxml v0.0.0-20220216072414-40ff56130850` — Word document generation
- `github.com/carmel/gooxml/color`, `/measurement`, `/schema/soo/wml` — styling primitives

**Storage**: N/A — file output only

**Testing**: `go test ./...` via `make test`; manual visual inspection of output `.docx`

**Target Platform**: macOS / Linux CLI

**Project Type**: CLI tool

**Performance Goals**: N/A — document generation is batch, not latency-sensitive

**Constraints**: Must not introduce a second OOXML library; no gooxml version change

**Scale/Scope**: Affects all three `cmd/` document-generating binaries

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. CLI-First Design | ✅ Pass | No changes to CLI surface |
| II. Environment-Driven Configuration | ✅ Pass | No new env vars |
| III. Package Separation | ✅ Pass | All changes stay in `internal/word` and `cmd/` callers |
| **IV. Consistent Document Output** | ⚠️ **Violation** | Feature intentionally updates the formatting standard — see Complexity Tracking |
| V. Observability | ✅ Pass | No logging changes needed |

**Constitution amendment required**: Principle IV must be updated in the same PR to record
the new canonical values (`#0070C0`, top/bottom margins only, auto-color borders).

## Project Structure

### Documentation (this feature)

```text
specs/003-unified-word-table-styling/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
└── tasks.md             # Phase 2 output (/speckit-tasks)
```

### Source Code (repository root)

```text
internal/word/
├── config.go            # TableConfig — updated (new fields + DefaultConfig values)
├── table.go             # Table — updated (accept config, apply reference design)
└── doc.go               # Doc — updated (page breaks + heading style)

cmd/get-month-issues-from-jira/main.go   # Updated (pass TableConfig with headers)
cmd/get-sprint-issues-from-jira/main.go  # Updated (pass TableConfig with headers)
cmd/get-sprint-label-report/main.go      # Updated (pass TableConfig with headers)

.specify/memory/constitution.md          # Updated (Principle IV new values)
```

**Structure Decision**: Single-project layout (Option 1). All changes are inside the
existing source tree; no new packages or directories needed.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| Principle IV formatting standard | Feature is explicitly about matching the EPAM reference document (`EPAM Delivery Report - 2026-05.docx`). The current standard was set before the reference document was analyzed. | Keeping `#365F91` and 0.2 cm margins would contradict the stated goal of "take table configuration from the reference document". The constitution must be updated to stay authoritative. |
